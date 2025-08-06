package driver

import (
	"time"

	dsModels "github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
)

// simulateAsyncReporting 构造一次 AsyncValues 并直接推送到 s.asyncCh
//
//	deviceName    设备名称
//	sourceName    上报用的 sourceName
//	resourceNames 要上报的资源名列表（仅取第一个），每个元素都会作为 NewCommandValue 的第一个参数
// func (d *LpMpDriver) AsyncReporting(
// 	deviceName string,
// 	sourceName string,
// 	resourceNames []string,
// ) {
// 	if len(resourceNames) == 0 {
// 		d.lc.Warn("simulateAsyncReporting: 未提供任何资源名，跳过上报")
// 		return
// 	}

// 	// 只取第一个资源名，构造一个 CommandValue
// 	name := resourceNames[0]
// 	origin := time.Now().UnixNano()
// 	cv, err := dsModels.NewCommandValue(
// 		name,
// 		common.ValueTypeInt32, // 如需支持其他类型，可根据 name 做映射
// 		rand.Int32(),          // 随机值或您自己的生成逻辑
// 	)
// 	if err != nil {
// 		d.lc.Error(fmt.Sprintf("NewCommandValue(%s) 失败: %v", name, err))
// 		return
// 	}
// 	cv.Origin = origin

// 	// 封装结果
// 	async := &dsModels.AsyncValues{
// 		DeviceName:    deviceName,
// 		SourceName:    sourceName,
// 		CommandValues: []*dsModels.CommandValue{cv},
// 	}

// 	d.asyncCh <- async
// 	d.lc.Debugf("AsyncValues pushed: device=%s source=%s value=%+v",
// 		deviceName, sourceName, cv)
// }

// func (d *LpMpDriver) AsyncReporting(
// 	deviceName string,
// 	sourceName string,
// 	values map[string]interface{},
// ) {
// 	fmt.Printf("[AsyncReporting] called! d=%p\n", d)
// 	fmt.Printf("[AsyncReporting] values=%#v", values)

// 	if len(values) == 0 {
// 		d.lc.Warn("AsyncReporting: 没有要上报的值")
// 		return
// 	}

// 	var cvs []*dsModels.CommandValue
// 	origin := time.Now().UnixNano()

// 	for name, val := range values {
// 		var cv *dsModels.CommandValue
// 		var err error

// 		switch v := val.(type) {
// 		case int32:
// 			cv, err = dsModels.NewCommandValue(name, common.ValueTypeInt32, v)
// 		case float32:
// 			cv, err = dsModels.NewCommandValue(name, common.ValueTypeFloat32, v)
// 		case string:
// 			cv, err = dsModels.NewCommandValue(name, common.ValueTypeString, v)
// 		default:
// 			d.lc.Errorf("不支持的类型: %T", v)
// 			continue
// 		}

// 		if err != nil {
// 			d.lc.Errorf("NewCommandValue(%s) 失败: %v", name, err)
// 			continue
// 		}
// 		cv.Origin = origin
// 		cvs = append(cvs, cv)
// 	}

// 	async := &dsModels.AsyncValues{
// 		DeviceName:    deviceName,
// 		SourceName:    sourceName,
// 		CommandValues: cvs,
// 	}

// 	d.asyncCh <- async
// 	d.lc.Debugf("AsyncValues pushed: device=%s source=%s values=%+v", deviceName, sourceName, cvs)
// }

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
		case int32:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeInt32, v)
		case int64:
			cv, err = dsModels.NewCommandValue(name, common.ValueTypeInt64, v)
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
