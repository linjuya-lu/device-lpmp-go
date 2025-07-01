package frameparser

import (
	"log"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

func handle_frame_ctl(frameCtl config.Frame) {

	// 1. 断言拿到原始 []byte
	raw, ok := frameCtl.Payload.([]byte)
	if !ok {
		log.Printf("[CTL] payload 类型不是 []byte，而是 %T，跳过", frameCtl.Payload)
		return
	}
	if len(raw) < 1 {
		log.Printf("[CTL] payload 长度不足，跳过")
		return
	}

	// 2. 解析第一个字节：高 7 位为 CtrlType，最低位为 RequestSetFlag
	head := raw[0]

	// 解析数据
	if handle, ok := config.LookupResponseHandle(head); ok {
		err := handle.Parse(raw[1:], frameCtl)
		if err != nil {
			log.Printf("❌ 参数解析失败 head=0x%02X: %v", head, err)
			// 如果你想忽略这次，就 continue 或 return
			return
		}

	} else {
		log.Printf("未找到解析函数")
	}

}
