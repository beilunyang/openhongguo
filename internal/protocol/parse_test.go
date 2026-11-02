package protocol

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/beilunyang/openhongguo/sdk/model"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "protocol", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestParseDiscoverSurfacesIDsAndTitles(t *testing.T) {
	page, err := ParseDiscover(fixture(t, "discover-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 {
		t.Fatalf("items = %d", len(page.Items))
	}
	if page.Items[0].ID != "7100000000000101" || page.Items[0].Title != "Fixture Lantern Court" {
		t.Fatalf("first = %+v", page.Items[0])
	}
	if page.Items[1].ID != "7100000000000102" || page.Items[1].Title != "Fixture River Market" {
		t.Fatalf("second = %+v", page.Items[1])
	}
}

func TestParseDiscoverKeepsDetailListTitle(t *testing.T) {
	raw := []byte(`{"code":0,"data":{"tab_item":[{"tab_type":16,"session_id":"feed-session","client_template":7,"cell_data":[{"cell_id_str":"7100000000000301","show_type":407,"cell_data":[{"video_detail":{"video_list":[{"series_id":"7100000000000101","title":"promo line"}],"video_detail_list":[{"series_id_str":"7100000000000101","series_title":"Fixture Lantern Court"}]}}]}]}]}}`)
	page, err := ParseDiscover(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "7100000000000101" || page.Items[0].Title != "Fixture Lantern Court" {
		t.Fatalf("page = %+v", page.Items)
	}
	next, ok := DiscoverContinuation(raw)
	if !ok || next.CellID != "7100000000000301" || next.TabType != "16" || next.ShowType != "407" || next.ClientTemplate != "7" || next.SessionID != "feed-session" {
		t.Fatalf("continuation = %+v ok=%v", next, ok)
	}
	if _, ok := DiscoverContinuation(fixture(t, "discover-response.json")); ok {
		t.Fatal("fixture tab has no unlimited handle")
	}
}

func TestParseCellChangeReadsUnlimitedPage(t *testing.T) {
	raw := []byte(`{"code":0,"data":{"has_more":true,"next_offset":6,"cell_view":{"cell_data":[{"video_data":{"series_id":"7100000000000102","video_detail":{"series_title":"Fixture River Market"}}},{"video_detail":{"video_detail_list":[{"series_id_str":"7100000000000103","series_title":"Fixture North Gate"}]}}]}}}`)
	page, err := ParseCellChange(raw)
	if err != nil {
		t.Fatal(err)
	}
	if !page.HasMore || len(page.Items) != 2 || page.Items[0].Title != "Fixture River Market" || page.Items[1].ID != "7100000000000103" {
		t.Fatalf("page = %+v has_more=%v", page.Items, page.HasMore)
	}
}

func TestFeedCursorRoundTrip(t *testing.T) {
	token, err := EncodeFeedCursor(FeedContinuation{
		CellID: "7100000000000301", SessionID: "feed-session", ClientTemplate: "7",
		ShowType: "407", TabType: "16", Offset: "6", Limit: 30,
	})
	if err != nil {
		t.Fatal(err)
	}
	next, err := DecodeFeedCursor(token)
	if err != nil {
		t.Fatal(err)
	}
	if next.Offset != "6" || next.CellID != "7100000000000301" || next.SessionID != "feed-session" || next.TabType != "16" || next.Limit != 30 {
		t.Fatalf("next = %+v", next)
	}
	if _, err := DecodeFeedCursor("20"); err == nil {
		t.Fatal("plain offset was accepted as a discover cursor")
	}
}

func TestResolveChannelAndSearchTab(t *testing.T) {
	ch, err := ResolveDiscoverChannel("漫剧")
	if err != nil || ch.RequestTab != "36" || ch.ResponseTab != 36 || ch.BottomTab != "7" {
		t.Fatalf("channel = %+v %v", ch, err)
	}
	home, err := ResolveDiscoverChannel("找剧")
	if err != nil || home.RequestTab != "8" || home.ResponseTab != 8 || home.BottomTab != "0" {
		t.Fatalf("找剧 = %+v %v", home, err)
	}
	community, err := ResolveDiscoverChannel("社区")
	if err != nil || community.RequestTab != "33" || community.BottomTab != "10" {
		t.Fatalf("社区 = %+v %v", community, err)
	}
	if homeComic, err := ResolveDiscoverChannel("首页漫剧"); err != nil || homeComic.RequestTab != "32" || homeComic.BottomTab != "0" {
		t.Fatalf("首页漫剧 = %+v %v", homeComic, err)
	}
	tab, err := ResolveSearchTab("短剧")
	if err != nil || tab != SearchTabShortPlay {
		t.Fatalf("tab = %s %v", tab, err)
	}
	if got, err := ResolveSearchTab("综合"); err != nil || got != SearchTabShortPlay {
		t.Fatalf("综合 = %s %v", got, err)
	}
	if got, err := ResolveSearchTab("听书"); err != nil || got != SearchTabVoice {
		t.Fatalf("听书 = %s %v", got, err)
	}
	if _, err := ResolveSearchTab("漫画"); err == nil {
		t.Fatal("comic search tab was accepted")
	}
	if _, err := ResolveDiscoverChannel("书架"); err == nil {
		t.Fatal("bookshelf channel was accepted")
	}
}

func TestParseDiscoverReadsBooks(t *testing.T) {
	raw := []byte(`{"code":0,"data":{"tab_item":[{"tab_type":5,"cell_data":[{"book_data":[{"book_id":"7100000000000401","book_name":"Fixture Audio","abstract":"A narration.","thumb_url":"https://example.test/cover","serial_count":12}]}]}]}}`)
	page, err := ParseDiscoverTab(raw, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "7100000000000401" || page.Items[0].Title != "Fixture Audio" || page.Items[0].EpisodeCount != 12 || page.Items[0].Description != "A narration." {
		t.Fatalf("page = %+v", page.Items)
	}
	asString := []byte(`{"code":0,"data":{"tab_item":[{"tab_type":5,"cell_data":[{"book_data":[{"book_id":"7100000000000401","book_name":"Fixture Audio","serial_count":"12"}]}]}]}}`)
	page, err = ParseDiscoverTab(asString, 5)
	if err != nil || len(page.Items) != 1 || page.Items[0].EpisodeCount != 12 {
		t.Fatalf("string count = %+v %v", page, err)
	}
}

func TestContinuationUsesChildlessCell(t *testing.T) {
	raw := []byte(`{"code":0,"data":{"tab_item":[{"tab_type":8,"session_id":"feed-session","cell_data":[{"cell_id_str":"7100000000000301","show_type":506}]}]}}`)
	next, ok := DiscoverContinuationTab(raw, 8)
	if !ok || next.CellID != "7100000000000301" || next.ShowType != "506" || next.TabType != "8" || next.SessionID != "feed-session" {
		t.Fatalf("continuation = %+v ok=%v", next, ok)
	}
}

func TestParseSearchReadsBookUserAndPost(t *testing.T) {
	raw := []byte(`{"code":0,"search_tabs":[{"tab_type":1,"data":[{"book_data":{"book_id":"7100000000000402","book_name":"Fixture Novel"}}]},{"tab_type":27,"data":[{"celebrity_user_list":[{"user_id":7100000000000403,"base_info":{"user_name":"Fixture Reader","open_id":"hidden"}}]}]},{"tab_type":31,"data":[{"post_data":{"post_id":"7100000000000404","title":"Fixture Thread"}}]}]}`)
	books, err := ParseSearchTab(raw, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(books.Items) != 1 || books.Items[0].Title != "Fixture Novel" {
		t.Fatalf("books = %+v", books.Items)
	}
	users, err := ParseSearchTab(raw, 27)
	if err != nil {
		t.Fatal(err)
	}
	if len(users.Items) != 1 || users.Items[0].ID != "7100000000000403" || users.Items[0].Title != "Fixture Reader" || strings.Contains(fmtUsers(users), "hidden") {
		t.Fatalf("users = %+v", users.Items)
	}
	posts, err := ParseSearchTab(raw, 31)
	if err != nil {
		t.Fatal(err)
	}
	if len(posts.Items) != 1 || posts.Items[0].ID != "7100000000000404" || posts.Items[0].Title != "Fixture Thread" {
		t.Fatalf("posts = %+v", posts.Items)
	}
}

func fmtUsers(page *CatalogPage) string {
	if page == nil || len(page.Items) == 0 {
		return ""
	}
	return page.Items[0].Title + page.Items[0].Description + page.Items[0].CoverURL
}

func TestParseSearchAcceptsSingleVideoObject(t *testing.T) {
	raw := []byte(`{"code":0,"search_tabs":[{"tab_type":11,"data":[{"show_type":532,"video_data":[{"series_id":"7100000000000199","title":"短剧"}]},{"show_type":369,"video_data":{"series_id":"7100000000000101","title":"noise","video_detail":{"series_title":"Fixture Lantern Court","episode_cnt":2}}}]}]}`)
	page, err := ParseSearch(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "7100000000000101" || page.Items[0].Title != "Fixture Lantern Court" {
		t.Fatalf("page = %+v", page.Items)
	}
}

func TestParseSearchVideoTabUsesDetailListTitle(t *testing.T) {
	raw := []byte(`{"code":0,"search_tabs":[{"tab_type":30,"data":[{"video_data":{"series_id":"7100000000000199","title":"promo line","video_detail":{"video_list":[{"title":"promo line"}],"video_detail_list":[{"series_id":"7100000000000108","series_title":"Fixture Clip","episode_cnt":3}]}}}]}]}`)
	page, err := ParseSearchTab(raw, 30)
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "7100000000000108" || page.Items[0].Title != "Fixture Clip" || page.Items[0].EpisodeCount != 3 {
		t.Fatalf("page = %+v", page.Items)
	}
}

func TestSignedSearchPairOrder(t *testing.T) {
	pairs := SignedSearchPairs(SearchIdentity{DeviceID: "1", InstallID: "2", CDID: "c", OpenUDID: "o"}, "短剧", 0)
	if pairs[0] != [2]string{"iid", "2"} || pairs[1] != [2]string{"device_id", "1"} {
		t.Fatalf("head = %v", pairs[:2])
	}
	if pairs[len(pairs)-1] != [2]string{"user_is_login", "0"} {
		t.Fatalf("tail = %v", pairs[len(pairs)-1])
	}
	got := map[string]string{}
	for _, pair := range pairs {
		got[pair[0]] = pair[1]
	}
	if got["query"] != "短剧" || got["tab_type"] != "11" || got["search_source"] != "1" || got["count"] != "20" || got["channel"] != "xiaomi_8662_64" || got["openudid"] != "o" {
		t.Fatalf("pairs = %v", got)
	}
}

func TestParseSearchSurfacesIDsAndTitles(t *testing.T) {
	page, err := ParseSearch(fixture(t, "search-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "7100000000000101" || page.Items[0].Title != "Fixture Lantern Court" {
		t.Fatalf("search = %+v", page.Items)
	}
}

func TestParseSeriesEpisodeNumbersStartAtOne(t *testing.T) {
	series, episodes, err := ParseSeries(fixture(t, "series-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	if series.ID != "7100000000000101" || series.Title != "Fixture Lantern Court" {
		t.Fatalf("series = %+v", series)
	}
	if len(episodes) != 2 || episodes[0].Number != 1 || episodes[1].Number != 2 {
		t.Fatalf("episodes = %+v", episodes)
	}
	if episodes[0].ID != "7100000000000201" || episodes[0].Title != "Episode One" {
		t.Fatalf("episode = %+v", episodes[0])
	}
}

func TestSharedSynopsisBecomesEpisodeNumber(t *testing.T) {
	raw := []byte(`{"code":0,"data":{"video_data":{"series_id_str":"7100000000000101","series_title":"Fixture Lantern Court","series_intro":"same synopsis","video_list":[{"vid":"7100000000000201","vid_index":1,"title":"same synopsis"},{"vid":"7100000000000202","vid_index":2,"title":"Cold Open"}]}}}`)
	_, episodes, err := ParseSeries(raw)
	if err != nil {
		t.Fatal(err)
	}
	if episodes[0].Title != "第1集" || episodes[1].Title != "Cold Open" {
		t.Fatalf("titles = %q %q", episodes[0].Title, episodes[1].Title)
	}
}

func TestZeroBasedEpisodeIndexBecomesOne(t *testing.T) {
	raw := []byte(`{"code":0,"data":{"video_data":{"series_id_str":"7100000000000101","series_title":"Fixture Lantern Court","video_list":[{"vid":"7100000000000201","vid_index":0,"title":"Episode One"},{"vid":"7100000000000202","vid_index":1,"title":"Episode Two"}]}}}`)
	_, episodes, err := ParseSeries(raw)
	if err != nil {
		t.Fatal(err)
	}
	if episodes[0].Number != 1 || episodes[1].Number != 2 {
		t.Fatalf("numbers = %d %d", episodes[0].Number, episodes[1].Number)
	}
}

func TestParsePlaybackMarksDRMWithoutKeys(t *testing.T) {
	playback, err := ParsePlayback(fixture(t, "drm-playback.json"), "7100000000000201")
	if err != nil {
		t.Fatal(err)
	}
	if !playback.Protected || playback.EpisodeID != "7100000000000201" || len(playback.Streams) != 1 {
		t.Fatalf("playback = %+v", playback)
	}
	stream := playback.Streams[0]
	if !stream.Protected || stream.Encryption != "cenc-aes-ctr" || stream.URL == "" {
		t.Fatalf("stream = %+v", stream)
	}
	// The wrapped key survives on the playback so the decrypt path can use it,
	// but no stream carries key material and JSON never prints it.
	rendered := fmt.Sprintf("%+v", playback.Streams)
	for _, banned := range []string{"fixture-kid", "fixture-spade", "gear_des_key", "spade_a", "check_info", "0:MP4"} {
		if strings.Contains(rendered, banned) {
			t.Fatalf("stream retained key material %q: %s", banned, rendered)
		}
	}
	if playback.KeyMaterial == "" {
		t.Fatal("wrapped key material was dropped")
	}
	encoded, err := json.Marshal(playback)
	if err != nil {
		t.Fatal(err)
	}
	for _, banned := range []string{"fixture-spade", "KeyMaterial", "key_material"} {
		if strings.Contains(string(encoded), banned) {
			t.Fatalf("json leaked %q: %s", banned, encoded)
		}
	}
}

func TestParsePlaybackReadsClearStreams(t *testing.T) {
	playback, err := ParsePlayback(fixture(t, "clear-playback.json"), "7100000000000201")
	if err != nil {
		t.Fatal(err)
	}
	if playback.Protected || len(playback.Streams) != 2 || playback.DurationSeconds != 42 {
		t.Fatalf("playback = %+v", playback)
	}
	best, err := playback.Select("")
	if err != nil || best.Definition != "1080p" {
		t.Fatalf("best = %+v err = %v", best, err)
	}
	worst, err := playback.Select("worst")
	if err != nil || worst.Definition != "360p" {
		t.Fatalf("worst = %+v err = %v", worst, err)
	}
	if _, err := playback.Select("720p"); err == nil || !strings.Contains(err.Error(), "360p") {
		t.Fatalf("missing quality err = %v", err)
	}
}

func TestParsePlaybackAcceptsObjectModelAndKeyedList(t *testing.T) {
	raw := []byte(`{"code":"0","data":{"video_model":{"status":10,"video_id":"v-object","video_list":{"540p":{"main_url":"https://media.invalid/540.mp4","video_meta":{"definition":"540p","vheight":960}}}}}}`)
	playback, err := ParsePlayback(raw, "7100000000000201")
	if err != nil {
		t.Fatal(err)
	}
	if len(playback.Streams) != 1 || playback.Streams[0].Definition != "540p" || playback.Protected {
		t.Fatalf("playback = %+v", playback)
	}
}

func TestParsePlaybackRejectsBusinessCodeAndEmptyList(t *testing.T) {
	if _, err := ParsePlayback([]byte(`{"code":10001,"message":"denied","data":{}}`), "7100000000000201"); err == nil || !strings.Contains(err.Error(), "10001") {
		t.Fatalf("business code err = %v", err)
	}
	if _, err := ParsePlayback([]byte(`{"code":0,"data":{"video_model":"{\"status\":10,\"video_list\":[]}"}}`), "7100000000000201"); err == nil {
		t.Fatal("empty video_list was accepted")
	}
}

func TestFixturesContainNoSecrets(t *testing.T) {
	dir := filepath.Join("..", "..", "testdata", "protocol")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Media URLs are public response fields and use an invalid host in fixtures.
	// Credentials, cookies, signatures, and play tokens stay banned.
	banned := []string{"cookie", "device_token", "X-SS-STUB", "Bearer ", "Authorization", "sessionid", "X-Gorgon", "play_auth"}
	for _, entry := range entries {
		body, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		text := string(body)
		for _, word := range banned {
			if strings.Contains(strings.ToLower(text), strings.ToLower(word)) {
				t.Fatalf("%s contains %s", entry.Name(), word)
			}
		}
	}
}

func TestAnonymousSearchOmitsAccountFields(t *testing.T) {
	q := SearchQuery("短剧", "7100000000000001", "7100000000000002", 1_700_000_000_123, 20)
	if q.Get("query") != "短剧" || q.Get("tab_type") != "11" || q.Get("search_source") != "1" || q.Get("user_is_login") != "0" {
		t.Fatalf("query = %s", q.Encode())
	}
	if q.Get("count") != "20" || q.Get("tab_name") != "feed" || q.Get("bookstore_tab") != "16" {
		t.Fatalf("query = %s", q.Encode())
	}
	headers := SearchHeaders(1_700_000_000_123, 7)
	for _, name := range CredentialHeaderNames {
		if _, ok := headers[name]; ok {
			t.Fatalf("header %s present", name)
		}
	}
	if headers["x-reading-request"] != "1700000000123-7" || headers["X-Xs-From-Web"] != "0" {
		t.Fatalf("headers = %#v", headers)
	}
}

func TestRegisterEnvelopeIsPortableTTData(t *testing.T) {
	body, err := BuildRegisterBody(RegisterIdentity{
		CDID: "11111111-1111-4111-8111-111111111111", OpenUDID: "abcdef0123456789",
		ClientUDID: "22222222-2222-4222-8222-222222222222", OAID: "33333333-3333-4333-8333-333333333333",
		ReqID: "44444444-4444-4444-8444-444444444444",
	}, fixtureTime())
	if err != nil {
		t.Fatal(err)
	}
	if len(body) < 6 || body[0] != 0x74 || body[1] != 0x63 || body[2] != 0x05 || body[3] != 0x10 {
		t.Fatalf("envelope prefix = %x", body[:6])
	}
	again, err := BuildRegisterBody(RegisterIdentity{
		CDID: "11111111-1111-4111-8111-111111111111", OpenUDID: "abcdef0123456789",
		ClientUDID: "22222222-2222-4222-8222-222222222222", OAID: "33333333-3333-4333-8333-333333333333",
		ReqID: "44444444-4444-4444-8444-444444444444",
	}, fixtureTime())
	if err != nil {
		t.Fatal(err)
	}
	if string(body) != string(again) {
		t.Fatal("register body is not deterministic for a fixed identity")
	}
	plain, err := BuildBootstrapPlaintext(registerHeader(RegisterIdentity{OAID: "33333333-3333-4333-8333-333333333333"}, fixtureTime()), fixtureTime())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(plain), `"magic_tag":"ss_app_log"`) || !strings.Contains(string(plain), `"package":"com.phoenix.read"`) {
		t.Fatal("plaintext missing anonymous registration fields")
	}
	if strings.Contains(string(plain), "device_token") || strings.Contains(string(plain), "Cookie") {
		t.Fatal("plaintext contains a credential")
	}
}

func TestRedactedURLDropsQuery(t *testing.T) {
	got := RedactedURL("https://reading.snssdk.com/reading/bookapi/search/tab/v?device_id=7100000000000001&query=x")
	if got != "https://reading.snssdk.com/reading/bookapi/search/tab/v" {
		t.Fatalf("redacted = %s", got)
	}
	if strings.Contains(got, "7100000000000001") {
		t.Fatal("device id leaked into redacted URL")
	}
}

func TestParsePlaybackReadsSegmentedMovie(t *testing.T) {
	playback, err := ParsePlayback(fixture(t, "dynamic-playback.json"), "7100000000000301")
	if err != nil {
		t.Fatal(err)
	}
	if !playback.Protected || playback.DurationSeconds != 1500 {
		t.Fatalf("playback = %+v", playback)
	}
	if len(playback.Streams) != 2 {
		t.Fatalf("streams = %d", len(playback.Streams))
	}
	for _, s := range playback.Streams {
		if !s.Dash || !s.Protected || s.URL == "" {
			t.Fatalf("stream = %+v", s)
		}
	}
	if playback.AudioURL != "https://media.invalid/movie-audio.mp4" {
		t.Fatalf("audio = %q", playback.AudioURL)
	}
	if playback.KeyMaterial != "fixture-movie-spade" {
		t.Fatalf("key material = %q", playback.KeyMaterial)
	}
	best, err := playback.Select("")
	if err != nil || best.Definition != "1080p" {
		t.Fatalf("best = %+v err = %v", best, err)
	}
}

func TestParseSeriesReadsStatusPaywallAndFlags(t *testing.T) {
	series, episodes, err := ParseSeries(fixture(t, "series-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	if series.Status != model.SeriesEnd || series.PayType != model.PayFree {
		t.Fatalf("status = %v pay = %v", series.Status, series.PayType)
	}
	if series.ContentType != model.ContentShortSeries || series.Platform != model.PlatformOuter {
		t.Fatalf("kind = %v platform = %v", series.ContentType, series.Platform)
	}
	if series.Heat != 120000 || series.PlayCount != 340000 || series.FollowCount != 5600 || series.DiggCount != 78 {
		t.Fatalf("counters = %+v", series)
	}
	if series.RightText != "全2集" || len(series.Subtitles) != 2 {
		t.Fatalf("badge = %q subtitles = %v", series.RightText, series.Subtitles)
	}
	if series.CreatedAt.IsZero() || series.CreatedAt.Unix() != 1735689600 {
		t.Fatalf("created = %v", series.CreatedAt)
	}
	if len(series.Rows) != 2 || series.Rows[0].Kind != model.InfoSeasonOrder || series.Rows[1].Kind != model.InfoCategory {
		t.Fatalf("rows = %+v", series.Rows)
	}
	if series.Rows[1].Schema == "" || series.Rows[1].Text != "古装" {
		t.Fatalf("row = %+v", series.Rows[1])
	}

	if len(episodes) != 2 {
		t.Fatalf("episodes = %d", len(episodes))
	}
	if !episodes[0].Playable || !episodes[0].Vertical || !episodes[0].IsNew {
		t.Fatalf("episode 1 = %+v", episodes[0])
	}
	if episodes[1].Playable {
		t.Fatalf("episode 2 should be locked: %+v", episodes[1])
	}
	if episodes[0].CoverURL != "https://media.invalid/ep1.jpg" || episodes[0].CommentCount != 12 || episodes[0].DiggCount != 34 {
		t.Fatalf("episode 1 detail = %+v", episodes[0])
	}
	if episodes[0].ContentType != model.ContentShortSeries {
		t.Fatalf("episode content type = %v", episodes[0].ContentType)
	}
}

func TestURLLifetimeReadsSignedExpiry(t *testing.T) {
	raw := "https://p3-reading-sign.fqnovelpic.com/novel-pic/x~tplv-aifit:400:0.heic?lk3s=k&x-expires=1797914570&x-signature=abc"
	if got := URLLifetime(raw); got.Unix() != 1797914570 {
		t.Fatalf("expiry = %v", got)
	}
	if got := URLLifetime("https://media.invalid/plain.jpg"); !got.IsZero() {
		t.Fatalf("plain url expiry = %v", got)
	}
	if got := URLLifetime(""); !got.IsZero() {
		t.Fatalf("empty url expiry = %v", got)
	}
}

func TestParseSuggestPrefersRichRows(t *testing.T) {
	got, err := ParseSuggest(fixture(t, "suggest-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("suggestions = %+v", got)
	}
	if got[0].Text != "Fixture Word One" || got[0].Keyword != "7100000000000101" || got[0].Kind != 2 {
		t.Fatalf("first = %+v", got[0])
	}
	// A text keyword stays the text.
	if got[1].Keyword != "Fixture Word Two" {
		t.Fatalf("second = %+v", got[1])
	}
}

func TestParseCategoryLandingReadsFiltersAndRows(t *testing.T) {
	page, err := ParseCategoryLanding(fixture(t, "category-landing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if page.CategoryID != "68" || page.CategoryName != "Fixture Category" {
		t.Fatalf("category = %+v", page)
	}
	if !page.HasMore || page.NextOffset != 20 {
		t.Fatalf("paging = %+v", page)
	}
	if len(page.Filters) != 4 {
		t.Fatalf("filters = %+v", page.Filters)
	}
	if page.Filters[0].Key != "book_status" || len(page.Filters[0].Options) != 2 {
		t.Fatalf("first filter = %+v", page.Filters[0])
	}
	if len(page.Items) != 1 {
		t.Fatalf("items = %+v", page.Items)
	}
	item := page.Items[0]
	if item.ID != "7100000000000301" || item.Author != "Fixture Author" || item.Words != 123456 {
		t.Fatalf("item = %+v", item)
	}
	if item.Description != "First line of the abstract." {
		t.Fatalf("abstract = %q", item.Description)
	}
	if item.CoverExpiresAt.IsZero() {
		t.Fatalf("cover expiry missing: %+v", item)
	}
}

func TestParseCategoryFrontReadsTabs(t *testing.T) {
	page, err := ParseCategoryFront(fixture(t, "category-front.json"))
	if err != nil {
		t.Fatal(err)
	}
	if page.DefaultTab != 8 || len(page.Tabs) != 3 {
		t.Fatalf("front = %+v", page)
	}
	if page.Tabs[0].Name != "全部" || page.Tabs[0].Value != "8" {
		t.Fatalf("tab = %+v", page.Tabs[0])
	}
}

func TestClassifyBusinessCodes(t *testing.T) {
	cases := []struct {
		code int
		msg  string
		want ErrorKind
	}{
		{101, "", KindEndOfList},
		{100, `{"minBanTime":60,"maxBanTime":120,"banRate":0.5}`, KindRateLimited},
		{101143, "", KindRateLimited},
		{101108, "", KindAuth},
		{101123, "", KindAuth},
		{101109, "", KindContentGone},
		{101001, "合集已下架", KindContentGone},
		{100103, "", KindParam},
		{111, "", KindChallenge},
		{112, "", KindClientRejected},
		{999999, "请求过于频繁", KindRateLimited},
		{999999, "内容已下架", KindContentGone},
		{999999, "", KindUnknown},
	}
	for _, c := range cases {
		if got := Classify(c.code, c.msg); got != c.want {
			t.Fatalf("Classify(%d,%q) = %v want %v", c.code, c.msg, got, c.want)
		}
	}
	if CodeName(101109) != "BOOK_REMOVE" || CodeName(123) != "" {
		t.Fatalf("CodeName table is wrong")
	}
}

func TestBanWindowReadsSvrBanModel(t *testing.T) {
	min, max, ok := BanWindow(`{"minBanTime":30,"maxBanTime":90,"banRate":0.5}`)
	if !ok || min.Seconds() != 30 || max.Seconds() != 90 {
		t.Fatalf("window = %v %v %v", min, max, ok)
	}
	if _, _, ok := BanWindow("请求成功"); ok {
		t.Fatal("plain message parsed as a ban window")
	}
}

func TestBaseStatusAndDeviceRejection(t *testing.T) {
	raw := []byte(`{"BaseResp":{"StatusCode":263175},"code":0}`)
	if got := BaseStatus(raw); got != 263175 {
		t.Fatalf("status = %d", got)
	}
	if !DeviceRejected(263175) || !DeviceRejected(263176) || DeviceRejected(0) {
		t.Fatal("device rejection table is wrong")
	}
}

func TestParsePlanCellFindsRelatedVideo(t *testing.T) {
	cells, err := ParsePlanCell(fixture(t, "plan-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(cells) != 1 || cells[0].ShowType != 160 || cells[0].CellName != "相关推荐" {
		t.Fatalf("cells = %+v", cells)
	}
	if len(cells[0].Items) != 2 || cells[0].Items[0].ID != "7100000000000101" {
		t.Fatalf("items = %+v", cells[0].Items)
	}
	if cells[0].Items[0].CoverExpiresAt.IsZero() {
		t.Fatal("cover expiry missing")
	}
	// The live cell carries no session id.
	if cells[0].SessionID != "" {
		t.Fatalf("session = %q", cells[0].SessionID)
	}
	// Older cells name the same fields series_title and series_cover.
	legacy, err := ParsePlanCell([]byte(`{"code":0,"session_id":"s1","data":[{"show_type":160,"video_data":[{"series_id":"7100000000000103","series_title":"Legacy","series_cover":"https://media.invalid/c.jpg"}]}]}`))
	if err != nil || len(legacy) != 1 || legacy[0].Items[0].Title != "Legacy" || legacy[0].SessionID != "s1" {
		t.Fatalf("legacy = %+v err %v", legacy, err)
	}
}

func TestParseRankingReadsListsAndItems(t *testing.T) {
	rank, err := ParseRanking(fixture(t, "ranking-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(rank.Lists) != 1 || rank.Lists[0].ID != "hongguo_actor_fans_ranklist" || rank.Lists[0].Size != 50 {
		t.Fatalf("lists = %+v", rank.Lists)
	}
	if len(rank.Items) != 1 {
		t.Fatalf("items = %+v", rank.Items)
	}
	item := rank.Items[0]
	if item.Rank != 1 || item.Votes != 120 || item.Title != "Fixture Play" || item.Actor != "Fixture Actor" {
		t.Fatalf("item = %+v", item)
	}
	if rank.ListName != "星粉榜" || rank.CanVote {
		t.Fatalf("ranking = %+v", rank)
	}
}

func TestParseRankRowsReadsPageLocalRanks(t *testing.T) {
	rank, err := ParseRankRows(fixture(t, "rank-rows.json"), 0)
	if err != nil {
		t.Fatal(err)
	}
	if !rank.HasMore || rank.NextOffset != 20 || rank.SessionID != "fixture-session" {
		t.Fatalf("paging = %+v", rank)
	}
	if len(rank.Items) != 2 {
		t.Fatalf("items = %+v", rank.Items)
	}
	if rank.Items[0].Rank != 1 || rank.Items[0].Score != "2550万热度" || rank.Items[0].Actor != "Fixture Author" {
		t.Fatalf("first = %+v", rank.Items[0])
	}
	if rank.Items[1].Rank != 2 {
		t.Fatalf("second = %+v", rank.Items[1])
	}
	// recommend_info may also arrive as a plain object with a numeric rank.
	plain, err := ParseRankRows([]byte(`{"code":0,"data":{"cell_view":{"book_data":[{"book_id":"7100000000000503","book_name":"Plain","recommend_info":{"rank":3}}]}}}`), 0)
	if err != nil || len(plain.Items) != 1 || plain.Items[0].Rank != 3 {
		t.Fatalf("plain = %+v err %v", plain, err)
	}
}

func TestParseRankGroupsWalksNestedCells(t *testing.T) {
	groups := ParseRankGroups(fixture(t, "rank-groups.json"))
	if len(groups) != 2 {
		t.Fatalf("groups = %+v", groups)
	}
	if groups[0].ID != "179" || groups[0].Name != "推荐榜" {
		t.Fatalf("first = %+v", groups[0])
	}
	if groups[1].ID != "183" {
		t.Fatalf("second = %+v", groups[1])
	}
}

func TestParseCueReadsHints(t *testing.T) {
	words, err := ParseCue(fixture(t, "cue-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(words) != 2 || words[0] != "书名" || words[1] != "主角" {
		t.Fatalf("words = %q", words)
	}
	if _, err := ParseCue([]byte(`{"code":100103,"message":"bad"}`)); err == nil {
		t.Fatal("error code accepted")
	}
}

func TestParseShortFilterReadsPanelAndRows(t *testing.T) {
	panel, err := ParseShortFilter(fixture(t, "short-filter-panel.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(panel.Rows) != 4 || panel.Rows[0].Key != "genre" || panel.Rows[0].Name != "全部体裁" {
		t.Fatalf("rows = %+v", panel.Rows)
	}
	if o := panel.Rows[1].Options; len(o) != 2 || o[0].Name != "悬疑" || o[0].Value != "cate_165" {
		t.Fatalf("theme options = %+v", o)
	}
	content, err := ParseShortFilter(fixture(t, "short-filter-content.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !content.HasMore || content.NextOffset != 18 || content.SessionID != "fixture-session" {
		t.Fatalf("paging = %+v", content)
	}
	if len(content.Items) != 2 || content.Items[0].ID != "7100000000000101" || content.Items[1].EpisodeCount != 30 {
		t.Fatalf("items = %+v", content.Items)
	}
	if content.Items[0].CoverExpiresAt.IsZero() {
		t.Fatal("cover expiry missing")
	}
}

func TestParseShortRankReadsCatalogAndPromotedRows(t *testing.T) {
	page, err := ParseShortRank(fixture(t, "short-rank.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !page.HasMore || page.NextOffset != 10 || page.Version != "1792811520" || page.SessionID != "fixture-session" {
		t.Fatalf("paging = %+v", page)
	}
	if len(page.Contents) != 2 || page.Contents[1].ID != "human" || !page.Contents[1].Selected {
		t.Fatalf("contents = %+v", page.Contents)
	}
	if b := page.Contents[1].Boards; len(b) != 2 || b[1].ID != "human_hot_play" || !b[1].Selected || b[0].Name != "推荐榜" {
		t.Fatalf("boards = %+v", b)
	}
	if len(page.Items) != 3 {
		t.Fatalf("items = %+v", page.Items)
	}
	first, inserted, second := page.Items[0], page.Items[1], page.Items[2]
	if first.Position != 1 || first.Rank != 1 || first.Promoted || first.Score != "7921万热度" || first.EpisodeCount != 60 {
		t.Fatalf("first = %+v", first)
	}
	// The inserted row reads 平台推荐 and copies the next row's rank.
	if inserted.Position != 2 || inserted.Rank != 2 || !inserted.Promoted {
		t.Fatalf("inserted = %+v", inserted)
	}
	if second.Position != 3 || second.Rank != 2 || second.Promoted {
		t.Fatalf("second = %+v", second)
	}
}

func TestShortCursorRoundTripsAndRejectsJunk(t *testing.T) {
	want := ShortCursor{Offset: 18, Seen: []string{"7100000000000101", "7100000000000102"}, Session: "s"}
	token, err := EncodeShortCursor(want)
	if err != nil {
		t.Fatal(err)
	}
	got, err := DecodeShortCursor(token)
	if err != nil || got.Offset != 18 || len(got.Seen) != 2 || got.Session != "s" {
		t.Fatalf("got %+v err %v", got, err)
	}
	if first, err := DecodeShortCursor(""); err != nil || first.Offset != 0 {
		t.Fatalf("empty = %+v err %v", first, err)
	}
	bad, _ := EncodeShortCursor(ShortCursor{Offset: 1, Seen: []string{"1,2"}})
	for _, token := range []string{"d1.abc", "s1.!!!", bad} {
		if _, err := DecodeShortCursor(token); err == nil {
			t.Fatalf("%q accepted", token)
		}
	}
}

func TestParseSeasonsReadsSeriesRow(t *testing.T) {
	seasons, err := ParseSeasons(fixture(t, "series-seasons.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(seasons) != 3 || seasons[1].SeriesID != "7100000000000502" || seasons[1].Index != 2 ||
		seasons[1].Title != "Fixture Saga 第二季" || seasons[1].CoverURL == "" || seasons[0].EpisodeCount == 0 {
		t.Fatalf("seasons = %+v", seasons)
	}
	// Other cells of the feed are not seasons.
	other, err := ParseSeasons(fixture(t, "plan-response.json"))
	if err != nil || len(other) != 0 {
		t.Fatalf("other = %+v err %v", other, err)
	}
}

func TestParseHotSearchReadsLynxLists(t *testing.T) {
	hot, err := ParseHotSearch(fixture(t, "hot-search.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(hot.Guesses) != 2 || hot.Guesses[0].SeriesID != "7100000000000801" || hot.Guesses[1].Position != 2 {
		t.Fatalf("guesses = %+v", hot.Guesses)
	}
	var names []string
	for _, b := range hot.Boards {
		names = append(names, b.Name)
	}
	if strings.Join(names, ",") != "短剧热搜榜,漫剧热搜榜,热点话题榜" {
		t.Fatalf("boards = %v", names)
	}
	item := hot.Boards[0].Items[0]
	if item.Position != 1 || item.SeriesID != "7100000000000811" || item.SearchScore != "389万热搜值" ||
		item.Heat != "4011万热度" || item.EpisodeCount == 0 || item.ContentType != 1 || item.CoverExpiresAt.IsZero() {
		t.Fatalf("item = %+v", item)
	}
	if hot.Boards[1].Items[0].ContentType != 1004 {
		t.Fatalf("comic item = %+v", hot.Boards[1].Items[0])
	}
	topic := hot.Boards[2].Topics[0]
	if len(hot.Boards[2].Items) != 0 || topic.ID != "7100000000001015" || topic.Query != "Fixture topic 15" || topic.Heat == "" {
		t.Fatalf("topics = %+v", hot.Boards[2])
	}
	// The other cells of the feed carry no lists.
	empty, err := ParseHotSearch(fixture(t, "plan-response.json"))
	if err != nil || len(empty.Boards) != 0 || len(empty.Guesses) != 0 {
		t.Fatalf("plan response = %+v err %v", empty, err)
	}
}

func TestParseShortRankReadsActors(t *testing.T) {
	res, err := ParseShortRank(fixture(t, "short-rank-actors.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 3 || !res.HasMore || res.NextOffset != 3 {
		t.Fatalf("result = %+v", res)
	}
	a := res.Items[0]
	if a.UserID != "7100000000001301" {
		t.Fatalf("user ID = %q", a.UserID)
	}
	if a.Position != 1 || a.ActorID != "7100000000001401" || a.Actor != "Fixture Actor A" || a.Score != "700万人气" ||
		a.AvatarURL == "" || a.Intro != "《Fixture Work 1》" || strings.Join(a.Details, "|") != "10部作品|双周追剧200万人" {
		t.Fatalf("actor = %+v", a)
	}
	if a.Change == nil || a.Change.Kind != model.RankUp || a.Change.Places != 1 {
		t.Fatalf("change = %+v", a.Change)
	}
	if c := res.Items[1].Change; c == nil || c.Kind != model.RankDown || c.Places != 2 {
		t.Fatalf("second change = %+v", c)
	}
	if c := res.Items[2].Change; c == nil || c.Kind != model.RankNew {
		t.Fatalf("third change = %+v", c)
	}
	var boards []string
	for _, c := range res.Contents {
		if c.Selected {
			for _, b := range c.Boards {
				boards = append(boards, b.ID)
			}
		}
	}
	if strings.Join(boards, ",") != "ranklist_series_actor,ranklist_actor_fans" {
		t.Fatalf("actor boards = %v", boards)
	}
}

func TestParseNewReleasesReadsOutAndAnnounced(t *testing.T) {
	res, err := ParseNewReleases(fixture(t, "new-releases.json"))
	if err != nil {
		t.Fatal(err)
	}
	if res.Date != "20261023" || len(res.Dates) != 3 || !res.HasMore || res.NextOffset != 2 || res.SessionID != "fixture-session" {
		t.Fatalf("page = %+v", res)
	}
	if len(res.Items) != 1 {
		t.Fatalf("items = %+v", res.Items)
	}
	out := res.Items[0]
	if !out.Online || out.SeriesID != "7100000000001501" || out.Title != "Fixture Out Today" || out.EpisodeCount != 60 ||
		out.Heat != "378万热度" || out.PublishAt.Unix() != 1792772281 || out.CoverExpiresAt.IsZero() {
		t.Fatalf("online = %+v", out)
	}
	// An announced title is the item itself; a placeholder without an ID is
	// dropped.
	soon, err := ParseNewReleases([]byte(`{"code":0,"data":{"subscribe_items":[
		{"is_online":false,"item_id":"7100000000001502","name":"Fixture Coming Soon","item_desc":"d",
		 "cover":"https://media.invalid/new-2.jpg","expected_rec_text":"159.7万人预约","categories":["玄幻脑洞","剧情"],
		 "schedule_publish_time":1792944480,"expected_publish_time":1792944480,"sub_info":"预计10月26日上线"},
		{"is_online":false,"item_id":"0","name":""}]}}`))
	if err != nil || len(soon.Items) != 1 {
		t.Fatalf("announced = %+v err %v", soon, err)
	}
	if r := soon.Items[0]; r.Online || r.SeriesID != "7100000000001502" || r.Title != "Fixture Coming Soon" || r.Heat != "159.7万人预约" ||
		r.Note != "预计10月26日上线" || r.PublishAt.Unix() != 1792944480 || strings.Join(r.Categories, ",") != "玄幻脑洞,剧情" {
		t.Fatalf("upcoming = %+v", r)
	}
}

func TestParseProfileAndWorks(t *testing.T) {
	p, err := ParseProfile(fixture(t, "person.json"))
	if err != nil {
		t.Fatal(err)
	}
	if p.UserID != "7100000000001701" || p.Name != "Fixture Actor" || p.Followers != 2603410 || p.Likes != 185960233 ||
		strings.Join(p.Tags, ",") != "女演员" || strings.Join(p.Labels, ",") != "水瓶座" || p.Description == "" {
		t.Fatalf("person = %+v", p)
	}
	w, err := ParseUserWorks(fixture(t, "person-works.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(w.Items) != 2 || !w.HasMore || !strings.Contains(w.Cursor, `"next_cursor"`) {
		t.Fatalf("works = %+v", w)
	}
	if one := w.Items[0]; one.SeriesID != "7100000000001801" || one.Title != "Fixture Work One" || one.EpisodeCount != 70 ||
		one.Heat != 52912861 || one.Plays != 7409454 || strings.Join(one.Tags, ",") != "爱情,70集" || one.CoverExpiresAt.IsZero() {
		t.Fatalf("work = %+v", one)
	}
}

func TestCursorsRejectForeignValues(t *testing.T) {
	token, err := EncodeShortCursor(ShortCursor{Offset: 2, Date: "20261024"})
	if err != nil {
		t.Fatal(err)
	}
	if c, err := DecodeShortCursor(token); err != nil || c.Date != "20261024" {
		t.Fatalf("cursor = %+v err %v", c, err)
	}
	bad, _ := EncodeShortCursor(ShortCursor{Date: "2026-10-24"})
	if _, err := DecodeShortCursor(bad); err == nil {
		t.Fatal("malformed date accepted")
	}
	works, err := EncodeWorksCursor(WorksCursor{UserID: "7100000000001701", Next: `{"has_more":true}`})
	if err != nil {
		t.Fatal(err)
	}
	if c, err := DecodeWorksCursor(works); err != nil || c.UserID != "7100000000001701" {
		t.Fatalf("works cursor = %+v err %v", c, err)
	}
	if _, err := DecodeWorksCursor(token); err == nil {
		t.Fatal("short cursor accepted as works cursor")
	}
}

func TestParseSeriesBatchReadsEachSeriesAndNovel(t *testing.T) {
	got, err := ParseSeriesBatch(fixture(t, "series-batch.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("series = %d", len(got))
	}
	one, two := got["7100000000000101"], got["7100000000000102"]
	if one.Series == nil || one.Series.Title != "Fixture Lantern Court" || len(one.Episodes) == 0 {
		t.Fatalf("first = %+v", one)
	}
	if two.Series == nil || two.Series.Title != "Fixture North Gate" || two.Episodes[0].SeriesID != "7100000000000102" || two.Series.Novel != nil {
		t.Fatalf("second = %+v", two)
	}
	n := one.Series.Novel
	if n == nil || n.BookID != "7100000000002001" || n.Title != "Fixture Source Novel" || n.ShortTitle != "Fixture Lantern Court" ||
		n.Status != model.NovelFinished || strings.Join(n.Tags, ",") != "悬疑,穿越" || n.Reads != 959435 || n.ShowAtEpisode != 2 || n.CoverExpiresAt.IsZero() {
		t.Fatalf("novel = %+v", n)
	}
	// The single detail read carries the same block.
	single, _, err := ParseSeries([]byte(`{"code":0,"data":{"video_data":{"series_id":"7100000000000103","series_title":"T"},"video_relate_book":{"book_info":{"book_id":"7100000000002002","book_name":"N","creation_status":"1"}}}}`))
	if err != nil || single.Novel == nil || single.Novel.Status != model.NovelOngoing || single.Novel.ShowAtEpisode != 0 {
		t.Fatalf("single = %+v err %v", single, err)
	}
	// No status reads as unknown rather than finished.
	bare, _, err := ParseSeries([]byte(`{"code":0,"data":{"video_data":{"series_id":"7100000000000103","series_title":"T"},"video_relate_book":{"book_info":{"book_id":"7100000000002002","book_name":"N"}}}}`))
	if err != nil || bare.Novel.Status.String() != "unknown" {
		t.Fatalf("bare = %+v err %v", bare.Novel, err)
	}
}

func TestOneDeviceDrivesQueryRegistrationAndUserAgent(t *testing.T) {
	pad := model.Device{Brand: "Xiaomi", Model: "21051182C", OSVersion: "11", OSAPI: 30, Build: "RKQ1.200826.002",
		Resolution: "1600*2524", DPI: 360, Pad: true, Channel: "xiaomi_8662_64"}
	query := map[string]string{}
	for _, pair := range DeviceQueryPairs(SearchIdentity{DeviceID: "1", InstallID: "2", Device: pad}) {
		query[pair[0]] = pair[1]
	}
	for key, want := range map[string]string{
		"device_type": "21051182C", "device_brand": "Xiaomi", "os_api": "30", "os_version": "11",
		"resolution": "1600*2524", "dpi": "360", "dragon_device_type": "pad", "is_android_pad_screen": "1", "channel": "xiaomi_8662_64",
	} {
		if query[key] != want {
			t.Fatalf("%s = %q, want %q", key, query[key], want)
		}
	}
	header := registerHeader(RegisterIdentity{Device: pad}, time.Unix(1, 0))
	for key, want := range map[string]any{
		"device_model": "21051182C", "device_brand": "Xiaomi", "device_manufacturer": "Xiaomi",
		"os_version": "11", "os_api": 30, "channel": "xiaomi_8662_64",
	} {
		if header[key] != want {
			t.Fatalf("register %s = %v, want %v", key, header[key], want)
		}
	}
	if ua := UserAgent(pad); ua != "com.phoenix.read/73932 (Linux; U; Android 11; zh_CN; 21051182C; Build/RKQ1.200826.002; Cronet/TTNetVersion:8d40f833 2026-03-03 QuicVersion:21ac1950 2025-11-18)" {
		t.Fatalf("user agent = %s", ua)
	}
	// The zero device is the default phone everywhere.
	if h := registerHeader(RegisterIdentity{}, time.Unix(1, 0)); h["device_model"] != model.DefaultDevice.Model || h["os_api"] != model.DefaultDevice.OSAPI {
		t.Fatalf("default register = %v", h)
	}
}
