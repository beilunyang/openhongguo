package sdk_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/beilunyang/openhongguo/adapters/httptransport"
	"github.com/beilunyang/openhongguo/adapters/memstore"
	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "testdata", "protocol", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

type captured struct {
	Method   string
	Path     string
	RawQuery string
	Header   http.Header
	Body     string
}

type logCapture struct{ lines []string }

func (l *logCapture) Printf(format string, args ...any) {
	l.lines = append(l.lines, fmt.Sprintf(format, args...))
}

func testClient(t *testing.T, handler http.Handler) (*sdk.Client, *logCapture) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	transport := httptransport.New()
	logs := &logCapture{}
	transport.SetLogger(logs)
	client, err := sdk.NewClient(sdk.Options{
		Transport:          transport,
		ProfileStore:       memstore.New(),
		APIOrigin:          srv.URL,
		RegisterOrigin:     srv.URL,
		DistributionOrigin: srv.URL,
		Logger:             logs,
		Clock:              func() time.Time { return time.UnixMilli(1_700_000_000_123).UTC() },
	})
	if err != nil {
		t.Fatal(err)
	}
	return client, logs
}

func happyHandler(t *testing.T, seen *[]captured) http.Handler {
	t.Helper()
	files := map[string]string{
		protocol.PathDiscover: "discover-response.json",
		protocol.PathSearch:   "search-response.json",
		protocol.PathSeries:   "series-response.json",
		protocol.PathPlayback: "drm-playback.json",
		protocol.PathRegister: "register-response.json",
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*seen = append(*seen, captured{Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery, Header: r.Header.Clone(), Body: string(body)})
		name, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, name))
	})
}

func TestAnonymousReadsExposeIDsAndTitles(t *testing.T) {
	var seen []captured
	client, logs := testClient(t, happyHandler(t, &seen))
	ctx := context.Background()

	discover, err := client.Discover(ctx, sdk.DiscoverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(discover.Items) != 2 || discover.Items[0].ID != "7100000000000101" || discover.Items[0].Title != "Fixture Lantern Court" {
		t.Fatalf("discover = %+v", discover.Items)
	}
	again, err := client.Discover(ctx, sdk.DiscoverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if again.Items[0].ID != discover.Items[0].ID || again.Items[1].Title != discover.Items[1].Title {
		t.Fatalf("second discover = %+v", again.Items)
	}

	search, err := client.Search(ctx, "短剧", sdk.SearchOptions{PageSize: 20})
	if err != nil {
		t.Fatal(err)
	}
	if search.Items[0].ID != "7100000000000101" || search.Items[0].Title != "Fixture Lantern Court" {
		t.Fatalf("search = %+v", search.Items)
	}

	series, err := client.GetSeries(ctx, "7100000000000101")
	if err != nil {
		t.Fatal(err)
	}
	if series.ID != "7100000000000101" || series.Title != "Fixture Lantern Court" {
		t.Fatalf("series = %+v", series)
	}
	episodes, err := client.ListEpisodes(ctx, "7100000000000101", sdk.EpisodeOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if episodes.Items[0].Number != 1 || episodes.Items[0].ID != "7100000000000201" || episodes.Items[0].Title != "Episode One" {
		t.Fatalf("episodes = %+v", episodes.Items)
	}
	if episodes.Items[1].Number != 2 {
		t.Fatalf("second number = %d", episodes.Items[1].Number)
	}

	playback, err := client.ResolvePlayback(ctx, "7100000000000201", sdk.PlaybackOptions{})
	if !errors.Is(err, sdk.ErrDRMProtected) {
		t.Fatalf("playback err = %v", err)
	}
	if playback == nil || !playback.Protected || len(playback.Streams) != 1 {
		t.Fatalf("playback = %+v", playback)
	}
	if errors.Is(err, sdk.ErrAuthenticationRequired) {
		t.Fatal("DRM was reported as authentication")
	}
	if strings.Contains(err.Error(), "cenc") || strings.Contains(err.Error(), "encrypt_info") || strings.Contains(err.Error(), "?") {
		t.Fatalf("DRM error leaked media material: %s", err)
	}

	assertAnonymousWire(t, seen)
	for _, line := range logs.lines {
		if strings.Contains(line, "device_id=") || strings.Contains(line, "X-SS-STUB") || strings.Contains(line, "7100000000000001") || strings.Contains(line, "?") {
			t.Fatalf("log leaked request material: %s", line)
		}
	}
}

func TestDiscoverFollowsUnlimitedCell(t *testing.T) {
	const session = "feed-session-fixture"
	tab := []byte(`{"code":0,"data":{"tab_item":[{"tab_type":16,"title":"推荐","session_id":"` + session + `","client_template":7,"cell_data":[{"cell_id_str":"7100000000000301","show_type":407,"cell_data":[{"video_data":{"series_id":"7100000000000101","video_detail":{"series_title":"Fixture Lantern Court"}}}]}]}]}}`)
	change := []byte(`{"code":0,"data":{"has_more":true,"cell_view":{"cell_data":[{"video_data":{"series_id":"7100000000000102","video_detail":{"series_title":"Fixture River Market"}}},{"video_data":{"series_id":"7100000000000101","video_detail":{"series_title":"Fixture Lantern Court"}}}]}}}`)
	var seen []captured
	client, logs := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		seen = append(seen, captured{Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery, Header: r.Header.Clone(), Body: string(body)})
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathDiscover:
			_, _ = w.Write(tab)
		case protocol.PathCellChange:
			_, _ = w.Write(change)
		default:
			http.NotFound(w, r)
		}
	}))
	page, err := client.Discover(context.Background(), sdk.DiscoverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || !page.HasMore || page.Items[0].ID != "7100000000000101" || page.Items[1].Title != "Fixture River Market" {
		t.Fatalf("page = %+v has_more=%v", page.Items, page.HasMore)
	}
	var followed bool
	for _, hit := range seen {
		if hit.Path != protocol.PathCellChange {
			continue
		}
		followed = true
		if hit.Header.Get("Cookie") != "" || hit.Header.Get("x-gorgon") == "" {
			t.Fatal("cell change wire is wrong")
		}
		q, _ := url.ParseQuery(hit.RawQuery)
		if q.Get("cell_id") != "7100000000000301" || q.Get("change_type") != "0" || q.Get("client_req_type") != "2" || q.Get("tab_type") != "16" || q.Get("show_type") != "407" || q.Get("limit") != "20" || q.Get("client_template") != "7" {
			t.Fatal("cell change query missing business fields")
		}
		if q.Get("session_id") != session {
			t.Fatal("cell change dropped the feed session")
		}
	}
	if !followed {
		t.Fatal("discover did not request the unlimited page")
	}
	for _, line := range logs.lines {
		if strings.Contains(line, session) || strings.Contains(line, "?") {
			t.Fatalf("log leaked the feed session: %s", line)
		}
	}
}

func TestDiscoverPageSizeUsesLimit(t *testing.T) {
	const session = "feed-session-fixture"
	tab := []byte(`{"code":0,"data":{"tab_item":[{"tab_type":16,"title":"推荐","session_id":"` + session + `","client_template":7,"cell_data":[{"cell_id_str":"7100000000000301","show_type":407,"cell_data":[{"video_data":{"series_id":"7100000000000101","video_detail":{"series_title":"Fixture Lantern Court"}}}]}]}]}}`)
	change := []byte(`{"code":0,"data":{"has_more":false,"cell_view":{"cell_data":[{"video_data":{"series_id":"7100000000000102","video_detail":{"series_title":"Fixture River Market"}}}]}}}`)
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathDiscover:
			_, _ = w.Write(tab)
		case protocol.PathCellChange:
			q, _ := url.ParseQuery(r.URL.RawQuery)
			if q.Get("limit") != "30" {
				t.Fatalf("limit = %q", q.Get("limit"))
			}
			_, _ = w.Write(change)
		default:
			http.NotFound(w, r)
		}
	}))
	if _, err := client.Discover(context.Background(), sdk.DiscoverOptions{PageSize: 30}); err != nil {
		t.Fatal(err)
	}
}

func TestDiscoverCursorLoadsNextOffset(t *testing.T) {
	const session = "feed-session-fixture"
	tab := []byte(`{"code":0,"data":{"tab_item":[{"tab_type":16,"session_id":"` + session + `","client_template":7,"cell_data":[{"cell_id_str":"7100000000000301","show_type":407,"cell_data":[{"video_data":{"series_id":"7100000000000101","video_detail":{"series_title":"Fixture Lantern Court","episode_cnt":2}}}]}]}]}}`)
	first := []byte(`{"code":0,"data":{"has_more":true,"next_offset":6,"session_id":"` + session + `","cell_view":{"cell_data":[{"video_data":{"series_id":"7100000000000102","video_detail":{"series_title":"Fixture River Market","episode_cnt":3}}}]}}}`)
	second := []byte(`{"code":0,"data":{"has_more":false,"next_offset":0,"cell_view":{"cell_data":[{"video_data":{"series_id":"7100000000000103","video_detail":{"series_title":"Fixture North Gate"}}}]}}}`)
	var offsets []string
	client, logs := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathDiscover:
			_, _ = w.Write(tab)
		case protocol.PathCellChange:
			q, _ := url.ParseQuery(r.URL.RawQuery)
			offsets = append(offsets, q.Get("offset"))
			if q.Get("offset") == "6" {
				_, _ = w.Write(second)
				return
			}
			_, _ = w.Write(first)
		default:
			http.NotFound(w, r)
		}
	}))
	page, err := client.Discover(context.Background(), sdk.DiscoverOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Cursor == "" || !page.HasMore || len(page.Items) != 2 {
		t.Fatalf("first page cursor=%q items=%d has_more=%v", page.Cursor, len(page.Items), page.HasMore)
	}
	next, err := client.Discover(context.Background(), sdk.DiscoverOptions{Cursor: page.Cursor})
	if err != nil {
		t.Fatal(err)
	}
	if next.HasMore || next.Cursor != "" || len(next.Items) != 1 || next.Items[0].Title != "Fixture North Gate" {
		t.Fatalf("second page = %+v cursor=%q", next.Items, next.Cursor)
	}
	if len(offsets) != 2 || offsets[0] != "0" || offsets[1] != "6" {
		t.Fatalf("offsets = %v", offsets)
	}
	for _, line := range logs.lines {
		if strings.Contains(line, session) || strings.Contains(line, page.Cursor) {
			t.Fatalf("log leaked the feed cursor: %s", line)
		}
	}
}

func TestDiscoverChannelUsesMotionTab(t *testing.T) {
	body := []byte(`{"code":0,"data":{"tab_item":[{"tab_type":36,"title":"漫剧","cell_data":[{"video_data":{"series_id":"7100000000000104","video_detail":{"series_title":"Fixture Motion","episode_cnt":4}}}]}]}}`)
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathDiscover:
			q, _ := url.ParseQuery(r.URL.RawQuery)
			if q.Get("tab_type") != "36" {
				t.Fatalf("tab_type = %s", q.Get("tab_type"))
			}
			_, _ = w.Write(body)
		default:
			http.NotFound(w, r)
		}
	}))
	page, err := client.Discover(context.Background(), sdk.DiscoverOptions{Channel: "漫剧"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Title != "Fixture Motion" || page.Items[0].EpisodeCount != 4 {
		t.Fatalf("page = %+v", page.Items)
	}
}

func TestDiscoverBookstoreChannelSendsBottomTab(t *testing.T) {
	open := []byte(`{"code":0,"data":{"tab_item":[{"tab_type":8,"session_id":"feed-session","cell_data":[{"cell_id_str":"7100000000000301","show_type":506}]}]}}`)
	change := []byte(`{"code":0,"data":{"has_more":false,"cell_view":{"cell_data":[{"video_data":{"series_id":"7100000000000107","title":"Fixture Find"}}]}}}`)
	var bottoms, tabs []string
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathDiscover:
			q, _ := url.ParseQuery(r.URL.RawQuery)
			bottoms = append(bottoms, q.Get("bottom_tab_type"))
			tabs = append(tabs, q.Get("tab_type"))
			_, _ = w.Write(open)
		case protocol.PathCellChange:
			_, _ = w.Write(change)
		default:
			http.NotFound(w, r)
		}
	}))
	page, err := client.Discover(context.Background(), sdk.DiscoverOptions{Channel: "找剧"})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Title != "Fixture Find" {
		t.Fatalf("page = %+v", page.Items)
	}
	if len(bottoms) != 1 || bottoms[0] != "0" || tabs[0] != "8" {
		t.Fatalf("bottom=%v tab=%v", bottoms, tabs)
	}
}

func TestSearchNovelTabReadsBook(t *testing.T) {
	body := []byte(`{"code":0,"search_tabs":[{"tab_type":1,"has_more":false,"data":[{"book_data":[{"book_id":"7100000000000402","book_name":"Fixture Novel","serial_count":3}]}]}]}`)
	var tab string
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathSearch:
			q, _ := url.ParseQuery(r.URL.RawQuery)
			tab = q.Get("tab_type")
			_, _ = w.Write(body)
		default:
			http.NotFound(w, r)
		}
	}))
	page, err := client.Search(context.Background(), "短剧", sdk.SearchOptions{Tab: "小说"})
	if err != nil {
		t.Fatal(err)
	}
	if tab != "1" || len(page.Items) != 1 || page.Items[0].Title != "Fixture Novel" || page.Items[0].EpisodeCount != 3 {
		t.Fatalf("tab=%s page=%+v", tab, page.Items)
	}
}

func TestSearchSendsNextOffset(t *testing.T) {
	first := []byte(`{"code":0,"search_tabs":[{"tab_type":11,"has_more":true,"next_offset":6,"data":[{"video_data":{"series_id":"7100000000000105","video_detail":{"series_title":"Fixture Lantern Court","episode_cnt":6}}}]}]}`)
	second := []byte(`{"code":0,"search_tabs":[{"tab_type":11,"has_more":false,"next_offset":0,"data":[{"video_data":{"series_id":"7100000000000106","video_detail":{"series_title":"Fixture North Gate"}}}]}]}`)
	var offsets []string
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathSearch:
			q, _ := url.ParseQuery(r.URL.RawQuery)
			offsets = append(offsets, q.Get("offset"))
			if q.Get("offset") == "6" {
				_, _ = w.Write(second)
				return
			}
			_, _ = w.Write(first)
		default:
			http.NotFound(w, r)
		}
	}))
	page, err := client.Search(context.Background(), "短剧", sdk.SearchOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if page.Cursor != "6" || len(page.Items) != 1 || page.Items[0].EpisodeCount != 6 {
		t.Fatalf("page = %+v cursor=%q", page.Items, page.Cursor)
	}
	next, err := client.Search(context.Background(), "短剧", sdk.SearchOptions{Cursor: page.Cursor})
	if err != nil {
		t.Fatal(err)
	}
	if next.Cursor != "" || next.Items[0].Title != "Fixture North Gate" {
		t.Fatalf("next = %+v cursor=%q", next.Items, next.Cursor)
	}
	if len(offsets) != 2 || offsets[0] != "0" || offsets[1] != "6" {
		t.Fatalf("offsets = %v", offsets)
	}
}

func assertAnonymousWire(t *testing.T, seen []captured) {
	t.Helper()
	var registered, discovered, searched, detailed, played bool
	for _, hit := range seen {
		if hit.Header.Get("Cookie") != "" || hit.Header.Get("Authorization") != "" || hit.Header.Get("X-Tt-Token") != "" {
			t.Fatalf("credential header on %s", hit.Path)
		}
		switch hit.Path {
		case protocol.PathRegister:
			registered = true
			if hit.Method != http.MethodPost {
				t.Fatalf("register method %s", hit.Method)
			}
			q, _ := url.ParseQuery(hit.RawQuery)
			if q.Get("tt_data") != "a" {
				t.Fatalf("register query = %s", hit.RawQuery)
			}
			if !strings.HasPrefix(hit.Body, "tc") && (len(hit.Body) < 2 || hit.Body[0] != 't' || hit.Body[1] != 'c') {
				if len(hit.Body) < 4 || hit.Body[0] != 0x74 || hit.Body[1] != 0x63 {
					t.Fatal("register body is not a tt-data envelope")
				}
			}
			if hit.Header.Get("X-SS-STUB") == "" {
				t.Fatal("portable content stub missing")
			}
		case protocol.PathDiscover:
			discovered = true
			if !strings.HasPrefix(hit.RawQuery, "iid=") || strings.Contains(hit.RawQuery, " ") {
				t.Fatal("discover query was re-encoded")
			}
			q, _ := url.ParseQuery(hit.RawQuery)
			if q.Get("device_id") != "7100000000000001" || q.Get("iid") != "7100000000000002" || q.Get("aid") != "8662" {
				t.Fatal("discover query missing device fields")
			}
			if q.Get("bottom_tab_type") != "7" || q.Get("tab_type") != "-1" || q.Get("client_req_type") != "3" {
				t.Fatal("discover tab query missing business fields")
			}
			if hit.Header.Get("x-gorgon") == "" || hit.Header.Get("X-Xs-From-Web") != "0" {
				t.Fatal("discover signature missing")
			}
		case protocol.PathSearch:
			searched = true
			if !strings.HasPrefix(hit.RawQuery, "iid=") || !strings.Contains(hit.RawQuery, "resolution=1200*2670") || strings.Contains(hit.RawQuery, " ") {
				t.Fatalf("search query was re-encoded")
			}
			q, _ := url.ParseQuery(hit.RawQuery)
			if q.Get("query") != "短剧" || q.Get("tab_type") != "11" || q.Get("user_is_login") != "0" || q.Get("search_source") != "1" {
				t.Fatalf("search query missing business fields")
			}
			if hit.Header.Get("X-Xs-From-Web") != "0" || hit.Header.Get("x-reading-request") == "" || hit.Header.Get("x-gorgon") == "" {
				t.Fatal("anonymous search headers missing")
			}
		case protocol.PathSeries:
			detailed = true
			if !strings.HasPrefix(hit.RawQuery, "iid=") || strings.Contains(hit.RawQuery, " ") {
				t.Fatal("series query was re-encoded")
			}
			if !strings.Contains(hit.Body, `"series_id":"7100000000000101"`) {
				t.Fatalf("series body = %s", hit.Body)
			}
			if hit.Header.Get("x-gorgon") == "" || hit.Header.Get("x-ss-stub") == "" {
				t.Fatal("series signature missing")
			}
		case protocol.PathPlayback:
			played = true
			if !strings.HasPrefix(hit.RawQuery, "iid=") || strings.Contains(hit.RawQuery, " ") {
				t.Fatal("playback query was re-encoded")
			}
			if !strings.Contains(hit.Body, `"video_id":"7100000000000201"`) || !strings.Contains(hit.Body, `"need_all_video_definition":true`) {
				t.Fatalf("playback body = %s", hit.Body)
			}
			if hit.Header.Get("x-gorgon") == "" || hit.Header.Get("x-ss-stub") == "" {
				t.Fatal("playback signature missing")
			}
		default:
			t.Fatalf("unexpected path %s", hit.Path)
		}
	}
	if !registered || !discovered || !searched || !detailed || !played {
		t.Fatalf("missing calls reg=%v discover=%v search=%v series=%v playback=%v", registered, discovered, searched, detailed, played)
	}
}

func TestAccountRequiredIsDistinctFromDRM(t *testing.T) {
	srvBody := fixture(t, "account-required.json")
	reg := fixture(t, "register-response.json")
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == protocol.PathRegister {
			_, _ = w.Write(reg)
			return
		}
		_, _ = w.Write(srvBody)
	}))
	_, err := client.Discover(context.Background(), sdk.DiscoverOptions{})
	if !errors.Is(err, sdk.ErrAuthenticationRequired) {
		t.Fatalf("err = %v", err)
	}
	if errors.Is(err, sdk.ErrDRMProtected) {
		t.Fatal("account failure reported as DRM")
	}

	client401, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == protocol.PathRegister {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(reg)
			return
		}
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write(srvBody)
	}))
	_, err = client401.Search(context.Background(), "短剧", sdk.SearchOptions{})
	if !errors.Is(err, sdk.ErrAuthenticationRequired) || errors.Is(err, sdk.ErrDRMProtected) {
		t.Fatalf("http 401 err = %v", err)
	}
}

func TestSearchSendsGuestCookieFromRegistration(t *testing.T) {
	const ticket = "fixture-ticket-value"
	reg := fixture(t, "register-response.json")
	body := fixture(t, "search-response.json")
	var searches int
	var registers int
	client, logs := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == protocol.PathRegister {
			registers++
			w.Header().Add("Set-Cookie", "store-region=cn; Path=/")
			w.Header().Add("Set-Cookie", "store-region-src=did; Path=/")
			w.Header().Add("Set-Cookie", "install_id=7100000000000002; Path=/")
			w.Header().Add("Set-Cookie", "ttreq="+ticket+"; Path=/")
			_, _ = w.Write(reg)
			return
		}
		if r.URL.Path != protocol.PathSearch {
			http.NotFound(w, r)
			return
		}
		searches++
		cookie := r.Header.Get("Cookie")
		if !strings.Contains(cookie, "passport_csrf_token=;") || !strings.Contains(cookie, "install_id=7100000000000002") || !strings.Contains(cookie, "ttreq="+ticket) || !strings.Contains(cookie, "store-region=cn") || !strings.Contains(cookie, "store-region-src=did") {
			t.Errorf("guest cookie missing")
		}
		if strings.Contains(strings.ToLower(cookie), "session") || r.Header.Get("Authorization") != "" {
			t.Errorf("account credential on search")
		}
		if !strings.HasPrefix(r.URL.RawQuery, "iid=7100000000000002&device_id=7100000000000001&") {
			t.Errorf("signed query order changed")
		}
		_, _ = w.Write(body)
	}))
	if _, err := client.Search(context.Background(), "短剧", sdk.SearchOptions{PageSize: 20}); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Search(context.Background(), "短剧", sdk.SearchOptions{PageSize: 20}); err != nil {
		t.Fatal(err)
	}
	if registers != 1 || searches != 2 {
		t.Fatalf("registers=%d searches=%d", registers, searches)
	}
	for _, line := range logs.lines {
		if strings.Contains(line, ticket) || strings.Contains(line, "ttreq") || strings.Contains(line, "?") || strings.Contains(line, "7100000000000001") {
			t.Fatalf("log leaked request material")
		}
	}
}

func TestEmptyAnonymousBodyIsTypedLimitation(t *testing.T) {
	reg := fixture(t, "register-response.json")
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == protocol.PathRegister {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write(reg)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	_, err := client.Search(context.Background(), "短剧", sdk.SearchOptions{})
	if !errors.Is(err, sdk.ErrAnonymousLimited) {
		t.Fatalf("err = %v", err)
	}
}

func TestClearPlaybackReturnsPlayableStream(t *testing.T) {
	reg := fixture(t, "register-response.json")
	clear := fixture(t, "clear-playback.json")
	signed := false
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(reg)
		case protocol.PathPlayback:
			signed = r.Header.Get("x-gorgon") != "" && r.Header.Get("x-ss-stub") != ""
			_, _ = w.Write(clear)
		default:
			http.NotFound(w, r)
		}
	}))
	playback, err := client.ResolvePlayback(context.Background(), "7100000000000201", sdk.PlaybackOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if playback.Protected || len(playback.Streams) != 2 {
		t.Fatalf("playback = %+v", playback)
	}
	best, err := playback.Select("")
	if err != nil || best.Definition != "1080p" || best.URL != "https://media.invalid/clear-1080.mp4" {
		t.Fatalf("best = %+v err = %v", best, err)
	}
	if !signed {
		t.Fatal("playback request was not signed")
	}
}

func TestPagerDropsRowsRepeatedAcrossPages(t *testing.T) {
	pages := []*sdk.Page[model.SeriesSummary]{
		{Items: []model.SeriesSummary{{ID: "1"}, {ID: "2"}}, Cursor: "c1", HasMore: true},
		{Items: []model.SeriesSummary{{ID: "2"}, {ID: "3"}}, Cursor: "c2", HasMore: true},
		{Items: []model.SeriesSummary{{ID: "3"}}, Cursor: "", HasMore: false},
	}
	var cursors []string
	pager := sdk.NewPager("", func(_ context.Context, cursor string) (*sdk.Page[model.SeriesSummary], error) {
		cursors = append(cursors, cursor)
		if len(pages) == 0 {
			return &sdk.Page[model.SeriesSummary]{}, nil
		}
		page := pages[0]
		pages = pages[1:]
		return page, nil
	})
	var ids []string
	for {
		items, more, err := pager.Next(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range items {
			ids = append(ids, item.ID)
		}
		if !more {
			break
		}
	}
	if strings.Join(ids, ",") != "1,2,3" {
		t.Fatalf("ids = %v", ids)
	}
	if strings.Join(cursors, ",") != ",c1,c2" {
		t.Fatalf("cursors = %v", cursors)
	}
	if pager.Seen() != 3 {
		t.Fatalf("seen = %d", pager.Seen())
	}
}

func TestRetriesRateLimitedRequest(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "device_register") {
			_, _ = w.Write(fixture(t, "register-response.json"))
			return
		}
		calls++
		if calls == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_, _ = w.Write(fixture(t, "discover-response.json"))
	}))
	t.Cleanup(srv.Close)
	transport := httptransport.New()
	client, err := sdk.NewClient(sdk.Options{
		Transport: transport, ProfileStore: memstore.New(),
		APIOrigin: srv.URL, RegisterOrigin: srv.URL, RetryAttempts: 3,
		Clock: func() time.Time { return time.UnixMilli(1_700_000_000_123).UTC() },
	})
	if err != nil {
		t.Fatal(err)
	}
	page, err := client.Discover(context.Background(), sdk.DiscoverOptions{})
	if err != nil {
		t.Fatalf("discover after retry: %v", err)
	}
	if calls < 2 || len(page.Items) == 0 {
		t.Fatalf("calls = %d items = %d", calls, len(page.Items))
	}
}

func TestDoesNotRetryBadRequest(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusBadRequest)
	}))
	t.Cleanup(srv.Close)
	transport := httptransport.New()
	client, err := sdk.NewClient(sdk.Options{
		Transport: transport, ProfileStore: memstore.New(),
		APIOrigin: srv.URL, RegisterOrigin: srv.URL, RetryAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Discover(context.Background(), sdk.DiscoverOptions{}); err == nil {
		t.Fatal("bad request succeeded")
	}
	if calls != 1 {
		t.Fatalf("calls = %d, want 1", calls)
	}
}

func TestCategoryLandingSendsFiltersAndCursor(t *testing.T) {
	var queries []url.Values
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathCategoryLanding:
			q, _ := url.ParseQuery(r.URL.RawQuery)
			queries = append(queries, q)
			_, _ = w.Write(fixture(t, "category-landing.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	ctx := context.Background()
	_, err := client.CategoryLanding(ctx, "68", sdk.LandingOptions{
		Tab:     "1",
		Filters: map[string]string{"book_status": "0", "sort_by": "76", "sub_category": "515", "word_type": "1"},
		Cursor:  "20",
	})
	if err != nil {
		t.Fatal(err)
	}
	q := queries[0]
	// Rule keys are sent under GetNewCategoryLandingPageRequest's wire names,
	// and the tab as query_gender.
	for key, want := range map[string]string{
		"category_id": "68", "query_gender": "1", "book_status": "0", "sort_by": "76",
		"sub_category_id": "515", "word_number": "1", "offset": "20",
	} {
		if q.Get(key) != want {
			t.Fatalf("%s = %q, want %q (query %v)", key, q.Get(key), want, q)
		}
	}
	for _, unsent := range []string{"tab_type", "sub_category", "word_type"} {
		if q.Has(unsent) {
			t.Fatalf("%s sent: %v", unsent, q)
		}
	}
	// 全部 sends no gender.
	if _, err := client.CategoryLanding(ctx, "68", sdk.LandingOptions{Tab: "8"}); err != nil {
		t.Fatal(err)
	}
	if queries[1].Has("query_gender") {
		t.Fatalf("全部 sent a gender: %v", queries[1])
	}
	for _, bad := range []sdk.LandingOptions{{Filters: map[string]string{"colour": "red"}}, {Tab: "5"}} {
		if _, err := client.CategoryLanding(ctx, "68", bad); err == nil {
			t.Fatalf("%+v accepted", bad)
		}
	}
	if len(queries) != 2 {
		t.Fatalf("refused options still sent %d requests", len(queries)-2)
	}
}

func TestLongBanWindowFailsFastInsteadOfSleeping(t *testing.T) {
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "device_register") {
			_, _ = w.Write(fixture(t, "register-response.json"))
			return
		}
		calls++
		// 100 FAST_REJECT carries the ban window in its message.
		_, _ = w.Write([]byte(`{"code":100,"message":"{\"minBanTime\":600,\"maxBanTime\":900,\"banRate\":1}"}`))
	}))
	t.Cleanup(srv.Close)
	transport := httptransport.New()
	client, err := sdk.NewClient(sdk.Options{
		Transport: transport, ProfileStore: memstore.New(),
		APIOrigin: srv.URL, RegisterOrigin: srv.URL, RetryAttempts: 3,
	})
	if err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	_, err = client.Discover(context.Background(), sdk.DiscoverOptions{})
	if !errors.Is(err, sdk.ErrRateLimited) || !strings.Contains(err.Error(), "wait about 12m30s") {
		t.Fatalf("err = %v", err)
	}
	if calls != 1 || time.Since(start) > 5*time.Second {
		t.Fatalf("calls = %d after %s", calls, time.Since(start))
	}
}

func TestRankRowsEmptyFirstPageIsUnavailableButLaterPageIsEnd(t *testing.T) {
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathRankCells:
			_, _ = w.Write([]byte(`{"code":0,"data":{"has_more":false,"cell_view":{"book_data":[]}}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	ctx := context.Background()
	if _, err := client.RankRows(ctx, "", 0); !errors.Is(err, sdk.ErrContentUnavailable) {
		t.Fatalf("first page err = %v", err)
	}
	page, err := client.RankRows(ctx, "", 40)
	if err != nil || len(page.Items) != 0 || page.HasMore {
		t.Fatalf("later page = %+v err %v", page, err)
	}
}

func TestShortFilterUsesDistributionOriginAndPages(t *testing.T) {
	var bodies []map[string]any
	dist := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != protocol.PathShortFilter {
			http.NotFound(w, r)
			return
		}
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		_ = json.Unmarshal(raw, &body)
		bodies = append(bodies, body)
		name := "short-filter-content.json"
		if body["req_type"] == "only_panel" {
			name = "short-filter-panel.json"
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, name))
	}))
	t.Cleanup(dist.Close)
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == protocol.PathRegister {
			_, _ = w.Write(fixture(t, "register-response.json"))
			return
		}
		// The reading host does not serve /reading/distribution/.
		http.NotFound(w, r)
	}))
	t.Cleanup(api.Close)
	transport := httptransport.New()
	client, err := sdk.NewClient(sdk.Options{
		Transport: transport, ProfileStore: memstore.New(),
		APIOrigin: api.URL, RegisterOrigin: api.URL, DistributionOrigin: dist.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	rows, err := client.ShortFilterPanel(ctx, "comic_series")
	if err != nil || len(rows) != 4 {
		t.Fatalf("rows = %+v err %v", rows, err)
	}
	if bodies[0]["req_scene"] != "comic_series" || bodies[0]["req_type"] != "only_panel" {
		t.Fatalf("panel body = %v", bodies[0])
	}
	page, err := client.ShortFilter(ctx, sdk.ShortFilterQuery{Selections: map[string][]string{
		"genre": {"short_play"}, "category_dim_theme": {"cate_165"},
	}})
	if err != nil {
		t.Fatal(err)
	}
	// 真人剧 is the default scene with the genre selected.
	body := bodies[1]
	sel, _ := body["select_items"].(map[string]any)
	if body["req_scene"] != "default" || body["req_type"] != "only_content" || body["offset"] != float64(0) || body["filter_ids"] != "" {
		t.Fatalf("content body = %v", body)
	}
	if fmt.Sprint(sel["genre"]) != "[short_play]" || fmt.Sprint(sel["category_dim_theme"]) != "[cate_165]" || fmt.Sprint(sel["sort"]) != "[]" {
		t.Fatalf("select_items = %v", sel)
	}
	if len(page.Items) != 2 || !page.HasMore || page.Cursor == "" {
		t.Fatalf("page = %+v", page)
	}
	// The next page sends the offset, the series shown, and the session.
	if _, err := client.ShortFilter(ctx, sdk.ShortFilterQuery{Cursor: page.Cursor}); err != nil {
		t.Fatal(err)
	}
	next := bodies[2]
	if next["offset"] != float64(18) || next["filter_ids"] != "7100000000000101,7100000000000102" || next["session_id"] != "fixture-session" {
		t.Fatalf("next body = %v", next)
	}
	// The cursor alone resumes the same filter.
	nextSel, _ := next["select_items"].(map[string]any)
	if next["req_scene"] != "default" || fmt.Sprint(nextSel["genre"]) != "[short_play]" || fmt.Sprint(nextSel["category_dim_theme"]) != "[cate_165]" {
		t.Fatalf("next selections = %v", next)
	}
	if _, err := client.ShortFilter(ctx, sdk.ShortFilterQuery{
		Selections: map[string][]string{"genre": {"comic_series"}}, Cursor: page.Cursor,
	}); err == nil || !strings.Contains(err.Error(), "different filter") {
		t.Fatalf("mismatched cursor err = %v", err)
	}
	for _, bad := range []sdk.ShortFilterQuery{
		{Selections: map[string][]string{"colour": {"red"}}},
		{Selections: map[string][]string{"genre": {"opera"}}},
		{Cursor: "junk"},
	} {
		if _, err := client.ShortFilter(ctx, bad); err == nil {
			t.Fatalf("%+v accepted", bad)
		}
	}
	if len(bodies) != 3 {
		t.Fatalf("refused queries still sent %d requests", len(bodies)-3)
	}
}

func TestShortRankSelectsBoardAndPages(t *testing.T) {
	var queries []url.Values
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathRankCells:
			q, _ := url.ParseQuery(r.URL.RawQuery)
			queries = append(queries, q)
			_, _ = w.Write(fixture(t, "short-rank.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	ctx := context.Background()
	page, err := client.ShortRank(ctx, sdk.ShortRankQuery{Content: "human", Board: "human_hot_play"})
	if err != nil {
		t.Fatal(err)
	}
	q := queries[0]
	for key, want := range map[string]string{
		"cell_id": protocol.ShortRankCell, "tab_type": "26", "selected_items": "human",
		"sub_selected_items": "human_hot_play", "offset": "0", "unlimited_selector_change_type": "1",
	} {
		if q.Get(key) != want {
			t.Fatalf("%s = %q, want %q", key, q.Get(key), want)
		}
	}
	if page.Content != "human" || page.Board != "human_hot_play" || len(page.Contents) != 2 || len(page.Items) != 3 || page.Cursor == "" {
		t.Fatalf("page = %+v", page)
	}
	// The cursor alone resumes the same list.
	next, err := client.ShortRank(ctx, sdk.ShortRankQuery{Cursor: page.Cursor})
	if err != nil {
		t.Fatal(err)
	}
	q = queries[1]
	if q.Get("offset") != "10" || q.Get("unlimited_selector_change_type") != "2" || q.Get("session_id") != "fixture-session" ||
		q.Get("filter_ids") != "7100000000000101,7100000000000103,7100000000000102" {
		t.Fatalf("next query = %v", q)
	}
	if q.Get("selected_items") != "human" || q.Get("sub_selected_items") != "human_hot_play" {
		t.Fatalf("resumed list = %v", q)
	}
	// Positions continue across pages.
	if next.Items[0].Position != 4 {
		t.Fatalf("next positions = %+v", next.Items)
	}
	if _, err := client.ShortRank(ctx, sdk.ShortRankQuery{Board: "human_new_rank", Cursor: page.Cursor}); err == nil {
		t.Fatal("cursor of another board accepted")
	}
	// An empty content is all, and no board leaves the choice to the service.
	if _, err := client.ShortRank(ctx, sdk.ShortRankQuery{}); err != nil {
		t.Fatal(err)
	}
	if q := queries[2]; q.Get("selected_items") != "all" || q.Has("sub_selected_items") {
		t.Fatalf("default query = %v", q)
	}
}

func TestSeasonsReadsSeriesRowOrFallsBackToOneSeason(t *testing.T) {
	var plans []url.Values
	searched := false
	single := false
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathSeries:
			_, _ = w.Write(fixture(t, "series-response.json"))
		case protocol.PathPlan:
			q, _ := url.ParseQuery(r.URL.RawQuery)
			plans = append(plans, q)
			if single {
				_, _ = w.Write([]byte(`{"code":101116,"message":"SERVICE_ERROR"}`))
				return
			}
			_, _ = w.Write(fixture(t, "series-seasons.json"))
		case protocol.PathSearch:
			searched = true
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	ctx := context.Background()
	album, err := client.Seasons(ctx, "7100000000000101")
	if err != nil {
		t.Fatal(err)
	}
	if q := plans[0]; q.Get("from") != protocol.PlanFromSeriesSeasons || q.Get("book_id") != "7100000000000101" || q.Get("filter_ids") != "7100000000000101" {
		t.Fatalf("plan query = %v", q)
	}
	if album.SeasonCount != 3 || album.Title != "Fixture Lantern Court" {
		t.Fatalf("album = %+v", album)
	}
	for i, s := range album.Seasons {
		if s.Index != i+1 {
			t.Fatalf("seasons out of order: %+v", album.Seasons)
		}
	}
	// The announced season has no episodes yet.
	if album.Seasons[0].SeriesID != "7100000000000101" || album.Seasons[2].EpisodeCount != 0 {
		t.Fatalf("seasons = %+v", album.Seasons)
	}

	// A title without other seasons is refused there and stays on its own,
	// without searching for look-alikes.
	single = true
	album, err = client.Seasons(ctx, "7100000000000101")
	if err != nil {
		t.Fatal(err)
	}
	if album.SeasonCount != 1 || album.Seasons[0].SeriesID != "7100000000000101" || searched {
		t.Fatalf("single album = %+v searched %v", album, searched)
	}
}

func TestSearchSendsFilters(t *testing.T) {
	var selected []string
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathSearch:
			q, _ := url.ParseQuery(r.URL.RawQuery)
			selected = append(selected, q.Get("selected_items"))
			_, _ = w.Write(fixture(t, "search-response.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	ctx := context.Background()
	if _, err := client.Search(ctx, "奇幻", sdk.SearchOptions{Filters: []string{
		sdk.SearchFilterAIComic, sdk.SearchFilterUnwatched, sdk.SearchFilterAIComic,
	}}); err != nil {
		t.Fatal(err)
	}
	if selected[0] != "content_type_ai_dynamic_comic,search_range_unread" {
		t.Fatalf("selected_items = %q", selected[0])
	}
	for _, opts := range []sdk.SearchOptions{
		{Filters: []string{"content_type_novel"}},
		{Tab: "用户", Filters: []string{sdk.SearchFilterFilm}},
	} {
		if _, err := client.Search(ctx, "奇幻", opts); err == nil {
			t.Fatalf("%+v accepted", opts)
		}
	}
	if len(selected) != 1 {
		t.Fatalf("refused searches were sent: %q", selected)
	}
}

func TestHotSearchRequestsSearchPage(t *testing.T) {
	var query url.Values
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathPlan:
			query, _ = url.ParseQuery(r.URL.RawQuery)
			_, _ = w.Write(fixture(t, "hot-search.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	hot, err := client.HotSearch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if query.Get("from") != "search_input_page" || query.Get("search_middle_page_version") != "4" || query.Get("new_search_middle_page") != "true" {
		t.Fatalf("query = %v", query)
	}
	if len(hot.Boards) != 3 || len(hot.Guesses) != 2 {
		t.Fatalf("hot = %+v", hot)
	}
}

func TestShortRankReadsActorsAndPagesWithoutFilterIDs(t *testing.T) {
	var queries []url.Values
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathRankCells:
			q, _ := url.ParseQuery(r.URL.RawQuery)
			queries = append(queries, q)
			_, _ = w.Write(fixture(t, "short-rank-actors.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	ctx := context.Background()
	page, err := client.ShortRank(ctx, sdk.ShortRankQuery{Content: protocol.ShortRankActors})
	if err != nil {
		t.Fatal(err)
	}
	if page.Content != protocol.ShortRankActors || page.Board != "ranklist_series_actor" || len(page.Items) != 3 || page.Cursor == "" {
		t.Fatalf("page = %+v", page)
	}
	if a := page.Items[0]; a.ActorID != "7100000000001401" || a.Actor != "Fixture Actor A" || a.SeriesID != "" {
		t.Fatalf("actor = %+v", a)
	}
	next, err := client.ShortRank(ctx, sdk.ShortRankQuery{Cursor: page.Cursor})
	if err != nil {
		t.Fatal(err)
	}
	// The captured client pages actors by offset and session alone.
	if q := queries[1]; q.Has("filter_ids") || q.Get("offset") != "3" || q.Get("session_id") != "fixture-session" || q.Get("selected_items") != protocol.ShortRankActors {
		t.Fatalf("next query = %v", q)
	}
	if next.Items[0].Position != 4 {
		t.Fatalf("next positions = %+v", next.Items)
	}
}

func TestNewReleasesSendsDateAndResumesFromCursor(t *testing.T) {
	var queries []url.Values
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathSubscribeList:
			q, _ := url.ParseQuery(r.URL.RawQuery)
			queries = append(queries, q)
			_, _ = w.Write(fixture(t, "new-releases.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	ctx := context.Background()
	page, err := client.NewReleases(ctx, sdk.NewReleaseQuery{Date: "20261024"})
	if err != nil {
		t.Fatal(err)
	}
	if q := queries[0]; q.Get("tab_type") != "5" || q.Get("target_date") != "20261024" || q.Get("offset") != "0" {
		t.Fatalf("query = %v", q)
	}
	if page.Date != "20261023" || len(page.Items) != 1 || page.Cursor == "" {
		t.Fatalf("page = %+v", page)
	}
	if _, err := client.NewReleases(ctx, sdk.NewReleaseQuery{Cursor: page.Cursor}); err != nil {
		t.Fatal(err)
	}
	// The cursor keeps the day the service answered.
	if q := queries[1]; q.Get("target_date") != "20261023" || q.Get("offset") != "2" || q.Get("session_id") != "fixture-session" {
		t.Fatalf("next query = %v", q)
	}
	// Today is the service's default day.
	if _, err := client.NewReleases(ctx, sdk.NewReleaseQuery{}); err != nil {
		t.Fatal(err)
	}
	if queries[2].Has("target_date") {
		t.Fatalf("default query = %v", queries[2])
	}
	for _, q := range []sdk.NewReleaseQuery{{Date: "2026-10-24"}, {Date: "20261025", Cursor: page.Cursor}} {
		if _, err := client.NewReleases(ctx, q); err == nil {
			t.Fatalf("%+v accepted", q)
		}
	}
	if len(queries) != 3 {
		t.Fatalf("refused queries were sent: %d", len(queries))
	}
}

func TestPersonAndWorks(t *testing.T) {
	var bodies []map[string]any
	var paths []string
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.URL.Path == protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case r.URL.Path == protocol.PathProfile:
			if r.URL.Query().Get("user_id") != "7100000000001701" {
				t.Errorf("profile query = %v", r.URL.Query())
			}
			_, _ = w.Write(fixture(t, "person.json"))
		case strings.HasPrefix(r.URL.Path, "/novel/commentapi/user/profile/"):
			paths = append(paths, r.URL.Path)
			body := map[string]any{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Error(err)
			}
			bodies = append(bodies, body)
			_, _ = w.Write(fixture(t, "person-works.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	ctx := context.Background()
	person, err := client.Person(ctx, "7100000000001701")
	if err != nil || person.Name != "Fixture Actor" {
		t.Fatalf("person = %+v err %v", person, err)
	}
	works, err := client.PersonWorks(ctx, "7100000000001701", "")
	if err != nil {
		t.Fatal(err)
	}
	if paths[0] != "/novel/commentapi/user/profile/7100000000001701/v1" {
		t.Fatalf("path = %s", paths[0])
	}
	if b := bodies[0]; b["tab_type"] != float64(23) || b["sub_tab_type"] != float64(22) || b["cursor"] != "" || b["target_user_id"] != "7100000000001701" {
		t.Fatalf("body = %v", b)
	}
	if len(works.Items) != 2 || works.Cursor == "" {
		t.Fatalf("works = %+v", works)
	}
	if _, err := client.PersonWorks(ctx, "7100000000001701", works.Cursor); err != nil {
		t.Fatal(err)
	}
	if c, _ := bodies[1]["cursor"].(string); !strings.Contains(c, `"next_cursor"`) {
		t.Fatalf("next cursor = %v", bodies[1]["cursor"])
	}
	if _, err := client.PersonWorks(ctx, "7100000000001702", works.Cursor); err == nil {
		t.Fatal("another person's cursor accepted")
	}
	if _, err := client.Person(ctx, "../x"); err == nil {
		t.Fatal("malformed user ID accepted")
	}
}

func TestGetSeriesBatchChunksAndKeepsOrder(t *testing.T) {
	var asked [][]string
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathSeriesBatch:
			var body struct {
				SeriesID string `json:"series_id"`
				Scene    string `json:"dr_scene"`
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.Scene != "player" {
				t.Errorf("body = %+v err %v", body, err)
			}
			asked = append(asked, strings.Split(body.SeriesID, ","))
			_, _ = w.Write(fixture(t, "series-batch.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	ids := []string{"7100000000000102", "7100000000000101", "7100000000000102"}
	for i := 0; i < 10; i++ {
		ids = append(ids, fmt.Sprintf("71000000000099%02d", i))
	}
	got, err := client.GetSeriesBatch(context.Background(), ids)
	if err != nil {
		t.Fatal(err)
	}
	// Twelve distinct IDs go in requests of ten and two.
	if len(asked) != 2 || len(asked[0]) != 10 || len(asked[1]) != 2 {
		t.Fatalf("asked = %v", asked)
	}
	// Request order, the repeat asked once, unknown IDs left out.
	if len(got) != 2 || got[0].Series.ID != "7100000000000102" || got[1].Series.ID != "7100000000000101" || got[1].Series.Novel == nil {
		t.Fatalf("got = %+v", got)
	}
	if _, err := client.GetSeriesBatch(context.Background(), []string{"x/1"}); err == nil {
		t.Fatal("malformed ID accepted")
	}
}

func TestRankingUsesDistributionOriginWithoutSigning(t *testing.T) {
	var query string
	client, _ := testClient(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRankActivity:
			query = r.URL.RawQuery
			_, _ = w.Write(fixture(t, "ranking-response.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	rank, err := client.Ranking(context.Background(), "", "")
	if err != nil {
		t.Fatal(err)
	}
	// The activity host answers 400 to the signed device query.
	if query != "" || len(rank.Items) == 0 {
		t.Fatalf("query %q rank %+v", query, rank)
	}
}

func TestDeviceIsRegisteredStoredAndPresented(t *testing.T) {
	type seen struct {
		path, ua string
		query    url.Values
	}
	var reqs []seen
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqs = append(reqs, seen{r.URL.Path, r.Header.Get("User-Agent"), r.URL.Query()})
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case protocol.PathRegister:
			_, _ = w.Write(fixture(t, "register-response.json"))
		case protocol.PathCue:
			_, _ = w.Write(fixture(t, "cue-response.json"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	transport := httptransport.New()
	newClient := func(store sdk.ProfileStore, d model.Device) *sdk.Client {
		t.Helper()
		c, err := sdk.NewClient(sdk.Options{Transport: transport, ProfileStore: store, APIOrigin: srv.URL, RegisterOrigin: srv.URL, Device: d})
		if err != nil {
			t.Fatal(err)
		}
		return c
	}
	pad := model.Device{Model: "21051182C", OSVersion: "11", OSAPI: 30, Build: "RKQ1.200826.002", Pad: true}
	store := memstore.New()
	if _, err := newClient(store, pad).SearchCues(context.Background()); err != nil {
		t.Fatal(err)
	}
	wantUA := "com.phoenix.read/73932 (Linux; U; Android 11; zh_CN; 21051182C; Build/RKQ1.200826.002; Cronet/TTNetVersion:8d40f833 2026-03-03 QuicVersion:21ac1950 2025-11-18)"
	if reqs[0].path != protocol.PathRegister || reqs[0].ua != wantUA {
		t.Fatalf("register = %+v", reqs[0])
	}
	cue := reqs[len(reqs)-1]
	if cue.ua != wantUA || cue.query.Get("device_type") != "21051182C" || cue.query.Get("os_api") != "30" ||
		cue.query.Get("dragon_device_type") != "pad" || cue.query.Get("device_brand") != "Xiaomi" || cue.query.Get("channel") != "xiaomi_8662_64" {
		t.Fatalf("cue = %+v", cue)
	}
	saved, _ := store.Load(context.Background())
	if saved.Device == nil || saved.Device.Model != "21051182C" || !saved.Device.Pad {
		t.Fatalf("saved device = %+v", saved.Device)
	}
	// A registered profile keeps its device when the client asks for another.
	// (The fixture registration sets no guest cookie, so each client
	// registers again; the profile's device is what it registers as.)
	reqs = nil
	if _, err := newClient(store, model.Device{Model: "OTHER"}).SearchCues(context.Background()); err != nil {
		t.Fatal(err)
	}
	if last := reqs[len(reqs)-1]; last.query.Get("device_type") != "21051182C" || reqs[0].ua != wantUA {
		t.Fatalf("kept device = %+v", reqs)
	}
	// A profile from before devices were stored pins the client's device.
	saved.Device = nil
	if err := store.Save(context.Background(), saved); err != nil {
		t.Fatal(err)
	}
	reqs = nil
	if _, err := newClient(store, model.Device{}).SearchCues(context.Background()); err != nil {
		t.Fatal(err)
	}
	if again, _ := store.Load(context.Background()); again.Device == nil || again.Device.Model != model.DefaultDevice.Model ||
		reqs[len(reqs)-1].query.Get("device_type") != model.DefaultDevice.Model {
		t.Fatalf("pinned device = %+v query %v", again.Device, reqs[len(reqs)-1].query)
	}
	if _, err := sdk.NewClient(sdk.Options{Transport: transport, ProfileStore: memstore.New(), Device: model.Device{Model: "x; evil"}}); !errors.Is(err, sdk.ErrInvalidOptions) {
		t.Fatalf("bad device err = %v", err)
	}
}

func TestContentKeyUnwrapsKeyMaterial(t *testing.T) {
	// Live vector: episode 7690261304497458201 of series 7690184933037329432.
	pb := &model.Playback{EpisodeID: "7690261304497458201", Protected: true, KeyMaterial: "nLwSxG6JI85zhT//c7AK4HauO+lzrg7TdZwO0XWrP+J0pDypqQ=="}
	key, err := sdk.ContentKey(pb)
	if err != nil {
		t.Fatal(err)
	}
	if key.Hex != "1be760855c48a8c86000eeb8c5bb1010" || key.Version != "1" {
		t.Fatalf("key = %+v", key)
	}
	if raw, _ := json.Marshal(key); string(raw) != `{"key":"1be760855c48a8c86000eeb8c5bb1010","version":"1"}` {
		t.Fatalf("json = %s", raw)
	}
	if _, err := sdk.ContentKey(&model.Playback{}); !errors.Is(err, sdk.ErrNotProtected) {
		t.Fatalf("clear model err = %v", err)
	}
	if _, err := sdk.ContentKey(&model.Playback{Protected: true, KeyMaterial: "fixture-spade"}); !errors.Is(err, sdk.ErrDRMProtected) {
		t.Fatalf("bad material err = %v", err)
	}
	if _, err := sdk.ContentKey(nil); err == nil {
		t.Fatal("nil model accepted")
	}
}
