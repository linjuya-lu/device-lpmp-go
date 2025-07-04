package serial

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

// Mock 从串口读到的 +TOP 响应。你需要替换成真正的读串口逻辑。
// 返回值格式举例：
// "+TOP:3,2,\n238A08200002,1,1,238A08201000,\n238A08200003,2,0,238A08201000\nOK"
func readTopoResponse() (string, error) {
	// … 实际应该从串口读一段完整的文本 …
	return "", nil
}

// parseTopoResponse 解析一次 +TOP 响应文本，返回总条目数、这批返回的节点列表
func parseTopoResponse(resp string) (total int, nodes []config.NodeTopology, err error) {
	lines := strings.Split(strings.TrimSpace(resp), "\n")
	// 第一行：+TOP:TotalNum,number,
	header := strings.TrimPrefix(lines[0], "+TOP:")
	parts := strings.Split(strings.TrimRight(header, ","), ",")
	if len(parts) < 2 {
		return 0, nil, fmt.Errorf("解析 header 失败: %s", lines[0])
	}
	total, err = strconv.Atoi(parts[0])
	if err != nil {
		return
	}
	number, err := strconv.Atoi(parts[1])
	if err != nil {
		return
	}
	// 接下来 number 行，每行一个记录
	for i := 1; i <= number && i < len(lines); i++ {
		cols := strings.Split(strings.TrimRight(lines[i], ","), ",")
		if len(cols) < 4 {
			continue
		}
		t, _ := strconv.Atoi(cols[1])
		s, _ := strconv.Atoi(cols[2])
		nodes = append(nodes, config.NodeTopology{
			EID:    cols[0],
			Type:   t,
			State:  s,
			Parent: cols[3],
		})
	}
	return
}

// QueryAllTopology 批量拉取并返回整个网络的拓扑表
func QueryAllTopology(batchSize int) ([]config.NodeTopology, error) {
	var result []config.NodeTopology
	startIndex := 0
	totalNum := -1

	for {
		// 1. 发送查询命令
		SendTopoQuery(startIndex, batchSize)
		// 2. 等待响应（这里简单 sleep，实际应阻塞读串口）
		time.Sleep(200 * time.Millisecond)

		resp, err := readTopoResponse()
		if err != nil {
			return nil, fmt.Errorf("读取拓扑响应失败: %w", err)
		}
		total, nodes, err := parseTopoResponse(resp)
		if err != nil {
			return nil, fmt.Errorf("解析拓扑响应失败: %w", err)
		}
		// 第一次读取时记录总数
		if totalNum < 0 {
			totalNum = total
		}
		// 累加结果
		result = append(result, nodes...)
		// 如果已拿齐或超出，就跳出
		if len(result) >= totalNum {
			break
		}
		// 否则继续下一批
		startIndex += batchSize
	}
	return result, nil
}

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

// nodes, err := topo.QueryAllTopology(10)
// if err != nil {
//     log.Fatalf("拉取网络拓扑失败: %v", err)
// }
// for _, n := range nodes {
//     fmt.Printf("EID=%s type=%d state=%d parent=%s\n",
//         n.EID, n.Type, n.State, n.Parent)
// }
