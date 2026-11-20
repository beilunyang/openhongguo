package protocol

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/beilunyang/openhongguo/sdk/model"
)

// CatalogPage is a parsed anonymous catalog response.
type CatalogPage struct {
	Items      []model.SeriesSummary
	HasMore    bool
	NextOffset int
	// SessionID is the feed handle returned with a cell/change page.
	// It is a continuation value, not an account credential.
	SessionID string
}

type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	}
	*f = flexString(b)
	return nil
}

func (f flexString) String() string { return strings.TrimSpace(string(f)) }

// flexInt reads a JSON number or a numeric string. A missing or non-numeric
// value stays 0 so one count field cannot drop the rest of the page.
type flexInt int

func (f *flexInt) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			return nil
		}
		n, err := strconv.Atoi(s)
		if err != nil {
			return nil
		}
		*f = flexInt(n)
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err != nil {
		return nil
	}
	*f = flexInt(n)
	return nil
}

// flexInt64 reads a JSON number or a numeric string for unix-second fields.
type flexInt64 int64

func (f *flexInt64) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return nil
		}
		if n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64); err == nil {
			*f = flexInt64(n)
		}
		return nil
	}
	var n float64
	if err := json.Unmarshal(b, &n); err != nil {
		return nil
	}
	*f = flexInt64(n)
	return nil
}

type catalogVideo struct {
	SeriesID    flexString `json:"series_id"`
	Title       string     `json:"title"`
	VideoDetail struct {
		Title        string            `json:"series_title"`
		Intro        string            `json:"series_intro"`
		Cover        string            `json:"series_cover"`
		EpisodeCount int               `json:"episode_cnt"`
		VideoList    []json.RawMessage `json:"video_list"`
		Cards        []seriesCard      `json:"video_detail_list"`
	} `json:"video_detail"`
}

type seriesCard struct {
	SeriesID     flexString `json:"series_id"`
	SeriesIDStr  flexString `json:"series_id_str"`
	Title        string     `json:"series_title"`
	Intro        string     `json:"series_intro"`
	Cover        string     `json:"series_cover"`
	EpisodeCount int        `json:"episode_cnt"`
}

type cellDetail struct {
	Cards []seriesCard `json:"video_detail_list"`
}

type catalogBook struct {
	BookID      flexString `json:"book_id"`
	Name        string     `json:"book_name"`
	Abstract    string     `json:"abstract"`
	Cover       string     `json:"thumb_url"`
	SerialCount flexInt    `json:"serial_count"`
}

type catalogUser struct {
	UserID flexString `json:"user_id"`
	Base   struct {
		Name string `json:"user_name"`
	} `json:"base_info"`
}

type catalogPost struct {
	PostID flexString `json:"post_id"`
	Title  string     `json:"title"`
}

type catalogCell struct {
	CellIDStr   flexString    `json:"cell_id_str"`
	ShowType    int           `json:"show_type"`
	CellData    []catalogCell `json:"cell_data"`
	VideoData   videoList     `json:"video_data"`
	VideoDetail *cellDetail   `json:"video_detail"`
	Books       bookList      `json:"book_data"`
}

// videoList accepts video_data as one object or as an array.
type videoList []catalogVideo

func (v *videoList) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		return nil
	}
	switch b[0] {
	case '[':
		var many []catalogVideo
		if err := json.Unmarshal(b, &many); err != nil {
			return err
		}
		*v = many
	case '{':
		var one catalogVideo
		if err := json.Unmarshal(b, &one); err != nil {
			return err
		}
		*v = videoList{one}
	default:
		return fmt.Errorf("video_data")
	}
	return nil
}

// bookList accepts book_data as one object or as an array.
type bookList []catalogBook

func (b *bookList) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	switch raw[0] {
	case '[':
		var many []catalogBook
		if err := json.Unmarshal(raw, &many); err != nil {
			return err
		}
		*b = many
	case '{':
		var one catalogBook
		if err := json.Unmarshal(raw, &one); err != nil {
			return err
		}
		*b = bookList{one}
	default:
		return fmt.Errorf("book_data")
	}
	return nil
}

// postList accepts post_data as one object or as an array.
type postList []catalogPost

func (p *postList) UnmarshalJSON(raw []byte) error {
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	switch raw[0] {
	case '[':
		var many []catalogPost
		if err := json.Unmarshal(raw, &many); err != nil {
			return err
		}
		*p = many
	case '{':
		var one catalogPost
		if err := json.Unmarshal(raw, &one); err != nil {
			return err
		}
		*p = postList{one}
	default:
		return fmt.Errorf("post_data")
	}
	return nil
}

// ParseDiscover reads the 推荐 tab (video_feed, 16) from a bookmall response.
func ParseDiscover(raw []byte) (*CatalogPage, error) {
	return ParseDiscoverTab(raw, 16)
}

// ParseDiscoverTab reads one bookmall tab_type into public series ids and titles.
func ParseDiscoverTab(raw []byte, tabType int) (*CatalogPage, error) {
	var env struct {
		Code    flexString `json:"code"`
		Message string     `json:"message"`
		Data    struct {
			TabItems []struct {
				TabType int           `json:"tab_type"`
				HasMore bool          `json:"has_more"`
				Cells   []catalogCell `json:"cell_data"`
			} `json:"tab_item"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("catalog code %s", env.Code.String())
	}
	page := &CatalogPage{Items: []model.SeriesSummary{}}
	for _, tab := range env.Data.TabItems {
		if tab.TabType != tabType {
			continue
		}
		page.HasMore = tab.HasMore
		for _, cell := range tab.Cells {
			appendCatalogCell(page, cell)
		}
	}
	return page, nil
}

func appendCatalogCell(page *CatalogPage, cell catalogCell) {
	for _, v := range cell.VideoData {
		title := v.VideoDetail.Title
		if title == "" {
			title = v.Title
		}
		id := v.SeriesID.String()
		if id == "" || title == "" {
			continue
		}
		count := v.VideoDetail.EpisodeCount
		if count == 0 {
			count = len(v.VideoDetail.VideoList)
		}
		page.Items = append(page.Items, model.SeriesSummary{
			ID: id, Title: title, Description: v.VideoDetail.Intro,
			CoverURL: v.VideoDetail.Cover, EpisodeCount: count,
		})
	}
	if cell.VideoDetail != nil {
		for _, card := range cell.VideoDetail.Cards {
			id := card.SeriesIDStr.String()
			if id == "" {
				id = card.SeriesID.String()
			}
			if id == "" || card.Title == "" {
				continue
			}
			page.Items = append(page.Items, model.SeriesSummary{
				ID: id, Title: card.Title, Description: card.Intro,
				CoverURL: card.Cover, EpisodeCount: card.EpisodeCount,
			})
		}
	}
	for _, book := range cell.Books {
		appendBook(page, book)
	}
	for _, nested := range cell.CellData {
		appendCatalogCell(page, nested)
	}
}

func appendBook(page *CatalogPage, book catalogBook) {
	id := book.BookID.String()
	title := strings.TrimSpace(book.Name)
	if id == "" || title == "" {
		return
	}
	page.Items = append(page.Items, model.SeriesSummary{
		ID: id, Title: title, Description: strings.TrimSpace(book.Abstract),
		CoverURL: book.Cover, EpisodeCount: int(book.SerialCount),
	})
}

// DiscoverContinuation reads the unlimited-feed handle from a bookmall tab response.
// The 推荐 tab (type 16) names that container and returns only its first cards.
func DiscoverContinuation(raw []byte) (FeedContinuation, bool) {
	return DiscoverContinuationTab(raw, 16)
}

// DiscoverContinuationTab reads the unlimited-feed handle for one bookmall tab.
func DiscoverContinuationTab(raw []byte, tabType int) (FeedContinuation, bool) {
	var env struct {
		Data struct {
			Tabs []struct {
				TabType  int           `json:"tab_type"`
				Session  string        `json:"session_id"`
				Template int           `json:"client_template"`
				Cells    []catalogCell `json:"cell_data"`
			} `json:"tab_item"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return FeedContinuation{}, false
	}
	var fallback FeedContinuation
	var haveFallback bool
	for _, tab := range env.Data.Tabs {
		if tab.TabType != tabType || tab.Session == "" {
			continue
		}
		for _, cell := range tab.Cells {
			id := cell.CellIDStr.String()
			if id == "" || cell.ShowType == 0 {
				continue
			}
			next := FeedContinuation{
				CellID: id, SessionID: tab.Session,
				ShowType: strconv.Itoa(cell.ShowType), TabType: strconv.Itoa(tab.TabType),
			}
			if tab.Template > 0 {
				next.ClientTemplate = strconv.Itoa(tab.Template)
			}
			// A cell with nested cards is the unlimited feed. A childless
			// cell is only used when the tab has no nested feed, which is
			// how 找剧 names its double-column page.
			if len(cell.CellData) > 0 {
				return next, true
			}
			if !haveFallback {
				fallback = next
				haveFallback = true
			}
		}
	}
	return fallback, haveFallback
}

// ParseCellChange reads the unlimited page from bookmall cell/change.
func ParseCellChange(raw []byte) (*CatalogPage, error) {
	var env struct {
		Code    flexString `json:"code"`
		Message string     `json:"message"`
		Data    struct {
			HasMore    bool        `json:"has_more"`
			NextOffset int         `json:"next_offset"`
			Session    string      `json:"session_id"`
			View       catalogCell `json:"cell_view"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("catalog code %s", env.Code.String())
	}
	page := &CatalogPage{
		Items: []model.SeriesSummary{}, HasMore: env.Data.HasMore,
		NextOffset: env.Data.NextOffset, SessionID: env.Data.Session,
	}
	appendCatalogCell(page, env.Data.View)
	return page, nil
}

// ParseSearch reads short-play search results (SearchTabType.ShortPlay, 11).
func ParseSearch(raw []byte) (*CatalogPage, error) {
	return ParseSearchTab(raw, 11)
}

// ParseSearchTab reads one search tab_type. Series cards keep series_title.
func ParseSearchTab(raw []byte, tabType int) (*CatalogPage, error) {
	var env struct {
		Code    flexString `json:"code"`
		Message string     `json:"message"`
		Tabs    []struct {
			TabType    int  `json:"tab_type"`
			HasMore    bool `json:"has_more"`
			NextOffset int  `json:"next_offset"`
			Data       []struct {
				VideoData videoList     `json:"video_data"`
				Books     bookList      `json:"book_data"`
				Users     []catalogUser `json:"celebrity_user_list"`
				Posts     postList      `json:"post_data"`
			} `json:"data"`
		} `json:"search_tabs"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("search code %s", env.Code.String())
	}
	page := &CatalogPage{Items: []model.SeriesSummary{}}
	for _, tab := range env.Tabs {
		if tab.TabType != tabType {
			continue
		}
		page.HasMore = tab.HasMore
		page.NextOffset = tab.NextOffset
		for _, row := range tab.Data {
			for _, v := range row.VideoData {
				// Series cards carry video_detail.series_title. Query and
				// discussion cells reuse title without that field.
				// The 视频 tab nests the series title under video_detail_list.
				title := strings.TrimSpace(v.VideoDetail.Title)
				id := v.SeriesID.String()
				if id != "" && title != "" {
					page.Items = append(page.Items, model.SeriesSummary{
						ID: id, Title: title, Description: v.VideoDetail.Intro, EpisodeCount: v.VideoDetail.EpisodeCount,
					})
					continue
				}
				for _, card := range v.VideoDetail.Cards {
					cid := card.SeriesIDStr.String()
					if cid == "" {
						cid = card.SeriesID.String()
					}
					ctitle := strings.TrimSpace(card.Title)
					if cid == "" || ctitle == "" {
						continue
					}
					page.Items = append(page.Items, model.SeriesSummary{
						ID: cid, Title: ctitle, Description: card.Intro,
						CoverURL: card.Cover, EpisodeCount: card.EpisodeCount,
					})
				}
			}
			for _, book := range row.Books {
				appendBook(page, book)
			}
			for _, user := range row.Users {
				id := user.UserID.String()
				name := strings.TrimSpace(user.Base.Name)
				if id == "" || name == "" {
					continue
				}
				page.Items = append(page.Items, model.SeriesSummary{ID: id, Title: name})
			}
			for _, post := range row.Posts {
				id := post.PostID.String()
				title := strings.TrimSpace(post.Title)
				if id == "" || title == "" {
					continue
				}
				page.Items = append(page.Items, model.SeriesSummary{ID: id, Title: title})
			}
		}
	}
	return page, nil
}

type wireEpisode struct {
	ID           flexString `json:"vid"`
	Number       int        `json:"vid_index"`
	Title        string     `json:"title"`
	Duration     int64      `json:"duration"`
	SeriesID     flexString `json:"series_id"`
	DisablePlay  bool       `json:"disable_play"`
	Vertical     bool       `json:"vertical"`
	IsNewlyUpd   bool       `json:"is_newly_update"`
	Cover        string     `json:"cover"`
	EpisodeCover string     `json:"episode_cover"`
	ContentType  int        `json:"content_type"`
	CommentCount int64      `json:"comment_count"`
	DiggCount    int64      `json:"digged_count"`
}

// wireSeriesPayload is one series of a detail response: the series and the
// novel it is adapted from.
type wireSeriesPayload struct {
	VideoData  wireSeriesData  `json:"video_data"`
	RelateBook *wireRelateBook `json:"video_relate_book"`
}

type wireSeriesData struct {
	ID          flexString    `json:"series_id_str"`
	NumericID   flexString    `json:"series_id"`
	Title       string        `json:"series_title"`
	Intro       string        `json:"series_intro"`
	Cover       string        `json:"series_cover"`
	Count       int           `json:"episode_cnt"`
	List        []wireEpisode `json:"video_list"`
	Status      int           `json:"series_status"`
	MultiSeason bool          `json:"is_multi_season"`
	SeasonIndex int           `json:"season_index"`
	AlbumID     flexString    `json:"related_album_id"`
	NovelID     flexString    `json:"related_novel_book_id"`
	SubTitles   []string      `json:"series_sub_title_list"`
	RightText   string        `json:"episode_right_text"`
	CreateTime  int64         `json:"create_time"`
	HotScore    int64         `json:"hot_score"`
	PlayCount   int64         `json:"series_play_cnt"`
	FollowCnt   int64         `json:"followed_cnt"`
	DiggCnt     int64         `json:"digg_cnt"`
	ContentTyp  int           `json:"content_type"`
	Platform    int           `json:"video_platform"`
	PayInfo     struct {
		PayType int `json:"pay_type"`
	} `json:"pay_info"`
	AgeGate struct {
		AgeGate int `json:"age_gate"`
	} `json:"age_gate_info"`
	SecondaryInfos []struct {
		Content  string `json:"content"`
		DataType int    `json:"data_type"`
		Schema   string `json:"schema"`
	} `json:"secondary_infos"`
}

// wireRelateBook is the source novel of a series (video_relate_book).
type wireRelateBook struct {
	ShowAt []int `json:"ShowInEpisodeIndex"`
	Book   struct {
		ID        flexString `json:"book_id"`
		Name      string     `json:"book_name"`
		ShortName string     `json:"book_short_name"`
		Status    flexString `json:"creation_status"`
		Reads     flexInt64  `json:"read_count"`
		Tags      string     `json:"tags"`
		Cover     string     `json:"thumb_url"`
	} `json:"book_info"`
}

// ParseSeries reads a video_detail response into the series and its episode list.
// Episode numbers shown to callers start at 1.
func ParseSeries(raw []byte) (*model.Series, []model.Episode, error) {
	var env struct {
		Code    flexString        `json:"code"`
		Message string            `json:"message"`
		Data    wireSeriesPayload `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, nil, fmt.Errorf("series code %s", env.Code.String())
	}
	return parseSeriesPayload(env.Data)
}

// ParseSeriesBatch reads a multi_video_detail response: the requested series
// keyed by ID. The service leaves out IDs it does not know.
func ParseSeriesBatch(raw []byte) (map[string]model.SeriesDetail, error) {
	var env struct {
		Code flexString                   `json:"code"`
		Data map[string]wireSeriesPayload `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("series code %s", env.Code.String())
	}
	out := make(map[string]model.SeriesDetail, len(env.Data))
	for key, payload := range env.Data {
		series, episodes, err := parseSeriesPayload(payload)
		if err != nil {
			return nil, fmt.Errorf("series %s: %w", key, err)
		}
		out[series.ID] = model.SeriesDetail{Series: series, Episodes: episodes}
	}
	return out, nil
}

// parseSeriesPayload reads one series and its episodes.
func parseSeriesPayload(payload wireSeriesPayload) (*model.Series, []model.Episode, error) {
	v := payload.VideoData
	id := v.ID.String()
	if id == "" {
		id = v.NumericID.String()
	}
	if id == "" || v.Title == "" {
		return nil, nil, fmt.Errorf("series response missing id or title")
	}
	series := &model.Series{SeriesSummary: model.SeriesSummary{
		ID: id, Title: v.Title, Description: v.Intro, CoverURL: v.Cover, EpisodeCount: v.Count,
		CoverExpiresAt: URLLifetime(v.Cover),
	}}
	series.Status = model.SeriesStatus(v.Status)
	series.PayType = model.PayType(v.PayInfo.PayType)
	series.ContentType = model.ContentType(v.ContentTyp)
	series.Platform = model.PlatformType(v.Platform)
	series.Heat = v.HotScore
	series.PlayCount = v.PlayCount
	series.FollowCount = v.FollowCnt
	series.DiggCount = v.DiggCnt
	series.RightText = strings.TrimSpace(v.RightText)
	series.Subtitles = trimAll(v.SubTitles)
	series.AgeGate = v.AgeGate.AgeGate
	series.IsMultiSeason = v.MultiSeason
	series.SeasonIndex = v.SeasonIndex
	series.RelatedAlbumID = v.AlbumID.String()
	series.RelatedNovelBookID = v.NovelID.String()
	if b := payload.RelateBook; b != nil && b.Book.ID.String() != "" && b.Book.ID.String() != "0" {
		novel := &model.Novel{
			BookID: b.Book.ID.String(), Title: strings.TrimSpace(b.Book.Name), ShortTitle: strings.TrimSpace(b.Book.ShortName),
			Status: model.NovelStatus(-1), Reads: int64(b.Book.Reads),
			CoverURL: b.Book.Cover, CoverExpiresAt: URLLifetime(b.Book.Cover),
		}
		if n, err := strconv.Atoi(b.Book.Status.String()); err == nil {
			novel.Status = model.NovelStatus(n)
		}
		for _, t := range strings.Split(b.Book.Tags, ",") {
			if t = strings.TrimSpace(t); t != "" {
				novel.Tags = append(novel.Tags, t)
			}
		}
		if len(b.ShowAt) > 0 {
			novel.ShowAtEpisode = b.ShowAt[0]
		}
		series.Novel = novel
	}
	if v.CreateTime > 0 {
		series.CreatedAt = time.Unix(v.CreateTime, 0)
	}
	rows := make([]model.InfoRow, 0, len(v.SecondaryInfos))
	for _, row := range v.SecondaryInfos {
		text := strings.TrimSpace(row.Content)
		if text == "" {
			continue
		}
		rows = append(rows, model.InfoRow{
			Kind: model.InfoKind(row.DataType), Text: text, Schema: strings.TrimSpace(row.Schema),
		})
	}
	series.Rows = rows
	if series.EpisodeCount == 0 {
		series.EpisodeCount = len(v.List)
	}
	shift := 0
	if len(v.List) > 0 {
		min := v.List[0].Number
		for _, e := range v.List[1:] {
			if e.Number < min {
				min = e.Number
			}
		}
		if min == 0 {
			shift = 1
		}
	}
	shared, sharedTitle := "", len(v.List) > 1
	for i, e := range v.List {
		title := strings.TrimSpace(e.Title)
		if i == 0 {
			shared = title
			continue
		}
		if title != shared {
			sharedTitle = false
			break
		}
	}
	episodes := make([]model.Episode, 0, len(v.List))
	for _, e := range v.List {
		eid := e.ID.String()
		if eid == "" {
			return nil, nil, fmt.Errorf("malformed episode in series %s", id)
		}
		number := e.Number + shift
		if number < 1 {
			return nil, nil, fmt.Errorf("episode number is not 1-based")
		}
		seriesID := e.SeriesID.String()
		if seriesID == "" {
			seriesID = id
		}
		title := strings.TrimSpace(e.Title)
		// The detail response copies the series synopsis into every episode title.
		// The app labels those rows by episode number.
		if title == "" || title == strings.TrimSpace(v.Intro) || (sharedTitle && title == shared) {
			title = fmt.Sprintf("第%d集", number)
		}
		cover := firstNonEmpty(e.EpisodeCover, e.Cover)
		episodes = append(episodes, model.Episode{
			ID: eid, SeriesID: seriesID, Number: number, Title: title,
			Duration:       time.Duration(e.Duration) * time.Second,
			Playable:       !e.DisablePlay,
			Vertical:       e.Vertical,
			IsNew:          e.IsNewlyUpd,
			CoverURL:       cover,
			CoverExpiresAt: URLLifetime(cover),
			ContentType:    model.ContentType(e.ContentType),
			CommentCount:   e.CommentCount,
			DiggCount:      e.DiggCount,
		})
	}
	if len(episodes) > 0 && episodes[0].Number != 1 && shift == 0 && episodes[0].Number > 1 {
		// A page that starts later keeps the wire index. The first full list starts at 1.
	}
	return series, episodes, nil
}

// ParsePlayback reads a video_model response into a media model.
//
// TTVideoEngine 模型位于旧响应的 data.video_model，或 V5 响应的
// data["1004"][episodeID].video_model / data[episodeID].video_model。
// A representation is marked protected when it carries encrypt_info,
// gear_des_key, spade_a, kid, check_info, encryption_method, or encrypt=true.
// Keys and per-segment check data are used only to set Protected; they are
// never copied into the returned model.
func ParsePlayback(raw []byte, episodeID string) (*model.Playback, error) {
	var env struct {
		Code    flexString                 `json:"code"`
		Message string                     `json:"message"`
		Data    map[string]json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("playback code %s", env.Code.String())
	}
	entry := env.Data
	if group, ok := entry["1004"]; ok {
		var grouped map[string]json.RawMessage
		if err := json.Unmarshal(group, &grouped); err != nil {
			return nil, fmt.Errorf("playback group is not an object: %w", err)
		}
		entry = grouped
	} else if group, ok := entry["video_model_data"]; ok {
		var grouped map[string]json.RawMessage
		if err := json.Unmarshal(group, &grouped); err != nil {
			return nil, fmt.Errorf("playback video_model_data is not an object: %w", err)
		}
		entry = grouped
	}
	if _, ok := entry["video_model"]; !ok {
		row, ok := entry[episodeID]
		if !ok {
			return nil, fmt.Errorf("playback model missing for requested episode")
		}
		var selected map[string]json.RawMessage
		if err := json.Unmarshal(row, &selected); err != nil {
			return nil, fmt.Errorf("playback entry is not an object: %w", err)
		}
		entry = selected
	}
	var envelopeExpiry int64
	for _, fields := range []map[string]json.RawMessage{entry, env.Data} {
		for _, name := range []string{"url_expire", "expire_time"} {
			if value, ok := fields[name]; ok {
				var expiry flexInt64
				if err := json.Unmarshal(value, &expiry); err != nil {
					return nil, err
				}
				if envelopeExpiry == 0 && expiry > 0 {
					envelopeExpiry = int64(expiry)
				}
			}
		}
	}
	body, err := playbackObject(entry["video_model"])
	if err != nil {
		return nil, err
	}
	var video struct {
		Status     flexInt         `json:"status"`
		VideoID    flexString      `json:"video_id"`
		MediaType  string          `json:"media_type"`
		Duration   float64         `json:"video_duration"`
		URLExpire  flexInt64       `json:"url_expire"`
		ExpireTime flexInt64       `json:"expire_time"`
		KeySeed    string          `json:"key_seed"`
		VideoList  json.RawMessage `json:"video_list"`
		Dynamic    json.RawMessage `json:"dynamic_video"`
	}
	if err := json.Unmarshal(body, &video); err != nil {
		return nil, fmt.Errorf("playback model is not JSON: %w", err)
	}
	items := []playbackItem{}
	if list := bytes.TrimSpace(video.VideoList); len(list) > 0 && string(list) != "null" && string(list) != "[]" {
		items, err = playbackItems(video.VideoList)
		if err != nil {
			return nil, err
		}
	}
	audioURL := ""
	if len(items) == 0 {
		// Full-length titles are delivered as a segmented (DASH) model: a set
		// of video renditions plus a separate audio track, with no video_list.
		var dynamic struct {
			VideoList []json.RawMessage `json:"dynamic_video_list"`
			AudioList []json.RawMessage `json:"dynamic_audio_list"`
		}
		if raw := nonEmptyJSON(video.Dynamic); len(raw) > 0 {
			if err := json.Unmarshal(raw, &dynamic); err != nil {
				return nil, fmt.Errorf("playback dynamic model is not JSON: %w", err)
			}
		}
		items, err = playbackItemsList(dynamic.VideoList)
		if err != nil {
			return nil, err
		}
		if len(dynamic.AudioList) > 0 {
			var track playbackItem
			if err := json.Unmarshal(dynamic.AudioList[0], &track); err == nil {
				audioURL, _ = playbackURL(track.MainURL, video.KeySeed)
			}
		}
	}
	out := &model.Playback{
		EpisodeID:       episodeID,
		VideoID:         video.VideoID.String(),
		MediaType:       strings.TrimSpace(video.MediaType),
		DurationSeconds: int(video.Duration + 0.5),
		ExpiresAt:       firstInt64(int64(video.URLExpire), int64(video.ExpireTime), envelopeExpiry),
		Streams:         make([]model.Stream, 0, len(items)),
	}
	protected := 0
	for _, item := range items {
		stream := item.stream(video.KeySeed)
		if stream.URL == "" {
			continue
		}
		if stream.Protected {
			protected++
		}
		if out.KeyMaterial == "" {
			out.KeyMaterial = item.wrappedKey()
		}
		out.Streams = append(out.Streams, stream)
	}
	if len(out.Streams) == 0 {
		return nil, fmt.Errorf("playback model unavailable")
	}
	out.AudioURL = audioURL
	out.Protected = protected == len(out.Streams)
	return out, nil
}

// playbackObject unwraps data.video_model. The service sends a JSON string;
// an object is accepted so fixtures and older responses parse the same way.
func playbackObject(raw json.RawMessage) ([]byte, error) {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 || string(b) == "null" {
		return nil, fmt.Errorf("playback model missing")
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return nil, fmt.Errorf("playback model is not a string: %w", err)
		}
		b = bytes.TrimSpace([]byte(s))
	}
	if len(b) == 0 || b[0] != '{' {
		return nil, fmt.Errorf("playback model is not an object")
	}
	return b, nil
}

// playbackItemsList parses a pre-decoded list of representation objects.
func playbackItemsList(raw []json.RawMessage) ([]playbackItem, error) {
	items := make([]playbackItem, 0, len(raw))
	for _, entry := range raw {
		var item playbackItem
		if err := json.Unmarshal(entry, &item); err != nil {
			return nil, fmt.Errorf("playback representation is not JSON: %w", err)
		}
		items = append(items, item)
	}
	return items, nil
}

// playbackItem is one representation. Resolution fields may sit on the item or
// inside video_meta; the shipped model uses video_meta.
type playbackItem struct {
	MainURL     string          `json:"main_url"`
	BackupURL   string          `json:"backup_url"`
	BackupURL1  string          `json:"backup_url_1"`
	BackupURL2  string          `json:"backup_url_2"`
	HLS         json.RawMessage `json:"hls"`
	M3U8        string          `json:"m3u8"`
	Encrypt     json.RawMessage `json:"encrypt"`
	EncryptInfo json.RawMessage `json:"encrypt_info"`
	GearDESKey  string          `json:"gear_des_key"`
	SpadeA      string          `json:"spade_a"`
	Kid         string          `json:"kid"`
	CheckInfo   json.RawMessage `json:"check_info"`
	Encryption  string          `json:"encryption_method"`
	Definition  string          `json:"definition"`
	Quality     string          `json:"quality"`
	QualityDesc string          `json:"quality_desc"`
	Codec       string          `json:"codec_type"`
	Width       int             `json:"vwidth"`
	Height      int             `json:"vheight"`
	Bitrate     int64           `json:"bitrate"`
	Size        int64           `json:"size"`
	VType       string          `json:"vtype"`
	Meta        struct {
		Definition  string `json:"definition"`
		Quality     string `json:"quality"`
		QualityDesc string `json:"quality_desc"`
		Codec       string `json:"codec_type"`
		Width       int    `json:"vwidth"`
		Height      int    `json:"vheight"`
		Bitrate     int64  `json:"bitrate"`
		Size        int64  `json:"size"`
		VType       string `json:"vtype"`
	} `json:"video_meta"`
}

func (it playbackItem) stream(seed string) model.Stream {
	s := model.Stream{
		Definition: firstNonEmpty(it.Meta.Definition, it.Definition),
		Quality:    firstNonEmpty(it.Meta.Quality, it.Quality),
		Codec:      firstNonEmpty(it.Meta.Codec, it.Codec),
		Width:      firstInt(it.Meta.Width, it.Width),
		Height:     firstInt(it.Meta.Height, it.Height),
		Bitrate:    firstInt64(it.Meta.Bitrate, it.Bitrate),
		Size:       firstInt64(it.Meta.Size, it.Size),
		HLS:        jsonTruthy(it.HLS) || strings.TrimSpace(it.M3U8) != "",
		Dash:       strings.EqualFold(it.Meta.VType, "dash") || strings.EqualFold(it.VType, "dash"),
	}
	for _, value := range []string{it.MainURL, it.BackupURL, it.BackupURL1, it.BackupURL2} {
		decoded, err := playbackURL(value, seed)
		if err != nil {
			continue
		}
		if s.URL == "" {
			s.URL = decoded
		} else if decoded != s.URL {
			s.BackupURL = decoded
			break
		}
	}
	protected, method := it.protection()
	s.Protected = protected
	s.Encryption = method
	if s.Quality == "" && it.QualityDesc != "" {
		s.Quality = it.QualityDesc
	}
	if s.Definition == "" && strings.EqualFold(it.VType, "m3u8") {
		s.HLS = true
	}
	return s
}

// wrappedKey returns the opaque wrapped content key the player consumes. It is
// kept out of Stream so a representation can never carry key material.
func (it playbackItem) wrappedKey() string {
	if raw := nonEmptyJSON(it.EncryptInfo); len(raw) > 0 && raw[0] == '{' {
		var info struct {
			SpadeA string `json:"spade_a"`
		}
		if err := json.Unmarshal(raw, &info); err == nil {
			if k := strings.TrimSpace(info.SpadeA); k != "" {
				return k
			}
		}
	}
	return strings.TrimSpace(it.SpadeA)
}

// protection reads the shipped client's encryption markers.
// It returns only the scheme name, never key material.
func (it playbackItem) protection() (bool, string) {
	method := strings.TrimSpace(it.Encryption)
	if raw := nonEmptyJSON(it.EncryptInfo); len(raw) > 0 && raw[0] == '{' {
		var info struct {
			Encrypt bool   `json:"encrypt"`
			Method  string `json:"encryption_method"`
			Kid     string `json:"kid"`
			SpadeA  string `json:"spade_a"`
		}
		if err := json.Unmarshal(raw, &info); err == nil {
			if method == "" {
				method = strings.TrimSpace(info.Method)
			}
			if info.Encrypt || method != "" || info.Kid != "" || info.SpadeA != "" {
				return true, method
			}
		}
	} else if len(raw) > 0 {
		// A non-object marker such as "protected" still marks the stream.
		return true, method
	}
	if it.GearDESKey != "" || it.SpadeA != "" || it.Kid != "" || method != "" {
		return true, method
	}
	if jsonTruthy(it.Encrypt) {
		return true, method
	}
	if len(nonEmptyJSON(it.CheckInfo)) > 0 {
		return true, method
	}
	return false, method
}

// playbackItems accepts video_list as an array or as an object keyed by
// definition. Object keys are sorted so output order is stable.
func playbackItems(raw json.RawMessage) ([]playbackItem, error) {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 || string(b) == "null" {
		return nil, fmt.Errorf("playback model unavailable")
	}
	var asList []playbackItem
	if err := json.Unmarshal(b, &asList); err == nil {
		if len(asList) == 0 {
			return nil, fmt.Errorf("playback model unavailable")
		}
		return asList, nil
	}
	var asMap map[string]playbackItem
	if err := json.Unmarshal(b, &asMap); err != nil {
		return nil, fmt.Errorf("playback model video_list is invalid: %w", err)
	}
	if len(asMap) == 0 {
		return nil, fmt.Errorf("playback model unavailable")
	}
	keys := make([]string, 0, len(asMap))
	for k := range asMap {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]playbackItem, 0, len(keys))
	for _, k := range keys {
		item := asMap[k]
		if item.Definition == "" && item.Meta.Definition == "" {
			item.Definition = k
		}
		out = append(out, item)
	}
	return out, nil
}

// nonEmptyJSON trims a raw JSON value and rejects null and empty containers.
func nonEmptyJSON(raw json.RawMessage) json.RawMessage {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 || string(b) == "null" || string(b) == `""` || string(b) == "{}" || string(b) == "[]" {
		return nil
	}
	return b
}

// jsonTruthy reads the boolean, numeric, and string spellings the model uses.
func jsonTruthy(raw json.RawMessage) bool {
	b := nonEmptyJSON(raw)
	if len(b) == 0 {
		return false
	}
	switch strings.ToLower(string(b)) {
	case "true", "1":
		return true
	case "false", "0":
		return false
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err == nil {
			return strings.TrimSpace(s) != ""
		}
	}
	if b[0] == '{' || b[0] == '[' {
		return true
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

func firstInt(values ...int) int {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}

func firstInt64(values ...int64) int64 {
	for _, v := range values {
		if v != 0 {
			return v
		}
	}
	return 0
}

// BusinessCode returns the envelope code when the body is JSON.
func BusinessCode(raw []byte) (code string, message string, ok bool) {
	var env struct {
		Code    json.RawMessage `json:"code"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(raw, &env); err != nil || len(env.Code) == 0 {
		return "", "", false
	}
	var asString string
	if err := json.Unmarshal(env.Code, &asString); err == nil {
		return asString, env.Message, true
	}
	var asNumber json.Number
	dec := json.NewDecoder(bytes.NewReader(env.Code))
	dec.UseNumber()
	if err := dec.Decode(&asNumber); err == nil {
		return asNumber.String(), env.Message, true
	}
	return strings.Trim(string(env.Code), `"`), env.Message, true
}

// AuthCode reports whether a business code is an account-required failure.
func AuthCode(code, message string) bool {
	switch code {
	case "auth_required", "401", "403":
		return true
	}
	if n, err := strconv.Atoi(code); err == nil && (n == 401 || n == 403) {
		return true
	}
	lower := strings.ToLower(message)
	return strings.Contains(message, "登录") || strings.Contains(lower, "login required") || strings.Contains(lower, "authentication required")
}

// trimAll trims every string and drops the empty ones, keeping order.
func trimAll(in []string) []string {
	out := make([]string, 0, len(in))
	for _, s := range in {
		if t := strings.TrimSpace(s); t != "" {
			out = append(out, t)
		}
	}
	return out
}

// ParseAlbum reads an album_detail response into the album and its seasons.
//
// The shipped client calls /novel/player/album_detail/v1/ for a collection
// whose detail response carries related_album_id. The response shape is
// observed on the wire; only the fields a client needs are read.
func ParseAlbum(raw []byte, albumID string) (*model.Album, error) {
	var env struct {
		Code    flexString `json:"code"`
		Message string     `json:"message"`
		Data    struct {
			AlbumData json.RawMessage `json:"album_data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		msg := strings.TrimSpace(env.Message)
		if msg == "" {
			msg = "album unavailable"
		}
		return nil, fmt.Errorf("album unavailable: %s (%s)", msg, env.Code.String())
	}
	body, err := playbackObject(env.Data.AlbumData)
	if err != nil {
		return nil, err
	}
	var album struct {
		ID       flexString      `json:"album_id_str"`
		Numeric  flexString      `json:"album_id"`
		Title    string          `json:"album_title"`
		Cover    string          `json:"album_cover"`
		Intro    string          `json:"album_intro"`
		SeasonNm flexInt         `json:"season_cnt"`
		List     json.RawMessage `json:"video_list"`
		Series   json.RawMessage `json:"series_list"`
	}
	if err := json.Unmarshal(body, &album); err != nil {
		return nil, fmt.Errorf("album model is not JSON: %w", err)
	}
	id := album.ID.String()
	if id == "" {
		id = album.Numeric.String()
	}
	if id == "" {
		id = albumID
	}
	out := &model.Album{
		ID: id, Title: album.Title, Description: album.Intro, CoverURL: album.Cover,
		CoverExpiresAt: URLLifetime(album.Cover), SeasonCount: int(album.SeasonNm),
	}
	entries := album.Series
	if len(bytes.TrimSpace(entries)) == 0 || string(bytes.TrimSpace(entries)) == "null" {
		entries = album.List
	}
	var rows []struct {
		ID     flexString `json:"series_id_str"`
		Num    flexString `json:"series_id"`
		Title  string     `json:"series_title"`
		Index  int        `json:"season_index"`
		Cover  string     `json:"series_cover"`
		Count  int        `json:"episode_cnt"`
		Status int        `json:"series_status"`
	}
	if len(bytes.TrimSpace(entries)) > 0 && string(bytes.TrimSpace(entries)) != "null" {
		if err := json.Unmarshal(entries, &rows); err != nil {
			return nil, fmt.Errorf("album season list is invalid: %w", err)
		}
	}
	seasons := make([]model.Season, 0, len(rows))
	for _, r := range rows {
		sid := r.ID.String()
		if sid == "" {
			sid = r.Num.String()
		}
		seasons = append(seasons, model.Season{
			SeriesID: sid, Title: r.Title, Index: r.Index,
			CoverURL: r.Cover, EpisodeCount: r.Count, Status: model.SeriesStatus(r.Status),
		})
	}
	out.Seasons = seasons
	if out.SeasonCount == 0 {
		out.SeasonCount = len(seasons)
	}
	return out, nil
}

// ParseRankRows reads the ranked rows of a bookmall ranking cell.
//
// The rank of a row is the page-local index the service returns in
// recommend_info.rank; it restarts at 1 on the next page, so callers show it
// as-is instead of computing a global position.
func ParseRankRows(raw []byte, offset int) (*model.Ranking, error) {
	var env struct {
		Code flexString `json:"code"`
		Data struct {
			HasMore   bool   `json:"has_more"`
			NextOff   int    `json:"next_offset"`
			SessionID string `json:"session_id"`
			CellView  struct {
				BookData []struct {
					ID     flexString      `json:"book_id"`
					Title  string          `json:"book_name"`
					Author string          `json:"author"`
					Cover  string          `json:"thumb_url"`
					Score  flexString      `json:"rank_score"`
					Rec    json.RawMessage `json:"recommend_info"`
				} `json:"book_data"`
			} `json:"cell_view"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("rank cells code %s", env.Code.String())
	}
	out := &model.Ranking{HasMore: env.Data.HasMore, NextOffset: env.Data.NextOff, SessionID: env.Data.SessionID,
		Items: []model.RankItem{}}
	for _, b := range env.Data.CellView.BookData {
		id := b.ID.String()
		if id == "" || strings.TrimSpace(b.Title) == "" {
			continue
		}
		out.Items = append(out.Items, model.RankItem{
			Rank: recommendRank(b.Rec), Title: b.Title, Actor: b.Author,
			SeriesID: id, CoverURL: b.Cover, Score: b.Score.String(),
		})
	}
	return out, nil
}

// recommendRank reads the page-local rank out of recommend_info, which the
// service sends either as an object or as a JSON string holding one.
func recommendRank(raw json.RawMessage) int {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 || string(b) == "null" {
		return 0
	}
	if b[0] == '"' {
		var inner string
		if err := json.Unmarshal(b, &inner); err != nil {
			return 0
		}
		b = bytes.TrimSpace([]byte(inner))
	}
	var rec struct {
		Rank flexInt `json:"rank"`
	}
	if err := json.Unmarshal(b, &rec); err != nil {
		return 0
	}
	return int(rec.Rank)
}

// ParseRanking reads an activity ranking response.
func ParseRanking(raw []byte) (*model.Ranking, error) {
	var env struct {
		Code flexString `json:"code"`
		Data struct {
			ActivityBasic struct {
				AllRankListBasic []struct {
					ID     string `json:"id"`
					Name   string `json:"name"`
					Type   int    `json:"type"`
					Number int    `json:"shortlisted_number"`
				} `json:"all_rank_list_basic"`
				ActivityStage int `json:"activity_stage"`
			} `json:"activity_basic"`
			RankList struct {
				RankListShortInfo struct {
					ID   string `json:"id"`
					Name string `json:"name"`
					Type int    `json:"type"`
				} `json:"rank_list_short_info"`
				Items []struct {
					RankItemBasic struct {
						ID         string `json:"id"`
						RankListID string `json:"rank_list_id"`
						Index      int    `json:"index"`
						VoteCount  int64  `json:"vote_count"`
					} `json:"rank_item_basic"`
					Type      int `json:"type"`
					ActorInfo struct {
						Name   string `json:"name"`
						Avatar string `json:"avatar"`
					} `json:"actor_info"`
					PlayletInfo struct {
						ID    string `json:"id"`
						Title string `json:"title"`
						Cover string `json:"cover"`
					} `json:"playlet_info"`
				} `json:"items"`
				TurnPageInfo struct {
					HasMore    bool `json:"has_more"`
					NextOffset int  `json:"next_offset"`
				} `json:"turn_page_info"`
			} `json:"rank_list"`
			CanVote bool `json:"can_vote"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("ranking code %s", env.Code.String())
	}
	out := &model.Ranking{
		Items:         []model.RankItem{},
		ActivityStage: env.Data.ActivityBasic.ActivityStage,
		CanVote:       env.Data.CanVote,
		ListName:      env.Data.RankList.RankListShortInfo.Name,
		HasMore:       env.Data.RankList.TurnPageInfo.HasMore,
		NextOffset:    env.Data.RankList.TurnPageInfo.NextOffset,
	}
	for _, r := range env.Data.ActivityBasic.AllRankListBasic {
		out.Lists = append(out.Lists, model.RankList{ID: r.ID, Name: r.Name, Type: r.Type, Size: r.Number})
	}
	for _, it := range env.Data.RankList.Items {
		entry := model.RankItem{
			Rank: it.RankItemBasic.Index, Votes: it.RankItemBasic.VoteCount,
			Kind: it.Type, Actor: it.ActorInfo.Name,
		}
		if it.PlayletInfo.Title != "" {
			entry.SeriesID = it.PlayletInfo.ID
			entry.Title = it.PlayletInfo.Title
			entry.CoverURL = it.PlayletInfo.Cover
		}
		if entry.Title == "" {
			entry.Title = it.ActorInfo.Name
		}
		out.Items = append(out.Items, entry)
	}
	return out, nil
}

// ParsePlanCell reads a "plan" feed page and returns the cells. The related
// video surface is one cell whose show_type is 160 (ShowType.DetailRelatedVideo;
// 58 is only the Kotlin ordinal).
func ParsePlanCell(raw []byte) ([]PlanCell, error) {
	var env struct {
		Code      flexString `json:"code"`
		SessionID string     `json:"session_id"`
		Cells     []struct {
			ShowType  int             `json:"show_type"`
			CellID    flexString      `json:"cell_id"`
			CellName  string          `json:"cell_name"`
			SessionID flexString      `json:"session_id"`
			CellData  json.RawMessage `json:"cell_data"`
			VideoData json.RawMessage `json:"video_data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("plan code %s", env.Code.String())
	}
	out := make([]PlanCell, 0, len(env.Cells))
	for _, c := range env.Cells {
		cell := PlanCell{
			ShowType: c.ShowType, CellID: c.CellID.String(), CellName: c.CellName,
			SessionID: firstNonEmpty(c.SessionID.String(), env.SessionID),
		}
		cell.Items = decodeSeriesList(c.VideoData)
		if len(cell.Items) == 0 {
			cell.Items = decodeSeriesList(c.CellData)
		}
		out = append(out, cell)
	}
	return out, nil
}

// ShowTypeSeriesSeasons is the plan cell type of the 系列剧 row (observed
// 2026-10-24).
const ShowTypeSeriesSeasons = 547

// ParseSeasons reads the 系列剧 cell of a plan response: every season of the
// series, the requested one included, with its season index. A season that is
// announced but not out yet has no episodes.
func ParseSeasons(raw []byte) ([]model.Season, error) {
	var env struct {
		Code  flexString `json:"code"`
		Cells []struct {
			ShowType int `json:"show_type"`
			Rows     []struct {
				ID    flexString `json:"series_id"`
				Title string     `json:"title"`
				Cover string     `json:"cover"`
				Count int        `json:"episode_cnt"`
				Index int        `json:"season_index"`
			} `json:"video_data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("plan code %s", env.Code.String())
	}
	var out []model.Season
	for _, c := range env.Cells {
		if c.ShowType != ShowTypeSeriesSeasons {
			continue
		}
		for _, r := range c.Rows {
			if r.ID.String() == "" {
				continue
			}
			out = append(out, model.Season{
				SeriesID: r.ID.String(), Title: r.Title, Index: r.Index,
				CoverURL: r.Cover, EpisodeCount: r.Count,
			})
		}
	}
	return out, nil
}

// ParseHotSearch reads the search page's plan response. The rows sit in a
// Lynx cell whose lynx_data is a JSON string of cells: a row cell (猜你想搜)
// with video_data, and a cell whose cell_data holds the hot lists.
func ParseHotSearch(raw []byte) (*model.HotSearch, error) {
	var env struct {
		Code  flexString `json:"code"`
		Cells []struct {
			LynxData json.RawMessage `json:"lynx_data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("plan code %s", env.Code.String())
	}
	type hotRow struct {
		ID      flexString `json:"series_id"`
		Title   string     `json:"title"`
		Cover   string     `json:"cover"`
		Count   flexInt    `json:"episode_cnt"`
		Type    flexInt    `json:"content_type"`
		Tips    string     `json:"card_tips"`
		RecText string     `json:"rec_text"`
		Tag     struct {
			Text string `json:"text"`
		} `json:"tag_info"`
	}
	type hotCell struct {
		Name   string   `json:"cell_name"`
		Rows   []hotRow `json:"video_data"`
		Topics []struct {
			ID    flexString `json:"topic_id"`
			Title string     `json:"title"`
			URL   string     `json:"url"`
			Info  []struct {
				Content string `json:"content"`
			} `json:"secondary_info_list"`
		} `json:"video_topic"`
		Boards []json.RawMessage `json:"cell_data"`
	}
	items := func(rows []hotRow) []model.HotItem {
		var out []model.HotItem
		for _, r := range rows {
			if r.ID.String() == "" {
				continue
			}
			out = append(out, model.HotItem{
				Position: len(out) + 1, SeriesID: r.ID.String(), Title: r.Title,
				CoverURL: r.Cover, CoverExpiresAt: URLLifetime(r.Cover), EpisodeCount: int(r.Count),
				ContentType: model.ContentType(r.Type), SearchScore: r.Tips, Heat: r.RecText, Tag: r.Tag.Text,
			})
		}
		return out
	}
	out := &model.HotSearch{}
	for _, c := range env.Cells {
		data := bytes.TrimSpace(c.LynxData)
		if len(data) == 0 || string(data) == "null" {
			continue
		}
		// lynx_data is a string holding JSON; accept it inline as well.
		if data[0] == '"' {
			var s string
			if err := json.Unmarshal(data, &s); err != nil {
				return nil, err
			}
			data = []byte(s)
		}
		var cells []hotCell
		if err := json.Unmarshal(data, &cells); err != nil {
			return nil, fmt.Errorf("lynx_data: %w", err)
		}
		for _, cell := range cells {
			if len(cell.Rows) > 0 {
				out.Guesses = append(out.Guesses, items(cell.Rows)...)
			}
			for _, rawBoard := range cell.Boards {
				var b hotCell
				if err := json.Unmarshal(rawBoard, &b); err != nil {
					return nil, err
				}
				board := model.HotBoard{Name: b.Name, Items: items(b.Rows)}
				for _, t := range b.Topics {
					topic := model.HotTopic{ID: t.ID.String(), Title: t.Title, Query: topicQuery(t.URL)}
					if len(t.Info) > 0 {
						topic.Heat = t.Info[0].Content
					}
					board.Topics = append(board.Topics, topic)
				}
				if len(board.Items) > 0 || len(board.Topics) > 0 {
					out.Boards = append(out.Boards, board)
				}
			}
		}
	}
	return out, nil
}

// topicQuery reads the search a topic opens from its dragon8662://search
// link (directQueryWord).
func topicQuery(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	return u.Query().Get("directQueryWord")
}

// NewReleaseResult is one page of the 上新 calendar with its paging state.
type NewReleaseResult struct {
	model.NewReleasePage
	NextOffset int
	SessionID  string
}

// ParseNewReleases reads a 上新 calendar page. A title that is out carries
// its row in subscribe_data; an announced one is the item itself, whose
// item_id is the series ID.
func ParseNewReleases(raw []byte) (*NewReleaseResult, error) {
	var env struct {
		Code flexString `json:"code"`
		Data struct {
			Items []struct {
				Online     bool       `json:"is_online"`
				ItemID     flexString `json:"item_id"`
				Name       string     `json:"name"`
				Desc       string     `json:"item_desc"`
				Cover      string     `json:"cover"`
				Expected   string     `json:"expected_rec_text"`
				SubInfo    string     `json:"sub_info"`
				Schedule   flexInt64  `json:"schedule_publish_time"`
				Planned    flexInt64  `json:"expected_publish_time"`
				Categories []string   `json:"categories"`
				RecTags    []struct {
					Content string `json:"content"`
				} `json:"rec_tags"`
				Row *struct {
					ID    flexString `json:"series_id"`
					Title string     `json:"title"`
					Desc  string     `json:"video_desc"`
					Cover string     `json:"cover"`
					Count flexInt    `json:"episode_cnt"`
				} `json:"subscribe_data"`
			} `json:"subscribe_items"`
			NextOffset int        `json:"next_offset"`
			HasMore    bool       `json:"has_more"`
			SessionID  flexString `json:"session_id"`
			Calendar   struct {
				Dates   []string `json:"date_list"`
				Default string   `json:"default_date"`
			} `json:"calendar_schema"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("subscribe list code %s", env.Code.String())
	}
	d := env.Data
	out := &NewReleaseResult{NextOffset: d.NextOffset, SessionID: d.SessionID.String()}
	out.Date, out.Dates, out.HasMore, out.Items = d.Calendar.Default, d.Calendar.Dates, d.HasMore, []model.NewRelease{}
	for _, it := range d.Items {
		r := model.NewRelease{Online: it.Online, Categories: it.Categories, Note: it.SubInfo}
		when := int64(it.Schedule)
		if when == 0 {
			when = int64(it.Planned)
		}
		if when > 0 {
			r.PublishAt = time.Unix(when, 0).UTC()
		}
		if it.Row != nil {
			r.SeriesID, r.Title, r.Description = it.Row.ID.String(), it.Row.Title, it.Row.Desc
			r.CoverURL, r.EpisodeCount = it.Row.Cover, int(it.Row.Count)
		} else {
			r.SeriesID, r.Title, r.Description, r.CoverURL = it.ItemID.String(), it.Name, it.Desc, it.Cover
		}
		if r.SeriesID == "" || r.SeriesID == "0" || strings.TrimSpace(r.Title) == "" {
			continue
		}
		r.CoverExpiresAt = URLLifetime(r.CoverURL)
		if it.Online && len(it.RecTags) > 0 {
			r.Heat = it.RecTags[0].Content
		} else if !it.Online {
			r.Heat = it.Expected
		}
		out.Items = append(out.Items, r)
	}
	return out, nil
}

// ParseProfile reads a person's page.
func ParseProfile(raw []byte) (*model.Person, error) {
	type label struct {
		Text string `json:"text"`
	}
	var env struct {
		Code flexString `json:"code"`
		Data struct {
			User struct {
				ID   flexString `json:"user_id"`
				Base struct {
					Name   string `json:"user_name"`
					Desc   string `json:"description"`
					Avatar string `json:"user_avatar"`
				} `json:"base_info"`
				Tag struct {
					Titles []label `json:"sub_title_list"`
				} `json:"user_tag"`
				Relation struct {
					Fans  flexInt64 `json:"fans_num"`
					Likes flexInt64 `json:"recv_digged_count"`
				} `json:"user_relation"`
				Expand struct {
					Labels []label `json:"sub_title_list"`
				} `json:"user_expand"`
			} `json:"ugc_user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("profile code %s", env.Code.String())
	}
	u := env.Data.User
	p := &model.Person{
		UserID: u.ID.String(), Name: strings.TrimSpace(u.Base.Name), Description: u.Base.Desc,
		AvatarURL: u.Base.Avatar, Followers: int64(u.Relation.Fans), Likes: int64(u.Relation.Likes),
	}
	for _, t := range u.Tag.Titles {
		if t.Text != "" {
			p.Tags = append(p.Tags, t.Text)
		}
	}
	for _, t := range u.Expand.Labels {
		if t.Text != "" {
			p.Labels = append(p.Labels, t.Text)
		}
	}
	return p, nil
}

// WorksResult is one page of a person's works and the service's cursor
// string for the next.
type WorksResult struct {
	Items   []model.Work
	HasMore bool
	Cursor  string
}

// worksTabName is the tab, or sub tab, of a person's page that lists the
// series they act in.
const worksTabName = "剧作"

// ParseWorksTab reads which tab of a person's page lists their works from
// the answer to UserWorksTabsBody. A sub tab named 剧作 wins (作品 › 剧作);
// otherwise a tab named 剧作 is used with its 全部 sub tab, or its first.
// ok is false when the page has no such tab.
func ParseWorksTab(raw []byte) (tab WorksTab, ok bool, err error) {
	type entry struct {
		Name string  `json:"name"`
		Type flexInt `json:"tab_type"`
		Subs []struct {
			Name string  `json:"name"`
			Type flexInt `json:"tab_type"`
		} `json:"sub_tabs"`
	}
	var env struct {
		Code flexString `json:"code"`
		Data struct {
			Tabs      []entry `json:"tabs"`
			Highlight *entry  `json:"hightlight"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return WorksTab{}, false, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return WorksTab{}, false, fmt.Errorf("works tabs code %s", env.Code.String())
	}
	tabs := env.Data.Tabs
	if len(tabs) == 0 && env.Data.Highlight != nil {
		tabs = []entry{*env.Data.Highlight}
	}
	for _, t := range tabs {
		for _, s := range t.Subs {
			if s.Name == worksTabName {
				return WorksTab{Tab: int(t.Type), Sub: int(s.Type)}, true, nil
			}
		}
	}
	for _, t := range tabs {
		if t.Name != worksTabName || len(t.Subs) == 0 {
			continue
		}
		sub := t.Subs[0]
		for _, s := range t.Subs {
			if s.Name == "全部" {
				sub = s
				break
			}
		}
		return WorksTab{Tab: int(t.Type), Sub: int(sub.Type)}, true, nil
	}
	return WorksTab{}, false, nil
}

// ParseUserWorks reads the 作品 tab of a person's page.
func ParseUserWorks(raw []byte) (*WorksResult, error) {
	var env struct {
		Code flexString `json:"code"`
		Data struct {
			List []struct {
				Video *struct {
					ID     flexString `json:"video_id"`
					Common struct {
						Title string `json:"title"`
						Intro string `json:"intro"`
						Cover string `json:"cover"`
					} `json:"common"`
					Stat struct {
						Hot   flexInt64 `json:"hot"`
						Plays flexInt64 `json:"play_cnt"`
					} `json:"stat"`
					Expand struct {
						Count  flexInt `json:"episode_cnt"`
						Titles []struct {
							Text string `json:"text"`
						} `json:"sub_title_list"`
					} `json:"expand"`
				} `json:"video"`
			} `json:"data_list"`
			Info struct {
				Cursor  string `json:"cursor"`
				HasMore bool   `json:"has_more"`
			} `json:"common_list_info"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("works code %s", env.Code.String())
	}
	out := &WorksResult{Items: []model.Work{}, HasMore: env.Data.Info.HasMore, Cursor: env.Data.Info.Cursor}
	for _, row := range env.Data.List {
		v := row.Video
		if v == nil || v.ID.String() == "" || strings.TrimSpace(v.Common.Title) == "" {
			continue
		}
		w := model.Work{
			SeriesID: v.ID.String(), Title: v.Common.Title, Description: v.Common.Intro,
			CoverURL: v.Common.Cover, CoverExpiresAt: URLLifetime(v.Common.Cover),
			EpisodeCount: int(v.Expand.Count), Heat: int64(v.Stat.Hot), Plays: int64(v.Stat.Plays),
		}
		for _, t := range v.Expand.Titles {
			if t.Text != "" {
				w.Tags = append(w.Tags, t.Text)
			}
		}
		out.Items = append(out.Items, w)
	}
	return out, nil
}

// PlanCell is one cell of a plan feed.
type PlanCell struct {
	ShowType  int
	CellID    string
	CellName  string
	SessionID string
	Items     []model.SeriesSummary
}

// decodeSeriesList reads a list of series rows from a cell payload.
func decodeSeriesList(raw json.RawMessage) []model.SeriesSummary {
	b := bytes.TrimSpace(raw)
	if len(b) == 0 || string(b) == "null" || b[0] != '[' {
		return nil
	}
	var rows []struct {
		ID      flexString `json:"series_id_str"`
		Numeric flexString `json:"series_id"`
		Vid     flexString `json:"vid"`
		Title   string     `json:"series_title"`
		RawTtl  string     `json:"title"`
		Cover   string     `json:"series_cover"`
		RawCvr  string     `json:"cover"`
		Intro   string     `json:"series_intro"`
		Count   int        `json:"episode_cnt"`
		Status  int        `json:"series_status"`
	}
	if err := json.Unmarshal(b, &rows); err != nil {
		return nil
	}
	out := make([]model.SeriesSummary, 0, len(rows))
	for _, r := range rows {
		id := r.ID.String()
		if id == "" {
			id = r.Numeric.String()
		}
		if id == "" {
			id = r.Vid.String()
		}
		title := firstNonEmpty(r.Title, r.RawTtl)
		if id == "" || strings.TrimSpace(title) == "" {
			continue
		}
		cover := firstNonEmpty(r.Cover, r.RawCvr)
		out = append(out, model.SeriesSummary{
			ID: id, Title: title, Description: oneLineText(r.Intro),
			CoverURL: cover, CoverExpiresAt: URLLifetime(cover), EpisodeCount: r.Count,
		})
	}
	return out
}

// ShortFilterResult is one answer of the short-drama filter page: its
// selector rows (a panel request) or matching series rows (a content request).
type ShortFilterResult struct {
	Rows       []model.FilterRow
	Items      []model.SeriesSummary
	NextOffset int
	HasMore    bool
	SessionID  string
}

// ParseShortFilter reads /reading/distribution/category/landpage/.
func ParseShortFilter(raw []byte) (*ShortFilterResult, error) {
	var env struct {
		Code flexString `json:"code"`
		Data struct {
			NextOffset int        `json:"next_offset"`
			HasMore    bool       `json:"has_more"`
			SessionID  flexString `json:"session_id"`
			Rows       []struct {
				Type  string `json:"type"`
				Name  string `json:"row_name"`
				Items []struct {
					Name string     `json:"show_name"`
					ID   flexString `json:"selector_item_id"`
				} `json:"items"`
			} `json:"selector_rows"`
			VideoData json.RawMessage `json:"video_data"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("short filter code %s", env.Code.String())
	}
	out := &ShortFilterResult{
		NextOffset: env.Data.NextOffset, HasMore: env.Data.HasMore, SessionID: env.Data.SessionID.String(),
		Items: decodeSeriesList(env.Data.VideoData),
	}
	if out.Items == nil {
		out.Items = []model.SeriesSummary{}
	}
	for _, r := range env.Data.Rows {
		row := model.FilterRow{Key: r.Type, Name: r.Name}
		for _, it := range r.Items {
			if id := it.ID.String(); id != "" {
				row.Options = append(row.Options, model.FilterOption{Name: it.Name, Value: id})
			}
		}
		if row.Key != "" {
			out.Rows = append(out.Rows, row)
		}
	}
	return out, nil
}

// ShortRankResult is one page of the short-drama ranking page.
type ShortRankResult struct {
	Contents   []model.RankContent
	Items      []model.RankItem
	NextOffset int
	HasMore    bool
	SessionID  string
	Version    string
}

// promotedText marks a row inserted into a short-drama ranking: the app shows
// it without a position, and its rank copies the next row's (observed).
const promotedText = "平台推荐"

// ParseShortRank reads cell/change on the short-drama ranking cell: the
// content/list catalog of the page selector, and the ranked rows of the
// selected list.
func ParseShortRank(raw []byte) (*ShortRankResult, error) {
	type option struct {
		Name     string     `json:"show_name"`
		ID       flexString `json:"selector_item_id"`
		Selected bool       `json:"is_selected"`
		Sub      struct {
			OuterRow struct {
				Items []struct {
					Name     string     `json:"show_name"`
					ID       flexString `json:"selector_item_id"`
					Selected bool       `json:"is_selected"`
				} `json:"items"`
			} `json:"outer_row"`
		} `json:"sub_cell_selector"`
	}
	var env struct {
		Code flexString `json:"code"`
		Data struct {
			NextOffset int        `json:"next_offset"`
			HasMore    bool       `json:"has_more"`
			SessionID  flexString `json:"session_id"`
			Version    flexString `json:"rank_version"`
			CellView   struct {
				Selector struct {
					OuterRow struct {
						Items []option `json:"items"`
					} `json:"outer_row"`
				} `json:"cell_selector"`
				CellData []struct {
					VideoData []struct {
						SeriesID flexString      `json:"series_id"`
						Title    string          `json:"title"`
						Cover    string          `json:"cover"`
						Count    flexInt         `json:"episode_cnt"`
						Rec      json.RawMessage `json:"recommend_info"`
						RecText  struct {
							Text string `json:"RecommendText"`
						} `json:"rec_text_item"`
					} `json:"video_data"`
					Users []struct {
						Info struct {
							Base struct {
								Name    string     `json:"user_name"`
								Avatar  string     `json:"user_avatar"`
								ActorID flexString `json:"actor_id_str"`
								UserID  flexString `json:"user_id"`
							} `json:"base_info"`
						} `json:"user_info"`
						RecText struct {
							Text string `json:"RecommendText"`
						} `json:"rec_text_item"`
						Intro     string `json:"intro"`
						SubTitles []struct {
							Content string `json:"content"`
						} `json:"sub_title_list"`
						Change *struct {
							Kind  flexInt `json:"rank_change_type"`
							Moved flexInt `json:"ChangeNum"`
						} `json:"rank_change_data"`
					} `json:"ugc_user_data"`
				} `json:"cell_data"`
			} `json:"cell_view"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("short rank code %s", env.Code.String())
	}
	d := env.Data
	out := &ShortRankResult{
		NextOffset: d.NextOffset, HasMore: d.HasMore, SessionID: d.SessionID.String(),
		Version: d.Version.String(), Items: []model.RankItem{},
	}
	for _, c := range d.CellView.Selector.OuterRow.Items {
		content := model.RankContent{ID: c.ID.String(), Name: c.Name, Selected: c.Selected}
		for _, l := range c.Sub.OuterRow.Items {
			content.Boards = append(content.Boards, model.RankBoard{ID: l.ID.String(), Name: l.Name, Selected: l.Selected})
		}
		if content.ID != "" {
			out.Contents = append(out.Contents, content)
		}
	}
	for _, cell := range d.CellView.CellData {
		for _, v := range cell.VideoData {
			id, title := v.SeriesID.String(), strings.TrimSpace(v.Title)
			if id == "" || title == "" {
				continue
			}
			out.Items = append(out.Items, model.RankItem{
				Position: len(out.Items) + 1, Rank: recommendRank(v.Rec), SeriesID: id, Title: title,
				CoverURL: v.Cover, Score: v.RecText.Text, EpisodeCount: int(v.Count),
				Promoted: v.RecText.Text == promotedText,
			})
		}
		// The actor ranking (ShortRankActors) lists people instead.
		for _, u := range cell.Users {
			b := u.Info.Base
			id := firstNonEmpty(b.ActorID.String(), b.UserID.String())
			name := strings.TrimSpace(b.Name)
			if id == "" || name == "" {
				continue
			}
			item := model.RankItem{
				Position: len(out.Items) + 1, ActorID: id, UserID: b.UserID.String(), Actor: name, AvatarURL: b.Avatar,
				Score: u.RecText.Text, Intro: u.Intro,
			}
			for _, s := range u.SubTitles {
				if s.Content != "" {
					item.Details = append(item.Details, s.Content)
				}
			}
			if u.Change != nil {
				item.Change = &model.RankChange{Kind: model.RankChangeKind(u.Change.Kind), Places: int(u.Change.Moved)}
			}
			out.Items = append(out.Items, item)
		}
	}
	return out, nil
}

// ParseCategoryLanding reads a category landing page: its filter options and
// the book rows for the current filter.
func ParseCategoryLanding(raw []byte) (*model.CategoryPage, error) {
	var env struct {
		Code flexString `json:"code"`
		Data struct {
			Offset  int             `json:"offset"`
			HasMore bool            `json:"has_more"`
			Rule    json.RawMessage `json:"rule"`
			Books   []struct {
				ID       flexString `json:"book_id"`
				Title    string     `json:"book_name"`
				Author   string     `json:"author"`
				Abstract string     `json:"abstract"`
				Cover    string     `json:"thumb_url"`
				Category string     `json:"category"`
				Words    flexInt64  `json:"word_number"`
				Score    flexString `json:"score"`
			} `json:"book_info"`
			Category struct {
				ID    flexInt `json:"id"`
				Name  string  `json:"name"`
				Desc  string  `json:"description"`
				Books flexInt `json:"book_num"`
			} `json:"category"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("category code %s", env.Code.String())
	}
	out := &model.CategoryPage{
		CategoryID:   strconv.Itoa(int(env.Data.Category.ID)),
		CategoryName: env.Data.Category.Name,
		Description:  env.Data.Category.Desc,
		HasMore:      env.Data.HasMore,
		NextOffset:   env.Data.Offset,
	}
	if raw := nonEmptyJSON(env.Data.Rule); len(raw) > 0 && raw[0] == '{' {
		var rule map[string]json.RawMessage
		if err := json.Unmarshal(raw, &rule); err == nil {
			for _, key := range []string{"book_status", "sort_by", "sub_category", "word_type"} {
				body, ok := rule[key]
				if !ok {
					continue
				}
				var opts []struct {
					Name  string `json:"name"`
					Value string `json:"value"`
				}
				if err := json.Unmarshal(body, &opts); err != nil {
					continue
				}
				group := model.FilterGroup{Key: key}
				for _, o := range opts {
					group.Options = append(group.Options, model.FilterOption{Name: o.Name, Value: o.Value})
				}
				out.Filters = append(out.Filters, group)
			}
		}
	}
	items := make([]model.SeriesSummary, 0, len(env.Data.Books))
	for _, b := range env.Data.Books {
		id := b.ID.String()
		if id == "" || strings.TrimSpace(b.Title) == "" {
			continue
		}
		items = append(items, model.SeriesSummary{
			ID: id, Title: b.Title, Description: oneLineText(b.Abstract),
			CoverURL: b.Cover, CoverExpiresAt: URLLifetime(b.Cover),
			Author: b.Author, Category: b.Category, Words: int64(b.Words),
		})
	}
	out.Items = items
	return out, nil
}

// oneLineText keeps the first sentence of a long abstract.
func oneLineText(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexAny(s, "\n"); i >= 0 {
		s = s[:i]
	}
	if len(s) > 120 {
		s = s[:120]
	}
	return strings.Join(strings.Fields(s), " ")
}

// ParseSuggest reads search suggestions.
//
// The live response carries data.query_result (plain strings) and
// data.query_result_v2 (rich rows with name, keyword, word_type and a
// highlighted snippet). The rich rows win because they keep the keyword the
// search page actually submits.
func ParseSuggest(raw []byte) ([]model.Suggestion, error) {
	var env struct {
		Code flexString `json:"code"`
		Data struct {
			QueryKey string          `json:"query_key"`
			Plain    []string        `json:"query_result"`
			Rich     json.RawMessage `json:"query_result_v2"`
			Suggest  json.RawMessage `json:"suggest_result"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("suggest code %s", env.Code.String())
	}
	for _, list := range []json.RawMessage{env.Data.Rich, env.Data.Suggest} {
		if len(bytes.TrimSpace(list)) == 0 || string(bytes.TrimSpace(list)) == "null" {
			continue
		}
		var rows []struct {
			Name    string `json:"name"`
			Keyword string `json:"keyword"`
			Type    int    `json:"word_type"`
		}
		if err := json.Unmarshal(list, &rows); err != nil {
			continue
		}
		out := make([]model.Suggestion, 0, len(rows))
		for _, r := range rows {
			text := strings.TrimSpace(r.Name)
			if text == "" {
				continue
			}
			keyword := strings.TrimSpace(r.Keyword)
			if keyword == "" || len(keyword) > 40 || !isDigits(keyword) {
				// A word_type row may carry the query text in keyword; a long
				// numeric keyword is a content ID and is kept as the target.
				keyword = text
			}
			out = append(out, model.Suggestion{Text: text, Keyword: keyword, Kind: r.Type})
		}
		if len(out) > 0 {
			return out, nil
		}
	}
	out := make([]model.Suggestion, 0, len(env.Data.Plain))
	for _, w := range env.Data.Plain {
		if text := strings.TrimSpace(w); text != "" {
			out = append(out, model.Suggestion{Text: text, Keyword: text})
		}
	}
	return out, nil
}

func isDigits(s string) bool {
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return s != ""
}

// ParseCategoryFront reads the category shell: its tab list. The item lists
// come from ParseCategoryLanding per tab.
func ParseCategoryFront(raw []byte) (*model.CategoryPage, error) {
	var env struct {
		Code flexString `json:"code"`
		Data struct {
			Config struct {
				DefaultTab flexInt   `json:"default_tab"`
				Names      []string  `json:"tab_name_list"`
				Tabs       []flexInt `json:"tab_type_list"`
			} `json:"category_tab_config"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("category code %s", env.Code.String())
	}
	out := &model.CategoryPage{DefaultTab: int(env.Data.Config.DefaultTab), Items: []model.SeriesSummary{}}
	names := env.Data.Config.Names
	for i, tab := range env.Data.Config.Tabs {
		name := ""
		if i < len(names) {
			name = names[i]
		}
		out.Tabs = append(out.Tabs, model.FilterOption{Name: name, Value: strconv.Itoa(int(tab))})
	}
	return out, nil
}

// ParseRankGroups reads the ranking channels a bookmall tab advertises.
//
// The 经典 tab carries a ranking cell whose rank_with_category_data.rank_algo_list
// names each ranking (rank_algo + rank_name). The walk is structural because the
// cell sits inside a nested cell tree.
func ParseRankGroups(raw []byte) []model.RankList {
	var root any
	if err := json.Unmarshal(raw, &root); err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []model.RankList
	var walk func(v any)
	walk = func(v any) {
		switch node := v.(type) {
		case map[string]any:
			if data, ok := node["rank_with_category_data"].(map[string]any); ok {
				if list, ok := data["rank_algo_list"].([]any); ok {
					for _, entry := range list {
						item, ok := entry.(map[string]any)
						if !ok {
							continue
						}
						algo := jsonNumber(item["rank_algo"])
						name, _ := item["rank_name"].(string)
						if algo == "" || name == "" || seen[algo] {
							continue
						}
						seen[algo] = true
						out = append(out, model.RankList{ID: algo, Name: name})
					}
				}
			}
			for _, child := range node {
				walk(child)
			}
		case []any:
			for _, child := range node {
				walk(child)
			}
		}
	}
	walk(root)
	return out
}

// jsonNumber renders a JSON number or numeric string as a plain string.
func jsonNumber(v any) string {
	switch n := v.(type) {
	case float64:
		return strconv.FormatInt(int64(n), 10)
	case string:
		return strings.TrimSpace(n)
	}
	return ""
}

// ParseCue reads the search box's rotating hints.
//
// The live response carries data_v1[] with {text, is_default} rows (for
// example 书名 / 主角 / 作者 / 书籍类型) plus a single data.text; cue_type
// variants may return a plain word list instead, which is accepted too.
func ParseCue(raw []byte) ([]string, error) {
	var env struct {
		Code   flexString `json:"code"`
		DataV1 []struct {
			Text string `json:"text"`
		} `json:"data_v1"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("cue code %s", env.Code.String())
	}
	out := make([]string, 0, len(env.DataV1))
	for _, row := range env.DataV1 {
		if t := strings.TrimSpace(row.Text); t != "" {
			out = append(out, t)
		}
	}
	return dedupeStrings(out), nil
}

// dedupeStrings keeps the first occurrence of each value.
func dedupeStrings(in []string) []string {
	seen := make(map[string]bool, len(in))
	out := make([]string, 0, len(in))
	for _, s := range in {
		if s == "" || seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
