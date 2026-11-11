package sdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/sdk"
)

func TestListDanmakuPagesToTheEpisodeEnd(t *testing.T) {
	var paths []string
	var bodies []map[string]any
	details := 0
	// next is the window the fake service answers after the first page.
	next := 30000
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case r.URL.Path == protocol.PathSeries:
			details++
			_, _ = w.Write(fixture(t, "series-response.json"))
		case strings.HasPrefix(r.URL.Path, "/novel/commentapi/comment/list/"):
			if r.Header.Get("x-ss-stub") == "" || r.URL.Query().Get("device_id") == "" {
				t.Errorf("unsigned danmaku request: %v", r.Header)
			}
			paths = append(paths, r.URL.Path)
			body := map[string]any{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			bodies = append(bodies, body)
			if len(bodies) == 1 {
				_, _ = w.Write(fixture(t, "danmaku-response.json"))
				return
			}
			// The service says there is more, but the next window starts at
			// the end of the episode.
			fmt.Fprintf(w, `{"code":0,"data":{"data_list":[],"common_list_info":{"cursor":"{}","has_more":true},"extra":{"next_query_danmaku_list_time":%d}}}`, next)
		default:
			http.NotFound(w, r)
		}
	}))
	ctx := context.Background()
	page, err := client.ListDanmaku(ctx, "7100000000000101", "7100000000000201", sdk.DanmakuOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if paths[0] != "/novel/commentapi/comment/list/7100000000000201/v1/" {
		t.Fatalf("path = %s", paths[0])
	}
	param, _ := bodies[0]["business_param"].(map[string]any)
	if bodies[0]["comment_type"] != float64(20) || bodies[0]["group_id"] != "7100000000000201" || bodies[0]["cursor"] != "" ||
		param["book_id"] != "7100000000000101" || param["start_offset_time"] != float64(0) ||
		param["playlet_item_duration"] != float64(91000) {
		t.Fatalf("body = %v", bodies[0])
	}
	if len(page.Items) != 2 || page.Items[0].Text != "Fixture danmaku one" ||
		page.StartMS != 0 || page.EndMS != 30000 || !page.HasMore || page.Cursor == "" {
		t.Fatalf("page = %+v", page)
	}

	next = 91000
	page, err = client.ListDanmaku(ctx, "7100000000000101", "7100000000000201", sdk.DanmakuOptions{Cursor: page.Cursor, PageSize: 150})
	if err != nil {
		t.Fatal(err)
	}
	param, _ = bodies[1]["business_param"].(map[string]any)
	if c, _ := bodies[1]["cursor"].(string); !strings.Contains(c, `"end_offset_time":30000`) ||
		param["start_offset_time"] != float64(30000) || param["playlet_item_duration"] != float64(91000) || bodies[1]["count"] != float64(150) {
		t.Fatalf("second body = %v", bodies[1])
	}
	if details != 1 {
		t.Fatalf("series detail read %d times; the cursor should carry the duration", details)
	}
	if page.HasMore || page.Cursor != "" || page.StartMS != 30000 {
		t.Fatalf("last page = %+v", page)
	}
}

func TestListDanmakuRejectsBadInput(t *testing.T) {
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathSeries:
			_, _ = w.Write(fixture(t, "series-response.json"))
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	ctx := context.Background()
	if _, err := client.ListDanmaku(ctx, "7100000000000101", "../x", sdk.DanmakuOptions{}); err == nil {
		t.Fatal("malformed episode ID accepted")
	}
	if _, err := client.ListDanmaku(ctx, "7100000000000101", "7100000000000299", sdk.DanmakuOptions{}); err == nil || !strings.Contains(err.Error(), "not in series") {
		t.Fatalf("episode outside the series: %v", err)
	}
	other, _ := protocol.EncodeDanmakuCursor(protocol.DanmakuCursor{EpisodeID: "7100000000000202", Duration: 96000, Start: 30000})
	if _, err := client.ListDanmaku(ctx, "7100000000000101", "7100000000000201", sdk.DanmakuOptions{Cursor: other}); err == nil {
		t.Fatal("another episode's cursor accepted")
	}
	if _, err := client.ListDanmaku(ctx, "7100000000000101", "7100000000000201", sdk.DanmakuOptions{Cursor: "w1.x"}); err == nil {
		t.Fatal("malformed cursor accepted")
	}
}

func TestListDanmakuStartsFromAPlayPosition(t *testing.T) {
	var bodies []map[string]any
	details := 0
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case r.URL.Path == protocol.PathSeries:
			details++
			_, _ = w.Write(fixture(t, "series-response.json"))
		case strings.HasPrefix(r.URL.Path, "/novel/commentapi/comment/list/"):
			body := map[string]any{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			bodies = append(bodies, body)
			// The service answers the window that holds the position.
			_, _ = w.Write([]byte(`{"code":0,"data":{"data_list":[],"common_list_info":{"has_more":true,
				"cursor":"{\"start_offset_time\":60000,\"end_offset_time\":90000}"},"extra":{"next_query_danmaku_list_time":90000}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	ctx := context.Background()
	page, err := client.ListDanmaku(ctx, "7100000000000101", "7100000000000201", sdk.DanmakuOptions{StartMs: 75000})
	if err != nil {
		t.Fatal(err)
	}
	param, _ := bodies[0]["business_param"].(map[string]any)
	if param["start_offset_time"] != float64(75000) || bodies[0]["cursor"] != "" || param["playlet_item_duration"] != float64(91000) {
		t.Fatalf("body = %v", bodies[0])
	}
	// The next window starts at 90 s, before the 91 s end, so there is more.
	if page.StartMS != 60000 || page.EndMS != 90000 || !page.HasMore || page.Cursor == "" {
		t.Fatalf("page = %+v", page)
	}

	// Seeking again, in this or another episode of the series, does not read
	// the series detail again.
	if _, err := client.ListDanmaku(ctx, "7100000000000101", "7100000000000202", sdk.DanmakuOptions{StartMs: 10000}); err != nil {
		t.Fatal(err)
	}
	if details != 1 {
		t.Fatalf("series detail read %d times", details)
	}
	param, _ = bodies[1]["business_param"].(map[string]any)
	if bodies[1]["group_id"] != "7100000000000202" || param["playlet_item_duration"] != float64(96000) {
		t.Fatalf("second episode body = %v", bodies[1])
	}

	// A position past the end has nothing to read.
	page, err = client.ListDanmaku(ctx, "7100000000000101", "7100000000000201", sdk.DanmakuOptions{StartMs: 91000})
	if err != nil || len(page.Items) != 0 || page.HasMore || len(bodies) != 2 {
		t.Fatalf("past the end: page %+v err %v, %d requests", page, err, len(bodies))
	}

	for _, opts := range []sdk.DanmakuOptions{{StartMs: -1}, {StartMs: 1000, Cursor: "m1.x"}} {
		if _, err := client.ListDanmaku(ctx, "7100000000000101", "7100000000000201", opts); !errors.Is(err, sdk.ErrInvalidOptions) {
			t.Fatalf("%+v: err = %v", opts, err)
		}
	}
}
