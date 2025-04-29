package modules

import (
	"agent/config"
	"agent/kits"
)

var (
	Encrypt = kits.NewEncrypt([]byte(config.CryptKey), 16)
)

func init() {
	go Scheduler()
}
