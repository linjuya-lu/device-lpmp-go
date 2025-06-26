package frameparser

import (
	"encoding/binary"
	"fmt"
	"log"
)

// FrameCtl 代表“传感器监测数据查询报文”
type FrameCtl struct {
	SensorID   string      // 传感器 ID，6 字节
	DataLen    int         // 参量个数，使用下位 4 位即可，或者直接用 uint32 存放 m
	FragInd    byte        // 分片指示，true=已分片, false=未分片
	PacketType byte        // 报文类型，3 字节，例：0x00,0x01,0x00 表示类型 100
	Payload    interface{} // 报文内容，接收端可根据 PacketType 做类型断言
	Check      uint16      // 校验位，2 字节 CRC
}

// ControlReportContent 表示“控制报文”中的子层内容
type ControlReportContent struct {
	// 控制报文类型：只用低 7 位
	CtrlType uint8
	// 参数配置类型标识：1 bit，0/1
	RequestSetFlag bool
	// 额外的业务负载，任意类型都可以往里存
	Payload interface{}
}

func handle_frame_ctl(frameCtl FrameCtl) {
	return

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
	ctrlType := head >> 1
	requestSet := (head & 0x1) == 1

	// 3. 剩余部分按 2 字节一对解析成参数类型列表
	//    协议说有 m 个类型码，每个 2 字节
	m := frameCtl.DataLen

	avail := len(raw) - 1 // 真正可用字节数
	count := avail / 2    // 能解析多少对
	if count > m {
		count = m
	}
	typeList := make([]uint16, count)
	for i := 0; i < count; i++ {
		off := 1 + i*2
		typeList[i] = binary.BigEndian.Uint16(raw[off : off+2])
	}

	// 4. 填充子层结构体
	content := ControlReportContent{
		CtrlType:       ctrlType,
		RequestSetFlag: requestSet,
		Payload:        typeList, // 也可以直接存 raw[1:]
	}

	// 5. 后续业务处理，分发
	fmt.Printf("[CTL] 解析到控制报文子层: CtrlType=%d, RequestSet=%t, TypeList=%v\n",
		content.CtrlType, content.RequestSetFlag, content.Payload)
}
