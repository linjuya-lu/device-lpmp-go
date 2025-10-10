package serial

import (
	"fmt"
	"log"
	"strings"
	"sync"

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
	TopoList []config.NodeTopology
	topoMu   sync.RWMutex
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
			// 校验完整块
			if !strings.HasPrefix(line, "+TOP:") || !strings.HasSuffix(line, "OK") {
				continue
			}
			// 解析
			payload := strings.TrimSuffix(strings.TrimPrefix(line, "+TOP:"), "OK")
			nodes, err := parseBuffer(payload)
			if err != nil {
				log.Printf("拓扑内容解析失败: %v", err)
				continue
			}
			// 更新
			topoMu.Lock()
			TopoList = nodes
			topoMu.Unlock()
			log.Printf("拓扑内容解析: %+v", nodes)
		}
	}()
}

// 解析拓扑
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
