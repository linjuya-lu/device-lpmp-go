package frameparser

import "encoding/binary"

// 告警参数查询/设置报文

func BuildAlarmParameterQueryFrame(sensorID [6]byte) ([]byte, error) {
	const (
		packetType     = 0x04 // 3bit = 100b
		ctrlAlarmQuery = 0x03 // 7bit，协议中“告警参数查询”对应的 CtrlType（示例值，按文档替换）
		dataLen        = 0x0F // 4bit = 1111b, 请求所有告警参数
		fragInd        = 0    // 1bit
		requestSetFlag = 0    // 1bit = 查询
	)
	// EID
	buf := make([]byte, 0, 6+1+1+2)
	buf = append(buf, sensorID[:]...)
	//head
	head := byte((dataLen&0x0F)<<4) |
		byte((fragInd&0x01)<<3) |
		byte(packetType&0x07)
	buf = append(buf, head)
	//ctrlByte
	ctrlByte := byte((ctrlAlarmQuery&0x7F)<<1) |
		byte(requestSetFlag&0x01)
	buf = append(buf, ctrlByte)
	// CRC16
	crc := CRC16(buf)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crc)
	buf = append(buf, crcBytes...)
	return buf, nil
}
