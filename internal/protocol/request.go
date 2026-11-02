package protocol

import (
	"net/url"
	"strconv"
	"strings"

	"github.com/beilunyang/openhongguo/sdk/model"
)

// CommonQuery is the anonymous device query shared by catalog reads.
func CommonQuery(deviceID, installID string, ticketMillis int64) url.Values {
	return url.Values{
		"device_id":       {deviceID},
		"iid":             {installID},
		"aid":             {AID},
		"app_name":        {AppName},
		"version_code":    {VersionCode},
		"version_name":    {VersionName},
		"channel":         {model.DefaultDevice.Channel},
		"device_platform": {DevicePlatform},
		"_rticket":        {strconv.FormatInt(ticketMillis, 10)},
	}
}

// DiscoverQuery is the bookmall tab request.
// Fields follow the anonymous discovery call: bottom tab 7 (short drama) and tab_type -1.
func DiscoverQuery(deviceID, installID string, ticketMillis int64) url.Values {
	q := CommonQuery(deviceID, installID, ticketMillis)
	// bottom_tab_type 7 is BottomTabBarItemType.VideoSeriesFeedTab, which
	// StaggeredFeedTab.fi returns for the series mall. client_req_type 3 is
	// ClientReqType.Open. offset 0 is the first page copied from request args.
	q.Set("bottom_tab_type_list", "7,0,1,2,3,4")
	q.Set("tab_type", "-1")
	q.Set("bottom_tab_type", "7")
	q.Set("offset", "0")
	q.Set("client_req_type", "3")
	return q
}

// SearchQuery is the anonymous first-page short-play search.
// user_is_login=0. Account headers are not part of this query.
func SearchQuery(keyword, deviceID, installID string, ticketMillis int64, count int) url.Values {
	if count <= 0 {
		count = 20
	}
	q := CommonQuery(deviceID, installID, ticketMillis)
	q.Set("query", keyword)
	q.Set("offset", "0")
	q.Set("count", strconv.Itoa(count))
	q.Set("tab_type", SearchTabShortPlay)
	q.Set("search_source", SearchSourceHome)
	q.Set("tab_name", SearchTabName)
	q.Set("bookstore_tab", "16")
	q.Set("bookshelf_search_plan", "4")
	q.Set("user_is_login", "0")
	return q
}

// SearchHeaders are the anonymous headers that do not depend on the signature.
// The signed search request adds the portable signature headers separately.
func SearchHeaders(ticketMillis int64, randomInt int32) map[string]string {
	return map[string]string{
		"Accept":                    "application/json; charset=utf-8",
		"x-reading-request":         ReadingRequestID(ticketMillis, randomInt),
		"X-Xs-From-Web":             "0",
		"x-vc-bdturing-sdk-version": "4.0.3.cn",
	}
}

// ReadingRequestID reproduces the client pattern millis + "-" + abs(randomInt).
func ReadingRequestID(unixMillis int64, randomInt int32) string {
	n := int64(randomInt)
	if randomInt < 0 {
		n = -n
	}
	return strconv.FormatInt(unixMillis, 10) + "-" + strconv.FormatInt(n, 10)
}

// SignedDiscoverPairs is the ordered bookmall query for the 推荐 open.
// Business fields match DiscoverQuery. ts and _rticket are appended by the signer.
func SignedDiscoverPairs(id SearchIdentity) [][2]string {
	return SignedDiscoverPairsFor(id, "-1")
}

// SignedDiscoverPairsFor is SignedDiscoverPairs with the bookmall tab_type.
// An empty tab type keeps the 推荐 open, BookstoreTabType.unknown_tab_type (-1).
// bottom_tab_type stays VideoSeriesFeedTab (7).
func SignedDiscoverPairsFor(id SearchIdentity, tabType string) [][2]string {
	return SignedDiscoverPairsIn(id, tabType, "7")
}

// SignedDiscoverPairsIn is SignedDiscoverPairsFor with an explicit bottom_tab_type.
// An empty bottom stays on the short-drama mall (7). The field is already on
// BookstoreTabRequest; only its value changes between mall and home channels.
func SignedDiscoverPairsIn(id SearchIdentity, tabType, bottom string) [][2]string {
	if tabType == "" {
		tabType = "-1"
	}
	if bottom == "" {
		bottom = "7"
	}
	return append(DeviceQueryPairs(id),
		[2]string{"tab_type", tabType},
		[2]string{"offset", "0"},
		[2]string{"client_req_type", "3"},
		[2]string{"bottom_tab_type", bottom},
		[2]string{"bottom_tab_type_list", "7,0,1,2,3,4"},
	)
}

// FeedContinuation is the unlimited 猜你喜欢 page carried by a bookmall tab response.
// SessionID is a feed cursor from that response. It is not an account credential.
type FeedContinuation struct {
	CellID         string
	SessionID      string
	ClientTemplate string
	ShowType       string
	TabType        string
	// Offset is the cell/change offset. The first unlimited page uses 0.
	// A later page uses next_offset from the previous cell/change response.
	Offset string
	// Limit is the cell/change page size (default 20 when zero).
	Limit int
}

// CellChangeLimit clamps a discover cell/change page size.
func CellChangeLimit(size int) int {
	if size <= 0 {
		return 20
	}
	if size > 50 {
		return 50
	}
	return size
}

// SignedCellChangePairs is the ordered GET for /reading/bookapi/bookmall/cell/change/v.
// The business field order is the one a live unlimited page accepted.
// ts and _rticket are appended by the signer.
func SignedCellChangePairs(id SearchIdentity, next FeedContinuation) [][2]string {
	offset := next.Offset
	if offset == "" {
		offset = "0"
	}
	pairs := append(DeviceQueryPairs(id),
		[2]string{"cell_id", next.CellID},
		[2]string{"offset", offset},
		[2]string{"limit", strconv.Itoa(CellChangeLimit(next.Limit))},
		[2]string{"change_type", "0"},
		[2]string{"tab_type", next.TabType},
		[2]string{"show_type", next.ShowType},
		[2]string{"client_req_type", "2"},
		[2]string{"session_id", next.SessionID},
	)
	if next.ClientTemplate != "" {
		pairs = append(pairs, [2]string{"client_template", next.ClientTemplate})
	}
	return pairs
}

// SearchIdentity is the local anonymous installation used to build a signed search query.
type SearchIdentity struct {
	DeviceID  string
	InstallID string
	CDID      string
	OpenUDID  string
	// Device is the phone the profile registered as; zero fields fall back
	// to model.DefaultDevice.
	Device model.Device
}

// DeviceQueryPairs is the ordered anonymous device query. Order is part of the signature.
func DeviceQueryPairs(id SearchIdentity) [][2]string {
	d := id.Device.OrDefault()
	kind, pad := "phone", "0"
	if d.Pad {
		kind, pad = "pad", "1"
	}
	return [][2]string{
		{"iid", id.InstallID},
		{"device_id", id.DeviceID},
		{"ac", "wifi"},
		{"channel", d.Channel},
		{"aid", AID},
		{"app_name", AppName},
		{"cdid", id.CDID},
		{"version_code", VersionCode},
		{"version_name", VersionName},
		{"device_platform", DevicePlatform},
		{"os", "android"},
		{"ssmix", "a"},
		{"device_type", d.Model},
		{"device_brand", d.Brand},
		{"language", "zh"},
		{"os_api", strconv.Itoa(d.OSAPI)},
		{"os_version", d.OSVersion},
		{"manifest_version_code", VersionCode},
		{"resolution", d.Resolution},
		{"dpi", strconv.Itoa(d.DPI)},
		{"update_version_code", VersionCode},
		{"host_abi", "arm64-v8a"},
		{"dragon_device_type", kind},
		{"pv_player", VersionCode},
		{"compliance_status", "0"},
		{"need_personal_recommend", "1"},
		{"player_so_load", "1"},
		{"is_android_pad_screen", pad},
		{"openudid", id.OpenUDID},
	}
}

// SignedSearchPairs is the ordered search query. Order is part of the signature.
// Business fields match SearchQuery. The extra device fields are the portable
// fingerprint a live anonymous search accepts. ts and _rticket are appended later.
func SignedSearchPairs(id SearchIdentity, keyword string, count int) [][2]string {
	return SignedSearchPairsFor(id, keyword, count, 0, SearchTabShortPlay)
}

// SignedSearchPairsFor is the ordered search query for one tab and offset.
// tab is a SearchTabType value. An empty tab keeps ShortPlay (11).
func SignedSearchPairsFor(id SearchIdentity, keyword string, count, offset int, tab string) [][2]string {
	if count <= 0 {
		count = 20
	}
	if offset < 0 {
		offset = 0
	}
	if tab == "" {
		tab = SearchTabShortPlay
	}
	return append(DeviceQueryPairs(id),
		[2]string{"query", keyword},
		[2]string{"tab_type", tab},
		[2]string{"count", strconv.Itoa(count)},
		[2]string{"offset", strconv.Itoa(offset)},
		[2]string{"use_correct", "true"},
		[2]string{"bookshelf_search_plan", "4"},
		[2]string{"search_source", SearchSourceHome},
		[2]string{"tab_name", SearchTabName},
		[2]string{"bookstore_tab", "16"},
		[2]string{"user_is_login", "0"},
	)
}

// SignedCuePairs is the ordered search-cue query (hot words).
func SignedCuePairs(id SearchIdentity, cueType, limit int) [][2]string {
	return append(DeviceQueryPairs(id),
		[2]string{"cue_type", strconv.Itoa(cueType)},
		[2]string{"limit", strconv.Itoa(limit)},
		[2]string{"search_source", SearchSourceHome},
		[2]string{"tab_name", SearchTabName},
		[2]string{"user_is_login", "0"},
	)
}

// SignedSuggestPairs is the ordered search-suggest query.
func SignedSuggestPairs(id SearchIdentity, keyword string) [][2]string {
	return append(DeviceQueryPairs(id),
		[2]string{"query", keyword},
		[2]string{"count", "10"},
		[2]string{"search_source", SearchSourceHome},
		[2]string{"user_is_login", "0"},
	)
}

// SignedRankCellPairs is the ordered ranking-cell query.
func SignedRankCellPairs(id SearchIdentity, cellID string, offset, limit int) [][2]string {
	pairs := append(DeviceQueryPairs(id),
		[2]string{"cell_id", cellID},
		[2]string{"show_type", "169"},
		[2]string{"offset", strconv.Itoa(offset)},
		[2]string{"limit", strconv.Itoa(limit)},
		[2]string{"change_type", "1"},
		[2]string{"tab_type", "11"},
		[2]string{"client_req_type", "3"},
		[2]string{"user_is_login", "0"},
	)
	return pairs
}

// SignedShortRankPairs is the ordered query of one short-drama ranking page:
// cell/change on ShortRankCell with the content (selected_items) and list
// (sub_selected_items) of the ranking page. Later pages carry the offset, the
// IDs already shown (the service skips them), and the session; the shipped
// page marks them with unlimited_selector_change_type 2.
// ShortRankActors is the ranking content that lists actors.
const ShortRankActors = "ranklist_celebrity"

func SignedShortRankPairs(id SearchIdentity, content, list string, cur ShortCursor) [][2]string {
	change := "1"
	if cur.Offset > 0 {
		change = "2"
	}
	pairs := append(DeviceQueryPairs(id),
		[2]string{"cell_id", ShortRankCell},
		[2]string{"tab_type", "26"},
		[2]string{"selected_items", content},
	)
	if list != "" {
		pairs = append(pairs, [2]string{"sub_selected_items", list})
	}
	pairs = append(pairs,
		[2]string{"offset", strconv.Itoa(cur.Offset)},
		[2]string{"limit", "0"},
		[2]string{"client_req_type", "2"},
		[2]string{"client_template", "2"},
		[2]string{"unlimited_selector_change_type", change},
	)
	// Seen holds actor ids on the actor ranking, which the captured client
	// pages without filter_ids.
	if len(cur.Seen) > 0 && content != ShortRankActors {
		pairs = append(pairs, [2]string{"filter_ids", strings.Join(cur.Seen, ",")})
	}
	if cur.Session != "" {
		pairs = append(pairs, [2]string{"session_id", cur.Session})
	}
	return append(pairs, [2]string{"user_is_login", "0"})
}

// ShortFilterKeys are the selector rows of the short-drama filter page, in
// the order the shipped client sends them in select_items.
var ShortFilterKeys = []string{"category_dim_epoch", "online_time", "gender", "category_dim_role", "genre", "creation_status", "sort", "category_dim_theme"}

// ShortFilterBody is the JSON body of the short-drama filter page. panel asks
// for the selector rows only (req_type only_panel); otherwise it asks for
// series rows matching selections, keyed by ShortFilterKeys.
func ShortFilterBody(scene string, selections map[string][]string, panel bool, cur ShortCursor) ([]byte, error) {
	if panel {
		return jsonMarshal(map[string]any{"req_scene": scene, "need_selector_panel": false, "req_type": "only_panel"})
	}
	items := make(map[string][]string, len(ShortFilterKeys))
	for _, k := range ShortFilterKeys {
		items[k] = append([]string{}, selections[k]...)
	}
	return jsonMarshal(map[string]any{
		"filter_ids":          strings.Join(cur.Seen, ","),
		"req_scene":           scene,
		"offset":              cur.Offset,
		"limit":               18,
		"need_selector_panel": false,
		"select_items":        items,
		"session_id":          cur.Session,
		"req_type":            "only_content",
	})
}

// RankActivityBody is the activity ranking request.
func RankActivityBody(activityID, rankListID string, offset, count int) ([]byte, error) {
	return jsonMarshal(map[string]any{
		"activity_id":  activityID,
		"rank_list_id": rankListID,
		"offset":       offset,
		"count":        count,
	})
}

// SignedPlanPairs is the ordered "plan" query. The detail page's related-video
// surface reuses this generic feed with from=select_panel_recommend and
// scene=10 (observed in the shipped client).
func SignedPlanPairs(id SearchIdentity, bookID, sessionUUID string) [][2]string {
	return SignedPlanPairsFrom(id, "select_panel_recommend", bookID, sessionUUID)
}

// SignedPlanPairsFrom is SignedPlanPairs with an explicit scene discriminator.
// The shipped client uses "select_panel_recommend" for the related-video tab and
// "series_filter_detail" for the 短剧 filter landing.
func SignedPlanPairsFrom(id SearchIdentity, from, bookID, sessionUUID string) [][2]string {
	pairs := append(DeviceQueryPairs(id),
		[2]string{"book_id", bookID},
		[2]string{"from", from},
		[2]string{"scene", "10"},
		[2]string{"session_uuid", sessionUUID},
		[2]string{"user_is_login", "0"},
	)
	return pairs
}

// PlanFromSeriesSeasons is the plan scene of the detail page's 系列剧 row.
const PlanFromSeriesSeasons = "single_col_same_ip_series"

// SignedSeasonPairs is the plan query of the 系列剧 row. The captured client
// (2026-10-24) also sends the series as filter_ids; the service still lists it.
func SignedSeasonPairs(id SearchIdentity, seriesID, sessionUUID string) [][2]string {
	return append(SignedPlanPairsFrom(id, PlanFromSeriesSeasons, seriesID, sessionUUID),
		[2]string{"filter_ids", seriesID})
}

// SignedHotSearchPairs is the plan query of the search page before a query
// (from=search_input_page, middle page version 4, as captured 2026-10-24).
// The client's history, last-read, and A/B fields are left out.
func SignedHotSearchPairs(id SearchIdentity) [][2]string {
	return append(DeviceQueryPairs(id),
		[2]string{"user_is_login", "0"},
		[2]string{"bookstore_tab", "16"},
		[2]string{"bookstore_tab_type", "16"},
		[2]string{"scene", "10"},
		[2]string{"limit", "0"},
		[2]string{"tab_name", SearchTabName},
		[2]string{"from", "search_input_page"},
		[2]string{"query_history_removed", "false"},
		[2]string{"offset", "0"},
		[2]string{"search_middle_page_version", "4"},
		[2]string{"search_source", SearchSourceHome},
		[2]string{"new_search_middle_page", "true"},
		[2]string{"hot_word_exchange", "false"},
	)
}

// SignedNewReleasePairs is the 上新 calendar query for one day (YYYYMMDD;
// empty is today), as the captured client sends it. Later pages carry the
// offset and session of the previous page.
func SignedNewReleasePairs(id SearchIdentity, date string, cur ShortCursor) [][2]string {
	pairs := append(DeviceQueryPairs(id),
		[2]string{"order_experiment", "descend"},
		[2]string{"offset", strconv.Itoa(cur.Offset)},
	)
	if date != "" {
		pairs = append(pairs, [2]string{"target_date", date})
	}
	return append(pairs,
		[2]string{"active_panel", "5"},
		[2]string{"session_id", cur.Session},
		[2]string{"version", "0"},
		[2]string{"gender_type", "2"},
		[2]string{"tab_type", "5"},
		[2]string{"filter_type", "gender"},
		[2]string{"subscribe_list_opt", "0"},
		[2]string{"tab_style", "2"},
		[2]string{"need_calendar_schema", "true"},
		[2]string{"user_is_login", "0"},
	)
}

// SignedProfilePairs is the person-page query.
func SignedProfilePairs(id SearchIdentity, userID string) [][2]string {
	return append(DeviceQueryPairs(id),
		[2]string{"user_id", userID},
		[2]string{"user_is_login", "0"},
	)
}

// UserWorksBody is the 作品 › 剧作 tab of a person's page (tab_type 23,
// sub_tab_type 22). cursor is the service's own cursor string; empty is the
// first page.
func UserWorksBody(userID, cursor string) ([]byte, error) {
	return jsonMarshal(map[string]any{
		"comment_source":    1506,
		"count":             18,
		"cursor":            cursor,
		"extra_param":       map[string]string{"is_video_tab_comment": "1"},
		"locate_post_id":    "",
		"profile_user_type": 0,
		"sort":              "time_desc",
		"sub_tab_type":      22,
		"tab_type":          23,
		"target_user_id":    userID,
	})
}

// SignedCategoryPairs is the ordered category-page query.
func SignedCategoryPairs(id SearchIdentity, categoryID string, offset, count int) [][2]string {
	return append(DeviceQueryPairs(id),
		[2]string{"category_id", categoryID},
		[2]string{"count", strconv.Itoa(count)},
		[2]string{"offset", strconv.Itoa(offset)},
		[2]string{"user_is_login", "0"},
	)
}

// LandingFilterParams maps each filter group of the landing page's rule to
// the query parameter GetNewCategoryLandingPageRequest sends it under, in
// sending order. Observed 2026-10-24: each changes the rows, and sending the
// rule key itself (word_type) is ignored.
var LandingFilterParams = [][2]string{
	{"book_status", "book_status"},
	{"sort_by", "sort_by"},
	{"sub_category", "sub_category_id"},
	{"word_type", "word_number"},
}

// SignedCategoryLandingPairs is the ordered category-landing query. gender is
// query_gender (1 男生, 0 女生) and is omitted for 全部. filters is keyed by
// rule key; groups without a value are omitted.
func SignedCategoryLandingPairs(id SearchIdentity, categoryID, gender string, filters map[string]string, offset, count int) [][2]string {
	pairs := append(DeviceQueryPairs(id), [2]string{"category_id", categoryID})
	if gender != "" {
		pairs = append(pairs, [2]string{"query_gender", gender})
	}
	for _, kv := range LandingFilterParams {
		if v := filters[kv[0]]; v != "" {
			pairs = append(pairs, [2]string{kv[1], v})
		}
	}
	return append(pairs,
		[2]string{"count", strconv.Itoa(count)},
		[2]string{"offset", strconv.Itoa(offset)},
		[2]string{"user_is_login", "0"},
	)
}

// SignedRankPairs is the ordered ranking query.
func SignedRankPairs(id SearchIdentity, rankID string, offset, count int) [][2]string {
	return append(DeviceQueryPairs(id),
		[2]string{"rank_id", rankID},
		[2]string{"count", strconv.Itoa(count)},
		[2]string{"offset", strconv.Itoa(offset)},
		[2]string{"user_is_login", "0"},
	)
}

// SeriesBody is the JSON body for video_detail. series_id is the public series id.
func SeriesBody(seriesID string) ([]byte, error) {
	return jsonMarshal(map[string]any{
		"series_id": seriesID,
		"biz_param": map[string]any{
			"caller_scene":              "single_col",
			"detail_page_version":       0,
			"from_video_id":             "",
			"need_all_video_definition": false,
			"need_mp4_align":            false,
			"source":                    4,
			"video_id_type":             1,
		},
	})
}

// SeriesBatchBody asks multi_video_detail for several series, comma-joined
// as the player sends them.
func SeriesBatchBody(seriesIDs []string) ([]byte, error) {
	return jsonMarshal(map[string]any{
		"series_id": strings.Join(seriesIDs, ","),
		"dr_scene":  "player",
		"biz_param": map[string]any{
			"detail_page_version":       0,
			"need_all_video_definition": false,
			"need_mp4_align":            false,
			"source":                    7,
			"use_os_player":             false,
			"use_server_dns":            false,
		},
	})
}

// PlaybackBody is the JSON body for video_model. The response is classified, not decrypted.
// AlbumBody is the anonymous album (multi-season) request.
// The shipped request class is GetVideoAlbumDetailRequest with albumId,
// seriesIds, needVideoDetailInfo and the shared bizParam block.
func AlbumBody(albumID string) ([]byte, error) {
	return jsonMarshal(map[string]any{
		"album_id":               albumID,
		"biz_param":              map[string]any{"video_platform": 3},
		"need_video_detail_info": true,
		"series_ids":             []int64{},
	})
}

func PlaybackBody(episodeID string) ([]byte, error) {
	return jsonMarshal(map[string]any{
		"video_id": episodeID,
		"biz_param": map[string]any{
			"video_id_type":             0,
			"need_all_video_definition": true,
			"video_platform":            3,
		},
	})
}
