package driver

import (
	"context"
	"time"

	"github.com/edgexfoundry/go-mod-core-contracts/v4/clients/logger"
	"github.com/linjuya-lu/device-lpmp-go/internal/serial"
)

// runOneHealthCheck 执行一次拓扑全量查询，并更新全局缓存。
// parentCtx 一般用 context.Background() / service 的主 ctx。
func runOneHealthCheck(parentCtx context.Context, lc logger.LoggingClient) {
	// 给这次健康检查设置一个总超时，比如 10 秒
	ctx, cancel := context.WithTimeout(parentCtx, 10*time.Second)
	defer cancel()

	topo, err := serial.QueryAllTopology(ctx)
	if err != nil {
		lc.Errorf("健康检查：拓扑查询失败: %v", err)
		return
	}

	serial.HealthTopoMu.Lock()
	serial.HealthTopo = topo
	serial.HealthTopoTime = time.Now()
	serial.HealthTopoMu.Unlock()

	lc.Infof("健康检查：拓扑刷新完成，节点数=%d", len(topo))
}

// startHealthCheckLoop 启动后台健康检查循环。
// 约定：在设备服务启动时调用一次即可。
func startHealthCheckLoop(ctx context.Context, lc logger.LoggingClient) {
	go func() {
		// 先立即跑一次，启动时就有数据
		runOneHealthCheck(ctx, lc)

		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				runOneHealthCheck(ctx, lc)

			case <-ctx.Done():
				lc.Infof("健康检查循环结束: %v", ctx.Err())
				return
			}
		}
	}()
}
