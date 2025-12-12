package driver

import (
	"fmt"
	"net/http"
	"time"

	"github.com/edgexfoundry/device-sdk-go/v4/pkg/interfaces"
	"github.com/labstack/echo/v4"
	"github.com/linjuya-lu/device-lpmp-go/internal/config"
	"github.com/linjuya-lu/device-lpmp-go/internal/serial"
)

func (d *LpMpDriver) handleLoadParamMap(c echo.Context) error {
	// Content-Type: multipart/form-data, 字段名为file
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
	// 小工具函数：过滤掉 SensorIDToDeviceName 里不存在的 EID，
	// 但模块自己的 EidStr 一律保留
	filterByMapping := func(nodes []config.NodeTopology) []config.NodeTopology {
		filtered := make([]config.NodeTopology, 0, len(nodes))
		for _, n := range nodes {
			// 1. 模块自身根节点：EID == EidStr，强制保留
			if n.EID == config.EidStr {
				filtered = append(filtered, n)
				continue
			}

			// 2. 其他节点：必须在 SensorIDToDeviceName 中有映射才保留
			if _, ok := config.LookupDeviceName(n.EID); ok {
				filtered = append(filtered, n)
			} else {
				d.lc.Debugf(
					"handleGetTopology: 丢弃脏拓扑节点 eid=%s（在 SensorIDToDeviceName 中无映射）",
					n.EID,
				)
			}
		}
		return filtered
	}

	topo, ts := serial.GetHealthTopology()

	if len(topo) == 0 {
		// 缓存没有，就查一次实时拓扑，然后一样做过滤
		ctx := c.Request().Context()
		rt, err := serial.QueryAllTopology(ctx)
		if err != nil {
			d.lc.Errorf("实时拓扑查询失败: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]any{
				"error": err.Error(),
			})
		}

		filtered := filterByMapping(rt)
		d.lc.Infof("返回实时拓扑（已过滤脏数据），原始=%d，过滤后=%d",
			len(rt), len(filtered))
		return c.JSON(http.StatusOK, filtered)
	}

	filtered := filterByMapping(topo)
	d.lc.Infof("返回健康缓存拓扑（已过滤脏数据），原始=%d，过滤后=%d，刷新时间=%s",
		len(topo), len(filtered), ts.Format(time.RFC3339))

	return c.JSON(http.StatusOK, filtered)
}
