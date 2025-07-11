// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2018-2022 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

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
