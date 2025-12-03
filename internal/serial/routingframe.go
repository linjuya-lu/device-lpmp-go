package serial

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

// 一页 TOP 返回结果
type TopoPage struct {
	Total  int                   // 节点总数
	Number int                   // 本页返回数量
	Nodes  []config.NodeTopology // 本页节点
}

var (
	topoRespCh = make(chan TopoPage, 1) // 一页结果

	topoSessionMu sync.Mutex // HTTP并发
)

// 拓扑查询
func SendTopoQuery(startIndex, numOfQuery int) {
	body := fmt.Sprintf("AT+TOP=%d,%d?", startIndex, numOfQuery)
	cmd := "\r" + body + "\r\n"
	fmt.Printf("拓扑查询: %s\n", body)
	config.WriteChan <- []byte(cmd)
}

// 解析拓扑
func parseBuffer(buf string) ([]config.NodeTopology, error) {
	buf = strings.TrimSpace(buf)
	if buf == "" {
		return nil, nil
	}

	// 按逗号拆分并清理空字段/空格
	raw := strings.Split(buf, ",")
	fields := make([]string, 0, len(raw))
	for _, f := range raw {
		f = strings.TrimSpace(f)
		if f != "" {
			fields = append(fields, f)
		}
	}

	if len(fields)%4 != 0 {
		return nil, fmt.Errorf("字段数 %d 不是 4 的倍数", len(fields))
	}

	// 小工具： 统一成十六进制大写（移除分隔符）
	normalizeHex12 := func(s string) (string, error) {
		s = strings.ToUpper(s)
		s = strings.ReplaceAll(s, ":", "")
		s = strings.ReplaceAll(s, "-", "")
		if len(s) != 12 {
			return "", fmt.Errorf("非法EID长度(%d): %q", len(s), s)
		}
		for i := 0; i < 12; i++ {
			c := s[i]
			if !('0' <= c && c <= '9' || 'A' <= c && c <= 'F') {
				return "", fmt.Errorf("EID包含非十六进制字符: %q", s)
			}
		}
		return s, nil
	}

	count := len(fields) / 4
	list := make([]config.NodeTopology, 0, count)
	for i := 0; i < count; i++ {
		j := i * 4

		eid, err := normalizeHex12(fields[j])
		if err != nil {
			return nil, fmt.Errorf("第 %d 个节点 EID 错误: %w", i, err)
		}
		parent, err := normalizeHex12(fields[j+3])
		if err != nil {
			return nil, fmt.Errorf("第 %d 个节点 Parent 错误: %w", i, err)
		}

		list = append(list, config.NodeTopology{
			EID:    eid,         // 12位HEX（大写）
			Type:   fields[j+1], // 原样保留（若需要可再做校验）
			State:  fields[j+2], // 原样保留（若需要可再做校验/映射）
			Parent: parent,      // 12位HEX（大写）
		})
	}
	return list, nil
}

func QueryAllTopology(ctx context.Context) ([]config.NodeTopology, error) {
	const pageSize = 10

	var (
		startIndex = 0
		total      = -1
		all        []config.NodeTopology
	)

	// 避免HTTP并发
	topoSessionMu.Lock()
	defer topoSessionMu.Unlock()

drain:
	for {
		select {
		case <-topoRespCh:
			// 丢弃旧的 TopoPage
		default:
			break drain
		}
	}

	for {
		// 发一页TOP命令
		SendTopoQuery(startIndex, pageSize)
		// 等这一页结果
		select {
		case page := <-topoRespCh:
			// 第一次拿到页结果时，初始化总数和预分配切片
			if total < 0 {
				total = page.Total
				// total 可能为 0（没有节点），这时 all = nil 也是合法的空结果
				all = make([]config.NodeTopology, 0, max(total, 0))
			}
			// 累加当前页节点
			all = append(all, page.Nodes...)
			// 计算下一页起始索引
			startIndex += page.Number
			// 已经取完所有节点，或者这一页 number=0，查询结束
			if startIndex >= total || page.Number == 0 {
				return all, nil
			}

		case <-ctx.Done():
			// 调用方主动取消
			return nil, ctx.Err()

		case <-time.After(3 * time.Second):
			// 单页等待超时
			return nil, fmt.Errorf("等待拓扑第 %d 页超时", startIndex/pageSize)
		}
	}
}

// 小工具
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
