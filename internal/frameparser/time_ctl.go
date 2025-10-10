package frameparser

// 时间参数查询/设置报文

import (
	"encoding/binary"
	"fmt"

	"github.com/linjuya-lu/device-lpmp-go/internal/serial"
)

const packetTypeControl = 0x04
const ctrlTypeTimeParam = 0x04

// 构造“时间参数查询/设置”控制报文
func BuildTimeParamFrame(sensorID [6]byte, requestSetFlag byte, timestamp uint32) ([]byte, error) {
	if requestSetFlag != 0 && requestSetFlag != 1 {
		return nil, fmt.Errorf("invalid requestSetFlag %d, must be 0 or 1", requestSetFlag)
	}
	buf := make([]byte, 0, 6+1+1+4+2)
	// EID
	buf = append(buf, sensorID[:]...)
	//head
	head := byte(0<<4) | byte(0<<3) | byte(packetTypeControl&0x07)
	buf = append(buf, head)
	//CtrlType+RequestSetFlag
	ctrlByte := byte((ctrlTypeTimeParam&0x7F)<<1) | (requestSetFlag & 0x01)
	buf = append(buf, ctrlByte)
	// 查询时 timestamp=0；设置时请传入需要下发的世纪秒
	tsBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(tsBytes, timestamp)
	buf = append(buf, tsBytes...)
	//CRC16 校验位
	crc := CRC16(buf)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crc)
	buf = append(buf, crcBytes...)
	return buf, nil
}

func RestCommandBuildFrame(eidStr string, sensorID [6]byte, requestSetFlag byte, timestamp uint32) error {
	if requestSetFlag != 0 && requestSetFlag != 1 {
		return fmt.Errorf("invalid requestSetFlag %d, must be 0 or 1", requestSetFlag)
	}
	// 预分配
	buf := make([]byte, 0, 6+1+1+4+2)
	// EID
	buf = append(buf, sensorID[:]...)
	// head
	head := byte(0<<4) | byte(0<<3) | byte(packetTypeControl&0x07)
	buf = append(buf, head)
	// CtrlType+RequestSetFlag
	ctrlByte := byte((ctrlTypeTimeParam&0x7F)<<1) | (requestSetFlag & 0x01)
	buf = append(buf, ctrlByte)
	// Timestamp
	tsBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(tsBytes, timestamp)
	buf = append(buf, tsBytes...)
	// CRC16 校验位
	crc := CRC16(buf)
	crcBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(crcBytes, crc)
	buf = append(buf, crcBytes...)
	// 发送帧
	serial.SendFrame(eidStr, buf)

	return nil
}
