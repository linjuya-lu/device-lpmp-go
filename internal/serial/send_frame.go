package serial

import (
	"fmt"
	"io"
	"strings"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

func WriteFrame(port io.ReadWriteCloser, frame []byte) error {
	// 转成字符串
	payload := string(frame)
	// 调试
	fmt.Printf(">> 发送字符串: %q\n", payload)
	// 发送
	n, err := port.Write([]byte(payload))
	if err != nil {
		return fmt.Errorf("写入串口失败：%w", err)
	}
	if n != len(payload) {
		return fmt.Errorf("写入字节数不完整：%d/%d", n, len(payload))
	}
	return nil
}

// StartWriteWorker 启动写入协程，持续从 writeChan 中读取数据并调用 WriteFrame 发送
func StartWriteWorker(port io.ReadWriteCloser) {
	go func() {
		for frame := range config.WriteChan {
			if err := WriteFrame(port, frame); err != nil {
				fmt.Println("写入错误：", err)
			}
		}
	}()
}

func SendFrame(dstAddr string, payload []byte) {
	// 逐字节格式化
	var parts []string
	for _, b := range payload {
		parts = append(parts, fmt.Sprintf("%02X", b))
	}
	hexStr := strings.Join(parts, "") // 合并字符串
	// AT 命令
	cmd := fmt.Sprintf("\rAT+DTX=%s,%s\r\n", dstAddr, hexStr)
	// 调试
	fmt.Printf(">> Sending AT command: %s", cmd)
	// 发送
	config.WriteChan <- []byte(cmd)
}

// 初始化时使用示例：
//
//     // 启动写协程
//     StartWriteWorker(port)
//
//     // 关闭通道
//     close(writeChan)
//
