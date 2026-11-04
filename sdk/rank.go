package sdk

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/internal/protocol/sign"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// Ranking 加载一个活动榜单。
//
// activity_api 主机独立于 reading 主机，
// 接受不带签名设备查询参数的普通 JSON POST。
//
// activity_api 系列接口位于 DistributionOrigin（实测：reading 主机返回 404）。
// 随 app 发布的客户端写死了星粉榜的活动 id 和榜单 id；
// 传空字符串即使用这些默认值。
func (c *Client) Ranking(ctx context.Context, activityID, rankListID string) (*model.Ranking, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.Ranking, error) {
		if activityID == "" {
			activityID = protocol.DefaultRankActivity
		}
		if rankListID == "" {
			rankListID = protocol.DefaultRankList
		}
		body, err := protocol.RankActivityBody(activityID, rankListID, 0, 50)
		if err != nil {
			return nil, err
		}
		// 活动主机接受普通 JSON POST：加上签名设备查询参数会让它返回 HTTP
		// 400（两种方式都实测过）。
		headers := map[string]string{"Content-Type": "application/json", "User-Agent": protocol.UserAgent(c.device)}
		raw, _, err := c.doRaw(ctx, "POST", c.distOrigin+protocol.PathRankActivity, headers, body)
		if err != nil {
			return nil, err
		}
		return protocol.ParseRanking(raw)
	})
}

// RankRows 加载书城榜单的一页（经典频道的榜单入口）。这些行是书，
// 不是短剧。请求要的是新的一批（change_type=1），
// 所以每次调用返回的集合都不同
// （2026-10-24 实测：调用三次，第 1 名三次都不同）。Rank 只是该行在本页的位置，
// 而不是稳定榜单上的名次。
func (c *Client) RankRows(ctx context.Context, cellID string, offset int) (*model.Ranking, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.Ranking, error) {
		if cellID == "" {
			cellID = protocol.DefaultRankCell
		}
		if offset < 0 {
			offset = 0
		}
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		ticket := c.clock().UnixMilli()
		rawQuery, headers := sign.SignGET(protocol.SignedRankCellPairs(c.identity(p), cellID, offset, 20), ticket, p.DeviceID, protocol.VersionName, nil)
		raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathRankCells+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
		if err != nil {
			return nil, err
		}
		ranking, err := protocol.ParseRankRows(raw, offset)
		if err != nil {
			return nil, err
		}
		// 首页为空说明该 cell 已不存在；后续页为空只表示列表到头了，
		// 与其他分页读取一致。
		if len(ranking.Items) == 0 && !ranking.HasMore && offset == 0 {
			return nil, ErrContentUnavailable
		}
		return ranking, nil
	})
}

// RankGroups 列出书城榜单的类型（经典频道的各个榜单）。
//
// 该 tab 提供一个榜单 cell，其 rank_with_category_data 列出所有榜单；
// 这些 id 就是 cell/change 调用所需的 rank_algo 值。
func (c *Client) RankGroups(ctx context.Context) ([]model.RankList, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) ([]model.RankList, error) {
		channel, err := protocol.ResolveDiscoverChannel("经典")
		if err != nil {
			return nil, err
		}
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		id := c.identity(p)
		ticket := c.clock().UnixMilli()
		rawQuery, headers := sign.SignGET(protocol.SignedDiscoverPairsIn(id, channel.RequestTab, channel.BottomTab), ticket, p.DeviceID, protocol.VersionName, nil)
		raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathDiscover+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
		if err != nil {
			return nil, err
		}
		groups := protocol.ParseRankGroups(raw)
		if len(groups) == 0 {
			return nil, ErrContentUnavailable
		}
		return groups, nil
	})
}

// ShortFilterQuery 在筛选页（筛选）上挑选短剧。
//
// Selections 把 FilterRow.Key 映射到选中的选项 Values；缺省的行即全部。
// Cursor 是上一页的 Cursor。
type ShortFilterQuery struct {
	Selections map[string][]string
	Cursor     string
}

// ShortFilterPanel 加载短剧筛选面板的各个选择行。
// 类型可为 short_play（真人剧）、comic_series（漫剧）、ai_series（AI剧），
// 空表示全部。不同类型的行和选项各不相同。
func (c *Client) ShortFilterPanel(ctx context.Context, genre string) ([]model.FilterRow, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) ([]model.FilterRow, error) {
		scene, err := shortFilterScene(genre)
		if err != nil {
			return nil, err
		}
		body, err := protocol.ShortFilterBody(scene, nil, true, protocol.ShortCursor{})
		if err != nil {
			return nil, err
		}
		res, err := c.shortFilter(ctx, body)
		if err != nil {
			return nil, err
		}
		if len(res.Rows) == 0 {
			return nil, ErrContentUnavailable
		}
		return res.Rows, nil
	})
}

// ShortFilter 按条件筛选短剧，一次一页。
//
// 该服务位于 DistributionOrigin。每页的 Cursor 都带有已展示过的短剧；
// 服务端会跳过它们，所以各页不会重复（2026-10-24 实测）。
func (c *Client) ShortFilter(ctx context.Context, q ShortFilterQuery) (*model.ShortFilterPage, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.ShortFilterPage, error) {
		for key, values := range q.Selections {
			if !slices.Contains(protocol.ShortFilterKeys, key) {
				return nil, fmt.Errorf("unknown short filter row %q", key)
			}
			for _, v := range values {
				if strings.TrimSpace(v) == "" {
					return nil, fmt.Errorf("empty value for short filter row %q", key)
				}
			}
		}
		cur, err := protocol.DecodeShortCursor(q.Cursor)
		if err != nil {
			return nil, err
		}
		// 游标只能续接它自己的查询；不同的查询无法续接。
		selections := q.Selections
		if q.Cursor != "" {
			if len(selections) == 0 {
				selections = cur.Selections
			} else if !sameSelections(selections, cur.Selections) {
				return nil, fmt.Errorf("the cursor continues a different filter; drop the selections or the cursor")
			}
		}
		genre := ""
		if g := selections["genre"]; len(g) == 1 {
			genre = g[0]
		}
		scene, err := shortFilterScene(genre)
		if err != nil {
			return nil, err
		}
		body, err := protocol.ShortFilterBody(scene, selections, false, cur)
		if err != nil {
			return nil, err
		}
		res, err := c.shortFilter(ctx, body)
		if err != nil {
			return nil, err
		}
		if len(res.Items) == 0 && cur.Offset == 0 {
			return nil, ErrContentUnavailable
		}
		page := &model.ShortFilterPage{Items: res.Items, HasMore: res.HasMore}
		ids := make([]string, 0, len(res.Items))
		for _, it := range res.Items {
			ids = append(ids, it.ID)
		}
		cur.Selections = selections
		page.Cursor, err = nextShortCursor(cur, res.HasMore, res.NextOffset, res.SessionID, ids)
		return page, err
	})
}

// sameSelections 比较两组筛选查询，忽略空行。
func sameSelections(a, b map[string][]string) bool {
	norm := func(m map[string][]string) map[string]string {
		out := map[string]string{}
		for k, v := range m {
			if len(v) > 0 {
				out[k] = strings.Join(v, ",")
			}
		}
		return out
	}
	na, nb := norm(a), norm(b)
	if len(na) != len(nb) {
		return false
	}
	for k, v := range na {
		if nb[k] != v {
			return false
		}
	}
	return true
}

// shortFilterScene 把类型映射到 req_scene。漫剧和 AI剧有各自的 scene；
// 真人剧使用 default scene 并选中该类型
// （服务端会忽略 short_play scene，实测）。
func shortFilterScene(genre string) (string, error) {
	switch genre {
	case "", "short_play":
		return "default", nil
	case "comic_series", "ai_series":
		return genre, nil
	}
	return "", fmt.Errorf("unknown short-drama genre %q; use short_play, comic_series, or ai_series", genre)
}

func (c *Client) shortFilter(ctx context.Context, body []byte) (*protocol.ShortFilterResult, error) {
	p, err := c.ensureSearchGuest(ctx)
	if err != nil {
		return nil, err
	}
	ticket := c.clock().UnixMilli()
	rawQuery, headers := sign.SignPOST(protocol.PathShortFilter, protocol.DeviceQueryPairs(c.identity(p)), ticket, p.DeviceID, protocol.VersionName, body, nil)
	raw, _, err := c.doRaw(ctx, "POST", c.distOrigin+protocol.PathShortFilter+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), body)
	if err != nil {
		return nil, err
	}
	return protocol.ParseShortFilter(raw)
}

// ShortRankQuery 选择一个短剧榜单。Content 是 RankContent ID（all、human、
// comic_series_rank、ai_playlet、series_album；空即 all），
// Board 是它的某个 RankBoard ID（空即服务端对该 content 的默认值）。
// Cursor 是上一页的 Cursor。
type ShortRankQuery struct {
	Content string
	Board   string
	Cursor  string
}

// ShortRank 加载短剧排行榜的一页，包括演员榜。
//
// 榜单页就是对某个书城 cell 调用 cell/change：
// 每页还带有该页的内容目录及各内容的榜单。行按页面顺序给出，
// RankItem.Position 就是这个顺序；Promoted 行是服务端插入的行（平台推荐）。
// 每个 Cursor 都带有已展示过的短剧，服务端会跳过它们（2026-10-24 实测）。
func (c *Client) ShortRank(ctx context.Context, q ShortRankQuery) (*model.ShortRanking, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.ShortRanking, error) {
		cur, err := protocol.DecodeShortCursor(q.Cursor)
		if err != nil {
			return nil, err
		}
		content, board := strings.TrimSpace(q.Content), strings.TrimSpace(q.Board)
		if q.Cursor != "" {
			// 游标只能续接它自己的榜单；不同的榜单无法续接。
			if (content != "" && content != cur.Content) || (board != "" && board != cur.Board) {
				return nil, fmt.Errorf("the cursor continues a different ranking; drop --content/--board or the cursor")
			}
			content, board = cur.Content, cur.Board
		}
		if content == "" {
			content = "all"
		}
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		ticket := c.clock().UnixMilli()
		rawQuery, headers := sign.SignGET(protocol.SignedShortRankPairs(c.identity(p), content, board, cur), ticket, p.DeviceID, protocol.VersionName, nil)
		raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathRankCells+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
		if err != nil {
			return nil, err
		}
		res, err := protocol.ParseShortRank(raw)
		if err != nil {
			return nil, err
		}
		if len(res.Items) == 0 && cur.Offset == 0 {
			return nil, ErrContentUnavailable
		}
		out := &model.ShortRanking{Contents: res.Contents, Version: res.Version, Items: res.Items, HasMore: res.HasMore}
		// 目录标出了服务端实际返回的内容；没有指定榜单时，这就是默认榜单。
		for _, c := range res.Contents {
			if !c.Selected {
				continue
			}
			out.Content = c.ID
			for _, b := range c.Boards {
				if b.Selected {
					out.Board = b.ID
				}
			}
		}
		if out.Content == "" {
			out.Content, out.Board = content, board
		}
		// 位置跨页连续编号。
		for i := range out.Items {
			out.Items[i].Position += len(cur.Seen)
		}
		ids := make([]string, 0, len(res.Items))
		for _, it := range res.Items {
			id := it.SeriesID
			if id == "" {
				id = it.ActorID
			}
			ids = append(ids, id)
		}
		// 后续页请求本页返回的那个榜单。
		cur.Content, cur.Board = out.Content, out.Board
		out.Cursor, err = nextShortCursor(cur, res.HasMore, res.NextOffset, res.SessionID, ids)
		return out, err
	})
}

// nextShortCursor 用一页的短剧扩展 cur；到列表末尾时返回 ""。
func nextShortCursor(cur protocol.ShortCursor, more bool, offset int, session string, ids []string) (string, error) {
	if !more || len(ids) == 0 || offset <= cur.Offset {
		return "", nil
	}
	next := cur
	next.Offset, next.Seen, next.Session = offset, append(append([]string{}, cur.Seen...), ids...), session
	return protocol.EncodeShortCursor(next)
}
