package config

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
)

type ParamKey struct {
	FeatureBits byte   // 高3位（参量特征）
	CodeBits    uint16 // 低11位（类型编码）
}

type ParamInfo struct {
	Name     string
	Unit     string
	ByteLen  int
	DataType string
	Parse    func([]byte) (any, error)
}

var paramMap = map[ParamKey]ParamInfo{
	{0b000, 0b00000000001}: {"长度", "m", 4, "float32", parseFloat32},
	{0b000, 0b00000000010}: {"电池剩余电量", "%", 2, "uint16", parseAndStoreBatteryLevel},
	{0b000, 0b00000000011}: {"voltage", "v", 4, "uint32", parseAndStoreVoltage},
	{0b000, 0b00000000100}: {"state", "0:其它,1:正常,2:异常", 1, "uint8", parseAndStoreDeviceStatus},
	{0b000, 0b00000000101}: {"温度", "℃", 4, "float32", parseFloat32},
	{0b000, 0b00000000110}: {"物质的量", "mol", 4, "float32", parseFloat32},
	{0b000, 0b00000000111}: {"发光强度", "cd", 4, "float32", parseFloat32},
	{0b000, 0b00000001000}: {"temperature", "℃", 4, "float32", parseAndStoreTemperature},
	{0b000, 0b00000001001}: {"humidity", "%RH", 2, "float32", parseAndStoreHumidity},
	{0b000, 0b00000111000}: {"心跳状态", "\\", 1, "uint8", parseUint8},
	{0b000, 0b00000111001}: {"battery-level", "%", 1, "uint8", parseUint8},
	{0b000, 0b00010100011}: {"water-level", "m", 4, "float32", parseAndStoreLevelHeight},
}

func LookupParamInfo(paramType uint16) (ParamInfo, bool) {
	feature := byte((paramType >> 11) & 0x07)
	code := paramType & 0x7FF
	fmt.Printf("🔍 TypeCode=0x%04X → Feature=%03b (0x%X), Code=%011b (0x%X)\n", paramType, feature, feature, code, code)

	key := ParamKey{feature, code}
	info, ok := paramMap[key]
	return info, ok
}

// ===================== 通用解析函数 =====================

func parseFloat32(data []byte) (any, error) {
	if len(data) != 4 {
		return nil, fmt.Errorf("期望4字节，实际%d", len(data))
	}
	bits := binary.LittleEndian.Uint32(data)
	val := math.Float32frombits(bits)
	return val, nil
}

func parseUint32(data []byte) (any, error) {
	if len(data) != 4 {
		return nil, fmt.Errorf("期望4字节，实际%d", len(data))
	}
	return binary.LittleEndian.Uint32(data), nil
}

func parseUint8(data []byte) (any, error) {
	if len(data) != 1 {
		return nil, fmt.Errorf("期望1字节，实际%d", len(data))
	}
	return uint8(data[0]), nil
}

func parseUint16(data []byte) (any, error) {
	if len(data) != 2 {
		return nil, fmt.Errorf("期望2字节，实际%d", len(data))
	}
	return binary.LittleEndian.Uint16(data), nil
}

func parseAndStoreTemperature(data []byte) (any, error) {
	valAny, err := parseFloat32(data)
	if err != nil {
		return nil, err
	}
	val := valAny.(float32)

	return val, nil
}

func parseAndStoreHumidity(data []byte) (any, error) {
	if len(data) != 2 {
		return nil, fmt.Errorf("期望2字节，实际%d", len(data))
	}
	val := float32(binary.LittleEndian.Uint16(data))

	return val, nil
}

func parseAndStoreVoltage(data []byte) (any, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("数据长度不足，期望 4 字节，实际 %d 字节", len(data))
	}

	bits := binary.LittleEndian.Uint32(data[:4])
	val := math.Float32frombits(bits)

	log.Printf("🔋 电池电压解析结果：%.4f V", val)

	return val, nil
}

func parseAndStoreBatteryLevel(data []byte) (any, error) {
	if len(data) < 2 {
		return nil, fmt.Errorf("数据长度不足，期望 2 字节，实际 %d 字节", len(data))
	}

	val := binary.LittleEndian.Uint16(data[:2])

	log.Printf("🔋 电池剩余电量解析结果：%d%%", val)

	return val, nil
}

func parseAndStoreDeviceStatus(data []byte) (any, error) {
	if len(data) < 1 {
		return nil, fmt.Errorf("数据长度不足，期望 1 字节，实际 %d 字节", len(data))
	}

	val := data[0]

	var statusDesc string
	switch val {
	case 0:
		statusDesc = "其它"
	case 1:
		statusDesc = "正常"
	case 2:
		statusDesc = "异常"
	default:
		statusDesc = "未知"
	}

	log.Printf("📡 设备状态解析结果：%d（%s）", val, statusDesc)

	return val, nil
}

func parseAndStoreLevelHeight(data []byte) (any, error) {
	if len(data) < 4 {
		return nil, fmt.Errorf("数据长度不足，期望 4 字节，实际 %d 字节", len(data))
	}

	bits := binary.LittleEndian.Uint32(data[:4])
	val := math.Float32frombits(bits)

	log.Printf("📏 液位高度解析结果：%.3f m", val)

	return val, nil
}
