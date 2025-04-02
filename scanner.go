// Package libudev implements a native udev library.
//
// Recursively parses devices in `/sys/devices/...` and reads `uevent` files (placed in `Env`).
// Based on the information received from `uevent` files, it tries to enrich the data based
// on the data received from the files that are on the same level as the `uevent` file (they are placed in `Attrs`),
// and also tries to find and read the files `/run/udev/data/...` (placed in `Env` or `Tags`).
//
// After building a list of devices, the library builds a device tree.
package libudev

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/qubesome/libudev/types"
)

const (
	maxDevSize = 128 * 1024 // 128KB
)

// Scanner represents a device scanner.
type scanner struct {
	opts *options
}

// NewScanner creates a new instance of the device scanner.
func NewScanner(opts ...Option) (*scanner, error) {
	s := &scanner{opts: &options{}}
	for _, opt := range opts {
		opt(s)
	}

	if s.opts.devicesRoot == nil {
		// ref: https://www.kernel.org/doc/Documentation/filesystems/sysfs.txt
		r, err := os.OpenRoot("/sys/devices")
		if err != nil {
			return nil, err
		}

		s.opts.devicesRoot = r
	}
	if s.opts.udevDataRoot == nil {
		r, err := os.OpenRoot("/run/udev/data")
		if err != nil {
			return nil, err
		}
		s.opts.udevDataRoot = r
	}

	return s, nil
}

// ScanDevices scans directories for `uevent` files and creates a device tree.
func (s *scanner) ScanDevices() ([]*types.Device, error) {
	devices := []*types.Device{}
	devicesMap := map[string]*types.Device{}

	err := fs.WalkDir(s.opts.devicesRoot.FS(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() || d.Name() != "uevent" {
			return nil
		}

		device := &types.Device{
			Devpath: filepath.Dir(path),
			Env:     map[string]string{},
			Attrs:   map[string]string{},
			Parent:  nil,
		}

		err = s.readAttrs(filepath.Dir(path), device)
		if err != nil {
			return err
		}

		err = s.readUeventFile(path, device)
		if err != nil {
			return nil
		}

		devicesMap[device.Devpath] = device
		return nil
	})
	if err != nil {
		return nil, err
	}

	// make tree
	for _, v := range devicesMap {
		parts := strings.Split(v.Devpath, "/")

		devpath := v.Devpath
		for i := len(parts) - 1; i >= 0; i-- {
			devpath = strings.TrimSuffix(devpath, "/"+parts[i])

			if device, ok := devicesMap[devpath]; ok {
				v.Parent = device
				device.Children = append(device.Children, v)
				break
			}
		}

		devices = append(devices, v)
	}

	return devices, err
}

func (s *scanner) getDevice(path string) (*types.Device, error) {
	attrs, err := s.readAttrs(filepath.Dir(path))
	if err != nil {
		return nil, err
	}

	device := &types.Device{
		Devpath: filepath.Dir(path),
		Env:     map[string]string{},
		Attrs:   attrs,
		Parent:  nil,
	}

	err = s.readUeventFile(path, device)
	if err != nil {
		return nil, err
	}

	return device, nil
}

func (s *scanner) readAttrs(path string) (map[string]string, error) {
	files, err := fs.ReadDir(s.opts.devicesRoot.FS(), path)
	if err != nil {
		return nil, err
	}

	attrs := map[string]string{}
	for _, f := range files {
		if f.IsDir() || f.Name() == "uevent" || f.Name() == "descriptors" {
			continue
		}

		data, err := fs.ReadFile(s.opts.devicesRoot.FS(), filepath.Join(path, f.Name()))
		if err != nil {
			continue
		}

		attrs[f.Name()] = strings.Trim(string(data), "\n\r\t ")
	}

	return attrs, nil
}

func (s *scanner) readUeventFile(path string, device *types.Device) error {
	_, err := s.opts.devicesRoot.Stat(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		return nil
	}

	f, err := s.opts.devicesRoot.Open(path)
	if err != nil {
		return err
	}

	defer func() {
		if err := f.Close(); err != nil {
			slog.Debug("cannot close uevent file", "error", err)
		}
	}()

	buf := bufio.NewScanner(f)
	for buf.Scan() {
		k, v, ok := strings.Cut(buf.Text(), "=")
		if !ok {
			continue
		}

		device.Env[k] = v
	}
	err = buf.Err()
	if err != nil {
		return err
	}

	devPath := filepath.Join(filepath.Dir(path), "dev")
	devString, err := s.readDevFile(devPath)
	if err != nil {
		return err
	}

	err = s.readUdevInfo(devString, device)
	if err != nil {
		return err
	}

	return nil
}

func (s *scanner) readDevFile(path string) (string, error) {
	_, err := s.opts.devicesRoot.Stat(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return "", err
		}

		return "", nil
	}

	f, err := s.opts.devicesRoot.Open(path)
	if err != nil {
		return "", err
	}

	defer func() {
		if err := f.Close(); err != nil {
			slog.Debug("cannot close dev file", "error", err)
		}
	}()

	d, err := io.ReadAll(io.LimitReader(f, maxDevSize))
	return strings.Trim(string(d), "\n\r\t "), err
}

func (s *scanner) readUdevInfo(devString string, d *types.Device) error {
	// The c prefix here defines a character device.
	path := fmt.Sprintf("c%s", devString)
	_, err := s.opts.udevDataRoot.Stat(path)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return err
		}

		return nil
	}

	f, err := s.opts.udevDataRoot.Open(path)
	if err != nil {
		return err
	}

	defer func() {
		if err := f.Close(); err != nil {
			slog.Debug("cannot close udev info file", "error", err)
		}
	}()

	buf := bufio.NewScanner(f)
	for buf.Scan() {
		k, v, ok := strings.Cut(buf.Text(), ":")
		if !ok {
			continue
		}

		if k == "I" {
			d.UsecInitialized = v
			continue
		}

		if k == "G" {
			d.Tags = append(d.Tags, v)
			continue
		}

		if k == "E" {
			ck, cv, ok := strings.Cut(v, "=")
			if !ok {
				continue
			}

			d.Env[ck] = cv
		}
	}

	err = buf.Err()
	if err != nil {
		return err
	}

	return nil
}
