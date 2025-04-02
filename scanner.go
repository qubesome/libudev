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
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/qubesome/libudev/types"
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

func (s *scanner) readAttrs(path string, device *types.Device) error {
	files, err := fs.ReadDir(s.opts.devicesRoot.FS(), path)
	if err != nil {
		return err
	}

	for _, f := range files {
		if f.IsDir() || f.Name() == "uevent" || f.Name() == "descriptors" {
			continue
		}

		data, err := fs.ReadFile(s.opts.devicesRoot.FS(), filepath.Join(path, f.Name()))
		if err != nil {
			continue
		}

		device.Attrs[f.Name()] = strings.Trim(string(data), "\n\r\t ")
	}

	return nil
}

func (s *scanner) readUeventFile(path string, device *types.Device) error {
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

	var line string
	for buf.Scan() {
		line = buf.Text()
		field := strings.SplitN(line, "=", 2)
		if len(field) != 2 {
			continue
		}

		device.Env[field[0]] = field[1]

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

func (s *scanner) readDevFile(path string) (data string, err error) {
	f, err := s.opts.devicesRoot.Open(path)
	if err != nil {
		return
	}

	defer func() {
		if err := f.Close(); err != nil {
			slog.Debug("cannot close dev file", "error", err)
		}
	}()

	d, err := io.ReadAll(f)
	return strings.Trim(string(d), "\n\r\t "), err
}

func (s *scanner) readUdevInfo(devString string, d *types.Device) error {
	path := fmt.Sprintf("c%s", devString)
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

	var line string
	for buf.Scan() {
		line = buf.Text()
		groups := strings.SplitN(line, ":", 2)
		if len(groups) != 2 {
			continue
		}

		if groups[0] == "I" {
			d.UsecInitialized = groups[1]
			continue
		}

		if groups[0] == "G" {
			d.Tags = append(d.Tags, groups[1])
			continue
		}

		if groups[0] == "E" {
			fields := strings.SplitN(groups[1], "=", 2)
			if len(fields) != 2 {
				continue
			}

			d.Env[fields[0]] = fields[1]
		}
	}

	return nil
}
