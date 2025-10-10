package driver

import (
	"encoding/hex"
	"fmt"
	"time"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
	"github.com/linjuya-lu/device-lpmp-go/internal/frameparser"
	"github.com/linjuya-lu/device-lpmp-go/internal/serial"
)

// 时间设置封装
func (d *LpMpDriver) handleTimeParameterSet(deviceName string) error {
	// EID
	eidValue, ok := config.GetDeviceValue(deviceName, "eid")
	if !ok {
		err := fmt.Errorf("设备 %s 的 EID 未初始化", deviceName)
		d.lc.Error(err.Error())
		return err
	}
	eidStr := "238A0841D828"
	eidBytes, err := hex.DecodeString(eidStr)
	if err != nil {
		err = fmt.Errorf("EID[%s] 转十六进制失败: %w", eidStr, err)
		d.lc.Error(err.Error())
		return err
	}
	if len(eidBytes) != 6 {
		err = fmt.Errorf("EID 长度不对，期望 6 字节，实际 %d 字节", len(eidBytes))
		d.lc.Error(err.Error())
		return err
	}
	var sensorID [6]byte
	copy(sensorID[:], eidBytes)
	// 复位帧
	loc := time.FixedZone("UTC-0", 0) // UTC
	ts := uint32(time.Now().In(loc).Unix())

	// 发送
	reqFrame, _ := frameparser.BuildTimeParamFrame(sensorID, 1, ts)
	eidStr, _ = eidValue.(string)
	serial.SendFrame(eidStr, reqFrame)
	d.lc.Infof("发送时间设置到设备 %s (EID: %s)", deviceName, eidStr)
	return nil
}

func (d *LpMpDriver) handleResetCommand(deviceName string) error {
	// EID
	eidValue, ok := config.GetDeviceValue(deviceName, "eid")
	if !ok {
		err := fmt.Errorf("设备 %s 的 EID 未初始化", deviceName)
		d.lc.Error(err.Error())
		return err
	}
	eidStr := "238A0841D828"
	// 解码6字节
	eidBytes, err := hex.DecodeString(eidStr)
	if err != nil {
		err = fmt.Errorf("EID[%s] 转十六进制失败: %w", eidStr, err)
		d.lc.Error(err.Error())
		return err
	}
	if len(eidBytes) != 6 {
		err = fmt.Errorf("EID 长度不对，期望 6 字节，实际 %d 字节", len(eidBytes))
		d.lc.Error(err.Error())
		return err
	}
	var sensorID [6]byte
	copy(sensorID[:], eidBytes)
	// 构建复位帧
	reqFrame, _ := frameparser.BuildResetRequest(sensorID)
	// 发送命令
	eidStr, _ = eidValue.(string)

	serial.SendFrame(eidStr, reqFrame)
	d.lc.Infof("发送复位命令到设备 %s (EID: %s)", deviceName, eidStr)
	return nil
}

func (d *LpMpDriver) handleTimeParameterQuery(deviceName string) error {
	// EID
	eidValue, ok := config.GetDeviceValue(deviceName, "eid")
	if !ok {
		err := fmt.Errorf("设备 %s 的 EID 未初始化", deviceName)
		d.lc.Error(err.Error())
		return err
	}
	eidStr := "238A0841D828"
	// 解码6字节
	eidBytes, err := hex.DecodeString(eidStr)
	if err != nil {
		err = fmt.Errorf("EID[%s] 转十六进制失败: %w", eidStr, err)
		d.lc.Error(err.Error())
		return err
	}
	if len(eidBytes) != 6 {
		err = fmt.Errorf("EID 长度不对，期望 6 字节，实际 %d 字节", len(eidBytes))
		d.lc.Error(err.Error())
		return err
	}
	var sensorID [6]byte
	copy(sensorID[:], eidBytes)
	// 构建时间请求帧
	reqFrame, _ := frameparser.BuildTimeParamFrame(sensorID, 0, 0)
	// 发送命令
	eidStr, _ = eidValue.(string)
	serial.SendFrame(eidStr, reqFrame)
	d.lc.Infof("发送时间请求帧到设备 %s (EID: %s)", deviceName, eidStr)
	return nil
}

func (d *LpMpDriver) handleIdMoniDataQuery(deviceName string) error {
	// EID
	eidValue, ok := config.GetDeviceValue(deviceName, "eid")
	if !ok {
		err := fmt.Errorf("设备 %s 的 EID 未初始化", deviceName)
		d.lc.Error(err.Error())
		return err
	}
	eidStr := "238A0841D828"
	// 解码成 6 字节
	eidBytes, err := hex.DecodeString(eidStr)
	if err != nil {
		err = fmt.Errorf("EID[%s] 转十六进制失败: %w", eidStr, err)
		d.lc.Error(err.Error())
		return err
	}
	if len(eidBytes) != 6 {
		err = fmt.Errorf("EID 长度不对，期望 6 字节，实际 %d 字节", len(eidBytes))
		d.lc.Error(err.Error())
		return err
	}
	var sensorID [6]byte
	copy(sensorID[:], eidBytes)
	//工况查询帧
	frame, err := frameparser.BuildMonitoringDataQueryFrame(sensorID)
	if err != nil {
		return fmt.Errorf("构造全部通用参数查询失败: %w", err)
	}
	eidStr, _ = eidValue.(string)
	//发送命令
	serial.SendFrame(eidStr, frame)
	d.lc.Infof("发送工况查询帧到设备 %s (EID: %s)", deviceName, eidStr)
	return nil
}
