package frameparser

import (
	"sync"
	"time"
)

// Frame 表示数据帧的结构，假设已有定义。
// 其中包含SensorID（6字节）、FragInd（是否为分片帧指示）、SSEQ（业务单元序号）、
// PSEQ（分片序号）、Flag（片段标志）、Data（负载数据）等字段。
type Frame struct {
	SensorID [6]byte // 传感器ID，6字节唯一标识传感器
	FragInd  uint8   // 分片指示: 1表示分片帧, 0表示完整帧
	SSEQ     uint8   // 业务单元序号 (6 bit有效位, 这里用byte表示0-63范围的值)
	PSEQ     uint8   // 分片序号 (7 bit有效位, 0-127范围)
	Flag     uint8   // 片段标志 (2 bit有效位: 00首片, 10中间片, 11尾片)
	Data     []byte  // 帧的有效载荷数据
	// 其他帧头字段如帧类型、长度等根据协议需要可加入
}

// SDUCache 结构保存正在拼接的某个传感器的一条SDU信息
type SDUCache struct {
	SSEQ        uint8            // 当前正在拼装的业务单元序号
	expectedSeq uint8            // 下一个期望收到的PSEQ序号
	finalSeq    uint8            // 最后尾片的序号（如果已知的话），0表示暂未确定
	dataBuffer  []byte           // 已接收片段的累计数据
	outOfOrder  map[uint8][]byte // 临时保存的乱序片段: key是PSEQ序号, value是该片段数据
	timer       *time.Timer      // 超时定时器，用于超时未完成时清理
}

// 全局缓存map: 按SensorID区分的SDUCache
var (
	sduCacheMap = make(map[[6]byte]*SDUCache)
	cacheMu     sync.Mutex
	// 这个通道用来把重组/未分片的 Frame 推给 StartParser 或上层逻辑
	FrameCh = make(chan *Frame, 100)
)

// 可配置的拼接超时时间，默认20秒
var reassembleTimeout = 20 * time.Second

// ProcessFrame 处理收到的单帧数据，根据是否分片进行缓存或直接解析
// 若非分片帧 (FragInd != 1)，直接通过通道发送，不进入缓存流程。
// 若是分片帧，根据是否已有缓存及片段类型分别处理：
// 首片处理： 创建新的缓存结构，初始化期望序号和数据缓冲，并启动超时定时器
// 重复首片或新消息首片冲突： 如已存在缓存，遇到新的首片，根据 SSEQ 判定是同一消息的重发还是新的消息开始，从而决定是重置当前缓存重新开始，还是丢弃旧缓存转入新消息的拼接。
// 中间/尾片处理： 检查 PSEQ 与期望序号的关系，采取顺序拼接、乱序暂存或重复忽略等措施，确保数据按序整合。收到尾片时记录最后序号，在确定所有片段齐全后进行最终拼装。
func ProcessFrame(frame *Frame) {
	// 如果不是分片帧，直接转发给下一阶段解析
	if frame.FragInd != 1 {
		FrameCh <- frame
		return
	}

	cacheMu.Lock() // 加锁保护全局缓存访问
	defer cacheMu.Unlock()

	// 获取该传感器对应的缓存（如果存在）
	sensorID := frame.SensorID
	sduCache, exists := sduCacheMap[sensorID]

	// 帧是分片帧的情况：
	if !exists {
		// 当前没有该传感器的缓存，表示这是新收到的分片数据
		if isFlagFirst(frame.Flag) {
			// 是首片，则创建新的SDUCache进行缓存
			sduCache = &SDUCache{
				SSEQ:        frame.SSEQ,
				expectedSeq: frame.PSEQ, // 首片的PSEQ通常为起始序号
				finalSeq:    0,          // 还未确定最后片序号
				dataBuffer:  make([]byte, 0),
				outOfOrder:  make(map[uint8][]byte),
			}
			// 缓存首片数据并更新期望下一个序号
			appendFragmentData(sduCache, frame.PSEQ, frame.Data)
			sduCache.expectedSeq = frame.PSEQ + 1

			// 启动超时定时器
			startReassembleTimer(sensorID, sduCache)
			// 将缓存保存到全局map
			sduCacheMap[sensorID] = sduCache

			// 检查该片是否同时也是尾片（首片==尾片的特殊情况）
			if isFlagLast(frame.Flag) {
				finalizeAndOutput(sensorID, sduCache)
			}
		} else {
			// 没有缓存且收到的不是首片，无法处理该片段（可能缺少前序片段）
			// 丢弃该片段（可记录警告日志）
			return
		}
	} else {
		// 已有该传感器的缓存正在拼接
		// 检查SSEQ是否匹配当前缓存的业务单元
		if frame.SSEQ != sduCache.SSEQ {
			// 收到不同SSEQ的分片帧
			if isFlagFirst(frame.Flag) {
				// 如果新来的帧是一个新的首片（新的消息开始）
				// 释放旧的未完成缓存，开始新的拼接
				cancelReassembleTimer(sduCache) // 停止旧定时器
				delete(sduCacheMap, sensorID)   // 删除旧缓存
				// 可在此记录日志: 丢弃旧SSEQ未完成的拼接数据

				// 使用新帧的信息创建新的缓存
				newCache := &SDUCache{
					SSEQ:        frame.SSEQ,
					expectedSeq: frame.PSEQ,
					finalSeq:    0,
					dataBuffer:  make([]byte, 0),
					outOfOrder:  make(map[uint8][]byte),
				}
				appendFragmentData(newCache, frame.PSEQ, frame.Data)
				newCache.expectedSeq = frame.PSEQ + 1
				startReassembleTimer(sensorID, newCache)
				sduCacheMap[sensorID] = newCache
				sduCache = newCache

				// 如果新首片同时也是尾片，则直接完成拼接输出
				if isFlagLast(frame.Flag) {
					finalizeAndOutput(sensorID, newCache)
				}
			} else {
				// 收到一个不属于当前缓存SSEQ的片段且不是新的首片，无法拼接，丢弃
				return
			}
		} else {
			// SSEQ匹配当前缓存，继续拼接流程
			if isFlagFirst(frame.Flag) {
				// 收到重复的首片（可能是发送端重传），重启拼接
				cancelReassembleTimer(sduCache) // 停止当前定时器
				delete(sduCacheMap, sensorID)   // 移除当前缓存
				// 创建新缓存（使用当前帧覆盖旧数据）
				newCache := &SDUCache{
					SSEQ:        frame.SSEQ,
					expectedSeq: frame.PSEQ,
					finalSeq:    0,
					dataBuffer:  make([]byte, 0),
					outOfOrder:  make(map[uint8][]byte),
				}
				appendFragmentData(newCache, frame.PSEQ, frame.Data)
				newCache.expectedSeq = frame.PSEQ + 1
				startReassembleTimer(sensorID, newCache)
				sduCacheMap[sensorID] = newCache
				sduCache = newCache

				// 检查是否同时为尾片
				if isFlagLast(frame.Flag) {
					finalizeAndOutput(sensorID, newCache)
				}
			} else {
				// 正常的中间片或尾片
				// 检查片段序号是否为期望的下一序号
				if frame.PSEQ < sduCache.expectedSeq {
					// 收到重复或过期的片段，直接忽略
					return
				}
				if frame.PSEQ > sduCache.expectedSeq {
					// 缺少中间片段，此片段超前了，将其暂存于乱序缓存
					sduCache.outOfOrder[frame.PSEQ] = frame.Data
					// 如果此片段是尾片，记录最后片序号
					if isFlagLast(frame.Flag) {
						sduCache.finalSeq = frame.PSEQ
					}
					return // 先返回，等待缺失的片段到达或超时
				}
				if frame.PSEQ == sduCache.expectedSeq {
					// 按顺序收到正确的下一片段
					appendFragmentData(sduCache, frame.PSEQ, frame.Data)
					sduCache.expectedSeq++ // 更新下一个期望序号

					// 如果是尾片，记录最后片序号
					if isFlagLast(frame.Flag) {
						sduCache.finalSeq = frame.PSEQ
					}
					// 尝试拼接乱序缓存中后续连续的片段
					for {
						data, ok := sduCache.outOfOrder[sduCache.expectedSeq]
						if !ok {
							break
						}
						// 找到按序衔接的片段，取出拼接
						appendFragmentData(sduCache, sduCache.expectedSeq, data)
						delete(sduCache.outOfOrder, sduCache.expectedSeq)
						sduCache.expectedSeq++
					}
					// 检查是否已完成整个SDU拼接：
					// 条件：已收到尾片且所有片段序号都已衔接到尾片
					if sduCache.finalSeq != 0 && sduCache.expectedSeq > sduCache.finalSeq {
						finalizeAndOutput(sensorID, sduCache)
					}
				}
			}
		}
	}
}

// 判断Flag是否标识首片 (2-bit 值 == 00)
func isFlagFirst(flag uint8) bool {
	// 低2位为标志位，00表示首片
	return flag&0x3 == 0x0
}

// 判断Flag是否标识尾片 (2-bit 值 == 11)
func isFlagLast(flag uint8) bool {
	return flag&0x3 == 0x3
}

// 将片段数据附加到缓存的dataBuffer中（根据需要可处理首片中的特殊字节）
func appendFragmentData(cache *SDUCache, pseq uint8, data []byte) {
	// 简单拼接数据片段
	cache.dataBuffer = append(cache.dataBuffer, data...)
	// （注：根据协议，可能需要在首片处处理协议头或长度字段，这里假设Data已经是纯净的SDU数据片段）
}

// 启动拼接超时定时器
func startReassembleTimer(sensorID [6]byte, cache *SDUCache) {
	cache.timer = time.AfterFunc(reassembleTimeout, func() {
		cacheMu.Lock()
		defer cacheMu.Unlock()
		// 定时器触发时再次检查：
		currentCache, ok := sduCacheMap[sensorID]
		if ok && currentCache == cache {
			// 若超时时该SensorID缓存仍是当前cache且尚未完成拼接，则丢弃
			delete(sduCacheMap, sensorID)
			// 记录超时日志（如需要）：fmt.Printf("拼接超时，丢弃传感器[%x]序号[%d]的未完成SDU\n", sensorID, cache.SSEQ)
		}
	})
}

// 取消并清除定时器
func cancelReassembleTimer(cache *SDUCache) {
	if cache.timer != nil {
		cache.timer.Stop()
		cache.timer = nil
	}
}

// 完成拼接后输出完整帧到解析通道
func finalizeAndOutput(sensorID [6]byte, cache *SDUCache) {
	// 在输出前先清除定时器和缓存，以免重复
	cancelReassembleTimer(cache)
	delete(sduCacheMap, sensorID)

	// 构造新的Frame，内容与首片帧类似但标记为非分片
	fullFrame := &Frame{
		SensorID: sensorID,         // **注意**：这里需要获取SensorID，本例中可以从传入参数sensorID获得或缓存中存储
		FragInd:  0,                // 标记为完整帧
		SSEQ:     cache.SSEQ,       // 沿用业务单元序号（可选，看后续解析是否需要）
		PSEQ:     0,                // 完整帧无分片序号
		Flag:     0,                // 完整帧无分片标志
		Data:     cache.dataBuffer, // 拼接后的完整SDU数据
	}
	// 通过frameCh通道发送给下一阶段解析
	FrameCh <- fullFrame
}
