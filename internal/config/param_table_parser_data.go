package config

import (
	"encoding/binary"
	"fmt"
	"math"
)

type ParamKey struct {
	FeatureBits byte   // 参量特征
	CodeBits    uint16 // 类型编码
}

type ParamInfo struct {
	Parse func([]byte) (any, error)
}

var paramMap = map[ParamKey]ParamInfo{
	{0b000, 0b00000000001}: {parseFloat32},
}

func LookupParamInfo(paramType uint16) (ParamInfo, bool) {
	feature := byte((paramType >> 11) & 0x07)
	code := paramType & 0x7FF
	fmt.Printf("TypeCode=0x%04X → Feature=%03b (0x%X), Code=%011b (0x%X)\n", paramType, feature, feature, code, code)

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

func parseUint8(data []byte) (any, error) {
	if len(data) != 1 {
		return nil, fmt.Errorf("期望1字节，实际%d", len(data))
	}
	return data[0], nil
}

func parseUint16(data []byte) (any, error) {
	if len(data) != 2 {
		return nil, fmt.Errorf("期望2字节，实际%d", len(data))
	}
	return binary.LittleEndian.Uint16(data), nil
}

func parseUint32(data []byte) (any, error) {
	if len(data) != 4 {
		return nil, fmt.Errorf("期望4字节，实际%d", len(data))
	}
	return binary.LittleEndian.Uint32(data), nil
}

func parsefloat32Array(data []byte) (any, error) {
	if len(data)%4 != 0 {
		return nil, fmt.Errorf("波形数据长度非4的倍数: %d", len(data))
	}
	n := len(data) / 4
	samples := make([]float32, n)
	for i := 0; i < n; i++ {
		bits := binary.LittleEndian.Uint32(data[i*4 : i*4+4])
		samples[i] = math.Float32frombits(bits)
	}
	return samples, nil
}

func parseUint16Array(data []byte) (any, error) {
	if len(data)%2 != 0 {
		return nil, fmt.Errorf("uint16 数组数据长度非2的倍数: %d", len(data))
	}
	n := len(data) / 2
	values := make([]uint16, n)
	for i := 0; i < n; i++ {
		values[i] = binary.LittleEndian.Uint16(data[i*2 : i*2+2])
	}
	return values, nil
}

func parseInt16(data []byte) (any, error) {
	if len(data) != 2 {
		return nil, fmt.Errorf("期望2字节，实际%d", len(data))
	}
	u := binary.LittleEndian.Uint16(data)
	val := int16(u)
	return val, nil
}
