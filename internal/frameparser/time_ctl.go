package frameparser

import (
	"encoding/binary"
	"fmt"

	"github.com/linjuya-lu/device-lpmp-go/internal/serial"
)

const (
	packetTypeControl = 0x04
	ctrlTypeTimeParam = 0x04
)

// 时间查询
func BuildTimeParamFrame(sensorID [6]byte, requestSetFlag byte, timestamp uint32) ([]byte, error) {
	if requestSetFlag != 0 && requestSetFlag != 1 {
		return nil, fmt.Errorf("无效参数 %d, 应为0或1", requestSetFlag)
	}
	buf := make([]byte, 0, 6+1+1+4+2)
	// EID
	buf = append(buf, sensorID[:]...)
	//头
	head := byte(0<<4) | byte(0<<3) | byte(packetTypeControl&0x07)
	buf = append(buf, head)
	//控制部分
	ctrlByte := byte((ctrlTypeTimeParam&0x7F)<<1) | (requestSetFlag & 0x01)
	buf = append(buf, ctrlByte)
	tsBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(tsBytes, timestamp)
	buf = append(buf, tsBytes...)
	//校验
	crc := CRC16(buf)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crc)
	buf = append(buf, crcBytes...)
	return buf, nil
}

// 时间同步
func RestCommandBuildFrame(eidStr string, sensorID [6]byte, requestSetFlag byte, timestamp uint32) error {
	if requestSetFlag != 0 && requestSetFlag != 1 {
		return fmt.Errorf("无效参数 %d, 应为0或1", requestSetFlag)
	}
	buf := make([]byte, 0, 6+1+1+4+2)
	// EID
	buf = append(buf, sensorID[:]...)
	// 头
	head := byte(0<<4) | byte(0<<3) | byte(packetTypeControl&0x07)
	buf = append(buf, head)
	// 控制部分
	ctrlByte := byte((ctrlTypeTimeParam&0x7F)<<1) | (requestSetFlag & 0x01)
	buf = append(buf, ctrlByte)
	// 时间戳
	tsBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(tsBytes, timestamp)
	buf = append(buf, tsBytes...)
	// 校验
	crc := CRC16(buf)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crc)
	buf = append(buf, crcBytes...)
	serial.SendFrame(eidStr, buf)

	return nil
}
