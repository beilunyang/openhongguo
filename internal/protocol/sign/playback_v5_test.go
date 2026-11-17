package sign

import (
	"strings"
	"testing"
)

func TestPlaybackV5SignKeepsTheTicketSlot(t *testing.T) {
	s := NewPlaybackV5Signer(1_794_000_000_000)
	body := []byte("gzip bytes")
	query, headers, err := s.Sign([][2]string{{"a", "1"}, {"_rticket", ""}, {"device_id", "7100000000000001"}}, 1_794_000_000_123, body, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(query, "a=1&_rticket=") || !strings.HasSuffix(query, "&device_id=7100000000000001") || strings.Count(query, "_rticket=") != 1 {
		t.Fatalf("query = %s", query)
	}
	if headers["x-medusa"] == "" || headers["x-helios"] == "" || headers["x-ss-stub"] == "" {
		t.Fatalf("headers = %v", headers)
	}
	query, _, err = s.Sign([][2]string{{"a", "1"}, {"device_id", "7100000000000001"}}, 1_794_000_000_123, body, nil)
	if err != nil || !strings.HasPrefix(query, "a=1&device_id=7100000000000001&_rticket=") {
		t.Fatalf("appended ticket query = %s err %v", query, err)
	}
}
