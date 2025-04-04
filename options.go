package libudev

import (
	"os"
	"regexp"

	"github.com/qubesome/libudev/matcher"
)

type Option func(*scanner)

type options struct {
	matcher *matcher.Matcher

	pathFilterPattern *regexp.Regexp

	devicesRoot  *os.Root
	udevDataRoot *os.Root
}

// WithPathFilterPattern sets a pattern to filter out device paths that
// are irrelevant to the query. This speeds up the scanning process, as
// the scanner won't deal with files (and devices) that do not match the
// regex.
//
// For example, when querying USB devices, this could be used:
// libudev.WithPathFilterPattern(regexp.MustCompile("(?i)^.*pci0000:00.*usb.*"))
func WithPathFilterPattern(p *regexp.Regexp) Option {
	return func(o *scanner) {
		o.opts.pathFilterPattern = p
	}
}

// WithMatcher sets a matcher to the scanner, so that only devices matching
// the rules are returned.
func WithMatcher(m *matcher.Matcher) Option {
	return func(o *scanner) {
		o.opts.matcher = m
	}
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
