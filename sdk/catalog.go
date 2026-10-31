package sdk

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/internal/protocol/sign"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// Page 是一页匿名结果。
type Page[T any] struct {
	Items   []T    `json:"items"`
	Cursor  string `json:"cursor,omitempty"`
	HasMore bool   `json:"has_more"`
}

// DiscoverOptions 选择一个发现页。Channel 是书城频道名，空频道即推荐。
// Cursor 是上一页给出的令牌。PageSize 是 cell/change 的 limit（0 表示 20，上限 50）。
type DiscoverOptions struct {
	Cursor   string
	Channel  string
	PageSize int
}

// SearchOptions 选择一个搜索页。Tab 取值为综合、剧集、视频、讨论、小说、
// 用户或听书，空 tab 即综合。Cursor 是下一个偏移量。
type SearchOptions struct {
	Cursor   string
	PageSize int
	Tab      string
	// Filters 把综合 tab 收窄到 SearchFilter* 值；选多个时互相放宽。
	// 游标不携带这些筛选，所以要连同游标传入相同的筛选。
	Filters []string
}

// 综合 tab 的搜索筛选项，与 app 筛选面板在 selected_items
// 中发送的值一致（2026-10-24 实测）。两个漫剧类筛选都返回 content_type 1004，
// 由服务端区分二者。
const (
	SearchFilterShortPlay = "content_type_short_play"       // 真人剧
	SearchFilterComic     = "content_type_dynamic_comic"    // 漫剧
	SearchFilterAIComic   = "content_type_ai_dynamic_comic" // AI剧
	SearchFilterFilm      = "content_type_long_video"       // 影视剧
	SearchFilterUnwatched = "search_range_unread"           // 未看过
)

// Discover 加载书城的一个频道，默认是“推荐”。首页由 tab 预加载和第一个不限量的
// cell/change 页组成。带游标时只加载下一个 cell/change 页。
func (c *Client) Discover(ctx context.Context, opts DiscoverOptions) (*Page[model.SeriesSummary], error) {
	channel, err := protocol.ResolveDiscoverChannel(opts.Channel)
	if err != nil {
		return nil, err
	}
	p, err := c.ensureSearchGuest(ctx)
	if err != nil {
		return nil, err
	}
	id := c.identity(p)
	if opts.Cursor != "" {
		next, err := protocol.DecodeFeedCursor(opts.Cursor)
		if err != nil {
			return nil, err
		}
		if next.TabType != strconvTab(channel.ResponseTab) {
			return nil, fmt.Errorf("discover cursor is not verified")
		}
		if next.Limit <= 0 {
			next.Limit = protocol.CellChangeLimit(opts.PageSize)
		}
		more, err := c.cellChange(ctx, p, id, next)
		if err != nil {
			return nil, err
		}
		return finishDiscover(more.Items, next, more)
	}
	ticket := c.clock().UnixMilli()
	rawQuery, headers := sign.SignGET(protocol.SignedDiscoverPairsIn(id, channel.RequestTab, channel.BottomTab), ticket, p.DeviceID, protocol.VersionName, nil)
	raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathDiscover+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
	if err != nil {
		if errors.Is(err, ErrEndOfList) {
			return &Page[model.SeriesSummary]{}, nil
		}
		return nil, err
	}
	parsed, err := protocol.ParseDiscoverTab(raw, channel.ResponseTab)
	if err != nil {
		return nil, err
	}
	if next, ok := protocol.DiscoverContinuationTab(raw, channel.ResponseTab); ok {
		next.Offset = "0"
		next.Limit = protocol.CellChangeLimit(opts.PageSize)
		more, err := c.cellChange(ctx, p, id, next)
		if err != nil {
			return nil, err
		}
		parsed.Items = mergeSeries(parsed.Items, more.Items)
		return finishDiscover(parsed.Items, next, more)
	}
	if len(parsed.Items) == 0 {
		return nil, ErrAnonymousLimited
	}
	return &Page[model.SeriesSummary]{Items: parsed.Items, HasMore: parsed.HasMore}, nil
}

func (c *Client) cellChange(ctx context.Context, p *DeviceProfile, id protocol.SearchIdentity, next protocol.FeedContinuation) (*protocol.CatalogPage, error) {
	ticket := c.clock().UnixMilli()
	changeQuery, changeHeaders := sign.SignGET(protocol.SignedCellChangePairs(id, next), ticket, p.DeviceID, protocol.VersionName, nil)
	moreRaw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathCellChange+"?"+changeQuery, anonymousHeaders(c, p, ticket, changeHeaders), nil)
	if err != nil {
		return nil, err
	}
	return protocol.ParseCellChange(moreRaw)
}

func finishDiscover(items []model.SeriesSummary, next protocol.FeedContinuation, more *protocol.CatalogPage) (*Page[model.SeriesSummary], error) {
	if len(items) == 0 {
		return nil, ErrAnonymousLimited
	}
	page := &Page[model.SeriesSummary]{Items: items, HasMore: more.HasMore}
	// 空的 change 页不会给出下一个偏移量。如有预加载行，仍然返回。
	if !more.HasMore || more.NextOffset <= 0 || len(more.Items) == 0 {
		return page, nil
	}
	if more.SessionID != "" {
		next.SessionID = more.SessionID
	}
	next.Offset = strconv.Itoa(more.NextOffset)
	cursor, err := protocol.EncodeFeedCursor(next)
	if err != nil {
		return page, nil
	}
	page.Cursor = cursor
	return page, nil
}

func strconvTab(n int) string {
	return strconv.Itoa(n)
}

func mergeSeries(base, extra []model.SeriesSummary) []model.SeriesSummary {
	seen := make(map[string]bool, len(base)+len(extra))
	out := make([]model.SeriesSummary, 0, len(base)+len(extra))
	for _, item := range append(base, extra...) {
		if item.ID == "" || seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		out = append(out, item)
	}
	return out
}

// Search 搜索短剧，可以按真人剧、漫剧等筛选。
func (c *Client) Search(ctx context.Context, query string, opts SearchOptions) (*Page[model.SeriesSummary], error) {
	if query == "" {
		return nil, fmt.Errorf("missing search query")
	}
	tab, err := protocol.ResolveSearchTab(opts.Tab)
	if err != nil {
		return nil, err
	}
	offset, err := protocol.SearchOffset(opts.Cursor)
	if err != nil {
		return nil, err
	}
	selected, err := searchFilters(opts.Filters, tab)
	if err != nil {
		return nil, err
	}
	p, err := c.ensureSearchGuest(ctx)
	if err != nil {
		return nil, err
	}
	count := opts.PageSize
	if count <= 0 {
		count = 20
	}
	pairs := protocol.SignedSearchPairsFor(c.identity(p), query, count, offset, tab)
	if selected != "" {
		pairs = append(pairs, [2]string{"selected_items", selected})
	}
	ticket := c.clock().UnixMilli()
	rawQuery, headers := sign.SignGET(pairs, ticket, p.DeviceID, protocol.VersionName, nil)
	raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathSearch+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
	if err != nil {
		if errors.Is(err, ErrEndOfList) {
			return &Page[model.SeriesSummary]{}, nil
		}
		return nil, err
	}
	tabType, err := strconv.Atoi(tab)
	if err != nil {
		return nil, err
	}
	parsed, err := protocol.ParseSearchTab(raw, tabType)
	if err != nil {
		return nil, err
	}
	if len(parsed.Items) == 0 {
		return nil, ErrAnonymousLimited
	}
	page := &Page[model.SeriesSummary]{Items: parsed.Items, HasMore: parsed.HasMore}
	if parsed.HasMore && parsed.NextOffset > offset {
		page.Cursor = strconv.Itoa(parsed.NextOffset)
	}
	return page, nil
}

// searchFilters 把 filters 拼成 selected_items 的值。
// 这些筛选只在综合 tab 上见过，所以其他 tab 会拒绝它们，而不是忽略。
func searchFilters(filters []string, tab string) (string, error) {
	if len(filters) == 0 {
		return "", nil
	}
	if tab != protocol.SearchTabShortPlay {
		return "", fmt.Errorf("search filters apply to the 综合 tab only")
	}
	seen := map[string]bool{}
	var out []string
	for _, f := range filters {
		switch f {
		case SearchFilterShortPlay, SearchFilterComic, SearchFilterAIComic, SearchFilterFilm, SearchFilterUnwatched:
		default:
			return "", fmt.Errorf("unknown search filter %q", f)
		}
		if !seen[f] {
			seen[f] = true
			out = append(out, f)
		}
	}
	return strings.Join(out, ","), nil
}

func anonymousHeaders(c *Client, p *DeviceProfile, ticket int64, headers map[string]string) map[string]string {
	if headers == nil {
		headers = map[string]string{}
	}
	headers["User-Agent"] = protocol.UserAgent(c.deviceOf(p))
	headers["x-ss-dp"] = protocol.AID
	headers["lc"] = "101"
	headers["x-reading-request"] = protocol.ReadingRequestID(ticket, c.randomInt31())
	if p != nil && p.TTReq != "" {
		headers["Cookie"] = guestCookie(p)
	}
	return headers
}

func validID(id string) bool {
	if len(id) < 4 || len(id) > 25 {
		return false
	}
	for _, c := range id {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// Pager 沿着游标翻页，并去掉之前的页已返回过的行。
// 服务端会跨页重复标题（在小说、漫剧和真人剧上实测），
// 所以跟随游标的调用方需要这个过滤。
type Pager struct {
	fetch func(ctx context.Context, cursor string) (*Page[model.SeriesSummary], error)
	seen  map[string]struct{}
	next  string
	done  bool
	// Pages 统计目前已拉取的页数。
	Pages int
}

// NewPager 基于任意翻页函数构建一个去重的 pager，让只持有接口的调用方（CLI、
// gomobile）也遵循同样的规则。
func NewPager(cursor string, fetch func(ctx context.Context, cursor string) (*Page[model.SeriesSummary], error)) *Pager {
	return &Pager{seen: map[string]struct{}{}, next: cursor, fetch: fetch}
}

// DiscoverPager 逐页遍历书城的一个频道，并去掉跨页重复的短剧。
// 遍历从 opts.Cursor 开始。
func (c *Client) DiscoverPager(opts DiscoverOptions) *Pager {
	return NewPager(opts.Cursor, func(ctx context.Context, cursor string) (*Page[model.SeriesSummary], error) {
		o := opts
		o.Cursor = cursor
		return c.Discover(ctx, o)
	})
}

// SearchPager 逐页遍历搜索结果，并去掉跨页重复的短剧。
// 遍历从 opts.Cursor 开始。
func (c *Client) SearchPager(query string, opts SearchOptions) *Pager {
	return NewPager(opts.Cursor, func(ctx context.Context, cursor string) (*Page[model.SeriesSummary], error) {
		o := opts
		o.Cursor = cursor
		return c.Search(ctx, query, o)
	})
}

// Next 加载下一页，返回之前没见过的行。more 表示是否还有下一页；
// 只含已知行的页会被跳过，调用方无需特殊处理。
func (p *Pager) Next(ctx context.Context) (items []model.SeriesSummary, more bool, err error) {
	for !p.done {
		page, err := p.fetch(ctx, p.next)
		if err != nil {
			return nil, false, err
		}
		p.Pages++
		p.next = page.Cursor
		p.done = !page.HasMore || page.Cursor == ""
		fresh := make([]model.SeriesSummary, 0, len(page.Items))
		for _, item := range page.Items {
			if _, ok := p.seen[item.ID]; ok {
				continue
			}
			p.seen[item.ID] = struct{}{}
			fresh = append(fresh, item)
		}
		if len(fresh) > 0 || p.done {
			return fresh, !p.done, nil
		}
	}
	return nil, false, nil
}

// Seen 返回 pager 已返回的不同行数。
func (p *Pager) Seen() int { return len(p.seen) }

// Suggest 返回某个前缀的搜索联想词。
//
// 路径取自随 app 发布的 sug 页面：
// GET /reading/bookapi/search/suggest/v:version/。响应结构按宽松方式解析，
// 因为不同 app 版本的列表名不同。
func (c *Client) Suggest(ctx context.Context, prefix string) ([]model.Suggestion, error) {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return nil, fmt.Errorf("missing search prefix")
	}
	p, err := c.ensureSearchGuest(ctx)
	if err != nil {
		return nil, err
	}
	ticket := c.clock().UnixMilli()
	rawQuery, headers := sign.SignGET(protocol.SignedSuggestPairs(c.identity(p), prefix), ticket, p.DeviceID, protocol.VersionName, nil)
	raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathSuggest+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
	if err != nil {
		return nil, err
	}
	return protocol.ParseSuggest(raw)
}

// SearchCues 加载搜索框里轮播的提示词。
//
// 路径和请求字段取自随 app 发布的客户端：
// GET /reading/bookapi/search/cue/v:version/，带 cue_type、limit、
// search_source 和 tab_name。响应不是热搜数据，
// 而是“书名”/“主角”/“作者”/“书籍类型”这类占位提示。
func (c *Client) SearchCues(ctx context.Context) ([]string, error) {
	p, err := c.ensureSearchGuest(ctx)
	if err != nil {
		return nil, err
	}
	ticket := c.clock().UnixMilli()
	rawQuery, headers := sign.SignGET(protocol.SignedCuePairs(c.identity(p), 1, 20), ticket, p.DeviceID, protocol.VersionName, nil)
	raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathCue+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
	if err != nil {
		return nil, err
	}
	return protocol.ParseCue(raw)
}

// HotSearch 加载搜索页的“猜你想搜”和各个热搜榜。
// 热搜榜有短剧热搜榜、漫剧热搜榜、热点话题榜，有时还有短剧新剧榜。
//
// 该页是带 from=search_input_page 的 plan feed；
// 其中的行位于一个 Lynx cell 的 lynx_data 中（2026-10-24 抓包）。
// 每个榜单只含页面展示的那几行。
func (c *Client) HotSearch(ctx context.Context) (*model.HotSearch, error) {
	p, err := c.ensureSearchGuest(ctx)
	if err != nil {
		return nil, err
	}
	ticket := c.clock().UnixMilli()
	rawQuery, headers := sign.SignGET(protocol.SignedHotSearchPairs(c.identity(p)), ticket, p.DeviceID, protocol.VersionName, nil)
	raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathPlan+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
	if err != nil {
		return nil, err
	}
	hot, err := protocol.ParseHotSearch(raw)
	if err != nil {
		return nil, err
	}
	if len(hot.Boards) == 0 && len(hot.Guesses) == 0 {
		return nil, ErrContentUnavailable
	}
	return hot, nil
}

// NewReleaseQuery 选择上新日历中的一天：Date 格式为 YYYYMMDD（空即今天），
// Cursor 是上一页的 Cursor，它带有所在的日期。
type NewReleaseQuery struct {
	Date   string
	Cursor string
}

// NewReleases 加载上新日历的一页。
// 一页包括某天上线的作品、预告在当天上线的作品，以及日历提供的日期。
//
// 该页是 /reading/user/subscribe/list/，带 tab_type 5 和
// target_date（2026-10-24 抓包）；匿名设备拿到的是公开日历，
// 而不是它自己的预约列表。
func (c *Client) NewReleases(ctx context.Context, q NewReleaseQuery) (*model.NewReleasePage, error) {
	cur, err := protocol.DecodeShortCursor(q.Cursor)
	if err != nil {
		return nil, err
	}
	date := q.Date
	switch {
	case date != "" && !protocol.ValidDate(date):
		return nil, fmt.Errorf("date must be YYYYMMDD")
	case q.Cursor != "" && date != "" && date != cur.Date:
		return nil, fmt.Errorf("the cursor belongs to %s, not %s", cur.Date, date)
	case q.Cursor != "":
		date = cur.Date
	}
	p, err := c.ensureSearchGuest(ctx)
	if err != nil {
		return nil, err
	}
	ticket := c.clock().UnixMilli()
	rawQuery, headers := sign.SignGET(protocol.SignedNewReleasePairs(c.identity(p), date, cur), ticket, p.DeviceID, protocol.VersionName, nil)
	raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathSubscribeList+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
	if err != nil {
		return nil, err
	}
	res, err := protocol.ParseNewReleases(raw)
	if err != nil {
		return nil, err
	}
	page := res.NewReleasePage
	if page.Date == "" {
		page.Date = date
	}
	if res.HasMore && len(res.Items) > 0 && res.NextOffset > cur.Offset {
		next := protocol.ShortCursor{Offset: res.NextOffset, Session: res.SessionID, Date: page.Date}
		if next.Session == "" {
			next.Session = cur.Session
		}
		if page.Cursor, err = protocol.EncodeShortCursor(next); err != nil {
			return nil, err
		}
	}
	return &page, nil
}
