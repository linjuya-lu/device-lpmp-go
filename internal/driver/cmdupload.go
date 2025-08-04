package driver

import (
	"fmt"
	"math/rand/v2"
	"time"

	dsModels "github.com/edgexfoundry/device-sdk-go/v4/pkg/models"
	"github.com/edgexfoundry/go-mod-core-contracts/v4/common"
)

// simulateAsyncReporting 构造一次 AsyncValues 并直接推送到 s.asyncCh
//
//	deviceName    设备名称
//	sourceName    上报用的 sourceName
//	resourceNames 要上报的资源名列表（仅取第一个），每个元素都会作为 NewCommandValue 的第一个参数
func (d *LpMpDriver) AsyncReporting(
	deviceName string,
	sourceName string,
	resourceNames []string,
) {
	if len(resourceNames) == 0 {
		d.lc.Warn("simulateAsyncReporting: 未提供任何资源名，跳过上报")
		return
	}

	// 只取第一个资源名，构造一个 CommandValue
	name := resourceNames[0]
	origin := time.Now().UnixNano()
	cv, err := dsModels.NewCommandValue(
		name,
		common.ValueTypeInt32, // 如需支持其他类型，可根据 name 做映射
		rand.Int32(),          // 随机值或您自己的生成逻辑
	)
	if err != nil {
		d.lc.Error(fmt.Sprintf("NewCommandValue(%s) 失败: %v", name, err))
		return
	}
	cv.Origin = origin

	// 封装结果
	async := &dsModels.AsyncValues{
		DeviceName:    deviceName,
		SourceName:    sourceName,
		CommandValues: []*dsModels.CommandValue{cv},
	}

	d.asyncCh <- async
	d.lc.Debugf("AsyncValues pushed: device=%s source=%s value=%+v",
		deviceName, sourceName, cv)
}
