package systeminfo

import (
	"fmt"
	"github.com/telemac/goutils/net"
	"os"
	"runtime"
)

type SystemInfo struct {
	Platform   string `json:"platform"` // ex: linux/amd64
	Docker     bool   `json:"docker"`   // true if runnint in docker container
	Hostname   string `json:"hostname"`
	MacAddress string `json:"mac_address"`
}

func NewSystemInfo() (*SystemInfo, error) {
	var systemInfo SystemInfo
	err := systemInfo.Fill()
	return &systemInfo, err
}

func (si *SystemInfo) Fill() error {
	var err error
	si.Platform = runtime.GOOS + "/" + runtime.GOARCH
	si.Docker = isRunningInDockerContainer()
	si.Hostname, err = os.Hostname()
	if err != nil {
		return fmt.Errorf("unable to get hostname: %v", err)
	}
	si.MacAddress, err = net.GetMACAddress()
	if err != nil {
		return fmt.Errorf("unable to get MAC address: %v", err)
	}
	return nil
}

// isRunningInDockerContainer
func isRunningInDockerContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	return false
}
