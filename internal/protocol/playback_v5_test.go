package protocol

import (
	"testing"

	"github.com/beilunyang/openhongguo/sdk/model"
)

func TestVideoTabPairsUseIdentityDevice(t *testing.T) {
	id := SearchIdentity{
		DeviceID:  "device-id",
		InstallID: "install-id",
		CDID:      "cdid",
		Device: model.Device{
			Channel:   "xiaomi_8662_64",
			Model:     "23127PN0CC",
			OSVersion: "14",
		},
	}

	got := make(map[string]string)
	for _, pair := range VideoTabPairs("episode-id", id) {
		got[pair[0]] = pair[1]
	}

	for key, want := range map[string]string{
		"channel":     "xiaomi_8662_64",
		"device_type": "23127PN0CC",
		"os_version":  "14",
	} {
		if got[key] != want {
			t.Errorf("VideoTabPairs[%q] = %q, want %q", key, got[key], want)
		}
	}
}
