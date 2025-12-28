package driver

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/linjuya-lu/device-lpmp-go/internal/config"
	"github.com/linjuya-lu/device-lpmp-go/internal/serial"
)

// 健康检查
func (d *LpMpDriver) runOneHealthCheck(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()
	topo, err := serial.QueryAllTopology(ctx)
	if err != nil {
		d.lc.Errorf("健康检查：拓扑查询失败: %v", err)
		return
	}
	serial.HealthTopoMu.Lock()
	serial.HealthTopo = topo
	serial.HealthTopoTime = time.Now()
	serial.HealthTopoMu.Unlock()
	d.lc.Infof("健康检查：拓扑刷新完成，节点数=%d", len(topo))
	d.reportDevicesStateFromTopo(topo)
}

func (d *LpMpDriver) startHealthCheckLoop(ctx context.Context, lc logger.LoggingClient) {
	go func() {
		d.runOneHealthCheck(ctx)
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				d.runOneHealthCheck(ctx)
			case <-ctx.Done():
				lc.Infof("健康检查循环结束: %v", ctx.Err())
				return
			}
		}
	}()
}

func (d *LpMpDriver) reportDevicesStateFromTopo(topo []config.NodeTopology) {
	if len(topo) == 0 {
		d.lc.Debug("healthCheck: topo 为空，跳过 state 上报")
		return
	}
	for _, n := range topo {
		// EID->deviceName
		deviceName, ok := config.LookupDeviceName(n.EID)
		if !ok {
			d.lc.Debugf("healthCheck: EID=%s 未绑定 EdgeX 设备，跳过", n.EID)
			continue
		}
		// 状态：1=在线，0=离线
		s := strings.TrimSpace(n.State)
		if s == "" {
			d.lc.Warnf("healthCheck: 设备 %s (EID=%s) state 为空，跳过", deviceName, n.EID)
			continue
		}
		u, err := strconv.ParseUint(s, 10, 8)
		if err != nil {
			d.lc.Warnf("healthCheck: 设备 %s (EID=%s) state=%q 解析失败: %v",
				deviceName, n.EID, n.State, err)
			continue
		}
		stateVal := uint8(u) // 0 或 1
		values := map[string]any{
			"state": stateVal,
		}
		d.AsyncReporting(deviceName, "state", values)
	}
}
