package sign

import (
	"encoding/hex"
	"strings"
	"testing"
)

// The vectors below are fixed inputs and outputs of the public Argus 551 and
// Ladon algorithms. They are the agreed golden vectors from two independent
// open-source implementations of the same wire protocol:
//
//   - zfdang/hongguo-open-player PR #92, test/unit/argus.test.js
//   - aston314/hongguo-tv commit 522d87b, testdata/playback-sign-vectors.json
//
// Only the inputs and expected outputs are reproduced; none of their code is.
// They pin this independent Go implementation to the same protocol.

// testArgus mirrors how SignPlaybackPOST derives the bean fields from a signed
// query, so these asserts exercise the same path as production.
func testArgus(query string, body []byte, ts, nonce uint32) string {
	bodyMD5 := [16]byte{}
	if len(body) > 0 {
		bodyMD5 = md5Raw(body)
	}
	aid := playbackAID(query)
	return argusHeader(query, bodyMD5, ts, nonce, aid, queryValue(query, "device_id"), queryValue(query, "iid"))
}

func TestSM3PlaybackVectors(t *testing.T) {
	cases := map[string]string{
		"":    "1ab21d8355cfa17f8e61194831e81a8f22bec8c728fefb747ed035eb5082aa2b",
		"abc": "66c7f0f462eeedd9d1f2d46bdc10e4e24167c4875cf2f7a2297da02b8f4ba8e0",
	}
	for in, want := range cases {
		if got := sm3Sum([]byte(in)); hex.EncodeToString(got[:]) != want {
			t.Fatalf("sm3(%q) = %x, want %s", in, got, want)
		}
	}
	var zeros16 [16]byte
	if got := sm3Sum(zeros16[:]); hex.EncodeToString(got[:]) != "106e34a2b8c7bb13156cfdd0d91379dcc47543dcf9787c68ae5eb582620ae6e8" {
		t.Fatalf("sm3(16 zero bytes) = %x", got)
	}
}

func TestLadonGoldenVectors(t *testing.T) {
	// nonce 0x04030201 is little-endian seed 01 02 03 04 in both references.
	const nonce uint32 = 0x04030201
	cases := []struct {
		ts        uint32
		aid, want string
	}{
		{1_700_000_000, "1967", "AQIDBKM7Y7bWnOIUXmTybD6/MpGbuge4sBwUJsyHVivhtMsH"},
		{1_800_000_000, "8662", "AQIDBPhgjvKhYEWp2lodHDSBnITF8APHUjBLe2ja98bNrRn+"},
	}
	for _, c := range cases {
		if got := ladonHeader(c.ts, nonce, c.aid); got != c.want {
			t.Fatalf("ladon ts=%d aid=%s = %s, want %s", c.ts, c.aid, got, c.want)
		}
	}
}

// Vectors from aston314/hongguo-tv (Go, query-derived). These bind the bean to
// aid, device_id, iid, version_name, body MD5, timestamp and nonce.
func TestArgusGoldenVectorsAston(t *testing.T) {
	cases := []struct {
		query, body string
		ts, nonce   uint32
		want        string
	}{
		{
			query: "aid=1967&device_id=test-device&iid=test-install",
			ts:    1_700_000_000, nonce: 659662259,
			want: "8oGwZRLe2V38m+1gfqIeLa2MoV9ZBWosW5Y0NXKHAB2aEYRF86eNA7rLWm6xUCiwYMBDI/yq5ajsCmDF+Zws9KOfZO0mqezSYjF9/D5C1EliRxXJBBuZBaQZJgT6gakwiRnJkm+8F1ZVFIFO9GxkdwLxdxYidY2sLQjAsG8q4241ayJ/0zOxsVJNlqzPs7PfJb83zY6J1eIUaugTB6QnbXOKEQUgLhiYC0Uszr1vBP2Vj1cgtWu0IgouGM/ie3CohEAkY2oOkN/W1leWheBszijv",
		},
		{
			query: "aid=8662&device_id=7000000000000000001&iid=7000000000000000002&version_name=7.1.3.32&_rticket=1800000000999&ts=1800000000",
			body:  `{"video_id":"123","biz_param":{"video_platform":3}}`,
			ts:    1_800_000_000, nonce: 4294967295,
			want: "8oGzD48Lx0I+DCXDwk+l2r54SBRLSexXn2DlWQdGZE1UFZXj1GWup1ztpbF3eGeKcJAItaddhQwgYm7odM2byEYHtJ0pNvNUAoumWtQPqducerNIGe0Hs3z5V7oJDVgEySh6Kqc69jI8T+zU13SblzavSSZGSegC030jul66JPvjsBGzZoerKy88GDYDbumRNPCtqnfztLNNlGwmnA+0Tv+dnBC1xJ9rIEBThybfhq8wL5YqxxstLwuLGffOIlbxoQdheycfZBDpJG5kiK4eX7b3gr+PVkmCZBkJleZIYDl22UQS7nQGJKyu/a0U2MC7oWo=",
		},
		{
			query: "",
			ts:    1_700_000_000, nonce: 0,
			want: "8oHyrVYEy1xDCDlx64ko40qC6JC9VYY/ZkjJzEl50iSFDw8Yfph1hupLxfKrA9GswSEz0Yi4goA7OQ4u+H8ECdOaHRbHCQV+QFNgNz4TTRpioctLgrN97GbIKIQgZJjNRNiwbGqa+jIbLonncbTbsGIMv2WphwNMhXazKSewxik7/ecf0/SUOIiVd/3LhLMGeCIy+/ngOQr9VXnvmCb8HyzUbr01+pgwc1OKSfU29+FP4vYSq2WdmLCpvopUNQZhB8U=",
		},
	}
	for i, c := range cases {
		if got := testArgus(c.query, []byte(c.body), c.ts, c.nonce); got != c.want {
			t.Fatalf("aston argus[%d] mismatch\n got %s\nwant %s", i, got, c.want)
		}
	}
}

// Vector from zfdang/hongguo-open-player PR #92. It passes the bean fields
// explicitly (note install id is empty while the query carries iid=111), so it
// exercises the low-level argusHeader with decoupled params.
func TestArgusGoldenVectorZfdang(t *testing.T) {
	const query = "aid=1967&device_id=dev_aaaaaaaa&iid=111"
	const want = "8oHyrVYEy1xDCDlx64ko40qC6JC9VYY/ZkjJzEl50iSFDw8Yfph1hupLxfKrA9GswSEz0Yi4goA7OQ4u+H8ECdOaQDL4GduQ3LULSq9Q4iSEGbS3L6G24GXChtnmPr6n46M2PLlKpXvF7uE0lhH4yD+X69FpdM41gQwx5Y5U3NdwXFaxlqYFszyp7dlUE6BEBWqwDsiAH8Jk9TGKr/jfHt4fVmMc9zRyNRdGeV0x6aP//x7aV+8Po8a/9KK3UNG5KEpmIgzRT6NUttkBP9gRNUh3"
	var noBody [16]byte
	got := argusHeader(query, noBody, 1_700_000_000, 659662259, "1967", "dev_aaaaaaaa", "")
	if got != want {
		t.Fatalf("zfdang argus mismatch\n got %s\nwant %s", got, want)
	}
}

func TestArgusBindsEveryInput(t *testing.T) {
	const query = "aid=8662&device_id=7000000000000000001&iid=7000000000000000002&version_name=7.3.9.32&ts=1700000000"
	body := []byte(`{"video_id":"7543196048185887806"}`)
	base := testArgus(query, body, 1_700_000_000, 7)
	variants := map[string]string{
		"query":     testArgus(query+"&extra=1", body, 1_700_000_000, 7),
		"body":      testArgus(query, append(body, ' '), 1_700_000_000, 7),
		"timestamp": testArgus(query, body, 1_700_000_001, 7),
		"nonce":     testArgus(query, body, 1_700_000_000, 8),
	}
	for name, got := range variants {
		if got == base {
			t.Fatalf("argus ignored %s", name)
		}
	}
}

func TestSignPlaybackPOSTShape(t *testing.T) {
	ent := &Entropy{NowSecs: 1_700_000_000, GorgonRandom: 0x1234, HeliosRandom: 1, MessageRandom: 1, Random: 1, PackedRandom: 1, EnvLaunch: 100, EnvPID: 10001}
	pairs := [][2]string{{"aid", "8662"}, {"device_id", "7000000000000000001"}, {"iid", "7000000000000000002"}, {"version_name", "7.3.9.32"}}
	body := []byte(`{"req_type":1}`)
	_, headers := SignPlaybackPOST("/novel/player/video_model/v1/", pairs, 1_700_000_000_123, "7000000000000000001", "7.3.9.32", body, ent)

	if _, ok := headers["x-medusa"]; ok {
		t.Fatal("playback request must not carry x-medusa")
	}
	for _, h := range []string{"x-argus", "x-ladon", "x-gorgon", "x-helios", "x-khronos"} {
		if headers[h] == "" {
			t.Fatalf("missing %s", h)
		}
	}
	// A real Argus is a long base64 blob beginning with the 0xf2 0x81 magic
	// ("8o..."), not the old 4-byte timestamp stub.
	if len(headers["x-argus"]) < 64 || !strings.HasPrefix(headers["x-argus"], "8o") {
		t.Fatalf("x-argus not a 551 envelope: %s", headers["x-argus"])
	}
	if headers["x-ss-stub"] == "" {
		t.Fatal("POST must keep x-ss-stub")
	}
}
