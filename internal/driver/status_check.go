package driver

import (
	"time"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

func startHealthCheckLoop() {
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for range ticker.C {
			now := time.Now().UnixNano()

			// 读取设备名称
			config.Mu.RLock()
			deviceNames := make([]string, 0, len(config.ValuesMap))
			for dev := range config.ValuesMap {
				deviceNames = append(deviceNames, dev)
			}
			config.Mu.RUnlock()

			// 检查每台设备
			for _, dev := range deviceNames {
				if dev == "Access-Node-01" {
					continue
				}

				rawTs, okTs := config.GetDeviceValue(dev, "resourceLastDataTimestamp")
				rawPr, okPr := config.GetDeviceValue(dev, "resourcePeriod")
				if !okTs || !okPr {
					continue
				}
				lastTs, ok1 := rawTs.(int64)
				period, ok2 := rawPr.(uint32)
				if !ok1 || !ok2 {
					continue
				}

				// 判断： now - lastTs > 2 * period 秒
				deadline := int64(period) * 2 * int64(time.Second)
				newState := uint8(0)
				if now-lastTs > deadline {
					newState = 1
				}

				config.SetDeviceValue(dev, "resourceState", newState)
			}
		}
	}()
}
