package driver

import (
	"fmt"
	"strconv"
	"time"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

func startHealthCheckLoop() {
	go func() {
		ticker := time.NewTicker(time.Second) // 检查间隔：1s
		defer ticker.Stop()

		for range ticker.C {
			nowNs := time.Now().UnixNano()

			config.Mu.RLock()
			deviceNames := make([]string, 0, len(config.ValuesMap))
			for dev := range config.ValuesMap {
				deviceNames = append(deviceNames, dev)
			}
			config.Mu.RUnlock()

			for _, dev := range deviceNames {
				if dev == "AccessNode01" {
					continue
				}

				rawTs, okTs := config.GetDeviceValue(dev, "LastDataTs")
				rawPr, okPr := config.GetDeviceValue(dev, "period")
				if !okTs || !okPr {
					fmt.Printf("[health] dev=%s missing key(s): ts=%v pr=%v\n", dev, okTs, okPr)
					continue
				}

				// 时间戳
				lastNs, okTsCast := asInt64Ns(rawTs)
				if !okTsCast {
					fmt.Printf("[health] dev=%s bad ts type=%T val=%v (expect ns int64)\n", dev, rawTs, rawTs)
					continue
				}

				// 周期
				periodSec, okPrCast := asUint64Seconds(rawPr)
				if !okPrCast {
					fmt.Printf("[health] dev=%s bad period type=%T val=%v (expect seconds)\n", dev, rawPr, rawPr)
					continue
				}
				if periodSec == 0 {
					fmt.Printf("[health] dev=%s period=0s -> skip\n", dev)
					continue
				}

				deadlineNs := int64(time.Duration(periodSec*2) * time.Second) // 2*period（单位ns）
				elapsedNs := nowNs - lastNs

				newState := uint8(1) // 在线=1
				if elapsedNs > deadlineNs {
					newState = 0 // 超时=离线
				}

				config.SetDeviceValue(dev, "state", newState)
			}
		}
	}()
}

// 将整型/字符串时间戳转为ns
func asInt64Ns(v interface{}) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case float64:
		return int64(t), true
	case string:
		// 字符串
		if n, err := strconv.ParseInt(t, 10, 64); err == nil {
			return n, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// 将周期秒统一到 uint64
func asUint64Seconds(v interface{}) (uint64, bool) {
	switch t := v.(type) {
	case uint16:
		return uint64(t), true
	case uint32:
		return uint64(t), true
	case uint64:
		return t, true
	case int:
		if t < 0 {
			return 0, false
		}
		return uint64(t), true
	case int32:
		if t < 0 {
			return 0, false
		}
		return uint64(t), true
	case int64:
		if t < 0 {
			return 0, false
		}
		return uint64(t), true
	case float64:
		if t < 0 {
			return 0, false
		}
		return uint64(t), true // 向下取整
	case string:
		if n, err := strconv.ParseUint(t, 10, 64); err == nil {
			return n, true
		}
		return 0, false
	default:
		return 0, false
	}
}
