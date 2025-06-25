// -*- Mode: Go; indent-tabs-mode: t -*-
//
// Copyright (C) 2019-2023 IOTech Ltd
//
// SPDX-License-Identifier: Apache-2.0

// Package driver provides an implementation of a ProtocolDriver interface.
package driver

import (
	"fmt"
	"sync"
	"time"

	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces"
	dsModels "github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/models"
	"github.com/linjuya-lu/device-lpmp-go/internal/config"
	"github.com/linjuya-lu/device-lpmp-go/internal/frameparser"
	"github.com/linjuya-lu/device-lpmp-go/internal/serial"
)

type LpMpDriver struct {
	lc      logger.LoggingClient
	asyncCh chan<- *dsModels.AsyncValues
	locker  sync.Mutex
	sdk     interfaces.DeviceServiceSDK
}

var once sync.Once
var driver *LpMpDriver

func NewVirtualDeviceDriver() interfaces.ProtocolDriver {
	once.Do(func() {
		driver = new(LpMpDriver)
	})
	return driver
}

func (d *LpMpDriver) Initialize(sdk interfaces.DeviceServiceSDK) error {
	d.sdk = sdk
	d.lc = sdk.LoggingClient()
	d.asyncCh = sdk.AsyncValuesChannel()

	return nil
}

func (d *LpMpDriver) Start() error {
	// —— 0. 配置文件和串口参数（可以硬编码，也可从 d.config 读取）
	const (
		devicesYAML = "../cmd/res/devices/devices.yaml"
		profilesDir = "../cmd/res/profiles"
	)
	portName := "/dev/ttyUSB0"
	baudRate := 115200

	// —— 1. 初始化静态资源定义 + 默认初始值
	if err := config.InitDeviceResources(devicesYAML, profilesDir); err != nil {
		return fmt.Errorf("初始化设备资源失败: %w", err)
	}

	// —— 2. 打开串口
	serialPort, err := serial.Open(portName, baudRate)
	if err != nil {
		return fmt.Errorf("打开串口 %s 失败: %w", portName, err)
	}

	// —— 3. 启动 AT+DRX 监听，把解析到的二进制帧推到 frameCh
	frameCh := make(chan []byte, 100)
	serial.StartDRXListener(serialPort, frameCh)

	// —— 4. 解析协程
	frameparser.StartParser(frameCh)

	d.lc.Infof("串口监听和解析已启动")
	return nil
}

func (d *LpMpDriver) HandleReadCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []dsModels.CommandRequest) (res []*dsModels.CommandValue, err error) {
	d.locker.Lock()
	defer d.locker.Unlock()

	d.lc.Infof("HandleReadCommands 调用: 设备=%s, 请求资源数=%d", deviceName, len(reqs))

	// 从 config 中取出当前所有资源的值快照
	values, ok := config.GetDeviceValues(deviceName)
	if !ok {
		d.lc.Errorf("设备 %s 未找到或无可用值", deviceName)
		return nil, fmt.Errorf("设备 %s 未找到或无可用值", deviceName)
	}

	results := make([]*dsModels.CommandValue, 0, len(reqs))
	for _, req := range reqs {
		resName := req.DeviceResourceName
		val, exists := values[resName]
		if !exists {
			d.lc.Errorf("设备 %s 上未找到资源 %s 的值", deviceName, resName)
			return nil, fmt.Errorf("设备 %s 上未找到资源 %s 的值", deviceName, resName)
		}

		// 构造 CommandValue
		cv := &dsModels.CommandValue{
			DeviceResourceName: resName,
			Type:               req.Type,
			Value:              val,
			Origin:             time.Now().UnixNano(),
			Tags:               map[string]string{},
		}
		results = append(results, cv)
		d.lc.Infof("读取值: %s.%s = %v", deviceName, resName, val)
	}

	return results, nil
}

func (d *LpMpDriver) HandleWriteCommands(deviceName string, protocols map[string]models.ProtocolProperties, reqs []dsModels.CommandRequest,
	params []*dsModels.CommandValue) error {
	d.locker.Lock()
	defer d.locker.Unlock()

	d.lc.Infof("HandleWriteCommands 调用: 设备=%s, 写入请求数=%d", deviceName, len(reqs))

	// 请求数与参数数必须一致
	if len(reqs) != len(params) {
		d.lc.Errorf("请求数与参数数不匹配: %d vs %d", len(reqs), len(params))
		return fmt.Errorf("请求数与参数数不匹配")
	}

	// 遍历每个请求，取出对应的值并写入 config
	for i, req := range reqs {
		resName := req.DeviceResourceName
		cv := params[i]

		// 直接使用 CommandValue.Value（已经是合适的 Go 类型）
		value := cv.Value

		// 并发安全地写入运行时值表
		config.SetDeviceValue(deviceName, resName, value)
		d.lc.Infof("写入值: %s.%s = %v", deviceName, resName, value)
	}

	return nil
}

func (d *LpMpDriver) Stop(force bool) error {
	d.lc.Info("VirtualDriver.Stop: device-virtual driver is stopping...")

	return nil
}

func (d *LpMpDriver) AddDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.lc.Debugf("a new Device is added: %s", deviceName)
	err := prepareVirtualResources(d, deviceName)
	return err
}

func (d *LpMpDriver) UpdateDevice(deviceName string, protocols map[string]models.ProtocolProperties, adminState models.AdminState) error {
	d.lc.Debugf("Device %s is updated", deviceName)
	err := prepareVirtualResources(d, deviceName)
	return err
}

func (d *LpMpDriver) RemoveDevice(deviceName string, protocols map[string]models.ProtocolProperties) error {
	d.lc.Debugf("Device %s is removed", deviceName)

	return nil
}

func prepareVirtualResources(driver *LpMpDriver, deviceName string) error {
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

		}
		// TODO another for loop to update the ENABLE_RANDOMIZATION field of virtual resource by device resource
		//  "EnableRandomization_{ResourceName}"
	}

	return nil
}

func (d *LpMpDriver) Discover() error {
	return fmt.Errorf("driver's Discover function isn't implemented")
}

func (d *LpMpDriver) ValidateDevice(device models.Device) error {
	d.lc.Debug("Driver's ValidateDevice function isn't implemented")
	return nil
}
