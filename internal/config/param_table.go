package config

//附录D表
import (
	"encoding/binary"
	"errors"
)

// Entry 表示一个参数在报文中的完整字段（不含后面的 CRC、帧头等）
// 它只包含：
// 1) head16：14bit 参数类型 + 2bit 长度指示位，按小端序写入报文时就是这 2 字节原样；
// 2) data：真正的参数内容，长度固定，由 lengthFlag 决定。
type Entry struct {
	head16 uint16 // (ParameterType<<2 | LengthFlag), 小端序存储到报文字段
	length int    // DataLen：0→4, 1→1, 2→2, 3→3 字节
	data   []byte // 参数的可变内容
}

// 全局表：参数名 → *Entry
var (
	table = map[string]*Entry{
		// 以下举例：假设有两个参数 "Temperature" 和 "Humidity"
		// 它们在协议里定义的 ParameterType 和 LengthFlag 已知：
		//  Temperature: 类型码 0x0005, 长度标志 0 → 数据固定 4 字节
		//  Humidity:    类型码 0x0009, 长度标志 1 → 数据固定 1 字节
		"Temperature": {
			head16: binary.LittleEndian.Uint16([]byte{0b00000101<<2 | 0b00, 0}), // (0x0005<<2)|0
			length: 4,
			data:   make([]byte, 4),
		},
		"Humidity": {
			head16: binary.LittleEndian.Uint16([]byte{0b00001001<<2 | 0b01, 0}), // (0x0009<<2)|1
			length: 1,
			data:   make([]byte, 1),
		},
		// 按照你的协议表继续添加……
	}
)

// UpdateData 用于并发安全地更新某个参数的 data 内容
// 要求 len(value) == entry.length，否则报错；
// data 会被完整拷贝到内部存储。
func UpdateData(name string, value []byte) error {
	mu.Lock()
	defer mu.Unlock()

	e, ok := table[name]
	if !ok {
		return errors.New("unknown parameter: " + name)
	}
	if len(value) != e.length {
		return errors.New("invalid data length for " + name)
	}
	// 拷贝到内部
	copy(e.data, value)
	return nil
}

// GetPacketFields 返回当前全量“头域+数据域”组合后的字节切片副本，map[key]=[]byte{head16_lo, head16_hi, ...data}
// head16 按小端序存储在前面 2 字节，后面紧跟 data。
func GetPacketFields() map[string][]byte {
	mu.RLock()
	defer mu.RUnlock()

	out := make(map[string][]byte, len(table))
	for name, e := range table {
		buf := make([]byte, 2+e.length)
		// 2 字节小端序 head16
		binary.LittleEndian.PutUint16(buf[0:2], e.head16)
		// 紧跟 data
		copy(buf[2:], e.data)
		out[name] = buf
	}
	return out
}

// GetEntryCopy 返回某个参数的当前 Entry 副本，包含 head16、length 和 data 副本
func GetEntryCopy(name string) (Entry, error) {
	mu.RLock()
	defer mu.RUnlock()

	e, ok := table[name]
	if !ok {
		return Entry{}, errors.New("unknown parameter: " + name)
	}
	dataCopy := make([]byte, len(e.data))
	copy(dataCopy, e.data)
	return Entry{
		head16: e.head16,
		length: e.length,
		data:   dataCopy,
	}, nil
}
