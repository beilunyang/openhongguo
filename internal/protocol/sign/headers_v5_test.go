package sign

import (
	"bufio"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The vectors are headers the real SDK produced in the emulator. Both carry
// their random in the clear: gorgon in bytes 2-3, helios in its first 4 bytes.
func TestGorgonHeliosV5MatchSDK(t *testing.T) {
	f, err := os.Open("testdata/headers_v5.tsv")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	n := 0
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		cols := strings.Split(line, "\t")
		rawURL, bodyHex, tsText, gorgon, helios := cols[0], cols[1], cols[2], cols[3], cols[4]
		var body []byte
		if bodyHex != "-" {
			body, _ = hex.DecodeString(bodyHex)
		}
		ts64, _ := strconv.ParseUint(tsText, 10, 32)
		ts := uint32(ts64)
		query := rawURL[strings.IndexByte(rawURL, '?')+1:]
		g, _ := hex.DecodeString(gorgon)
		if got := xGorgon(query, body, body != nil, ts, uint32(g[2])|uint32(g[3])<<8); got != gorgon {
			t.Errorf("x-gorgon = %s, want %s", got, gorgon)
		}
		h, _ := base64.StdEncoding.DecodeString(helios)
		if got := heliosV5(ts, binary.LittleEndian.Uint32(h[:4])); got != helios {
			t.Errorf("x-helios = %s, want %s", got, helios)
		}
		n++
	}
	if n < 8 {
		t.Fatalf("only %d vectors", n)
	}
}
