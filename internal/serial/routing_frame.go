package serial

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

// 拓扑查询命令
func SendTopoQuery(startIndex, numOfQuery int) {
	body := fmt.Sprintf("AT+TOP=%d,%d?", startIndex, numOfQuery)
	cmd := "\r" + body + "\r\n"
	fmt.Printf("拓扑查询: %s\n", body)
	config.WriteChan <- []byte(cmd)
}

var (
	TopoList    []config.NodeTopology
	topoIndex   = map[string]int{}  // EID -> index
	topoLastAt  time.Time           // 最近合并时间
	topoIdleTTL = 600 * time.Second // 超过这个空闲视为新一轮

	topoMu sync.RWMutex
)

// 读取拓扑
func GetTopoList() []config.NodeTopology {
	topoMu.RLock()
	defer topoMu.RUnlock()
	cloned := make([]config.NodeTopology, len(TopoList))
	copy(cloned, TopoList)
	return cloned
}

// 拓扑内容解析
func StartTopoProcessor(rawCh <-chan string) {
	go func() {
		for block := range rawCh {
			line := strings.TrimSpace(block)
			if line == "" {
				continue
			}

			// 截取最后一次出现的 "+TOP:" 与其后的 "OK"
			start := strings.LastIndex(line, "+TOP:")
			end := strings.LastIndex(line, "OK")
			if start == -1 || end == -1 || end <= start+len("+TOP:") {
				continue
			}

			// 取出负载，形如：EID,Type,State,Parent,EID,Type,State,Parent,...
			payload := strings.TrimSpace(line[start+len("+TOP:") : end])

			nodes, err := parseBuffer(payload)
			if err != nil {
				log.Printf("拓扑内容解析失败: %v | 原始: %q", err, payload)
				continue
			}

			// 如果这类文本块代表“完整快照”，可以直接覆盖；
			// 若你希望按 EID 增量合并，改成：snapshot := mergeTopo(nodes)
			topoMu.Lock()
			TopoList = nodes
			topoMu.Unlock()

			log.Printf("拓扑内容解析成功: %d 节点", len(nodes))
		}
	}()
}

// 解析拓扑：把 "EID,Type,State,Parent,..." 文本转成结构体切片
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

	// 小工具：把 EID/Parent 统一成 12 位大写十六进制（移除分隔符）
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

// 清空但保留底层容量
func ClearTopo() (prev int) {
	topoMu.Lock()
	prev = len(TopoList)
	TopoList = TopoList[:0] // 只清长度，保留容量
	topoIndex = make(map[string]int)
	topoLastAt = time.Time{} // 清掉时间戳
	topoMu.Unlock()
	return
}
