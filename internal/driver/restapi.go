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
	topo, ts := serial.GetHealthTopology()
	if len(topo) == 0 {
		// 缓存还没准备好，或者刚启动，可以选择：
		// 1) 直接查一次实时的
		// 2) 返回 503 提示“拓扑未准备好”
		ctx := c.Request().Context()
		rt, err := serial.QueryAllTopology(ctx)
		if err != nil {
			d.lc.Errorf("实时拓扑查询失败: %v", err)
			return c.JSON(http.StatusInternalServerError, map[string]any{
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusOK, rt)
	}

	d.lc.Infof("返回健康缓存拓扑，总数=%d，刷新时间=%s", len(topo), ts.Format(time.RFC3339))
	return c.JSON(http.StatusOK, topo)
}
