package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"gopkg.in/yaml.v3"
)

// 设备与模型
type DeviceEntry struct {
	Name        string `yaml:"name"`
	ProfileName string `yaml:"profileName"`
}

// 设备列表
type devicesYAML struct {
	DeviceList []DeviceEntry `yaml:"deviceList"`
}

// 资源属性
type ResourceProperty struct {
	ValueType    string `yaml:"valueType"`
	ReadWrite    string `yaml:"readWrite"`
	Units        string `yaml:"units"`
	DefaultValue string `yaml:"defaultValue"`
}

// 设备资源
type DeviceResource struct {
	Name        string           `yaml:"name"`
	IsHidden    bool             `yaml:"isHidden"`
	Description string           `yaml:"description"`
	Properties  ResourceProperty `yaml:"properties"`
}

// 资源列表
type profileYAML struct {
	DeviceResources []DeviceResource `yaml:"deviceResources"`
}

var (
	Mu           sync.RWMutex
	resourcesMap = make(map[string][]DeviceResource)
	// 资源表  设备名称 → (资源名称 → value)
	ValuesMap = make(map[string]map[string]interface{})
)

// 字符串类型转换
func parseDefaultValue(valStr, vt string) interface{} {
	switch vt {
	case "Float32":
		if f, err := strconv.ParseFloat(valStr, 32); err == nil {
			return float32(f)
		}
	case "Uint16":
		if u, err := strconv.ParseUint(valStr, 10, 16); err == nil {
			return uint16(u)
		}
	case "Uint8":
		if u, err := strconv.ParseUint(valStr, 10, 8); err == nil {
			return uint8(u)
		}
	case "Bool":
		if b, err := strconv.ParseBool(valStr); err == nil {
			return b
		}
	case "Float32Array":
		var arr []float32
		if err := json.Unmarshal([]byte(valStr), &arr); err == nil {
			return arr
		}
	case "Object":
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(valStr), &obj); err == nil {
			return obj
		}
	}
	return valStr
}

// 资源初始化
func InitDeviceResources(devicesPath, profilesDir string) error {
	// 读取并解析文件
	raw, err := os.ReadFile(devicesPath)
	if err != nil {
		return fmt.Errorf("无法读取设备列表文件 %s：%w", devicesPath, err)
	}
	var devs devicesYAML
	if err := yaml.Unmarshal(raw, &devs); err != nil {
		return fmt.Errorf("解析 devices.yaml 失败：%w", err)
	}
	Mu.Lock()
	defer Mu.Unlock()
	// 初始化资源
	for _, entry := range devs.DeviceList {
		profileFile := filepath.Join(profilesDir, entry.ProfileName+".yaml")
		rawProfile, err := os.ReadFile(profileFile)
		if err != nil {
			return fmt.Errorf("无法读取 Profile 文件 %s：%w", profileFile, err)
		}
		var prof profileYAML
		if err := yaml.Unmarshal(rawProfile, &prof); err != nil {
			return fmt.Errorf("解析 Profile 文件 %s 失败：%w", profileFile, err)
		}
		resourcesMap[entry.Name] = prof.DeviceResources
		ValuesMap[entry.Name] = make(map[string]interface{}, len(prof.DeviceResources))
		for _, dr := range prof.DeviceResources {
			ValuesMap[entry.Name][dr.Name] = parseDefaultValue(dr.Properties.DefaultValue, dr.Properties.ValueType)
		}
	}
	return nil
}

// 获取设备资源
func GetDeviceResources(deviceName string) ([]DeviceResource, bool) {
	Mu.RLock()
	defer Mu.RUnlock()
	res, ok := resourcesMap[deviceName]
	return res, ok
}

// 写入资源
func SetDeviceValue(deviceName, resourceName string, value interface{}) {
	Mu.Lock()
	defer Mu.Unlock()
	if _, ok := ValuesMap[deviceName]; !ok {
		ValuesMap[deviceName] = make(map[string]interface{})
	}
	ValuesMap[deviceName][resourceName] = value
}

// 获取资源
func GetDeviceValue(deviceName, resourceName string) (interface{}, bool) {
	Mu.RLock()
	defer Mu.RUnlock()
	deviceValues, ok := ValuesMap[deviceName]
	if !ok {
		return nil, false
	}
	value, exists := deviceValues[resourceName]
	return value, exists
}

// 获取所有资源
func GetDeviceValues(deviceName string) (map[string]interface{}, bool) {
	Mu.RLock()
	defer Mu.RUnlock()
	vals, ok := ValuesMap[deviceName]
	if !ok {
		return nil, false
	}
	copyMap := make(map[string]interface{}, len(vals))
	for k, v := range vals {
		copyMap[k] = v
	}
	return copyMap, true
}

// 初始化资源
func DeviceInit(deviceName, resourceName, defaultValue, valueType string) error {
	Mu.Lock()
	defer Mu.Unlock()
	// 映射是否存在
	if _, exists := ValuesMap[deviceName]; !exists {
		ValuesMap[deviceName] = make(map[string]interface{})
	}
	// 转换默认值
	parsedValue := parseDefaultValue(defaultValue, valueType)
	ValuesMap[deviceName][resourceName] = parsedValue
	return nil
}

// 删除设备资源
func DeleteDeviceValues(deviceName string) error {
	Mu.Lock()
	defer Mu.Unlock()
	// 设备是否存在
	if _, exists := ValuesMap[deviceName]; !exists {
		return fmt.Errorf("设备 %s 不存在于运行时值表中", deviceName)
	}
	// 删除
	delete(ValuesMap, deviceName)
	return nil
}

// 删除EID映射
func DeleteSensorIDMappingsByDevice(deviceName string) error {
	toDelete := make([]string, 0)
	for sensorID, mappedDeviceName := range SensorIDToDeviceName {
		if mappedDeviceName == deviceName {
			toDelete = append(toDelete, sensorID)
		}
	}
	// 删除映射
	for _, sensorID := range toDelete {
		delete(SensorIDToDeviceName, sensorID)
	}
	return nil
}
