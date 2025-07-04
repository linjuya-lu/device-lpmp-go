package frameparser

// 封装 7.3 节 监测参数查询/设置报文

import (
	"encoding/binary"
)

// BuildMonitoringDataQueryFrame 构造 “传感器监测数据查询” 控制报文。
//
//	sensorID: 原始 6 字节传感器 ID。
//
// 返回值：含 CRC16 的完整报文字节 slice，或出错。
func BuildMonitoringDataQueryFrame(sensorID [6]byte) ([]byte, error) {
	const (
		packetType       = 0x04 // 3bit = 100b
		ctrlTypeMonitor  = 0x02 // 7bit，协议中“请求监测数据”对应的 CtrlType
		dataLenAllParams = 0x0F // 4bit = 1111b, 表示请求所有可采集参数
		fragInd          = 0    // 1bit，未分片
		requestSetFlag   = 0    // 1bit，查询
	)

	// 1. 拼前 6 字节 SensorID
	buf := make([]byte, 0, 6+1+1+2)
	buf = append(buf, sensorID[:]...)

	// 2. 拼 head：DataLen(4) | FragInd(1) | PacketType(3)
	head := byte((dataLenAllParams&0x0F)<<4) |
		byte((fragInd&0x01)<<3) |
		byte(packetType&0x07)
	buf = append(buf, head)

	// 3. 拼 ctrlByte：CtrlType(7) | RequestSetFlag(1)
	ctrlByte := byte((ctrlTypeMonitor&0x7F)<<1) |
		byte(requestSetFlag&0x01)
	buf = append(buf, ctrlByte)

	// （不带 TypeList，因为请求所有参数）

	// 4. 计算 CRC16 并以大端序追加 2 字节
	crc := CRC16(buf)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crc)
	buf = append(buf, crcBytes...)

	return buf, nil
}
