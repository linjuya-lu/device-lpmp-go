package serial

import (
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

// SendTopoQuery 向全局通道投递一个 AT+TOP 拓扑查询命令。
//
//	startIndex：起始序号（从 0 开始）
//	numOfQuery：本次要查询的节点数量（一次最多 10 个）
//
// 输出命令格式：\rAT+TOP=<startIndex>,<numOfQuery>?\r\n
func SendTopoQuery(startIndex, numOfQuery int) {
	body := fmt.Sprintf("AT+TOP=%d,%d?", startIndex, numOfQuery)
	cmd := "\r" + body + "\r\n"
	fmt.Printf(">> Sending Topology Query: %s\n", body)
	config.WriteChan <- []byte(cmd)
}

// topoList 存储最新一批解析出的 NodeTopology 列表
var (
	TopoList []config.NodeTopology
	topoMu   sync.RWMutex
)

// GetTopoList 返回当前缓存
func GetTopoList() []config.NodeTopology {
	topoMu.RLock()
	defer topoMu.RUnlock()
	cloned := make([]config.NodeTopology, len(TopoList))
	copy(cloned, TopoList)
	return cloned
}

// StartTopoProcessor 从 rawCh 读取完整的 +TOP:…OK 块，直接解析并更新TopoList
func StartTopoProcessor(rawCh <-chan string) {
	go func() {
		// 一次性读取并解析完整块
		for block := range rawCh {
			line := strings.TrimSpace(block)
			// 只处理以 +TOP: 开头并以 OK 结尾的完整块
			if !strings.HasPrefix(line, "+TOP:") || !strings.HasSuffix(line, "OK") {
				continue
			}
			// 去掉前缀和尾部标志
			payload := strings.TrimSuffix(strings.TrimPrefix(line, "+TOP:"), "OK")
			// 解析 payload
			nodes, err := parseBuffer(payload)
			if err != nil {
				log.Printf("[StartTopo] 解析失败: %v", err)
				continue
			}
			// 更新
			topoMu.Lock()
			TopoList = nodes
			topoMu.Unlock()
			log.Printf("[StartTopo] 更新全局 TopoList: %+v", nodes)
		}
	}()
}

// parseBuffer 把 "E1,t1,s1,p1,E2,t2,s2,p2,..." 拆成 []NodeTopology
func parseBuffer(buf string) ([]config.NodeTopology, error) {
	buf = strings.Trim(buf, ",")
	fields := strings.Split(buf, ",")
	if len(fields)%4 != 0 {
		return nil, fmt.Errorf("字段数 %d 不是 4 的倍数", len(fields))
	}
	count := len(fields) / 4
	list := make([]config.NodeTopology, 0, count)
	for i := 0; i < count; i++ {
		j := i * 4
		list = append(list, config.NodeTopology{
			EID:    fields[j],
			Type:   fields[j+1],
			State:  fields[j+2],
			Parent: fields[j+3],
		})
	}
	return list, nil
}
