package dispatch_test

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/beilunyang/openhongguo/adapters/httptransport"
	"github.com/beilunyang/openhongguo/adapters/memstore"
	"github.com/beilunyang/openhongguo/bindings/internal/dispatch"
	"github.com/beilunyang/openhongguo/bindings/internal/fixtures"
	"github.com/beilunyang/openhongguo/sdk"
)

func newClient(t *testing.T, origin string) *sdk.Client {
	t.Helper()
	tr := httptransport.New()
	opts := sdk.Options{Transport: tr, ProfileStore: memstore.New()}
	cfg, err := dispatch.ParseConfig(`{"apiOrigin":"` + origin + `","registerOrigin":"` + origin + `","retryAttempts":1}`)
	if err != nil {
		t.Fatal(err)
	}
	cfg.Apply(&opts)
	c, err := sdk.NewClient(opts)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// call 调用 name，并把结果经 JSON 往返解码到 out。
func call(t *testing.T, c *sdk.Client, name, args string, out any) error {
	t.Helper()
	v, err := dispatch.Call(context.Background(), c, name, args)
	if err != nil {
		return err
	}
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(b, out); err != nil {
		t.Fatal(err)
	}
	return nil
}

func TestCall(t *testing.T) {
	srv := httptest.NewServer(fixtures.Handler("../../../testdata/protocol"))
	defer srv.Close()
	c := newClient(t, srv.URL)

	var page struct {
		Items []struct{ ID, Title string } `json:"items"`
	}
	if err := call(t, c, "search", `["短剧",{"pageSize":20}]`, &page); err != nil {
		t.Fatal(err)
	}
	if len(page.Items) == 0 || page.Items[0].ID != "7100000000000101" {
		t.Fatalf("search = %+v", page)
	}

	var episodes struct {
		Items []struct {
			ID     string `json:"id"`
			Number int    `json:"number"`
		} `json:"items"`
	}
	if err := call(t, c, "listEpisodes", `["7100000000000101"]`, &episodes); err != nil {
		t.Fatal(err)
	}
	if len(episodes.Items) == 0 || episodes.Items[0].Number != 1 {
		t.Fatalf("listEpisodes = %+v", episodes)
	}

	var pb struct {
		Playback        struct{ Protected bool } `json:"playback"`
		ContentKey      *struct{}                `json:"contentKey"`
		ContentKeyError struct{ Code string }    `json:"contentKeyError"`
	}
	if err := call(t, c, "resolvePlayback", `["7100000000000201"]`, &pb); err != nil {
		t.Fatal(err)
	}
	// 样本里的密钥材料是假的，所以推导失败，但播放模型照常返回。
	if !pb.Playback.Protected || pb.ContentKey != nil || pb.ContentKeyError.Code != "DRM_PROTECTED" {
		t.Fatalf("resolvePlayback = %+v", pb)
	}
}

func TestCallErrors(t *testing.T) {
	srv := httptest.NewServer(fixtures.Handler("../../../testdata/protocol"))
	defer srv.Close()

	tests := []struct {
		name, origin, method, args, code string
		retryable                        bool
	}{
		{"rate limited", srv.URL + "/limited", "search", `["短剧"]`, "RATE_LIMITED", true},
		{"unknown method", srv.URL, "nope", `[]`, "INVALID_OPTIONS", false},
		{"bad arguments", srv.URL, "search", `[1]`, "INVALID_OPTIONS", false},
		{"bad JSON", srv.URL, "search", `{`, "INVALID_OPTIONS", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := dispatch.Call(context.Background(), newClient(t, tt.origin), tt.method, tt.args)
			if err == nil {
				t.Fatal("no error")
			}
			e := dispatch.Error(err)
			if e["code"] != tt.code || e["retryable"] != tt.retryable {
				t.Fatalf("Error(%v) = %v", err, e)
			}
		})
	}
}

func TestCallCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := dispatch.Call(ctx, newClient(t, "http://127.0.0.1:1"), "search", `["短剧"]`)
	if got := dispatch.Error(err)["code"]; got != "CANCELLED" {
		t.Fatalf("code = %v (%v)", got, err)
	}
}

func TestParseConfigRejectsBadJSON(t *testing.T) {
	_, err := dispatch.ParseConfig("{")
	if dispatch.Error(err)["code"] != "INVALID_OPTIONS" {
		t.Fatalf("err = %v", err)
	}
}

func TestDeviceResetsPassesThrough(t *testing.T) {
	for raw, want := range map[string]*int{
		`{}`:                 nil,
		`{"deviceResets":0}`: sdk.Int(0),
		`{"deviceResets":3}`: sdk.Int(3),
	} {
		cfg, err := dispatch.ParseConfig(raw)
		if err != nil {
			t.Fatal(err)
		}
		var opts sdk.Options
		cfg.Apply(&opts)
		if (want == nil) != (opts.DeviceResets == nil) || (want != nil && *want != *opts.DeviceResets) {
			t.Errorf("%s: DeviceResets = %v, want %v", raw, opts.DeviceResets, want)
		}
	}
}
