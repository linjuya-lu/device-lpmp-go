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
				if dev == "Access-Node-01" {
					continue
				}

				rawTs, okTs := config.GetDeviceValue(dev, "resourceLastDataTimestamp")
				rawPr, okPr := config.GetDeviceValue(dev, "resourcePeriod")
				if !okTs || !okPr {
					fmt.Printf("[health] dev=%s missing key(s): ts=%v pr=%v\n", dev, okTs, okPr)
					continue
				}

				// lastTs: 期望纳秒
				lastNs, okTsCast := asInt64Ns(rawTs)
				if !okTsCast {
					fmt.Printf("[health] dev=%s bad ts type=%T val=%v (expect ns int64)\n", dev, rawTs, rawTs)
					continue
				}

				// period: 期望“秒”，但容忍多种类型
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

				// fmt.Printf("[health] dev=%s now=%d lastTs=%d elapsed=%s deadline=%s -> state=%d\n",
				// 	dev,
				// 	nowNs,
				// 	lastNs,
				// 	time.Duration(elapsedNs),
				// 	time.Duration(deadlineNs),
				// 	newState,
				// )

				config.SetDeviceValue(dev, "resourceState", newState)
			}
		}
	}()
}

// ---------- 辅助转换 ----------

// 将各种常见整型/字符串时间戳转为 ns（你的 ts 就是 UnixNano，直接透传）
func asInt64Ns(v interface{}) (int64, bool) {
	switch t := v.(type) {
	case int64:
		return t, true
	case int:
		return int64(t), true
	case float64:
		return int64(t), true
	case string:
		// 有些默认值可能是字符串
		if n, err := strconv.ParseInt(t, 10, 64); err == nil {
			return n, true
		}
		return 0, false
	default:
		return 0, false
	}
}

// 将 period（秒）统一到 uint64，容忍 Uint16/Uint32/Uint64/int/float/string
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
