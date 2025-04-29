package modules

import (
	"agent/config"
	"agent/kits"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/duke-git/lancet/v2/system"
	"github.com/spf13/cast"
	"net"
	"os"
	"strconv"
	"strings"
)

func AgentBus() {
	for {
		func() {
			var (
				err error
				d   map[string]interface{}
			)
			defer func() {
				if r := recover(); r != nil {
					kits.WriteLog(fmt.Sprint(r))
				}
				if err != nil {
					kits.WriteLog(err.Error())
				}
			}()
			msg := <-config.RecvChan
			err = json.Unmarshal([]byte(msg), &d)
			if err == nil {
				for k, v := range d {
					D, err := Encrypt.DecryptString(v.(string), true)
					if err == nil {
						switch k {
						case "heartbeat":
							go HeartbeatHandle(D)
						case "monitor":
							go MonitorHandle(D)
						case "jobShell":
							go JobShellHandle(D)
						case "jobFile":
							go JobFileHandle(D)
						}
					}
				}
			}
		}()
	}
}

func HeartbeatHandle(data []byte) {
	var (
		err           error
		HeartbeatData = config.HeartbeatJson{}
	)
	defer func() {
		if r := recover(); r != nil {
			kits.WriteLog(fmt.Sprint(r))
		}
		if err != nil {
			kits.WriteLog(err.Error())
		}
	}()
	err = json.Unmarshal(data, &HeartbeatData)
	if err == nil {
		config.HeartbeatErr = nil
		if HeartbeatData.Uninstall {
			config.Uninstall = true
			msg := AgentDogConn("uninstall")
			if msg == config.DogVersion {
				kits.WriteLog("agent下架处理")
				kits.UninstallAgent()
				config.Qch <- "进程(" + strconv.Itoa(os.Getpid()) + ")下架处理,退出"
			}
		}
		msg := config.DogVersion
		if HeartbeatData.AgentVersion != config.Version && HeartbeatData.Upgrade {
			//agent版本更新
			kits.WriteLog("agent当前版本: " + config.Version)
			kits.WriteLog("agent最新版本: " + HeartbeatData.AgentVersion)
			config.HeartbeatErr = errors.New("agent开始更新到最新版本")
			// 发送版本信息到agentDog
			_ = fileutil.RemoveFile(config.AgentFile)
			_ = fileutil.RemoveFile(config.AgentPidFile)
			config.Qch <- "进程(" + strconv.Itoa(os.Getpid()) + ")更新版本,退出"
		} else {
			msg = AgentDogConn("upgrade:" + config.DogVersion)
		}
		config.AssetAgentRun = cast.ToBool(HeartbeatData.AssetAgentRun)
		config.MonitorAgentRun = cast.ToBool(HeartbeatData.MonitorAgentRun)
		if HeartbeatData.HeartBeatInterval > 0 && config.HeartBeatInterval != HeartbeatData.HeartBeatInterval {
			config.HeartBeatInterval = HeartbeatData.HeartBeatInterval
			kits.WriteLog("变更心跳检测周期为" + cast.ToString(HeartbeatData.HeartBeatInterval) + "秒")
		}
		if HeartbeatData.AssetInterval > 0 && config.AssetInterval != HeartbeatData.AssetInterval {
			config.AssetInterval = HeartbeatData.AssetInterval
			kits.WriteLog("变更配置上报周期为" + cast.ToString(HeartbeatData.AssetInterval) + "分钟")
		}
		if HeartbeatData.MonitorInterval > 0 && config.MonitorInterval != HeartbeatData.MonitorInterval {
			config.MonitorInterval = HeartbeatData.MonitorInterval
			kits.WriteLog("变更监控上报周期为" + cast.ToString(HeartbeatData.MonitorInterval) + "秒")
		}
		if msg == "" {
			if config.AgentDogPid > 0 && !HeartbeatData.Upgrade {
				_ = system.KillProcess(config.AgentDogPid)
			}
			kits.WriteLog("agent-dog(" + strconv.Itoa(config.AgentDogPid) + ")运行异常")
		} else {
			kits.WriteLog("agent(" + config.Version + ")心跳检测正常")
		}
	}
}

func MonitorHandle(data []byte) {
	var (
		err error
		d   map[string]interface{}
	)
	defer func() {
		if r := recover(); r != nil {
			kits.WriteLog(fmt.Sprint(r))
		}
		if err != nil {
			kits.WriteLog(err.Error())
		}
	}()
	err = json.Unmarshal(data, &d)
	if err == nil {
		_, ok := d["process"]
		if ok {
			if cast.ToStringSlice(d["process"]) != nil {
				if config.Process == nil {
					kits.WriteLog("开启进程监控:" + strings.Join(cast.ToStringSlice(d["process"]), ","))
				}
				config.Process = cast.ToStringSlice(d["process"])
			}
		}
		_, ok = d["custom_metrics"]
		if ok {
			CustomMetrics := make(map[string]string)
			if len(config.CustomMetrics) == 0 {
				kits.WriteLog("开启自定义监控指标数据采集")
			}
			for k, v := range d["custom_metrics"].(map[string]interface{}) {
				CustomMetrics[k] = cast.ToString(v)
			}
			config.CustomMetrics = CustomMetrics
		} else {
			if len(config.CustomMetrics) > 0 {
				kits.WriteLog("关闭自定义监控指标数据采集")
				config.CustomMetrics = make(map[string]string)
			}
		}
	}
}

func JobShellHandle(data []byte) {
	var (
		err    error
		stdout string
		stderr string
		status bool
		d      map[string]interface{}
	)
	defer func() {
		if r := recover(); r != nil {
			kits.WriteLog(fmt.Sprint(r))
		}
		if err != nil {
			kits.WriteLog(err.Error())
		}
	}()
	err = json.Unmarshal(data, &d)
	if err == nil {
		defer func() {
			if cast.ToString(d["job_type"]) == "job_script" {
				kits.WriteLog("删除临时脚本文件:" + cast.ToString(d["file"]))
				_ = os.Remove(cast.ToString(d["file"]))
			}
		}()
		cmd := cast.ToString(d["exec"])
		kits.WriteLog("接收到执行命令:" + cmd)
		switch cmd {
		case "install clamAv":
			stdout, stderr = InstallClamAv()
		case "remove clamAv":
			stdout, stderr = RemoveClamAv()
		default:
			stdout, stderr, _ = system.ExecCommand(cmd)
		}
		if stderr == "" {
			status = true
		}
		s, err := json.Marshal(map[string]interface{}{"host_id": cast.ToString(d["host_id"]),
			"job_id":   cast.ToString(d["job_id"]),
			"job_type": cast.ToString(d["job_type"]),
			"stdout":   stdout,
			"stderr":   stderr,
			"status":   status,
		})
		if err == nil && config.WscAlive {
			config.SendChan <- kits.MapToJson(map[string]interface{}{"jobShell": Encrypt.EncryptString(string(s), true)})
		}
	}
}

func JobFileHandle(data []byte) {
	var (
		err     error
		message string
		status  bool
		d       map[string]interface{}
	)
	defer func() {
		if r := recover(); r != nil {
			kits.WriteLog(fmt.Sprint(r))
		}
		if err != nil {
			kits.WriteLog(err.Error())
		}
	}()
	err = json.Unmarshal(data, &d)
	if err == nil {
		_ = os.MkdirAll(cast.ToString(d["dst_path"]), 0755)
		file := cast.ToString(d["dst_path"]) + cast.ToString(d["file_name"])
		err = fileutil.WriteBytesToFile(file, d["file_content"].([]byte))
		if err == nil {
			kits.WriteLog("接收到分发文件:" + file)
			if fileutil.IsExist(file) {
				message = file + "文件已验证"
				status = true
			}
		} else {
			message = err.Error()
		}
		s, err := json.Marshal(map[string]interface{}{"host_id": cast.ToString(d["host_id"]),
			"job_id":   cast.ToString(d["job_id"]),
			"job_type": cast.ToString(d["job_type"]),
			"file":     file,
			"message":  message,
			"status":   status})
		if err == nil && config.WscAlive {
			config.SendChan <- kits.MapToJson(map[string]interface{}{"jobFile": Encrypt.EncryptString(string(s), true)})
		}
	}
}

func AgentDogConn(msg string) string {
	var (
		rec string
		buf [1024]byte
	)
	defer func() {
		if r := recover(); r != nil {
			kits.WriteLog(fmt.Sprint(r))
		}
	}()
	conn, err := net.Dial("tcp", "127.0.0.1:54321")
	if conn != nil {
		_, err = conn.Write([]byte(msg))
		if err == nil {
			n, err := conn.Read(buf[:])
			if err == nil {
				rec = string(buf[:n])
			} else {
				kits.WriteLog(err.Error())
			}
			if err != nil {
				kits.WriteLog(err.Error())
			}
		}
	}
	if err != nil {
		kits.WriteLog(err.Error())
	}
	return rec
}
