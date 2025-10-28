package frameparser

import (
	"encoding/binary"
)

// 工况查询
func BuildMonitoringDataQueryFrame(sensorID [6]byte) ([]byte, error) {
	const (
		packetType       = 0x04 // 100b
		ctrlTypeMonitor  = 0x02 // 控制类型
		dataLenAllParams = 0x0F // 1111b, 所有可采集参数
		fragInd          = 0    // 未分片
		requestSetFlag   = 0    // 查询
	)
	// EID
	buf := make([]byte, 0, 6+1+1+2)
	buf = append(buf, sensorID[:]...)
	// 头
	head := byte((dataLenAllParams&0x0F)<<4) |
		byte((fragInd&0x01)<<3) |
		byte(packetType&0x07)
	buf = append(buf, head)
	// 控制
	ctrlByte := byte((ctrlTypeMonitor&0x7F)<<1) |
		byte(requestSetFlag&0x01)
	buf = append(buf, ctrlByte)
	// 校验
	crc := CRC16(buf)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crc)
	buf = append(buf, crcBytes...)
	return buf, nil
}
