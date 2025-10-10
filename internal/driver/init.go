package driver

import "github.com/linjuya-lu/device-lpmp-go/internal/config"

// 初始化AccessNode01的state
func InitDevice() {
	deviceName := "AccessNode01"
	resourceName := "state"
	value := 1

	config.SetDeviceValue(deviceName, resourceName, value)
}
