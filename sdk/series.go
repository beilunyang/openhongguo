package sdk

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/internal/protocol/sign"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// EpisodeOptions 选择一个分集页。游标不在第 1 阶段的范围内。
type EpisodeOptions struct {
	Cursor   string
	PageSize int
}

// GetSeries 加载一部短剧的详情。
func (c *Client) GetSeries(ctx context.Context, seriesID string) (*model.Series, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.Series, error) {
		series, _, err := c.seriesDetail(ctx, seriesID)
		return series, err
	})
}

// seriesBatchSize 是一次批量请求查询的短剧数。服务端接受 20，
// 拒绝 30 并返回 100001（2026-10-24 实测）；取 10 可让响应保持在约 1 MB 以内。
const seriesBatchSize = 10

// GetSeriesBatch 加载多部短剧及其分集，按请求顺序返回。
// 服务端不认识的 ID 会被略去；重复的 ID 只查询一次。
//
// 播放器的 multi_video_detail 读取接口为每部短剧返回与 GetSeries 相同的结构；
// 更长的列表按 seriesBatchSize 分批请求。
func (c *Client) GetSeriesBatch(ctx context.Context, seriesIDs []string) ([]model.SeriesDetail, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) ([]model.SeriesDetail, error) {
		var ids []string
		seen := map[string]bool{}
		for _, id := range seriesIDs {
			if !validID(id) {
				return nil, fmt.Errorf("invalid series ID %q", id)
			}
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
		found := make(map[string]model.SeriesDetail, len(ids))
		for start := 0; start < len(ids); start += seriesBatchSize {
			chunk := ids[start:min(start+seriesBatchSize, len(ids))]
			body, err := protocol.SeriesBatchBody(chunk)
			if err != nil {
				return nil, err
			}
			p, err := c.ensureSearchGuest(ctx)
			if err != nil {
				return nil, err
			}
			ticket := c.clock().UnixMilli()
			rawQuery, headers := sign.SignPOST(protocol.PathSeriesBatch, protocol.DeviceQueryPairs(c.identity(p)), ticket, p.DeviceID, protocol.VersionName, body, nil)
			raw, _, err := c.doRaw(ctx, "POST", c.apiOrigin+protocol.PathSeriesBatch+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), body)
			if err != nil {
				return nil, err
			}
			got, err := protocol.ParseSeriesBatch(raw)
			if err != nil {
				return nil, err
			}
			for id, d := range got {
				found[id] = d
			}
		}
		out := make([]model.SeriesDetail, 0, len(found))
		for _, id := range ids {
			if d, ok := found[id]; ok {
				out = append(out, d)
			}
		}
		return out, nil
	})
}

// ListEpisodes 加载一部短剧的分集列表。集数从 1 开始。
func (c *Client) ListEpisodes(ctx context.Context, seriesID string, opts EpisodeOptions) (*Page[model.Episode], error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*Page[model.Episode], error) {
		if opts.Cursor != "" || opts.PageSize < 0 {
			return nil, fmt.Errorf("episode pagination is not verified")
		}
		_, episodes, err := c.seriesDetail(ctx, seriesID)
		if err != nil {
			return nil, err
		}
		if len(episodes) == 0 {
			return nil, ErrAnonymousLimited
		}
		return &Page[model.Episode]{Items: episodes}, nil
	})
}

func (c *Client) seriesDetail(ctx context.Context, seriesID string) (*model.Series, []model.Episode, error) {
	if !validID(seriesID) {
		return nil, nil, fmt.Errorf("invalid series ID")
	}
	p, err := c.ensureSearchGuest(ctx)
	if err != nil {
		return nil, nil, err
	}
	body, err := protocol.SeriesBody(seriesID)
	if err != nil {
		return nil, nil, err
	}
	ticket := c.clock().UnixMilli()
	rawQuery, headers := sign.SignPOST(protocol.PathSeries, protocol.DeviceQueryPairs(c.identity(p)), ticket, p.DeviceID, protocol.VersionName, body, nil)
	raw, _, err := c.doRaw(ctx, "POST", c.apiOrigin+protocol.PathSeries+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), body)
	if err != nil {
		return nil, nil, err
	}
	return protocol.ParseSeries(raw)
}

// AlbumOptions 留作选择合集界面之用。
type AlbumOptions struct{}

// GetAlbum 加载一个合集（多季合集）及其各季。
//
// 路径和请求类来自对 APK 的观察：GetVideoAlbumDetailRequest 携带 albumId、
// seriesIds 和 needVideoDetailInfo；响应列出该合集的各季。
func (c *Client) GetAlbum(ctx context.Context, albumID string, _ AlbumOptions) (*model.Album, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.Album, error) {
		if !validID(albumID) {
			return nil, fmt.Errorf("invalid album ID")
		}
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		body, err := protocol.AlbumBody(albumID)
		if err != nil {
			return nil, err
		}
		ticket := c.clock().UnixMilli()
		rawQuery, headers := sign.SignPOST(protocol.PathAlbum, protocol.DeviceQueryPairs(c.identity(p)), ticket, p.DeviceID, protocol.VersionName, body, nil)
		raw, _, err := c.doRaw(ctx, "POST", c.apiOrigin+protocol.PathAlbum+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), body)
		if err != nil {
			return nil, err
		}
		return protocol.ParseAlbum(raw, albumID)
	})
}

// SeriesRecommend 加载一部短剧的相关推荐。
//
// 该界面没有专门的 RPC：客户端复用通用的 plan feed（GET
// /reading/bookapi/plan/v:version/），带上 from=select_panel_recommend 和
// scene=10，再取 show_type 为 160 的那个 cell。匿名时，
// 服务端返回一个通用的更多好剧 cell（show_type 401），其中的行不随该短剧而定，
// 每次调用都会变化，也不带 session id。
func (c *Client) SeriesRecommend(ctx context.Context, seriesID string) (*model.Recommend, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.Recommend, error) {
		if !validID(seriesID) {
			return nil, fmt.Errorf("invalid series ID")
		}
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		ticket := c.clock().UnixMilli()
		session := randomUUID()
		rawQuery, headers := sign.SignGET(protocol.SignedPlanPairs(c.identity(p), seriesID, session), ticket, p.DeviceID, protocol.VersionName, nil)
		raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathPlan+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
		if err != nil {
			return nil, err
		}
		cells, err := protocol.ParsePlanCell(raw)
		if err != nil {
			return nil, err
		}

		for _, cell := range cells {
			if cell.ShowType == model.ShowTypeDetailRelatedVideo && len(cell.Items) > 0 {
				return &model.Recommend{CellTitle: cell.CellName, SessionID: cell.SessionID, Items: cell.Items}, nil
			}
		}
		// 匿名界面返回它自己的 cell 类型（实测：401 “更多好剧”），而不是 160，
		// 所以任何有内容的 cell 都是可用的结果。
		for _, cell := range cells {
			if len(cell.Items) > 0 {
				return &model.Recommend{CellTitle: cell.CellName, SessionID: cell.SessionID, Items: cell.Items}, nil
			}
		}
		return nil, ErrContentUnavailable
	})
}

// Seasons 列出多季短剧的全部季。
//
// 详情页的系列剧行列出所有季，包括即将上线的季。
// 没有其他季的作品在那里会得到 101116。读不到该行时，
// 本方法先退回合集接口（在所有抽样的合集上，它都以 101001 拒绝匿名读取），
// 再退回搜索：搜索基础标题，读取候选项的详情，
// 保留合集 id 相同的那些（服务端没有发布合集 id 时，保留基础标题相同的那些）。
// 结果按季序号排序。
func (c *Client) Seasons(ctx context.Context, seriesID string) (*model.Album, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.Album, error) {
		series, err := c.GetSeries(ctx, seriesID)
		if err != nil {
			return nil, err
		}
		if seasons, err := c.seriesSeasons(ctx, series.ID); err == nil && len(seasons) > 0 {
			for i := range seasons {
				if seasons[i].SeriesID == series.ID {
					seasons[i].Status = series.Status
				}
			}
			sortSeasons(seasons)
			return &model.Album{
				ID: series.RelatedAlbumID, Title: seasonBaseTitle(series.Title),
				SeasonCount: len(seasons), Seasons: seasons,
			}, nil
		} else if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if series.RelatedAlbumID != "" {
			if album, err := c.GetAlbum(ctx, series.RelatedAlbumID, AlbumOptions{}); err == nil && len(album.Seasons) > 0 {
				return album, nil
			}
		}
		base := seasonBaseTitle(series.Title)
		out := &model.Album{ID: series.RelatedAlbumID, Title: base}
		if !series.IsMultiSeason && series.RelatedAlbumID == "" {
			// 单季作品只是与无关条目同名；去搜索会把相似条目误当成别的季。
			out.Seasons = []model.Season{{
				SeriesID: series.ID, Title: series.Title, Index: 1,
				CoverURL: series.CoverURL, EpisodeCount: series.EpisodeCount, Status: series.Status,
			}}
			out.SeasonCount = 1
			return out, nil
		}
		out.Seasons = append(out.Seasons, model.Season{
			SeriesID: series.ID, Title: series.Title, Index: series.SeasonIndex,
			CoverURL: series.CoverURL, EpisodeCount: series.EpisodeCount, Status: series.Status,
		})
		// 完整的长标题只会搜出被查询的那一季，
		// 所以查找同系列其他季时搜索一个短前缀，再保留基础标题匹配的行。
		query := base
		if runes := []rune(base); len(runes) > 6 {
			query = string(runes[:6])
		}
		page, err := c.Search(ctx, query, SearchOptions{PageSize: 20})
		if err != nil && !errors.Is(err, ErrAnonymousLimited) {
			return out, err
		}
		// 基础标题匹配的候选项最多取十个，一次批量读取。
		var candidates []string
		if page != nil {
			for _, item := range page.Items {
				if item.ID == series.ID || len(candidates) >= seriesBatchSize {
					continue
				}
				candBase := seasonBaseTitle(item.Title)
				if candBase != base && !strings.HasPrefix(candBase, base) && !strings.HasPrefix(base, candBase) {
					continue
				}
				candidates = append(candidates, item.ID)
			}
		}
		if len(candidates) > 0 {
			details, err := c.GetSeriesBatch(ctx, candidates)
			if err != nil && ctx.Err() != nil {
				return nil, ctx.Err()
			}
			for _, d := range details {
				detail := d.Series
				if series.RelatedAlbumID != "" && detail.RelatedAlbumID != series.RelatedAlbumID {
					continue
				}
				out.Seasons = append(out.Seasons, model.Season{
					SeriesID: detail.ID, Title: detail.Title, Index: detail.SeasonIndex,
					CoverURL: detail.CoverURL, EpisodeCount: detail.EpisodeCount, Status: detail.Status,
				})
			}
		}
		sortSeasons(out.Seasons)
		out.SeasonCount = len(out.Seasons)
		return out, nil
	})
}

// seriesSeasons 读取详情页的系列剧行，即带 from=single_col_same_ip_series 的
// plan feed。
func (c *Client) seriesSeasons(ctx context.Context, seriesID string) ([]model.Season, error) {
	p, err := c.ensureSearchGuest(ctx)
	if err != nil {
		return nil, err
	}
	ticket := c.clock().UnixMilli()
	rawQuery, headers := sign.SignGET(protocol.SignedSeasonPairs(c.identity(p), seriesID, randomUUID()), ticket, p.DeviceID, protocol.VersionName, nil)
	raw, _, err := c.doRaw(ctx, "GET", c.apiOrigin+protocol.PathPlan+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), nil)
	if err != nil {
		return nil, err
	}
	return protocol.ParseSeasons(raw)
}

func sortSeasons(seasons []model.Season) {
	sort.SliceStable(seasons, func(i, j int) bool {
		if seasons[i].Index != seasons[j].Index {
			return seasons[i].Index < seasons[j].Index
		}
		return seasons[i].SeriesID < seasons[j].SeriesID
	})
}

// seasonSuffix 匹配标题末尾的季标记。
var seasonSuffix = regexp.MustCompile(`(?s)\s*(第\s*[0-9一二三四五六七八九十百]+\s*[季部期]|[0-9]+)\s*$`)

// seasonBaseTitle 去掉标题末尾的季标记（“…第二季”、“…3”），
// 让同系列各季的标题共享一个可搜索的基础标题。
func seasonBaseTitle(title string) string {
	t := strings.TrimSpace(title)
	if t == "" {
		return t
	}
	if loc := seasonSuffix.FindStringIndex(t); loc != nil && loc[0] > 0 {
		t = strings.TrimSpace(t[:loc[0]])
	}
	return t
}
