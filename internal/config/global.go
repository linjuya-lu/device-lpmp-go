package config

var (
	//写入通道
	WriteChan = make(chan []byte, 100)
	//接入节点EID
	EidStr = "238A0841D828"
	//资源路径
	DevicesYAML = "../cmd/res/devices/devices.yaml"
	ProfilesDir = "../cmd/res/profiles"
	//串口信息
	PortName = "/dev/ttyS8"
	BaudRate = 115200
)
