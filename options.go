package libudev

import (
	"os"

	"github.com/qubesome/libudev/matcher"
)

type Option func(*scanner)

type options struct {
	matcher *matcher.Matcher

	devicesRoot  *os.Root
	udevDataRoot *os.Root
}

// WithDevicesRoot provides a way to set a different os.Root to be used
// as the Devices dir. When not provided, defaults to an os.Root pointing
// to /sys/devices.
func WithDevicesRoot(r *os.Root) Option {
	return func(o *scanner) {
		o.opts.devicesRoot = r
	}
}

// WithUDevDataRoot provides a way to set a different os.Root to be used
// as the udev data dir. When not provided, defaults to an os.Root pointing
// to /run/udev/data.
func WithUDevDataRoot(r *os.Root) Option {
	return func(o *scanner) {
		o.opts.udevDataRoot = r
	}
}
