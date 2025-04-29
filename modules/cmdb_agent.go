package modules

import (
	"agent/config"
	"agent/kits"
	"encoding/json"
	"fmt"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/duke-git/lancet/v2/system"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"github.com/yumaojun03/dmidecode"
	"strings"
	"time"
)

func HardWare() config.HardWareConf {
	defer func() {
		if r := recover(); r != nil {
			kits.WriteLog(fmt.Sprint(r))
		}
	}()
	//获取硬件信息
	conf := config.HardWareConf{}
	//获取CPU信息
	core, err := cpu.Counts(true)
	if err == nil {
		conf.CpuCore = core
	}
	info, err := cpu.Info()
	if err == nil {
		conf.CpuInfo = info[0].ModelName
	}
	//获取内存信息
	ms, err := mem.VirtualMemory()
	if err == nil {
		if ms != nil {
			conf.MemSize = ms.Total
		}
	}
	//获取磁盘信息
	ds := make(map[string]config.DiskInfo)
	dk, err := disk.Partitions(false)
	if err == nil {
		for _, k := range dk {
			if strings.Contains(k.Device, "/dev/") {
				_, ok := ds[k.Device]
				if !ok {
					di := config.DiskInfo{}
					di.FsType = k.Fstype
					di.MountPoint = k.Mountpoint
					d, err := disk.Usage(k.Mountpoint)
					if err == nil {
						di.Size = d.Total
					}
					ds[k.Device] = di
				}
			}
		}
		conf.Disk = ds
	}
	//获取网卡信息
	ns := make(map[string]config.NetInfo)
	nt, err := net.Interfaces()
	if err == nil {
		for _, k := range nt {
			ni := config.NetInfo{}
			ni.HardwareAddr = k.HardwareAddr
			for _, i := range k.Addrs {
				ni.Addrs = append(ni.Addrs, i.Addr)
			}
			ns[k.Name] = ni
		}
		conf.Net = ns
	}
	//获取bios信息
	dmi, err := dmidecode.New()
	if dmi != nil {
		infos, err := dmi.System()
		conf.Manufacturer = infos[0].Manufacturer
		conf.ProductName = infos[0].ProductName
		conf.SerialNumber = infos[0].SerialNumber
		if err == nil {
			conf.UUID = infos[0].UUID
		}
	}
	return conf
}
func HostSystem() config.HostSystemConf {
	//获取操作系统信息
	defer func() {
		if r := recover(); r != nil {
			kits.WriteLog(fmt.Sprint(r))
		}
	}()
	conf := config.HostSystemConf{}
	h, err := host.Info()
	if err == nil {
		conf.HostID = h.HostID
		conf.Hostname = h.Hostname
		conf.Uptime = h.Uptime
		conf.OS = h.OS
		conf.Platform = h.Platform
		conf.PlatformVersion = h.PlatformVersion
		conf.KernelVersion = h.KernelVersion
		conf.VirtualizationSystem = h.VirtualizationSystem
		conf.InternetIp = strings.TrimSpace(kits.GetInternetIp())
	} else {
		kits.WriteLog(err.Error())
	}
	return conf
}
func IpmiInfo() config.IpmiConf {
	//获取IPMI信息
	var err error
	defer func() {
		if r := recover(); r != nil {
			kits.WriteLog(fmt.Sprint(r))
		}
		if err != nil {
			kits.WriteLog(err.Error())
		}
	}()
	conf := config.IpmiConf{}
	if fileutil.IsExist("/dev/ipmi0") || fileutil.IsExist("/dev/ipmi/0") ||
		fileutil.IsExist("/dev/ipmidev/0") {
		if fileutil.IsExist("/usr/bin/ipmitool") {
			stdout, _, _ := system.ExecCommand("ipmitool -I open lan print")
			if stdout != "" {
				for _, v := range strings.Split(stdout, "\n") {
					if strings.Contains(v, "IP Address") {
						ip := strings.TrimSpace(strings.Split(v, ":")[1])
						if len(strings.Split(ip, ".")) >= 3 {
							conf.Ip = ip
						}
					}
				}
			}
		} else {
			_, _, _ = system.ExecCommand("yum -y install ipmitool")
			for _, arg := range []string{"ipmi_watchdog", "ipmi_poweroff", "ipmi_devintf", "ipmi_si", "ipmi_msghandler"} {
				_, _, _ = system.ExecCommand("modprobe " + arg)
			}
		}
	}
	return conf
}
func CmdbAgent() {
	for {
		if config.AssetAgentRun && config.HeartbeatErr == nil {
			func() {
				defer func() {
					if r := recover(); r != nil {
						kits.WriteLog(fmt.Sprint(r))
					}
				}()
				// 数据加密
				Ipmi := IpmiInfo()
				ad, _ := json.Marshal(config.AssetData{HardWare: HardWare(), System: HostSystem(), Ipmi: Ipmi})
				data := Encrypt.EncryptString(string(ad), true)
				// 上传数据
				s, err := json.Marshal(map[string]interface{}{"cmdb": data})
				if err == nil && config.WscAlive {
					config.SendChan <- string(s)
					kits.WriteLog("上报设备配置信息成功")
				} else {
					kits.WriteLog(err.Error())
				}
				defer func() {
					if r := recover(); r != nil {
						kits.WriteLog(fmt.Sprint(r))
					}
				}()
			}()
		}
		if config.AssetInterval < 5 {
			config.AssetInterval = 5
		}
		time.Sleep(time.Duration(config.AssetInterval) * time.Minute)
	}
}
