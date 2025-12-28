package main

import (
	"github.com/edgexfoundry/device-sdk-go/v4/pkg/startup"
	"github.com/linjuya-lu/device-lpmp-go/internal/driver"
)

const (
	serviceName string = "device-lpmp"
	Version     string = "HYV1.0"
)

func main() {
	d := driver.LpMpDeviceDriver()
	startup.Bootstrap(serviceName, Version, d)
}
