package frameparser

import (
	"encoding/binary"
	"encoding/hex"
	"log"
	"strings"

	"github.com/linjuya-lu/device-lpmp-go/internal/config"
)

// StartParser 从 frameCh 通道中持续读取完整帧，启动一个后台协程进行业务数据解析。
// 依照《Q/GDW 12184—2021》附录 D 业务报文格式，实现以下功能：
// 1. 提取 SensorID、报文类型（仅处理业务数据：监测和告警）
// 2. 根据 DataLen（4bit）、FragInd（1bit）、PacketType（3bit）判断是否处理
// 3. 跳过分片帧（FragInd=1），不拼接，仅打印提示
// 4. 按照参量个数逐个解析 ParamType(14bit)+LengthFlag(2bit) + 可选长度字段 + 数据
// 5. 将数值按表大端转换为 float32/float64/int8等基本类型
// 6. 针对已知 SensorID（如"238A08262319"水位传感器），调用 config.SetDeviceValue 存储解析结果
// 7. 异常或格式不符时跳过本帧，确保解析循环不中断
func StartParser(frameCh <-chan []byte) {
	go func() {
		for frame := range frameCh {
			// 最小长度校验：6字节ID +1字节头 +2字节CRC
			if len(frame) < 9 {
				log.Println("帧长度不足，跳过解析")
				continue
			}
			// CRC 校验：最后 2 字节为 CRC-16
			payload := frame[:len(frame)-2]
			recvCRC := binary.BigEndian.Uint16(frame[len(frame)-2:])
			if CRC16(payload) != recvCRC {
				log.Println("CRC 校验失败，跳过解析")
				continue
			}
			// 1. 读取6字节SensorID，使用Hex字符串表示
			sidBytes := frame[0:6]
			sensorID := strings.ToUpper(hex.EncodeToString(sidBytes))
			deviceName, hasDevice := config.LookupDeviceName(sensorID)
			if !hasDevice {
				log.Printf("未知 SensorID=%s，跳过本帧", sensorID)
				continue
			}
			// 2. 读取头部：4bit DataLen、1bit FragInd、3bit PacketType
			head := frame[6]
			dataCount := int(head >> 4)  // 参量个数
			fragInd := (head >> 3) & 0x1 // 分片指示
			packetType := head & 0x07    // 报文类型

			// 只处理业务数据报文（监测=0、告警=2）
			if packetType != 0 && packetType != 2 {
				continue
			}

			// 分片帧不拼接，仅打印提示并跳过
			if fragInd == 1 {
				log.Printf("检测到分片帧 SensorID=%s，暂不拼接，跳过解析", sensorID)
				continue
			}

			// 3. 从第7字节开始解析参数数据，末尾2字节为CRC
			idx := 7
			parsed := 0
			for parsed < dataCount {
				// 参数头2字节
				if idx+2 > len(frame)-2 {
					log.Printf("参数头越界 SensorID=%s，跳过本帧", sensorID)
					break
				}
				head16 := binary.BigEndian.Uint16(frame[idx : idx+2])
				idx += 2
				paramType := head16 >> 2       // 14bit类型码
				lenFlag := uint8(head16 & 0x3) // 2bit长度指示

				// 计算真实数据长度
				var dataLen uint32
				switch lenFlag {
				case 0:
					dataLen = 4 // 默认4字节
				case 1:
					dataLen = uint32(frame[idx])
					idx++
				case 2:
					dataLen = uint32(binary.BigEndian.Uint16(frame[idx : idx+2]))
					idx += 2
				case 3:
					dataLen = uint32(frame[idx])<<16 | uint32(frame[idx+1])<<8 | uint32(frame[idx+2])
					idx += 3
				}

				// 数据越界校验
				if idx+int(dataLen) > len(frame)-2 {
					log.Printf("参数数据越界 SensorID=%s，跳过本帧", sensorID)
					break
				}

				// 提取原始值字节
				valBytes := frame[idx : idx+int(dataLen)]
				idx += int(dataLen)

				// 解析数据
				if info, ok := config.LookupParamInfo(head16); ok {
					val, err := info.Parse(valBytes)
					if err != nil {
						log.Printf("❌ 参数 %s.%s 解析失败: %v", deviceName, info.Name, err)
					} else {
						// 写入运行时值表
						config.SetDeviceValue(deviceName, info.Name, val)
						log.Printf("✅ 写入值 %s.%s = %v %s", deviceName, info.Name, val, info.Unit)
					}
				} else {
					log.Printf("未找到参数类型信息 type=0x%X", paramType)
				}

				parsed++
			}

			// 若未完全解析，跳过后续逻辑
			if parsed < dataCount {
				continue
			}
		}
	}()
}
