package libudev

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/qubesome/libudev/matcher"
)

func TestNewScanner(t *testing.T) {
	s, _ := NewScanner()
	_, ok := interface{}(s).(*scanner)
	if !ok {
		t.Fatal("Structure does not equal Scanner")
	}
}

func TestScanDevices(t *testing.T) {
	dir := t.TempDir()
	err := unzip("./assets/fixtures/demo_tree.zip", dir)
	if err != nil {
		t.Fatal(err)
	}

	devRoot, err := os.OpenRoot(filepath.Join(dir, "demo_tree/sys/devices"))
	if err != nil {
		t.Fatal(err)
	}

	udevDataRoot, err := os.OpenRoot(filepath.Join(dir, "demo_tree/run/udev/data"))
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewScanner(WithDevicesRoot(devRoot),
		WithUDevDataRoot(udevDataRoot))
	if err != nil {
		t.Fatal("failed to create scanner", err)
	}

	devices, err := s.ScanDevices()
	if err != nil {
		t.Fatal("failed to scan the demo tree", err)
	}

	if len(devices) != 11 {
		t.Fatalf("wanted 11 devices got %d", len(devices))
	}

	m := matcher.NewMatcher()
	m.AddRule(matcher.NewRuleAttr("dev", "189:133"))
	dFiltered := m.Matches(devices)
	if len(dFiltered) != 1 {
		t.Fatal("Not found device by Attr `dev` = `189:133`")
	}

	if len(dFiltered[0].Children) != 1 {
		t.Fatal("Device (`dev` = `189:133`) children count not equal 1")
	}

	if dFiltered[0].Children[0].Env["DEVNAME"] != "usb/lp0" {
		t.Fail()
	}

	if dFiltered[0].Parent == nil {
		t.Fatal("Not found parent device for device (`dev` = `189:133`)")
	}

	if dFiltered[0].Parent.Attrs["idProduct"] != "0024" {
		t.Fail()
	}
}

func TestScanDevicesNotFound(t *testing.T) {
	devRoot, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	udevDataRoot, err := os.OpenRoot(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	s, err := NewScanner(WithDevicesRoot(devRoot),
		WithUDevDataRoot(udevDataRoot))
	if err != nil {
		t.Fatal("failed to create scanner", err)
	}

	devices, err := s.ScanDevices()
	if len(devices) != 0 {
		t.Fatalf("wanted 0 devices but got %d", len(devices))
	}

	if err != nil {
		t.Fatalf("failed to scan empty dirs: %v", err)
	}
}

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := r.Close(); err != nil {
			panic(err)
		}
	}()

	err = os.MkdirAll(dest, 0o700)
	if err != nil {
		return err
	}

	// Closure to address file descriptors issue with all the deferred .Close() methods
	extractAndWriteFile := func(f *zip.File) error {
		rc, err := f.Open()
		if err != nil {
			return err
		}
		defer func() {
			if err := rc.Close(); err != nil {
				panic(err)
			}
		}()

		path := filepath.Join(dest, f.Name)

		if f.FileInfo().IsDir() {
			err = os.MkdirAll(path, f.Mode())
			if err != nil {
				return err
			}
		} else {
			err = os.MkdirAll(filepath.Dir(path), f.Mode())
			if err != nil {
				return err
			}

			f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
			if err != nil {
				return err
			}
			defer func() {
				if err := f.Close(); err != nil {
					panic(err)
				}
			}()

			_, err = io.Copy(f, rc)
			if err != nil {
				return err
			}
		}
		return nil
	}

	for _, f := range r.File {
		err := extractAndWriteFile(f)
		if err != nil {
			return err
		}
	}

	return nil
}

func TestScanDevicesWithMatcher(t *testing.T) {
	dir := t.TempDir()
	err := unzip("./assets/fixtures/demo_tree.zip", dir)
	if err != nil {
		t.Fatal(err)
	}

	devRoot, err := os.OpenRoot(filepath.Join(dir, "demo_tree/sys/devices"))
	if err != nil {
		t.Fatal(err)
	}

	udevDataRoot, err := os.OpenRoot(filepath.Join(dir, "demo_tree/run/udev/data"))
	if err != nil {
		t.Fatal(err)
	}

	m := matcher.NewMatcher()
	m.AddRule(matcher.NewRuleEnv("ID_MODEL_ENC", "USB\\\\x20Optical\\\\x20Mouse"))

	s, err := NewScanner(WithDevicesRoot(devRoot),
		WithUDevDataRoot(udevDataRoot),
		WithMatcher(m))
	if err != nil {
		t.Fatal("failed to create scanner", err)
	}

	devices, err := s.ScanDevices()
	if err != nil {
		t.Fatal("failed to scan the demo tree", err)
	}

	if len(devices) != 3 {
		t.Fatalf("wanted 3 devices got %d", len(devices))
	}

	for _, dev := range devices {
		if dev.Env["ID_MODEL"] != "USB_Optical_Mouse" {
			t.Errorf("want ID_MODEL %s got %v", "USB_Optical_Mouse", dev.Env["ID_MODEL"])
		}
		if dev.VendorID != "046d" {
			t.Errorf("want VendorID %s got %v", "046d", dev.VendorID)
		}
		if dev.ProductID != "c05b" {
			t.Errorf("want ProductID %s got %v", "c05b", dev.ProductID)
		}
	}
}
