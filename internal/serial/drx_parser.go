package serial

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"syscall"

	goserial "go.bug.st/serial.v1"
)

// 打开串口
func Open(portName string, baudRate int) (io.ReadWriteCloser, error) {
	mode := &goserial.Mode{BaudRate: baudRate}
	return goserial.Open(portName, mode)
}

// 解析+DRX
func ParseDRXLine(line string) ([]byte, error) {
	// 处理 +DRX:
	if !strings.HasPrefix(line, "+DRX:") {
		return nil, fmt.Errorf("不是 DRX 数据行：%s", line)
	}
	// 分割成三部分：prefix、length、payload
	parts := strings.SplitN(line, ",", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("DRX 行字段数不对：%s", line)
	}
	payload := parts[2]
	if len(payload)%2 != 0 {
		return nil, fmt.Errorf("payload 长度不是偶数：%s", payload)
	}
	// 解码
	n := len(payload) / 2
	buf := make([]byte, n)
	for i := 0; i < n; i++ {
		hexByte := payload[i*2 : i*2+2]
		v, err := strconv.ParseUint(hexByte, 16, 8)
		if err != nil {
			return nil, fmt.Errorf("解析 hex %s 失败：%w", hexByte, err)
		}
		buf[i] = byte(v)
	}
	return buf, nil
}

// DRX数据通道
var DrxChan = make(chan []byte, 100)

// TOP原始行通道
var TopoChan = make(chan string, 100)

// Lora初步解析
func StartSerialScanner(port io.Reader) {
	go func() {
		reader := bufio.NewReader(port)
		var (
			collectingTopo bool
			topoBuf        strings.Builder
		)
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
			// DRX 处理
			if strings.HasPrefix(line, "+DRX:") {
				data, err := ParseDRXLine(line)
				if err == nil {
					DrxChan <- data
				}
				continue
			}
			// TOP处理
			switch {
			case strings.HasPrefix(line, "+TOP:"):
				collectingTopo = true
				topoBuf.Reset()
				parts := strings.SplitN(line[len("+TOP:"):], ",", 3)
				if len(parts) >= 3 {
					topoBuf.WriteString(parts[2])
				}
			case collectingTopo && line == "OK":
				block := "+TOP:" + topoBuf.String() + "OK"
				TopoChan <- block
				collectingTopo = false
			case collectingTopo:
				payload := strings.TrimSuffix(line, ",")
				topoBuf.WriteString("," + payload)
			default:
			}
		}
		close(TopoChan)
		close(DrxChan)
	}()
}
