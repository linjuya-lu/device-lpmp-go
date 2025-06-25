package config

// sensorIDToDeviceName 是传感器 6 字节 ID（大写十六进制）到本地逻辑设备名的映射
var sensorIDToDeviceName = map[string]string{
	"238A08262319": "WaterLevelSensor01",
	// 在此处继续添加： "<SensorID>": "<DeviceName>",
}

// LookupDeviceName 根据大写十六进制的 SensorID 返回逻辑设备名
func LookupDeviceName(sensorID string) (deviceName string, ok bool) {
	deviceName, ok = sensorIDToDeviceName[sensorID]
	return
}
