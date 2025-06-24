// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2019-2023 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

// Package driver provides an implementation of a ProtocolDriver interface.
package driver

import (
	"fmt"
	"reflect"
	"sync"

	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces"
	dsModels "github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/linjuya-lu/device-lpmp-go/internal/config"
	"github.com/linjuya-lu/device-lpmp-go/internal/frameparser"
	"github.com/linjuya-lu/device-lpmp-go/internal/serial"
)

type VirtualDriver struct {
	lc             logger.LoggingClient
	asyncCh        chan<- *dsModels.AsyncValues
	virtualDevices sync.Map
	db             *db
	locker         sync.Mutex
	sdk            interfaces.DeviceServiceSDK
}

var once sync.Once
var driver *VirtualDriver

func NewVirtualDeviceDriver() interfaces.ProtocolDriver {
	once.Do(func() {
		driver = new(VirtualDriver)
	})
	return driver
}

func (d *VirtualDriver) retrieveVirtualDevice(deviceName string) (vdv *virtualDevice, err error) {
	vd, _ := d.virtualDevices.LoadOrStore(deviceName, newVirtualDevice())
	var ok bool
	if vdv, ok = vd.(*virtualDevice); !ok {
		d.lc.Errorf("retrieve virtualDevice by name: %s, the returned value has to be a reference of "+
			"virtualDevice struct, but got: %s", deviceName, reflect.TypeOf(vd))
	}
	return vdv, err
}

func (d *VirtualDriver) Initialize(sdk interfaces.DeviceServiceSDK) error {
	d.sdk = sdk
	d.lc = sdk.LoggingClient()
	d.asyncCh = sdk.AsyncValuesChannel()

	d.db = getDb()

	if err := initVirtualResourceTable(d); err != nil {
		return fmt.Errorf("failed to initial virtual resource table: %v", err)
	}

	return nil
}

func (d *VirtualDriver) Start() error {
	// —— 0. 配置文件和串口参数（可以硬编码，也可从 d.config 读取）
	const (
		devicesYAML = "cmd/res/devices/devices.yaml"
		profilesDir = "cmd/res/profiles"
	)
	// portName := d.config.PortName   // e.g. "/dev/ttyUSB0"
	// baudRate := d.config.BaudRate   // e.g. 115200
	portName := "/dev/ttyUSB0"
	baudRate := 115200
	// —— 1. 初始化静态资源定义 + 默认初始值
	if err := config.InitDeviceResources(devicesYAML, profilesDir); err != nil {
		return fmt.Errorf("初始化设备资源失败: %w", err)
	}

	// —— 2. 为每台设备准备虚拟资源（在 valuesMap 中分配槽位）
	for _, dev := range d.sdk.Devices() {
		if err := prepareVirtualResources(d, dev.Name); err != nil {
			return fmt.Errorf("prepareVirtualResources(%s) 失败: %w", dev.Name, err)
		}
	}

	// —— 3. 打开串口
	serialPort, err := serial.Open(portName, baudRate)
	if err != nil {
		return fmt.Errorf("打开串口 %s 失败: %w", portName, err)
	}

	// —— 4. 启动 AT+DRX 监听
	frameCh := make(chan []byte, 100)
	serial.StartDRXListener(serialPort, frameCh)

	// —— 5. 启动解析协程：从 frameCh 取原始帧，交给切帧 + 业务解析
	go func() {
		var backlog []byte
		for raw := range frameCh {
			backlog = append(backlog, raw...)
			// 不断尝试切帧
			for {
				frame, rest, err := frameparser.Parse(backlog)
				if err != nil {
					// 协议错误：丢弃整个缓存
					backlog = rest
					break
				}
				if frame == nil {
					// 半帧不足，保留缓存
					backlog = rest
					break
				}
				// 业务 TLV／浮点／整型解析，返回 map[resourceName]value
				values := frameparser.ParseBusiness(frame)
				for name, val := range values {
					config.SetDeviceValue(frameparser.DeviceName(frame), name, val)
				}
				backlog = rest
			}
		}
	}()

	return nil
}

func (d *VirtualDriver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []dsModels.CommandRequest) (res []*dsModels.CommandValue, err error) {
	d.locker.Lock()
	defer driver.locker.Unlock()

	vd, err := d.retrieveVirtualDevice(deviceName)
	if err != nil {
		return nil, err
	}

	res = make([]*dsModels.CommandValue, len(reqs))

	for i, req := range reqs {
		if dr, ok := d.sdk.DeviceResource(deviceName, req.DeviceResourceName); ok {
			if v, err := vd.read(deviceName, req.DeviceResourceName, dr.Properties.ValueType, dr.Properties.Minimum, dr.Properties.Maximum, d.db); err != nil {
				return nil, err
			} else {
				res[i] = v
			}
		} else {
			return nil, fmt.Errorf("cannot find device resource %s from device %s in cache", req.DeviceResourceName, deviceName)
		}
	}

	return res, nil
}

func (d *VirtualDriver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []dsModels.CommandRequest,
	params []*dsModels.CommandValue) error {
	d.locker.Lock()
	defer driver.locker.Unlock()

	vd, err := d.retrieveVirtualDevice(deviceName)
	if err != nil {
		return err
	}

	for _, param := range params {
		if err := vd.write(param, deviceName, d.db); err != nil {
			return err
		}
	}
	return nil
}

func (d *VirtualDriver) Stop(force bool) error {
	d.lc.Info("VirtualDriver.Stop: device-virtual driver is stopping...")
	if err := d.db.closeDb(); err != nil {
		d.lc.Errorf("ql DB closed ungracefully, error: %v", err)
	}
	return nil
}

func (d *VirtualDriver) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.lc.Debugf("a new Device is added: %s", deviceName)
	err := prepareVirtualResources(d, deviceName)
	return err
}

func (d *VirtualDriver) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.lc.Debugf("Device %s is updated", deviceName)
	err := prepareVirtualResources(d, deviceName)
	return err
}

func (d *VirtualDriver) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	d.lc.Debugf("Device %s is removed", deviceName)
	err := deleteVirtualResources(d, deviceName)
	return err
}

func initVirtualResourceTable(driver *VirtualDriver) error {
	if err := driver.db.init(); err != nil {
		driver.lc.Errorf("failed to init storage: %v", err)
		return err
	}

	return nil
}

func prepareVirtualResources(driver *VirtualDriver, deviceName string) error {
	driver.locker.Lock()
	defer driver.locker.Unlock()

	device, err := driver.sdk.GetDeviceByName(deviceName)
	if err != nil {
		return err
	}
	if device.ProfileName == "" {
		return nil
	}
	profile, err := driver.sdk.GetProfileByName(device.ProfileName)
	if err != nil {
		return err
	}

	for _, dr := range profile.DeviceResources {
		if dr.Properties.ReadWrite == common.ReadWrite_R || dr.Properties.ReadWrite == common.ReadWrite_RW {
			/*
				d.Name <-> VIRTUAL_RESOURCE.deviceName
				dr.Name <-> VIRTUAL_RESOURCE.CommandName, VIRTUAL_RESOURCE.ResourceName
				ro.DeviceResource <-> VIRTUAL_RESOURCE.DeviceResourceName
				dr.Properties.Value.Type <-> VIRTUAL_RESOURCE.DataType
				dr.Properties.Value.DefaultValue <-> VIRTUAL_RESOURCE.Value
			*/
			if dr.Properties.ValueType == common.ValueTypeBinary {
				continue
			}
			if err := driver.db.addResource(device.Name, dr.Name, dr.Name, true, dr.Properties.ValueType,
				dr.Properties.DefaultValue); err != nil {
				driver.lc.Errorf("failed to add resource: %v", err)
				return err
			}
		}
		// TODO another for loop to update the ENABLE_RANDOMIZATION field of virtual resource by device resource
		//  "EnableRandomization_{ResourceName}"
	}

	return nil
}

func deleteVirtualResources(driver *VirtualDriver, deviceName string) error {
	driver.locker.Lock()
	defer driver.locker.Unlock()

	if err := driver.db.deleteResources(deviceName); err != nil {
		driver.lc.Errorf("failed to delete virtual resources of device %s: %v", deviceName, err)
		return err
	} else {
		return nil
	}
}

func (d *VirtualDriver) Discover() error {
	return fmt.Errorf("driver's Discover function isn't implemented")
}

func (d *VirtualDriver) ValidateDevice(device models.Device) error {
	d.lc.Debug("Driver's ValidateDevice function isn't implemented")
	return nil
}
