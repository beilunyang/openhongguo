package sign

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestSignPOSTCoversBody(t *testing.T) {
	ent := &Entropy{NowSecs: 1_700_000_000, GorgonRandom: 0x1234, HeliosRandom: 1, MessageRandom: 1, Random: 1, PackedRandom: 1, EnvLaunch: 100, EnvPID: 10001}
	body := []byte(`{"series_id":"1"}`)
	pairs := [][2]string{{"aid", "8662"}}
	query, headers := SignPOST("/novel/player/video_detail/v1/", pairs, 1_700_000_000_123, "1", "7.3.9.32", body, ent)
	sum := md5Raw(body)
	if headers["x-ss-stub"] != strings.ToUpper(hex.EncodeToString(sum[:])) || headers["Content-Type"] == "" {
		t.Fatalf("stub headers missing")
	}
	if !strings.Contains(query, "aid=8662") || !strings.Contains(query, "ts=") {
		t.Fatalf("query = %s", query)
	}
	_, getHeaders := SignGET(pairs, 1_700_000_000_123, "1", "7.3.9.32", ent)
	if getHeaders["x-gorgon"] == headers["x-gorgon"] || getHeaders["x-ss-stub"] != "" {
		t.Fatal("POST signature did not cover the body")
	}
}

func TestSM3ABC(t *testing.T) {
	sum := sm3Sum([]byte("abc"))
	const want = "66c7f0f462eeedd9d1f2d46bdc10e4e24167c4875cf2f7a2297da02b8f4ba8e0"
	if hex.EncodeToString(sum[:]) != want {
		t.Fatalf("sm3 = %x", sum)
	}
}

// The expected values were checked against the reference algorithm; x-gorgon
// and x-helios have SDK vectors in TestGorgonHeliosV5MatchSDK.
func TestMedusaKeyHashAndXmxorMatchReference(t *testing.T) {
	hash, seed := keyHash(medusaSignKey(), 2_378_451_149)
	if hex.EncodeToString(hash[:]) != "1b35023431b99a949ab7b70fc50a30216076492a6229d102a7a33fa3a1104170" {
		t.Fatalf("key hash = %x", hash)
	}
	if hex.EncodeToString(seed[:]) != "b0dff9ff" {
		t.Fatalf("seed = %x", seed)
	}
	mixed := xmxor([]byte("0123456789abcdefghij"), hash[:])
	if hex.EncodeToString(mixed) != "5c69c37694106e8cbdb2e074619a1dda6c5a5a65" {
		t.Fatalf("xmxor = %x", mixed)
	}
}
