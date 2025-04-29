package kits

import (
	"agent/config"
	"encoding/json"
	"fmt"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/golang-module/carbon"
	"github.com/levigross/grequests"
	"github.com/mitchellh/go-ps"
	"github.com/spf13/cast"
	"os"
	"strconv"
	"strings"
	"time"
)

type YamlFile struct {
	LogFile   string `yaml:"LogFile"`
	ErrorFile string `yaml:"ErrorFile"`
}

func Count(value []uint64) uint64 {
	var data uint64
	for _, val := range value {
		data = data + val
	}
	return data
}
func MapToJson(param map[string]interface{}) string {
	dataType, _ := json.Marshal(param)
	return string(dataType)
}
func GetInternetIp() string {
	var ip string
	var ipc = make(chan string, 1)
	urls := []string{"http://ip.sb", "http://ident.me", "http://inet-ip.info"}
	for _, u := range urls {
		go func(u string) {
			resp, err := grequests.Get(u, &grequests.RequestOptions{})
			if resp != nil && err == nil && resp.Ok {
				if strings.Count(cast.ToString(resp.Bytes()), ".") == 3 {
					ip = cast.ToString(resp.Bytes())
					if ip != "" {
						ipc <- ip
					}
				}
				if strings.Count(cast.ToString(resp.Bytes()), ":") == 5 {
					ip = cast.ToString(resp.Bytes())
					if ip != "" {
						ipc <- ip
					}
				}
			}
		}(u)
	}
	select {
	case sip := <-ipc:
		ip = sip
		break
	case <-time.After(1 * time.Minute):
	}
	return ip
}
func WritePid() {
	_ = fileutil.WriteStringToFile(config.AgentPidFile, strconv.Itoa(os.Getpid()), false)
}
func CheckAgentDogPid() bool {
	f, _ := fileutil.ReadFileToString(config.AgentDogPidFile)
	if f != "" {
		p, err := strconv.Atoi(f)
		if err == nil {
			config.AgentDogPid = p
			p, err := ps.FindProcess(p)
			if err == nil && p != nil {
				return true
			}
		}
	}
	return false
}
func CheckAgentPid() {
	f, _ := fileutil.ReadFileToString(config.AgentPidFile)
	if f != "" {
		v, err := strconv.Atoi(f)
		if err == nil {
			p, err := ps.FindProcess(v)
			if err == nil && p != nil {
				if p.Pid() != os.Getpid() {
					config.Qch <- "进程(" + strconv.Itoa(os.Getpid()) + ")其它agent进程在运行,退出"
				}
			}
		}
	} else {
		WritePid()
	}
}
func UninstallAgent() {
	for _, f := range []string{config.AgentService, config.AgentFile, config.AgentPidFile,
		config.AgentDogPidFile, config.AgentPidFile, config.AgentConfig,
		config.LogFile + "." + carbon.Now().ToDateString()} {
		_ = fileutil.RemoveFile(f)
	}
	_ = os.RemoveAll(config.LogPath)
	_ = os.RemoveAll(config.AgentPath)
}
func ExcludeNetName(name string, extend []string) bool {
	result := true
	extend = append(extend, []string{"lo", "docker", "cni", "tunl", "vir", "cali", "flannel", "br-", "vnet", "veth", "kube-ip"}...)
	for _, n := range extend {
		if strings.Contains(name, n) == true {
			result = false
			break
		}
	}
	return result
}
func WriteLog(msg string) {
	if !strings.Contains(msg, "exit status") {
		go func(msg string) {
			logfile := config.LogFile + "." + carbon.Now().ToDateString()
			if fileutil.IsExist(logfile) {
				_ = fileutil.WriteStringToFile(logfile, carbon.Now().ToDateTimeString()+" opsone-agent "+fmt.Sprintln(msg), true)
			} else {
				file, _ := os.Create(logfile)
				defer func(file *os.File) {
					_ = file.Close()
				}(file)
			}
		}(msg)
	}
}
