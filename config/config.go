package config

const (
	CryptKey        = "4797979a2929ca11b7575755ahf88a2d"
	Version         = "2025041401"
	DogVersion      = "2025033102"
	AgentPath       = "/opt/opsone"
	PidPath         = "/var/run"
	LogPath         = "/var/log/opsone"
	AgentService    = "/etc/systemd/system/opsone.service"
	AgentConfig     = AgentPath + "/config.ini"
	AgentFile       = AgentPath + "/opsone-agent"
	AgentDogFile    = AgentPath + "/opsone-dog"
	AgentPidFile    = PidPath + "/opsone-agent.pid"
	AgentDogPidFile = PidPath + "/opsone-dog.pid"
	LogFile         = LogPath + "/opsone-agent.log"
)

var (
	AgentUrl          string
	Uninstall         bool
	AssetInterval     int64 = 15 //单位分钟
	MonitorInterval   int64 = 60 //单位秒
	HeartBeatInterval int64 = 10 //单位秒
	AssetAgentRun     bool
	MonitorAgentRun   bool
	SendChan          = make(chan string, 10) //定义channel大小
	RecvChan          = make(chan string, 10) //定义channel大小
	Qch               = make(chan string, 1)
	Process           []string
	CustomMetrics     = make(map[string]string)
	HeartbeatErr      error
	WscAlive          bool
	ServerAddr        string
	AgentDogPid       int
	ClamDay           int
	ClamRun           string
	WscPath           = "/api/v1/heartbeat/ws/"
)

type HeartbeatJson struct {
	AgentVersion      string `json:"AgentVersion"`
	AssetAgentRun     int    `json:"AssetAgentRun"`
	MonitorAgentRun   int    `json:"MonitorAgentRun"`
	HeartBeatInterval int64  `json:"HeartBeatInterval"`
	AssetInterval     int64  `json:"AssetInterval"`
	MonitorInterval   int64  `json:"MonitorInterval"`
	Upgrade           bool   `json:"Upgrade"`
	Uninstall         bool   `json:"Uninstall"`
}

type DiskInfo struct {
	MountPoint string `json:"mount_point"`
	FsType     string `json:"fs_type"`
	Size       uint64 `json:"size"`
}

type NetInfo struct {
	HardwareAddr string   `json:"hardwareaddr"`
	Addrs        []string `json:"addrs"`
}

type HardWareConf struct {
	UUID         string              `json:"uuid"`
	SerialNumber string              `json:"serial_number"`
	Manufacturer string              `json:"manufacturer"`
	ProductName  string              `json:"product_name"`
	CpuCore      int                 `json:"cpu_core"`
	CpuInfo      string              `json:"cpu_info"`
	MemSize      uint64              `json:"mem_size"`
	Disk         map[string]DiskInfo `json:"disk"`
	Net          map[string]NetInfo  `json:"net"`
}

type HostSystemConf struct {
	HostID               string `json:"host_id"`
	Hostname             string `json:"host_name"`
	OS                   string `json:"os"`
	Platform             string `json:"platform"`
	PlatformVersion      string `json:"platformVersion"`
	KernelVersion        string `json:"kernelVersion"`
	VirtualizationSystem string `json:"VirtualizationSystem"`
	InternetIp           string `json:"internet_ip"`
	Uptime               uint64 `json:"uptime"`
}
type IpmiConf struct {
	Ip string `json:"ip"`
}
type AssetData struct {
	HardWare HardWareConf   `json:"hardware"`
	System   HostSystemConf `json:"system"`
	Ipmi     IpmiConf       `json:"ipmi"`
}

type MonitorPostData struct {
	HostID  string   `json:"host_id"`
	Process []string `json:"process"`
}

type MonitorData struct {
	Tags            map[string]string                 `json:"tags"`
	Fields          map[string]interface{}            `json:"fields"`
	Process         map[string]map[string]interface{} `json:"process"`
	CustomMetrics   map[string]interface{}            `json:"custom_metrics"`
	ProcessTop      map[string]interface{}            `json:"process_top"`
	MonitorInterval int64
}
