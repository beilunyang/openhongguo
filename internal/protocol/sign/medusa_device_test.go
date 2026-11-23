package sign

import (
	"bytes"
	"encoding/hex"
	"strings"
	"testing"
)

const deviceQuery = "device_platform=android&device_type=23127PN0CC&device_brand=Xiaomi&os_version=14&version_name=7.4.3.32&device_id=1234567890123456"

func TestMedusaDeviceOfReadsTheQuery(t *testing.T) {
	if d := medusaDeviceOf("aid=8662&device_id=1234567890123456", "1234567890123456"); d != nil {
		t.Fatalf("query without a model gave %+v", d)
	}
	d := medusaDeviceOf(deviceQuery, "1234567890123456")
	if d == nil || d.Model != "23127PN0CC" || d.Brand != "Xiaomi" || d.OSVersion != "14" || d.AppVersion != "7.4.3.32" {
		t.Fatalf("device = %+v", d)
	}
	if again := medusaDeviceOf(deviceQuery, "1234567890123456"); *again != *d {
		t.Fatalf("one device ID gave two devices: %+v, %+v", d, again)
	}
	if other := medusaDeviceOf(deviceQuery, "1234567890123457"); other.InstallMillis == d.InstallMillis && other.StorageFreeGB == d.StorageFreeGB {
		t.Fatal("two device IDs gave the same derived values")
	}
	if d.InstallMillis >= medusaInstallAnchor || d.StorageFreeGB >= d.StorageGB || d.MemoryFreeGB >= d.MemoryGB {
		t.Fatalf("implausible device: %+v", d)
	}
}

// A device-shaped message carries what the request states and the flags of
// a real device; the emulator's layout is unchanged without a device.
func TestMedusaV5MessageDeviceLayout(t *testing.T) {
	const deviceID = "1234567890123456"
	in, _ := newMedusaV5Input(deviceQuery, nil, 1763460000) // the digest variant does not matter here
	env := DefaultMedusaEnv(1763459000000)
	emulator := medusaV5Message(in, deviceID, env, MedusaRandom{})
	env.Device = medusaDeviceOf(deviceQuery, deviceID)
	device := medusaV5Message(in, deviceID, env, MedusaRandom{})

	for name, want := range map[string][]byte{
		"app version":   protoString(7, "7.4.3.32"),
		"flags":         protoBytes(10, medusaDeviceFlags),
		"status":        protoSInt(21, medusaDeviceStatus),
		"model":         protoString(30, "23127PN0CC"),
		"brand":         protoString(31, "Xiaomi"),
		"os version":    protoString(22, "14"),
		"uptime":        protoSInt(1, 1000),
		"no permission": protoString(6, noPerm),
	} {
		if !bytes.Contains(device, want) {
			t.Errorf("device message lacks %s", name)
		}
	}
	if !bytes.Contains(emulator, protoBytes(10, medusaEmulatorFlags)) || bytes.Contains(emulator, []byte("23127PN0CC")) {
		t.Error("emulator layout changed")
	}
	if n := strings.Count(string(device), notSet); n >= strings.Count(string(emulator), notSet) {
		t.Errorf("device message has %d placeholders", n)
	}
	first := medusaV5Message(in, deviceID, env, MedusaRandom{})
	second := medusaV5Message(in, deviceID, env, MedusaRandom{})
	if bytes.Equal(first, second) {
		t.Error("field 15's counter did not advance between signatures")
	}
}

// x-gorgon carries the first two flag bytes and is keyed by them.
func TestGorgonFlags(t *testing.T) {
	emulator := xGorgon("a=1", nil, false, 1763460000, 0x1234)
	if got := xGorgonFlags("a=1", nil, false, 1763460000, 0x1234, medusaEmulatorFlags); got != emulator {
		t.Fatalf("emulator flags changed x-gorgon: %s != %s", got, emulator)
	}
	device, _ := hex.DecodeString(xGorgonFlags("a=1", nil, false, 1763460000, 0x1234, medusaDeviceFlags))
	if device[4] != 0x04 || device[5] != 0x05 {
		t.Fatalf("flag bytes = %x", device[4:6])
	}
	if hex.EncodeToString(device[6:]) == emulator[12:] {
		t.Fatal("flags do not key the cipher")
	}
}

func TestSignedRequestsUseTheDeviceLayout(t *testing.T) {
	pairs := [][2]string{{"device_type", "23127PN0CC"}, {"device_brand", "Xiaomi"}, {"os_version", "14"}, {"version_name", "7.4.3.32"}, {"device_id", "1234567890123456"}}
	_, headers := SignGET(pairs, 1763460000138, "1234567890123456", "7.4.3.32", nil)
	if g := headers["x-gorgon"]; g[8:12] != "0405" {
		t.Fatalf("x-gorgon = %s", g)
	}
}
