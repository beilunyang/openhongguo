package sdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/beilunyang/openhongguo/adapters/httptransport"
	"github.com/beilunyang/openhongguo/adapters/memstore"
	"github.com/beilunyang/openhongguo/sdk"
)

// fakeAPI 应答设备注册和搜索，并记下搜索请求的签名头。
func fakeAPI(t *testing.T) (*httptest.Server, func() http.Header) {
	var mu sync.Mutex
	var searched http.Header
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "device_register"):
			w.Write(fixture(t, "register-response.json"))
		case strings.Contains(r.URL.Path, "search"):
			mu.Lock()
			searched = r.Header.Clone()
			mu.Unlock()
			w.Write(fixture(t, "search-response.json"))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
		}
	}))
	t.Cleanup(srv.Close)
	return srv, func() http.Header { mu.Lock(); defer mu.Unlock(); return searched }
}

func TestSignerURLReplacesSignatureHeaders(t *testing.T) {
	api, searched := fakeAPI(t)
	var in map[string]string
	var token string
	signer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/feature-hash" {
			t.Errorf("signer path = %s", r.URL.Path)
		}
		token = r.Header.Get("X-Signer-Token")
		json.NewDecoder(r.Body).Decode(&in)
		w.Write([]byte(`{"headers":{"X-Medusa":"emu-medusa","X-Gorgon":"emu-gorgon","X-Khronos":"123"}}`))
	}))
	t.Cleanup(signer.Close)

	client, err := sdk.NewClient(sdk.Options{
		Transport: httptransport.New(), ProfileStore: memstore.New(),
		APIOrigin: api.URL, RegisterOrigin: api.URL,
		SignerURL: signer.URL + "/", SignerToken: "secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Search(context.Background(), "短剧", sdk.SearchOptions{}); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(in["url"], api.URL+"/") || !strings.Contains(in["url"], "device_id=") {
		t.Errorf("signer got url %q", in["url"])
	}
	if token != "secret" {
		t.Errorf("X-Signer-Token = %q", token)
	}
	h := searched()
	if h.Get("X-Medusa") != "emu-medusa" || h.Get("X-Gorgon") != "emu-gorgon" || h.Get("X-Khronos") != "123" {
		t.Errorf("search headers = %v", h)
	}
	// 签名服务没返回的签名头不能留着内置实现算的旧值。
	if h.Get("X-Argus") != "" || h.Get("X-Helios") != "" {
		t.Errorf("stale signature headers kept: %v", h)
	}
}

func TestSignerURLErrors(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		want       string
	}{
		{"refused", `{"error":"openhongguo-signer: signer refused the request","neptune":"-7"}`, 422, "refused the request (X-Neptune -7)"},
		{"unauthorized", `{"error":"openhongguo-signer: unauthorized"}`, 401, "openhongguo-signer: unauthorized"},
		{"no json", `oops`, 502, "HTTP 502"},
		{"no medusa", `{"headers":{}}`, 200, "no signature returned"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			api, searched := fakeAPI(t)
			signer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tc.status)
				w.Write([]byte(tc.body))
			}))
			t.Cleanup(signer.Close)

			client, err := sdk.NewClient(sdk.Options{
				Transport: httptransport.New(), ProfileStore: memstore.New(),
				APIOrigin: api.URL, RegisterOrigin: api.URL, SignerURL: signer.URL,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Search(context.Background(), "短剧", sdk.SearchOptions{})
			// SDK 不能把签名服务的错误吞成笼统的传输错误。
			if !errors.Is(err, sdk.ErrSignerUnavailable) || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("err = %v", err)
			}
			if searched() != nil {
				t.Fatal("search sent despite signing failure")
			}
		})
	}
}

func TestSignerURLMustBeHTTP(t *testing.T) {
	for _, u := range []string{"signer.example.com", "ftp://x", "http://"} {
		_, err := sdk.NewClient(sdk.Options{
			Transport: httptransport.New(), ProfileStore: memstore.New(), SignerURL: u,
		})
		if !errors.Is(err, sdk.ErrInvalidOptions) {
			t.Errorf("SignerURL %q: err = %v", u, err)
		}
	}
}
