// WriteFrame 往已打开的串口写出一帧数据
package serial

import (
	"fmt"
	"io"
)

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
