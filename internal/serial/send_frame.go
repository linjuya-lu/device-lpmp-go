package serial

import (
	"fmt"
	"io"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

// WriteFrame 保持原样，不修改参数
func WriteFrame(port io.ReadWriteCloser, frame []byte) error {
	n, err := port.Write(frame)
	if err != nil {
		return fmt.Errorf("写入串口失败：%w", err)
	}
	if n != len(frame) {
		return fmt.Errorf("写入字节数不完整：%d/%d", n, len(frame))
	}
	return nil
}

// StartWriteWorker 启动写入协程，持续从 writeChan 中读取数据并调用 WriteFrame 发送
func StartWriteWorker(port io.ReadWriteCloser) {
	go func() {
		for frame := range config.WriteChan {
			if err := WriteFrame(port, frame); err != nil {
				// 处理写错误，例如记录日志或重试
				fmt.Println("写入错误：", err)
			}
		}
	}()
}

// SendFrame 向全局通道投递数据，供写协程发送
func SendFrame(frame []byte) {
	config.WriteChan <- frame
}

// 初始化时使用示例：

//
//     // 启动写协程
//     StartWriteWorker(port)
//
//     // 发送帧示例
//     SendFrame([]byte{0xAA, 0xBB, 0xCC})
//
//     // 关闭通道
//     close(writeChan)
//
