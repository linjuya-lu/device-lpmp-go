package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"gopkg.in/yaml.v3"
)

// DeviceEntry 表示 devices.yaml 中的单个设备条目
// 包含设备逻辑名称和对应的 Profile 名称，不含 .yaml 后缀
type DeviceEntry struct {
	Name        string `yaml:"name"`
	ProfileName string `yaml:"profileName"`
}

// devicesYAML 对应 devices.yaml 的顶层结构，用于批量读取 deviceList 字段
type devicesYAML struct {
	DeviceList []DeviceEntry `yaml:"deviceList"`
}

// ResourceProperty 保存设备资源属性配置
// 包含值类型、权限、单位和默认值等
type ResourceProperty struct {
	ValueType    string `yaml:"valueType"`
	ReadWrite    string `yaml:"readWrite"`
	Units        string `yaml:"units"`
	DefaultValue string `yaml:"defaultValue"`
}

// DeviceResource 对应 Profile 文件中的单个资源条目
// 包含名称、隐藏标志、描述和属性字段
type DeviceResource struct {
	Name        string           `yaml:"name"`
	IsHidden    bool             `yaml:"isHidden"`
	Description string           `yaml:"description"`
	Properties  ResourceProperty `yaml:"properties"`
}

// profileYAML 对应 Profile 文件顶层，仅解析 deviceResources 列表
type profileYAML struct {
	DeviceResources []DeviceResource `yaml:"deviceResources"`
}

var (
	// mu 保护下面的静态资源表和运行时值表
	mu sync.RWMutex
	// resourcesMap 存储所有设备的静态资源定义，key 为设备逻辑名称
	resourcesMap = make(map[string][]DeviceResource)
	// valuesMap 存储所有设备的运行时资源值，key: 设备名称 → (资源名称 → value)
	valuesMap = make(map[string]map[string]interface{})
)

// parseDefaultValue 根据 ValueType 将 DefaultValue 字符串转换为对应类型
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
	}
	// 其它类型保留字符串
	return valStr
}

// InitDeviceResources 初始化静态资源定义及默认运行时值：
// 1. 读取并解析 devices.yaml，获取所有设备条目
// 2. 遍历每个 entry，根据 ProfileName 加载 Profile 文件，解析 deviceResources
// 3. 填充全局 maps，并将 DefaultValue 作为初始值写入 valuesMap
func InitDeviceResources(devicesPath, profilesDir string) error {
	// 读取 devices.yaml
	raw, err := os.ReadFile(devicesPath)
	if err != nil {
		return fmt.Errorf("无法读取设备列表文件 %s：%w", devicesPath, err)
	}
	// 解析 YAML
	var devs devicesYAML
	if err := yaml.Unmarshal(raw, &devs); err != nil {
		return fmt.Errorf("解析 devices.yaml 失败：%w", err)
	}

	mu.Lock()
	defer mu.Unlock()
	// 加载并写入静态资源和默认值表
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
		// 保存静态定义
		resourcesMap[entry.Name] = prof.DeviceResources
		// 初始化运行时值为 DefaultValue
		valuesMap[entry.Name] = make(map[string]interface{}, len(prof.DeviceResources))
		for _, dr := range prof.DeviceResources {
			valuesMap[entry.Name][dr.Name] = parseDefaultValue(dr.Properties.DefaultValue, dr.Properties.ValueType)
		}
	}
	return nil
}

// GetDeviceResources 并发安全地获取指定设备的静态资源列表
// 返回值: []DeviceResource, bool(是否存在)
func GetDeviceResources(deviceName string) ([]DeviceResource, bool) {
	mu.RLock()
	defer mu.RUnlock()
	res, ok := resourcesMap[deviceName]
	return res, ok
}

// SetDeviceValue 并发安全地写入解析后的单个资源值
func SetDeviceValue(deviceName, resourceName string, value interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if _, ok := valuesMap[deviceName]; !ok {
		valuesMap[deviceName] = make(map[string]interface{})
	}
	valuesMap[deviceName][resourceName] = value
}

// GetDeviceValues 并发安全地获取指定设备的所有运行时资源值
// 返回值: map[resourceName]value, bool(是否存在)
func GetDeviceValues(deviceName string) (map[string]interface{}, bool) {
	mu.RLock()
	defer mu.RUnlock()
	vals, ok := valuesMap[deviceName]
	if !ok {
		return nil, false
	}
	// 返回副本防止外部修改原表
	copyMap := make(map[string]interface{}, len(vals))
	for k, v := range vals {
		copyMap[k] = v
	}
	return copyMap, true
}
