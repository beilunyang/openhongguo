package sign

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The vectors come from calling the SDK's own field-13 function
// (so+0x2aac34) in the emulator, with random inputs and timestamps across
// the uint32 range, for each of the 16 variants.
func TestF13V5MatchesSDK(t *testing.T) {
	f, err := os.Open("testdata/f13_v5.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	heads, tails := 0, 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.Split(line, ":")
		raw, err1 := hex.DecodeString(parts[0])
		variant, err2 := strconv.Atoi(parts[1])
		want, err3 := hex.DecodeString(parts[2])
		if err1 != nil || err2 != nil || err3 != nil || len(raw) != 52 || len(want) != 20 {
			t.Fatalf("bad vector %q", line)
		}
		var sm3 [32]byte
		var body, head [16]byte
		copy(sm3[:], raw[:32])
		copy(body[:], raw[32:48])
		copy(head[:], want[:16])
		ts := binary.LittleEndian.Uint32(raw[48:])

		if got := f13V5Tail(head); hex.EncodeToString(got) != parts[2][32:] {
			t.Errorf("variant %d tail(%x) = %x, want %s", variant, head, got, parts[2][32:])
		}
		tails++
		var got [16]byte
		switch variant {
		case 5:
			got = f13V5HeadMD5(sm3, body, ts)
		case 12:
			got = f13V5HeadSHA(sm3, body, ts)
		default:
			continue
		}
		if got != head {
			t.Errorf("variant %d head(%s) = %x, want %x", variant, parts[0], got, head)
		}
		heads++
	}
	if heads < 60 || tails < 320 {
		t.Fatalf("too few vectors: %d heads, %d tails", heads, tails)
	}
}

func TestF13V5RejectsUnsupportedVariants(t *testing.T) {
	for ts := uint32(1763460000); ts < 1763460100; ts++ {
		var sm3 [32]byte
		var body [16]byte
		_, ok := f13V5(sm3, body, ts)
		if v := f13V5Variant(sm3, body, ts); ok != f13V5Supported(v) {
			t.Fatalf("ts %d: variant %d, ok %v", ts, v, ok)
		}
	}
}
