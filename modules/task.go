package modules

import (
	"agent/config"
	"agent/kits"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/duke-git/lancet/v2/netutil"
	"github.com/duke-git/lancet/v2/system"
	"github.com/jakecoffman/cron"
	"os"
)

func Scheduler() {
	c := cron.New()
	c.AddFunc("*/30 * * * * *", CheckAgentDog, "CheckAgentDog")
	c.Start()
}
func CheckAgentDog() {
	_ = os.MkdirAll(config.AgentPath, 0755)
	if !config.Uninstall {
		kits.CheckAgentDogPid()
		if !fileutil.IsExist(config.AgentPath + "/config.ini") {
			_ = netutil.DownloadFile(config.AgentPath+"/config.ini", config.AgentUrl+"/api/v1/conf/config.ini")
		}
		if !fileutil.IsExist(config.AgentDogFile) {
			_ = netutil.DownloadFile(config.AgentDogFile, config.AgentUrl+"/api/v1/ag/opsone-dog")
		}
		if fileutil.IsExist(config.AgentDogFile) {
			_ = os.Chmod(config.AgentDogFile, 0755)
			if fileutil.IsExist(config.AgentDogPidFile) {
				if !kits.CheckAgentDogPid() {
					_, stderr, err := system.ExecCommand(config.AgentDogFile + " start")
					if err != nil || stderr != "" {
						_ = fileutil.RemoveFile(config.AgentDogFile)
					}
				}
			} else {
				_, stderr, err := system.ExecCommand(config.AgentDogFile + " start")
				if err != nil || stderr != "" {
					_ = fileutil.RemoveFile(config.AgentDogFile)
				}
			}
		}
		if !fileutil.IsExist(config.AgentService) {
			err := netutil.DownloadFile(config.AgentService, config.AgentUrl+"/api/v1/ag/opsone.service")
			if err == nil {
				_ = os.Chmod(config.AgentService, 0755)
				_, _, _ = system.ExecCommand("systemctl daemon-reload")
				_, _, _ = system.ExecCommand("systemctl enable opsone")
			}
		}
	}
}
