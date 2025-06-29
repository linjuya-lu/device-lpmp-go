package frameparser

// 封装 7.2 节 传感器通用参数查询/设置报文

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

const (

	// CtrlType: 通用参数查询/设置 (7bit)，协议附录B 定义
	ctrlTypeGeneralParams = 0x03 // TODO: 请替换为协议中实际的 CtrlType

	// 最大支持一次下发/查询的参数数量
	maxParams = 16
)

// BuildGeneralParamFrame 构造“通用参数查询/设置”报文。
//
//	sensorID:        6 字节传感器 ID
//	requestSetFlag:  0 = 查询所有参数（此时 paramsMap 应传 nil 或 empty，DataLen=0xF 且无 ParameterList）
//	                 1 = 按 paramsOrder & paramsMap 中指定的参数组合 ParameterList
//	paramsOrder:     设 requestSetFlag=1 时，按此顺序列出要查询/设置的参数名
//	paramsMap:       map[参数名]→[]byte（对应参数的数据内容）
//
// 返回：完整帧字节切片（含 CRC16）
func BuildGeneralParamFrame(sensorID [6]byte, requestSetFlag byte, paramsOrder []string, paramsMap map[string][]byte) ([]byte, error) {
	// 1. 确定 DataLen 和 ParameterList
	var dataLen byte
	var parameterList []byte

	if requestSetFlag == 0 {
		// 查询所有通用参数：DataLen=0b1111，不附带 ParameterList
		dataLen = 0x0F
	} else {
		m := len(paramsOrder)
		if m == 0 || m > maxParams {
			return nil, fmt.Errorf("参数个数必须 1~%d, got %d", maxParams, m)
		}
		dataLen = byte(m & 0x0F)

		// 构造 ParameterList: 每个参数名对应 head16(2B little-endian) + data
		buf := &bytes.Buffer{}
		for _, name := range paramsOrder {
			// 先拿到当前表中对应的 entry 副本
			entry, err := config.GetEntryCopy(name)
			if err != nil {
				return nil, err
			}
			// 再更新 entry.data 为调用者传来的值
			val, ok := paramsMap[name]
			if !ok {
				return nil, fmt.Errorf("缺少参数 %q 的值", name)
			}
			if len(val) != entry.Length {
				return nil, fmt.Errorf("参数 %q 长度错误: want %d, got %d", name, entry.Length, len(val))
			}
			entry.Data = make([]byte, entry.Length)
			copy(entry.Data, val)

			// 将 head16(小端) 写入
			le := make([]byte, 2)
			binary.LittleEndian.PutUint16(le, entry.Head16)
			buf.Write(le)
			// 将 data 写入
			buf.Write(entry.Data)
		}
		parameterList = buf.Bytes()
	}

	// 2. 构建前导头：SensorID(6B) + head(1B)
	//    head = DataLen(4b)<<4 | FragInd(1b=0)<<3 | PacketType(3b)
	head := byte((dataLen&0x0F)<<4) | byte(packetTypeControl&0x07)

	// 3. 构建 CtrlType+RequestSetFlag(1b)
	ctrlByte := byte((ctrlTypeGeneralParams&0x7F)<<1) | (requestSetFlag & 0x01)

	// 4. 汇总所有字段，准备计算 CRC
	buf := &bytes.Buffer{}
	buf.Write(sensorID[:])
	buf.WriteByte(head)
	buf.WriteByte(ctrlByte)
	if requestSetFlag == 1 {
		buf.Write(parameterList)
	}

	// 5. 计算并追加 CRC16（大端）
	crc := CRC16(buf.Bytes())
	crcb := make([]byte, 2)
	binary.BigEndian.PutUint16(crcb, crc)
	buf.Write(crcb)

	return buf.Bytes(), nil
}
