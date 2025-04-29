package modules

import (
	"agent/config"
	"agent/kits"
	"encoding/json"
	"fmt"
	"github.com/duke-git/lancet/v2/netutil"
	"github.com/duke-git/lancet/v2/system"
	"github.com/shirou/gopsutil/process"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	gonet "github.com/shirou/gopsutil/v4/net"
	"github.com/spf13/cast"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

var (
	BeforeLanInTraffic      uint64
	BeforeLanOutTraffic     uint64
	BeforeWanInTraffic      uint64
	BeforeWanOutTraffic     uint64
	BeforeReadBytes         uint64
	BeforeWriteBytes        uint64
	BeforeProcessInTraffic  = make(map[string]uint64)
	BeforeProcessOutTraffic = make(map[string]uint64)
	BeforeProcessReadBytes  = make(map[string]uint64)
	BeforeProcessWriteByte  = make(map[string]uint64)
)

func MonitorCollection() config.MonitorData {
	//获取系统监控信息
	var (
		LanBytesSent     []uint64
		LanBytesRecv     []uint64
		WanBytesSent     []uint64
		WanBytesRecv     []uint64
		ReadBytes        []uint64
		WriteBytes       []uint64
		cpuTops          = map[string]float64{}
		memTops          = map[string]float32{}
		processBytesSent = make(map[string][]uint64)
		processBytesRecv = make(map[string][]uint64)
		md               = config.MonitorData{}
		tags             = make(map[string]string)
		fields           = make(map[string]interface{})
	)
	defer func() {
		if r := recover(); r != nil {
			kits.WriteLog(fmt.Sprint(r))
		}
	}()
	h, _ := host.Info()
	tags["host_id"] = h.HostID
	//如果设置的间隔时间大于5分钟均以5分钟间隔进行执行
	if config.MonitorInterval > 300 {
		config.MonitorInterval = 300
	}
	//获取系统load值
	info, _ := load.Avg()
	fields["cpu_loadavg"] = info.Load5
	//获取cpu平均使用率
	c2, _ := cpu.Percent(0, false)
	fields["cpu_usage"] = c2[0]
	//获取内存信息
	m, _ := mem.VirtualMemory()
	fields["mem_used"] = m.Used
	fields["mem_available"] = m.Free
	fields["mem_pavailable"] = 100 - m.UsedPercent
	fields["mem_pused"] = m.UsedPercent
	//获取磁盘信息
	fields["disk_usage"] = 0.0
	for _, dr := range []string{"/", "/home"} {
		d, err := disk.Usage(dr)
		if err == nil {
			fields["disk_usage"] = fields["disk_usage"].(float64) + d.UsedPercent
		}
	}
	fields["disk_usage"] = fields["disk_usage"].(float64) / 2
	p, _ := disk.Partitions(false)
	for _, v := range p {
		dm := strings.Split(v.Device, "/")
		sd, _ := disk.IOCounters()
		ReadBytes = append(ReadBytes, sd[dm[len(dm)-1]].ReadBytes)
		WriteBytes = append(WriteBytes, sd[dm[len(dm)-1]].WriteBytes)
	}
	Wan := make(map[string]struct{})
	Lan := make(map[string]struct{})
	nt, err := gonet.Interfaces()
	if err == nil {
		for _, k := range nt {
			if kits.ExcludeNetName(k.Name, []string{}) && len(k.Addrs) > 0 {
				for _, i := range k.Addrs {
					ip, _, err := net.ParseCIDR(i.Addr)
					if err == nil && ip.To4() != nil {
						if netutil.IsInternalIP(ip) {
							Lan[k.Name] = struct{}{}
						}
						if netutil.IsPublicIP(ip) {
							Wan[k.Name] = struct{}{}
						}
					}
				}
			}
		}
	}
	n, err := gonet.IOCounters(true)
	if err == nil {
		for _, v := range n {
			_, ok := Wan[v.Name]
			if ok {
				WanBytesSent = append(WanBytesSent, v.BytesSent)
				WanBytesRecv = append(WanBytesRecv, v.BytesRecv)
			}
			_, ok = Lan[v.Name]
			if ok {
				LanBytesSent = append(LanBytesSent, v.BytesSent)
				LanBytesRecv = append(LanBytesRecv, v.BytesRecv)
			}
		}
		lanInTraffic := kits.Count(LanBytesRecv)
		lanOutTraffic := kits.Count(LanBytesSent)
		WanInTraffic := kits.Count(WanBytesRecv)
		WanOutTraffic := kits.Count(WanBytesSent)
		//获取内网网络流量信息
		if BeforeLanInTraffic <= 0 {
			BeforeLanInTraffic = lanInTraffic
		}
		if BeforeLanOutTraffic <= 0 {
			BeforeLanOutTraffic = lanOutTraffic
		}
		fields["lan_intraffic"] = (lanInTraffic - BeforeLanInTraffic) * 8 / uint64(config.MonitorInterval)
		fields["lan_outtraffic"] = (lanOutTraffic - BeforeLanOutTraffic) * 8 / uint64(config.MonitorInterval)
		BeforeLanInTraffic = lanInTraffic
		BeforeLanOutTraffic = lanOutTraffic
		//获取公网网络流量信息
		if BeforeWanInTraffic <= 0 {
			BeforeWanInTraffic = WanInTraffic
		}
		if BeforeWanOutTraffic <= 0 {
			BeforeWanOutTraffic = WanOutTraffic
		}
		fields["wan_intraffic"] = (WanInTraffic - BeforeWanInTraffic) * 8 / uint64(config.MonitorInterval)
		fields["wan_outtraffic"] = (WanOutTraffic - BeforeWanOutTraffic) * 8 / uint64(config.MonitorInterval)
		BeforeWanInTraffic = WanInTraffic
		BeforeWanOutTraffic = WanOutTraffic
	}
	//磁盘读写量
	newReadBytes := kits.Count(ReadBytes)
	newWriteBytes := kits.Count(WriteBytes)
	if BeforeReadBytes <= 0 {
		BeforeReadBytes = newReadBytes
	}
	if BeforeWriteBytes <= 0 {
		BeforeWriteBytes = newWriteBytes
	}
	fields["disk_read_traffic"] = (newReadBytes - BeforeReadBytes) / uint64(config.MonitorInterval)
	fields["disk_write_traffic"] = (newWriteBytes - BeforeWriteBytes) / uint64(config.MonitorInterval)
	BeforeReadBytes = newReadBytes
	BeforeWriteBytes = newWriteBytes
	//获取tcp连接数
	Est := 0
	e, err := gonet.Connections("tcp")
	if err == nil {
		for _, k := range e {
			if k.Status == "ESTABLISHED" {
				Est++
			}
		}
	}
	fields["tcp_estab"] = Est
	infos, _ := process.Processes()
	//获取进行top
	for _, pid := range infos {
		name, _ := pid.Name()
		Cpu, _ := pid.CPUPercent()
		if Cpu > 1 {
			_, ok := cpuTops[name]
			if ok {
				cpuTops[name] = cpuTops[name] + Cpu
			} else {
				cpuTops[name] = Cpu
			}
		}
		Mem, _ := pid.MemoryPercent()
		if Mem > 1 {
			_, ok := memTops[name]
			if ok {
				memTops[name] = memTops[name] + Mem
			} else {
				memTops[name] = Mem
			}
		}
	}
	//获取进程信息
	go func(ps []*process.Process) {
		var Cpu float64
		for _, p := range ps {
			if p.Pid == int32(os.Getpid()) {
				C, _ := p.CPUPercent()
				Cpu = Cpu + C
			}
		}
		//cpu占用率超过80%自动退出
		if Cpu >= 80 {
			config.Qch <- "进程(" + strconv.Itoa(os.Getpid()) + ")cpu占用过高,退出"
		}
	}(infos)
	if len(infos) > 0 {
		if len(config.Process) > 0 {
			pro := map[string]map[string]interface{}{}
			for _, name := range config.Process {
				pro[name] = map[string]interface{}{"cpu_usage": 0.0, "mem_pused": 0.0, "disk_read_traffic": 0,
					"disk_write_traffic": 0, "process_numbers": 0, "lan_intraffic": 0, "lan_outtraffic": 0, "alive": 0}
				processBytesSent[name] = []uint64{}
				processBytesRecv[name] = []uint64{}
				for _, pid := range infos {
					if na, _ := pid.Name(); na == name {
						Cpu, _ := pid.CPUPercent()
						pro[name]["cpu_usage"] = cast.ToFloat64(pro[name]["cpu_usage"]) + Cpu
						Mem, _ := pid.MemoryPercent()
						pro[name]["mem_pused"] = cast.ToFloat32(pro[name]["mem_pused"]) + Mem
						pro[name]["process_numbers"] = cast.ToInt(pro[name]["process_numbers"]) + 1
						pro[name]["alive"] = 1
						//获取进程网络流量
						pn, _ := pid.NetIOCounters(true)
						for _, v := range pn {
							processBytesSent[name] = append(processBytesSent[name], v.BytesSent)
							processBytesRecv[name] = append(processBytesRecv[name], v.BytesRecv)
						}
						ProcessInTraffic := kits.Count(processBytesRecv[name])
						ProcessOutTraffic := kits.Count(processBytesSent[name])
						if _, ok := BeforeProcessInTraffic[cast.ToString(pid)]; ok {
							if BeforeProcessInTraffic[cast.ToString(pid)] <= 0 {
								BeforeProcessInTraffic[cast.ToString(pid)] = ProcessInTraffic
							}
							if BeforeProcessOutTraffic[cast.ToString(pid)] <= 0 {
								BeforeProcessOutTraffic[cast.ToString(pid)] = ProcessOutTraffic
							}
							pro[name]["lan_intraffic"] = (ProcessInTraffic - BeforeProcessInTraffic[cast.ToString(pid)]) * 8 / uint64(config.MonitorInterval)
							pro[name]["lan_outtraffic"] = (ProcessOutTraffic - BeforeProcessOutTraffic[cast.ToString(pid)]) * 8 / uint64(config.MonitorInterval)
						}
						BeforeProcessInTraffic[cast.ToString(pid)] = ProcessInTraffic
						BeforeProcessOutTraffic[cast.ToString(pid)] = ProcessOutTraffic
						//获取进程磁盘IO
						pi, _ := pid.IOCounters()
						if _, ok := BeforeProcessReadBytes[cast.ToString(pid)]; ok {
							if BeforeProcessReadBytes[cast.ToString(pid)] <= 0 {
								BeforeProcessReadBytes[cast.ToString(pid)] = pi.ReadBytes
							}
							if BeforeProcessWriteByte[cast.ToString(pid)] <= 0 {
								BeforeProcessWriteByte[cast.ToString(pid)] = pi.WriteBytes
							}
							pro[name]["disk_read_traffic"] = (pi.ReadBytes - BeforeProcessReadBytes[cast.ToString(pid)]) / uint64(config.MonitorInterval)
							pro[name]["disk_write_traffic"] = (pi.WriteBytes - BeforeProcessWriteByte[cast.ToString(pid)]) / uint64(config.MonitorInterval)
						}
						BeforeProcessReadBytes[cast.ToString(pid)] = pi.ReadBytes
						BeforeProcessWriteByte[cast.ToString(pid)] = pi.WriteBytes
					}
				}
			}
			md.Process = pro
		}
	}
	//获取自定义监控数据
	if len(config.CustomMetrics) > 0 {
		Custom := make(map[string]interface{})
		for k, v := range config.CustomMetrics {
			kits.WriteLog("执行自定义监控指标数据采集:" + k)
			//执行自定义监控指标数据采集
			stdout, stderr, err := system.ExecCommand(v)
			if err != nil {
				return config.MonitorData{}
			}
			if stderr != "" {
				kits.WriteLog(stderr)
			}
			if stdout != "" {
				var c string
				if strings.Count(stdout, "\n") <= 1 {
					c = strings.Trim(stdout, "\n")
				} else {
					s := strings.Split(stdout, "\n")
					c = s[len(s)-1]
				}
				if strings.Split(c, ".")[0] == "" {
					c = "0" + c
				}
				s, err := strconv.ParseFloat(c, 64)
				if err == nil {
					Custom[k] = s
				} else {
					kits.WriteLog(err.Error())
				}
			}
		}
		if len(Custom) > 0 {
			md.CustomMetrics = Custom
		}
	}
	md.Tags = tags
	md.Fields = fields
	md.ProcessTop = map[string]interface{}{"cpu": cpuTops, "mem": memTops}
	md.MonitorInterval = config.MonitorInterval
	return md
}

func MonitorAgent() {
	for {
		if config.MonitorAgentRun && config.HeartbeatErr == nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						kits.WriteLog(fmt.Sprint(r))
					}
				}()
				mc, err := json.Marshal(MonitorCollection())
				if err == nil {
					data := Encrypt.EncryptString(string(mc), true)
					s, err := json.Marshal(map[string]interface{}{"monitor": data})
					if err == nil && config.WscAlive {
						config.SendChan <- string(s)
						kits.WriteLog("上报监控数据成功")
					} else {
						kits.WriteLog(err.Error())
					}
				}
				defer func() {
					if err != nil {
						kits.WriteLog(err.Error())
					}
				}()
			}()
		}
		time.Sleep(time.Duration(config.MonitorInterval) * time.Second)
	}
}
