//go:build android && !cgo

package include

import (
	"os/exec"
	"strings"
	"time"
)

func init() {
	for _, getprop := range []string{"/system/bin/getprop", "getprop"} {
		output, err := exec.Command(getprop, "persist.sys.timezone").Output()
		if err != nil {
			continue
		}
		name := strings.TrimSpace(string(output))
		if name == "" {
			continue
		}
		location, err := time.LoadLocation(name)
		if err != nil {
			continue
		}
		time.Local = location
		return
	}
}
