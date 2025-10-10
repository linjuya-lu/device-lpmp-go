package driver

import (
	"time"

	dsModels "github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

func (d *LpMpDriver) AsyncReporting(deviceName string, sourceName string, values map[string]interface{}) {
	d.lc.Infof("[AsyncReporting] values=%#v", values)

	if len(values) == 0 {
		d.lc.Infof("AsyncReporting: 没有要上报的值")
		return
	}

	var cvs []*dsModels.CommandValue
	origin := time.Now().UnixNano()

	for name, val := range values {
		d.lc.Infof("[AsyncReporting] processing: name=%s type=%T value=%v", name, val, val)

		var cv *dsModels.CommandValue
		var err error

		switch v := val.(type) {
		case int8:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeInt8, v)
		case uint8:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeUint8, v)
		case int16:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeInt16, v)
		case uint16:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeUint16, v)
		case int32:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeInt32, v)
		case uint32:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeUint32, v)
		case int64:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeInt64, v)
		case uint64:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeUint64, v)
		case float32:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeFloat32, v)
		case float64:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeFloat64, v)
		case string:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeString, v)
		default:
			d.lc.Infof("不支持的类型: %T", v)
			continue
		}

		if err != nil {
			d.lc.Infof("NewCommandValue(%s) 失败: %v", name, err)
			continue
		}
		cv.Origin = origin
		cvs = append(cvs, cv)
	}

	if len(cvs) == 0 {
		d.lc.Infof("AsyncReporting: 没有有效的 CommandValue，跳过上报")
		return
	}

	// 封装 AsyncValues
	asyncValues := &dsModels.AsyncValues{
		DeviceName:    deviceName,
		SourceName:    sourceName,
		CommandValues: cvs,
	}

	// 推送到 SDK 异步通道
	d.asyncCh <- asyncValues

	d.lc.Infof("AsyncValues pushed: device=%s source=%s count=%d",
		deviceName, sourceName, len(cvs))
}

// 心跳上传
func (d *LpMpDriver) StartAsyncReporter() {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for range ticker.C {
			config.Mu.RLock()
			for deviceName, resMap := range config.ValuesMap {
				if resMap == nil {
					continue
				}

				stateVal, ok := resMap["state"]
				if !ok {
					continue
				}
				values := map[string]interface{}{
					"state": stateVal,
				}
				d.AsyncReporting(deviceName, "hBeat", values)
			}
			config.Mu.RUnlock()
		}
	}()
}
