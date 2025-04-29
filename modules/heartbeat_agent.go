package modules

import (
	"agent/config"
	"agent/kits"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/shirou/gopsutil/v4/host"
	"time"
)

func Heartbeat() {
	for {
		func() {
			h, err := host.Info()
			defer func() {
				if r := recover(); r != nil {
					err = errors.New(fmt.Sprint(r))
				}
			}()
			if err == nil && config.WscAlive {
				// 发送心跳数据包
				s, _ := json.Marshal(map[string]string{"heartbeat": Encrypt.EncryptString(kits.MapToJson(
					map[string]interface{}{"host_id": h.HostID, "host_name": h.Hostname, "clamAv": ClamAv(),
						"clamRun": config.ClamRun, "agent_version": config.Version}), true)})
				config.SendChan <- string(s)
			}
			defer func() {
				if err != nil {
					kits.WriteLog(err.Error())
					config.HeartbeatErr = err
				}
			}()
		}()
		time.Sleep(time.Duration(config.HeartBeatInterval) * time.Second)
	}
}
