# libudev
Golang native implementation Udev library

[![GoDoc](https://godoc.org/github.com/qubesome/libudev?status.svg)](https://godoc.org/github.com/qubesome/libudev)
[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/qubesome/libudev/badge)](https://scorecard.dev/viewer/?uri=github.com/qubesome/libudev)

Installation
------------
    go get github.com/qubesome/libudev

Usage
-----

### Scanning devices
```go
sc := libudev.NewScanner()
err, devices := s.ScanDevices()
```

### Filtering devices
```go
m := matcher.NewMatcher()
m.SetStrategy(matcher.StrategyOr)
m.AddRule(matcher.NewRuleAttr("dev", "189:133"))
m.AddRule(matcher.NewRuleEnv("DEVNAME", "usb/lp0"))

filteredDevices := m.Match(devices)
```

### Getting parent device
```go
if device.Parent != nil {
    fmt.Printf("%s\n", device.Parent.Devpath)
}
```

### Getting children devices
```go
fmt.Printf("Count children devices %d\n", len(device.Children))
```

Features
--------
* 100% Native code
* Without external dependencies
* Code is covered by tests

Documentation
-------------

You can read package documentation [here](http:godoc.org/github.com/qubesome/libudev) or read tests.

Testing
-------
Unit-tests:
```bash
make test
```

Contributing
------------
* Fork
* Write code
* Run unit test: `make test`
* Run format checks: `make verify`
* Commit changes
* Create pull-request
