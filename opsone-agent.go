package main

import (
	"agent/config"
	_ "agent/daemon"
	"agent/kits"
	"agent/modules"
	"fmt"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/golang-module/carbon"
	"net"
	"os"
	"time"
)

func init() {
	go func() {
		for {
			func() {
				defer func() {
					if r := recover(); r != nil {
						kits.WriteLog(fmt.Sprint(r))
					}
				}()
				kits.CheckAgentPid()
				for _, p := range []string{config.AgentPath, config.LogPath} {
					if !fileutil.IsExist(p) {
						_ = fileutil.CreateDir(p)
					}
				}
				fileNames, err := fileutil.ListFileNames(config.LogPath)
				if err == nil {
					for _, fileName := range fileNames {
						if fileName != "opsone-agent.log."+carbon.Now().ToDateString() {
							_ = fileutil.RemoveFile(config.LogPath + "/" + fileName)
						}
					}
				}
				if fileutil.IsExist(config.AgentConfig) {
					remoteIp, _ := fileutil.ReadFileToString(config.AgentConfig)
					if net.ParseIP(remoteIp) != nil {
						config.ServerAddr = remoteIp
						config.AgentUrl = "http://" + config.ServerAddr
						_ = fileutil.WriteStringToFile(config.AgentConfig, config.ServerAddr, false)
					}
				}
			}()
			time.Sleep(30 * time.Second)
		}
	}()
}
func main() {
	Wsc := modules.NewWsClientManager()
	go kits.WritePid()
	go Wsc.Start()
	go modules.AgentBus()
	go modules.Heartbeat()
	go modules.CmdbAgent()
	go modules.MonitorAgent()
	// pprof
	select {
	case msg := <-config.Qch:
		kits.WriteLog(msg)
		os.Exit(0) //pid检测异常结束
	}
}
