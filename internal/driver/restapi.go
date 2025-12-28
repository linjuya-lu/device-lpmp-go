package driver

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces"
	"github.com/labstack/echo/v4"
	"github.com/linjuya-lu/device-lpmp-go/internal/config"
	"github.com/linjuya-lu/device-lpmp-go/internal/serial"
)

func (d *LpMpDriver) handleLoadParamMap(c echo.Context) error {
	fh, err := c.FormFile("file")
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "missing form file field 'file': " + err.Error(),
		})
	}
	src, err := fh.Open()
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": "open uploaded file failed: " + err.Error(),
		})
	}
	defer src.Close()
	if err := config.LoadParamMapFromReader(src, fh.Filename); err != nil {
		d.lc.Errorf("LoadParamMapFromReader error: %v", err)
		return c.JSON(http.StatusBadRequest, map[string]any{
			"ok":    false,
			"error": err.Error(),
		})
	}
	return c.JSON(http.StatusOK, map[string]any{"ok": true})
}

func (d *LpMpDriver) addCustomRoutes() error {
	if err := d.sdk.AddCustomRoute(
		"/custom/load-param-map",
		interfaces.Unauthenticated,
		d.handleLoadParamMap,
		http.MethodPost,
	); err != nil {
		return fmt.Errorf("register load-param-map route failed: %w", err)
	}
	if err := d.sdk.AddCustomRoute(
		"/custom/topology",
		interfaces.Unauthenticated,
		d.handleGetTopology,
		http.MethodGet,
	); err != nil {
		return fmt.Errorf("register topology route failed: %w", err)
	}
	return nil
}

func (d *LpMpDriver) handleGetTopology(c echo.Context) error {
	fillDescIfPossible := func(node *config.NodeTopology) {
		devName, ok := config.LookupDeviceName(node.EID)
		if !ok {
			d.lc.Infof("handleGetTopology: eid=%s 无EID映射，Desc不填充", node.EID)
			return
		}
		dev, err := d.sdk.GetDeviceByName(devName)
		if err != nil {
			d.lc.Infof("handleGetTopology: 根据设备名%s获取设备失败:%v，Desc 不填充", devName, err)
			return
		}
		desc := strings.TrimSpace(dev.Description)
		if desc == "" {
			d.lc.Debugf("handleGetTopology: 设备 %s (eid=%s) 未配置 description，Desc 不填充", devName, node.EID)
			return
		}
		node.Desc = desc
	}

	filterByMapping := func(nodes []config.NodeTopology) []config.NodeTopology {
		filtered := make([]config.NodeTopology, 0, len(nodes))
		for _, n := range nodes {
			// EID==EidStr，保留
			if n.EID == config.EidStr {
				fillDescIfPossible(&n)
				filtered = append(filtered, n)
				continue
			}
			// 其他节点
			if _, ok := config.LookupDeviceName(n.EID); !ok {
				d.lc.Debugf(
					"handleGetTopology: 丢弃脏拓扑节点eid=%s",
					n.EID,
				)
				continue
			}
			// 有映射
			fillDescIfPossible(&n)
			filtered = append(filtered, n)
		}
		return filtered
	}
	topo, ts := serial.GetHealthTopology()
	if len(topo) == 0 {
		ctx := c.Request().Context()
		rt, err := serial.QueryAllTopology(ctx)
		if err != nil {
			d.lc.Errorf("实时拓扑查询失败: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]any{
				"error": err.Error(),
			})
		}
		filtered := filterByMapping(rt)
		d.lc.Infof("返回实时拓扑，原始=%d，过滤后=%d", len(rt), len(filtered))
		return c.JSON(http.StatusOK, filtered)
	}
	filtered := filterByMapping(topo)
	d.lc.Infof("返回健康缓存拓扑，原始=%d，过滤后=%d，刷新时间=%s", len(topo), len(filtered), ts.Format(time.RFC3339))
	return c.JSON(http.StatusOK, filtered)
}
