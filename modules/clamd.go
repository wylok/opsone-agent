package modules

import (
	"agent/config"
	"agent/kits"
	"context"
	"github.com/duke-git/lancet/v2/datetime"
	"github.com/duke-git/lancet/v2/fileutil"
	"github.com/duke-git/lancet/v2/system"
	"github.com/lyimmi/go-clamd"
	"github.com/shirou/gopsutil/v4/host"
	"time"
)

var (
	socket1  = "/run/clamd.scan/clamd.sock"
	socket2  = "/run/clamav/clamd.ctl"
	platform = map[string]string{"centos": socket1, "rocky": socket1, "ubuntu": socket2, "debian": socket2}
)

func ClamAv() string {
	var (
		discover string
		ctx      = context.Background()
	)
	h, _ := host.Info()
	_, ok := platform[h.Platform]
	if ok && fileutil.IsExist(platform[h.Platform]) {
		c := clamd.NewClamd(clamd.WithUnix(platform[h.Platform]))
		ok, _ := c.Ping(ctx)
		if ok {
			discover = "clamAv"
			if time.Now().Day() != config.ClamDay && time.Now().Hour() == 1 {
				config.ClamDay = time.Now().Day()
				go func(c *clamd.Clamd) {
					if !fileutil.IsExist("/tmp/clam_file/") {
						_ = fileutil.CreateDir("/tmp/clam_file/")
					}
					kits.WriteLog("开始启动病毒扫描任务......")
					config.ClamRun = datetime.GetNowDateTime()
					ctx := context.Background()
					_, _, _ = system.ExecCommand("freshclam")
					_, _ = c.Reload(ctx)
					_, _, _ = system.ExecCommand("clamscan -r --move=/tmp/clam_file/ /")
					names, err := fileutil.ListFileNames("/tmp/clam_file/")
					if len(names) > 0 && err == nil {
						config.ClamRun = "发现病毒,感染文件已移至/tmp/clam_file"
					}
				}(c)
			}
		}
	}
	return discover
}
func InstallClamAv() (stdout, stderr string) {
	var (
		err           error
		scanFile      = "/etc/clamd.d/scan.conf"
		clamdFile     = "/etc/clamd.d/clamd.conf"
		freshclamFile = "/etc/freshclam.conf"
	)
	h, _ := host.Info()
	_, ok := platform[h.Platform]
	if ok {
		if h.Platform == "centos" || h.Platform == "rocky" {
			if fileutil.IsExist("/etc/yum.repos.d/epel.repo") {
				stdout, stderr, err = system.ExecCommand("yum -y install clamav-server clamav-data clamav-update" +
					" clamav-filesystem clamav clamav-scanner-systemd clamav-devel clamav-lib clamav-server-systemd")
				if err == nil && fileutil.IsExist(freshclamFile) && fileutil.IsExist(scanFile) {
					stdout, stderr, _ = system.ExecCommand("setsebool -P antivirus_can_scan_system 1")
					stdout, stderr, _ = system.ExecCommand("sed -i -e \"s/^Example/#Example/\" " + scanFile)
					stdout, stderr, _ = system.ExecCommand("sed -i \"s/#LocalSocket/LocalSocket/g\" " + scanFile)
					stdout, stderr, _ = system.ExecCommand("sed -i \"s/User clamscan/User root/g\" " + scanFile)
					stdout, stderr, _ = system.ExecCommand("sed -i -e \"s/^Example/#Example/\" " + freshclamFile)
					if fileutil.IsExist(clamdFile) {
						_ = fileutil.ClearFile(clamdFile)
					} else {
						fileutil.CreateFile(clamdFile)
					}
					for _, c := range []string{"LocalSocket " + platform[h.Platform], "LocalSocketMode 660"} {
						_ = fileutil.WriteStringToFile(clamdFile, c+"\n", true)
					}
					for _, c := range []string{"systemctl start clamav-freshclam", "systemctl enable clamav-freshclam",
						"systemctl enable clamd@scan", "systemctl start clamd@scan", "systemctl status clamd@scan"} {
						stdout, stderr, _ = system.ExecCommand(c)
					}
				}
			} else {
				stdout, stderr, err = system.ExecCommand("yum install -y epel-release")
			}
		}
		if h.Platform == "ubuntu" || h.Platform == "debian" {
			stdout, stderr, err = system.ExecCommand("apt install -y clamav-daemon clamav")
			if err == nil {
				for _, c := range []string{"systemctl start clamav-daemon", "systemctl enable clamav-daemon"} {
					stdout, stderr, _ = system.ExecCommand(c)
				}
			}
		}
	} else {
		stderr = "暂不支持" + h.Platform + "系统安装ClamAv"
	}
	return stdout, stderr
}

func RemoveClamAv() (stdout, stderr string) {
	h, _ := host.Info()
	_, ok := platform[h.Platform]
	if ok {
		if h.Platform == "centos" || h.Platform == "rocky" {
			for _, c := range []string{"systemctl stop clamav-freshclam", "systemctl disable clamav-freshclam",
				"systemctl disable clamd@scan", "systemctl stop clamd@scan"} {
				stdout, stderr, _ = system.ExecCommand(c)
			}
			stdout, stderr, _ = system.ExecCommand("yum -y remove clamav-server clamav-data clamav-update" +
				" clamav-filesystem clamav clamav-scanner-systemd clamav-devel clamav-lib clamav-server-systemd")
			err := fileutil.RemoveFile(platform[h.Platform])
			if err == nil {
				stdout = "already removed"
				stderr = ""
			}
		}
		if h.Platform == "ubuntu" || h.Platform == "debian" {
			for _, c := range []string{"systemctl stop clamav-daemon", "systemctl disable clamav-daemon"} {
				stdout, stderr, _ = system.ExecCommand(c)
			}
			stdout, stderr, _ = system.ExecCommand("apt remove -y clamav-daemon clamav")
			err := fileutil.RemoveFile(platform[h.Platform])
			if err == nil {
				stdout = "already removed"
				stderr = ""
			}
		}
	} else {
		stderr = "暂不支持" + h.Platform + "系统安装ClamAv"
	}
	return stdout, stderr
}
