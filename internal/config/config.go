package config

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func ParseDefaultValue(valStr, vt string) any {
	vt = strings.TrimSpace(vt)

	switch vt {
	case "Float32":
		// 默认值没写，直接给 0.0
		if valStr == "" {
			return float32(0)
		}
		if f, err := strconv.ParseFloat(valStr, 32); err == nil {
			return float32(f)
		}
		// 解析失败也不要返回 string，直接给 0
		return float32(0)

	case "Uint16":
		if valStr == "" {
			return uint16(0)
		}
		if u, err := strconv.ParseUint(valStr, 10, 16); err == nil {
			return uint16(u)
		}
		return uint16(0)

	case "Uint8":
		if valStr == "" {
			return uint8(0)
		}
		if u, err := strconv.ParseUint(valStr, 10, 8); err == nil {
			return uint8(u)
		}
		return uint8(0)

	case "Bool":
		if valStr == "" {
			return false
		}
		if b, err := strconv.ParseBool(valStr); err == nil {
			return b
		}
		return false

	case "Float32Array":
		if valStr == "" {
			return []float32{}
		}
		var arr []float32
		if err := json.Unmarshal([]byte(valStr), &arr); err == nil {
			return arr
		}
		return []float32{}

	case "Object":
		if valStr == "" {
			return map[string]any{}
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(valStr), &obj); err == nil {
			return obj
		}
		return map[string]any{}

	case "String":
		// 明确是字符串时才原样返回
		return valStr
	}

	// 未知类型统统按 string 处理
	return valStr
}

func SetDeviceValue(deviceName, resourceName string, value any) {
	Mu.Lock()
	defer Mu.Unlock()
	if _, ok := ValuesMap[deviceName]; !ok {
		ValuesMap[deviceName] = make(map[string]any)
	}
	ValuesMap[deviceName][resourceName] = value
}

func GetDeviceValue(deviceName, resourceName string) (any, bool) {
	Mu.RLock()
	defer Mu.RUnlock()
	deviceValues, ok := ValuesMap[deviceName]
	if !ok {
		return nil, false
	}
	value, exists := deviceValues[resourceName]
	return value, exists
}

func GetDeviceValues(deviceName string) (map[string]any, bool) {
	Mu.RLock()
	defer Mu.RUnlock()
	vals, ok := ValuesMap[deviceName]
	if !ok {
		return nil, false
	}

	copyMap := make(map[string]any, len(vals))
	for k, v := range vals {
		copyMap[k] = v
	}
	return copyMap, true
}

func DeviceInit(deviceName, resourceName, defaultValue, valueType string) error {
	Mu.Lock()
	defer Mu.Unlock()
	if _, exists := ValuesMap[deviceName]; !exists {
		ValuesMap[deviceName] = make(map[string]any)
	}
	parsedValue := ParseDefaultValue(defaultValue, valueType)
	ValuesMap[deviceName][resourceName] = parsedValue
	return nil
}

func DeleteDeviceValues(deviceName string) error {
	Mu.Lock()
	defer Mu.Unlock()
	if _, exists := ValuesMap[deviceName]; !exists {
		return fmt.Errorf("DeleteDeviceValues 设备 %s 不存在于运行时值表中", deviceName)
	}
	delete(ValuesMap, deviceName)
	return nil
}
