package sign

import (
	"crypto/sha256"
	"encoding/binary"
	"sync"
	"sync/atomic"
	"time"
)

// MedusaDevice is the phone a request claims to come from. With it,
// x-medusa carries the environment in the shape the SDK reports on a real
// device (decoded from the 7.4.5.32 app on 2026-11-23) instead of the
// emulator's "!notset!" placeholders.
//
// Only the layout and the values the request itself states (model, brand,
// OS and app version) are known. The storage, memory and install-time
// fields are plausible values derived from the device ID, so one device
// reports the same ones in every process. What the app fills with data
// this package cannot produce (two device tokens, field 15's fifth
// counter, environment fields 2, 3, 5, 13, 16, 17, 22 and 23, the board
// names) stays absent or "!notset!".
type MedusaDevice struct {
	AppVersion, Model, Brand, OSVersion string
	// InstallMillis is when the app was installed (environment device
	// fields 25 and 27).
	InstallMillis int64
	// StorageGB, StorageFreeGB, MemoryGB and MemoryFreeGB are device
	// fields 16-19; fields 20 and 21 are twice the storage ones.
	StorageGB, StorageFreeGB, MemoryGB, MemoryFreeGB float32
}

const (
	noPerm    = "!noperm!"
	noPermInt = -888888
	// medusaInstallAnchor is 2026-11-23 13:33 UTC, the day of the capture;
	// install times are spread over the 20 days before it.
	medusaInstallAnchor = 1795440816000
)

var (
	medusaEmulatorFlags = []byte{0x08, 0, 0, 0, 0, 0, 0, 0}
	// medusaDeviceFlags is message field 10 on a real device. Its first two
	// bytes are also bytes 4 and 5 of x-gorgon.
	medusaDeviceFlags = []byte{0x04, 0x05, 0, 0, 0, 0, 0, 0}
)

// flags is message field 10.
func (env MedusaEnv) flags() []byte {
	if env.Device != nil {
		return medusaDeviceFlags
	}
	return medusaEmulatorFlags
}

// medusaDeviceOf reads the device a signed query describes. It returns nil
// for a query without a model, which keeps the emulator's layout.
func medusaDeviceOf(query, deviceID string) *MedusaDevice {
	model := queryValue(query, "device_type")
	if model == "" {
		return nil
	}
	h := sha256.Sum256([]byte("medusa-device\x00" + deviceID))
	pick := func(i int, n uint32) uint32 { return binary.LittleEndian.Uint32(h[i*4:]) % n }
	storage := []float32{106.83, 108.57, 229.36, 233.91, 487.62}[pick(0, 5)] + float32(pick(1, 400))/1000
	memory := []float32{5.37, 7.42, 11.18, 15.07}[pick(2, 4)] + float32(pick(3, 90))/1000
	const day = 24 * 60 * 60 * 1000
	return &MedusaDevice{
		AppVersion:    queryValue(query, "version_name"),
		Model:         model,
		Brand:         queryValue(query, "device_brand"),
		OSVersion:     queryValue(query, "os_version"),
		InstallMillis: medusaInstallAnchor - day - int64(pick(4, 20*day)),
		StorageGB:     storage,
		StorageFreeGB: storage * (0.25 + float32(pick(5, 500))/1000),
		MemoryGB:      memory,
		MemoryFreeGB:  0.35 + float32(pick(6, 900))/1000,
	}
}

var (
	medusaCounterOnce sync.Once
	medusaCounter     atomic.Int64
)

// nextMedusaCount is message field 15's first counter: on a device it grows
// by one with each signature of the process.
func nextMedusaCount() int64 {
	medusaCounterOnce.Do(func() { medusaCounter.Store(20 + time.Now().UnixNano()%100) })
	return medusaCounter.Add(1)
}

// medusaV5DeviceBlock is environment field 12 in a real device's layout.
func medusaV5DeviceBlock(deviceID string, env MedusaEnv, nowMillis int64) []byte {
	d := env.Device
	str := func(tag uint32, s string) []byte {
		if s == "" {
			s = notSet
		}
		return protoString(tag, s)
	}
	return concat(
		protoSInt(1, 1),
		protoString(3, appID),
		protoString(4, deviceID),
		protoString(5, notSet),
		protoString(6, noPerm),
		protoSInt(7, noPermInt),
		protoSInt(8, noPermInt),
		protoSInt(9, 2),
		protoSInt(10, noPermInt),
		protoString(11, notSet),
		protoString(12, "Asia/Shanghai,8"),
		protoString(13, "zh_CN"),
		protoSInt(14, 8),
		protoFloat(16, d.StorageGB),
		protoFloat(17, d.StorageFreeGB),
		protoFloat(18, d.MemoryGB),
		protoFloat(19, d.MemoryFreeGB),
		protoFloat(20, d.StorageGB*2),
		protoFloat(21, d.StorageFreeGB*2),
		str(22, d.OSVersion),
		protoSInt(23, 60),
		protoSInt(24, 10),
		protoSInt(25, d.InstallMillis),
		protoSInt(26, env.LaunchMillis+540),
		protoSInt(27, d.InstallMillis),
		protoSInt(28, nowMillis),
		protoSInt(29, 1),
		str(30, d.Model),
		str(31, d.Brand),
		protoString(32, notSet),
		protoString(33, notSet),
		protoString(34, notSet),
		str(35, d.Brand),
		protoString(36, notSet),
		protoSInt(38, 3),
		protoSInt(40, env.LaunchMillis),
	)
}
