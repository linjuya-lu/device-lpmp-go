package serial

import (
	"fmt"
	"io"
	"strings"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

func WriteFrame(port io.ReadWriteCloser, frame []byte) error {
	payload := string(frame)
	fmt.Printf("发送字符串: %q\n", payload)
	n, err := port.Write([]byte(payload))
	if err != nil {
		return fmt.Errorf("写入串口失败：%w", err)
	}
	if n != len(payload) {
		return fmt.Errorf("写入字节数不完整：%d/%d", n, len(payload))
	}
	return nil
}

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
	var parts []string
	for _, b := range payload {
		parts = append(parts, fmt.Sprintf("%02X", b))
	}
	hexStr := strings.Join(parts, "")
	cmd := fmt.Sprintf("\rAT+DTX=%s,%s\r\n", dstAddr, hexStr)
	fmt.Printf(">> Sending AT command: %s", cmd)
	config.WriteChan <- []byte(cmd)
}
