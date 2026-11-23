package sign

import (
	"bufio"
	"encoding/hex"
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The vectors are x-medusa values the real SDK produced in the emulator
// with its clock and rand() pinned; the MedusaEnv and MedusaRandom fields
// were read back from those values.
func TestAssembleMedusaV5MatchesSDK(t *testing.T) {
	f, err := os.Open("testdata/medusa_v5.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	n := 0
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 1<<16), 1<<20)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cols := strings.Split(line, "\t")
		if len(cols) != 7 {
			t.Fatalf("bad vector: %d columns", len(cols))
		}
		rawURL, bodyHex, tsText, deviceID, envJSON, rndJSON, want := cols[0], cols[1], cols[2], cols[3], cols[4], cols[5], cols[6]
		var body []byte
		if bodyHex != "-" {
			if body, err = hex.DecodeString(bodyHex); err != nil {
				t.Fatal(err)
			}
		}
		ts, err := strconv.ParseUint(tsText, 10, 32)
		if err != nil {
			t.Fatal(err)
		}
		var env MedusaEnv
		var rnd MedusaRandom
		if err := json.Unmarshal([]byte(envJSON), &env); err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(rndJSON), &rnd); err != nil {
			t.Fatal(err)
		}
		env.Status = medusaEmulatorStatus // the vectors come from the emulator
		query := rawURL[strings.IndexByte(rawURL, '?')+1:]
		in, ok := newMedusaV5Input(query, body, uint32(ts))
		if !ok {
			t.Fatalf("vector %d selects an unsupported field-13 variant", n)
		}
		if got := assembleMedusaV5(in, deviceID, env, rnd); got != want {
			t.Errorf("vector %d (ts %d, body %v): x-medusa differs\n got %s\nwant %s", n, ts, body != nil, got, want)
		}
		n++
	}
	if n < 8 {
		t.Fatalf("only %d vectors", n)
	}
}

// The service answers playback requests whose x-medusa carries the
// emulator's status with an empty body (observed 2026-11-23), so the
// environment this package signs with must report a real device's.
func TestDefaultMedusaEnvReportsDeviceStatus(t *testing.T) {
	if got := DefaultMedusaEnv(0).Status; got != medusaDeviceStatus || got == medusaEmulatorStatus {
		t.Fatalf("status = %d, want %d", got, medusaDeviceStatus)
	}
	in, _ := newMedusaV5Input("device_id=1234567890123456", nil, 1763460000) // the digest variant does not matter here
	device := medusaV5Message(in, "1234567890123456", DefaultMedusaEnv(0), MedusaRandom{})
	emulator := DefaultMedusaEnv(0)
	emulator.Status = medusaEmulatorStatus
	if string(device) == string(medusaV5Message(in, "1234567890123456", emulator, MedusaRandom{})) {
		t.Fatal("status is not part of the message")
	}
	if want := string(protoSInt(21, medusaDeviceStatus)); !strings.Contains(string(device), want) {
		t.Fatalf("message lacks field 21 = %d", medusaDeviceStatus)
	}
}
