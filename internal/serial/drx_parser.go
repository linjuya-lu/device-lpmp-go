// Package serial 提供对 AT+DRX 串口响应的解析功能，
// 将 ASCII 格式的接收指令转为二进制帧。
package serial

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"

	goserial "go.bug.st/serial.v1"
)

// Open 打开一个串口，并以 io.ReadWriteCloser 的形式返回
func Open(portName string, baudRate int) (io.ReadWriteCloser, error) {
	mode := &goserial.Mode{BaudRate: baudRate}
	return goserial.Open(portName, mode)
}

// ParseDRXLine 解析一行形如 "+DRX:<deviceId>,<length>,<hexPayload>"
// 的串口输出，提取出 hexPayload 并将其解码为字节切片。
// 例如："+DRX:238A08262319,3,111111" → []byte{0x11,0x11,0x11}
func ParseDRXLine(line string) ([]byte, error) {
	// 只处理以 +DRX: 开头的行
	if !strings.HasPrefix(line, "+DRX:") {
		return nil, fmt.Errorf("不是 DRX 数据行：%s", line)
	}
	// 分割成三部分：prefix、length、payload
	parts := strings.SplitN(line, ",", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("DRX 行字段数不对：%s", line)
	}
	payload := parts[2]
	// payload 必须是偶数长度，每两个字符表示一个字节
	if len(payload)%2 != 0 {
		return nil, fmt.Errorf("payload 长度不是偶数：%s", payload)
	}
	// 解码 hexPayload
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

// DRXReader 从 io.Reader 按行读取串口输出，过滤 +DRX 响应，
// 并将 payload 解码后通过 ReadFrame 返回。
// ReadFrame 会阻塞直到读取到下一条完整 DRX 行或遇到 io.EOF / 错误。
type DRXReader struct {
	s *bufio.Scanner
}

// NewDRXReader 创建一个 DRXReader，对给定的 io.Reader 进行封装
func NewDRXReader(r io.Reader) *DRXReader {
	return &DRXReader{s: bufio.NewScanner(r)}
}

// ReadFrame 读取下一条 DRX 响应，返回解码后的字节切片
func (r *DRXReader) ReadFrame() ([]byte, error) {
	for r.s.Scan() {
		line := r.s.Text()
		if !strings.HasPrefix(line, "+DRX:") {
			continue
		}
		data, err := ParseDRXLine(line)
		if err != nil {
			// 出错也跳过本行，继续读取下一行
			continue
		}
		return data, nil
	}
	if err := r.s.Err(); err != nil {
		return nil, err
	}
	return nil, io.EOF
}

// StartDRXListener 启动一个 goroutine，从 io.Reader 读取 AT+DRX 响应帧，
// 并将解码后的二进制帧推送到 frameCh。
// 调用示例（在初始化时）：
//
//	frameCh := make(chan []byte, 100)
//	serial.StartDRXListener(port, frameCh)
//
// 后续可在其他协程中：
//
//	for frame := range frameCh {
//	    // 处理 frame
//	}
func StartDRXListener(port io.Reader, frameCh chan<- []byte) {
	go func() {
		r := NewDRXReader(port)
		for {
			frame, err := r.ReadFrame()
			if err != nil {
				if err == io.EOF {
					close(frameCh)
					return
				}
				// 解析错误或临时错误，跳过本次
				continue
			}
			frameCh <- frame
		}
	}()
}
