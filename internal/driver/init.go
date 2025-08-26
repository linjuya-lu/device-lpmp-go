package driver

import "github.com/linjuya-lu/device-lpmp-go/internal/config"

// InitDevice 初始化 Access-Node-01 的 resourceState
func InitDevice() {
	deviceName := "Access-Node-01"
	resourceName := "resourceState"
	value := 1

	config.SetDeviceValue(deviceName, resourceName, value)
}
