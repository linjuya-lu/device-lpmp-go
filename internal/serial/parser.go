package serial

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
	goserial "go.bug.st/serial.v1"
)

var (
	HealthTopoMu   sync.RWMutex
	HealthTopo     []config.NodeTopology // 拓扑
	HealthTopoTime time.Time             // 时间戳
)

type topoParser struct {
	collecting bool            // 是否处在一段 +TOP ... OK 之间
	buf        strings.Builder // 节点字段临时缓冲区
	total      int             // 本次TOP报文声明的总节点数
	number     int             // 本页返回的节点数
}

func GetHealthTopology() ([]config.NodeTopology, time.Time) {
	HealthTopoMu.RLock()
	defer HealthTopoMu.RUnlock()

	if len(HealthTopo) == 0 {
		return nil, time.Time{}
	}

	cloned := make([]config.NodeTopology, len(HealthTopo))
	copy(cloned, HealthTopo)
	return cloned, HealthTopoTime
}

func Open(portName string, baudRate int) (io.ReadWriteCloser, error) {
	mode := &goserial.Mode{BaudRate: baudRate}
	return goserial.Open(portName, mode)
}

// DRX解析
func ParseDRXLine(line string) ([]byte, error) {
	// 前缀校验
	if !strings.HasPrefix(line, "+DRX:") {
		return nil, fmt.Errorf("不是 DRX 数据行：%s", line)
	}
	line = strings.TrimSpace(line[len("+DRX:"):])
	parts := strings.SplitN(line, ",", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("DRX行字段数不对：%s", line)
	}
	payload := strings.TrimSpace(parts[2])
	//字符串解码成[]byte
	data, err := hex.DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("解析 DRX payload 失败：%w", err)
	}
	return data, nil
}

// 串口解析
func StartSerialScanner(port io.Reader) {
	go func() {
		reader := bufio.NewReader(port)
		var tp topoParser
		for {
			rawLine, err := reader.ReadString('\n')
			if err != nil {
				if errors.Is(err, syscall.EINTR) {
					continue
				}
				if err != io.EOF {
					log.Printf("串口读取出错: %v", err)
				}
				break
			}
			line := strings.TrimSpace(rawLine)
			if line == "" {
				continue
			}
			// DRX处理
			if strings.HasPrefix(line, "+DRX:") {
				data, err := ParseDRXLine(line)
				if err == nil {
					config.DrxChan <- data
				}
				continue
			}
			switch {
			case strings.HasPrefix(line, "+TOP:"): // TOP 头行：+TOP:<TotalNum>,<number>,[节点字段]
				tp.collecting = true
				tp.buf.Reset()
				tp.total, tp.number = 0, 0

				header := strings.TrimSpace(line[len("+TOP:"):])
				parts := strings.SplitN(header, ",", 3) // TotalNum, number, [剩余节点字段...]
				if len(parts) < 2 {
					log.Printf("TOP头字段太少: %q", line)
					tp.collecting = false
					continue
				}
				// 解析 TotalNum, number
				total, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				number, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err1 != nil {
					log.Printf("TOP TotalNum 解析失败: %v, line=%q", err1, line)
					tp.collecting = false
					continue
				}
				if err2 != nil {
					log.Printf("TOP number 解析失败: %v, line=%q", err2, line)
					tp.collecting = false
					continue
				}
				tp.total = total
				tp.number = number
				// 部分节点字段（parts[2]）
				if len(parts) == 3 {
					payload := strings.TrimSpace(parts[2])
					if payload != "" {
						tp.buf.WriteString(payload)
					}
				}
			case tp.collecting && line == "OK": // TOP结束行：OK
				nodePart := strings.TrimSpace(tp.buf.String())
				// 解析节点段："EID,Type,State,Parent,..."
				nodes, err := parseBuffer(nodePart)
				if err != nil {
					log.Printf("TOP节点段解析失败:%v|原始=%q", err, nodePart)
					tp.collecting = false
					continue
				}
				if len(nodes) != tp.number {
					log.Printf("TOP number=%d,实际节点数=%d", tp.number, len(nodes))
				}
				page := TopoPage{
					Total:  tp.total,
					Number: tp.number,
					Nodes:  nodes,
				}
				select {
				case topoRespCh <- page: // 丢给QueryAllTopology
				default:
				}
				tp.collecting = false
			case tp.collecting: // TOP 中间行：纯节点内容
				payload := strings.TrimSuffix(line, ",")
				payload = strings.TrimSpace(payload)
				if payload != "" {
					if tp.buf.Len() > 0 {
						tp.buf.WriteString(",")
					}
					tp.buf.WriteString(payload)
				}
			default:
			}
		}
		close(config.DrxChan)
	}()
}
