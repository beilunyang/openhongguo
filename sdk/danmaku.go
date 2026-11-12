package sdk

import (
	"context"
	"fmt"
	"sync"

	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/internal/protocol/sign"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// DanmakuOptions 选择一页弹幕。
//
// Cursor 是上一页给出的令牌。Cursor 为空时，从 StartMs（这一集的播放位置，
// 单位毫秒）读起，0 即开头；播放器拖动进度条后，用新的位置重新读。
// 服务端从包含这个位置的那一段开始返回，所以第一页可能带着这个位置之前
// 一点的弹幕。Cursor 和 StartMs 只能给一个。
//
// PageSize 是一页最多几条，0 取官方播放器的 90；服务端每页给出的条数有上限，
// 再大也不会更多。
type DanmakuOptions struct {
	Cursor   string
	StartMs  int64
	PageSize int
}

// ListDanmaku 加载一集的弹幕，一次一页。seriesID 是短剧 id，episodeID 是
// 这一集的 id（Episode.ID）。
//
// 弹幕按播放时间分页：一页是服务端划出的一段，条目按出现时间排序。
// 一段里没有弹幕时，这一页是空的，但 HasMore 仍为真，所以要一直翻到
// HasMore 为假。不带 Cursor 的调用要知道这一集的时长，以便翻到片尾为止：
// 客户端第一次读某部短剧时读一次它的详情，之后记住各集的时长。
//
// 服务端只下发精选的一部分弹幕，而且每台设备拿到的不一样：2026-11-11 实测，
// 一集统计有 8093 条（DanmakuPage.Total），不同的匿名设备翻完全部页取到
// 35 到 297 条。
//
// 请求就是官方播放器读取弹幕的评论列表调用。弹幕不带发送者的身份。
func (c *Client) ListDanmaku(ctx context.Context, seriesID, episodeID string, opts DanmakuOptions) (*model.DanmakuPage, error) {
	return withFreshDevice(ctx, c, false, func(ctx context.Context) (*model.DanmakuPage, error) {
		if !validID(seriesID) {
			return nil, fmt.Errorf("invalid series ID")
		}
		if !validID(episodeID) {
			return nil, fmt.Errorf("invalid episode ID")
		}
		// StartMs breaks Go's initialism style on purpose: the bindings name
		// the option after it, and startMs matches the page's startMs and the
		// item's offsetMs there.
		if opts.StartMs < 0 {
			return nil, fmt.Errorf("%w: StartMs is negative", ErrInvalidOptions)
		}
		if opts.Cursor != "" && opts.StartMs != 0 {
			return nil, fmt.Errorf("%w: pass Cursor or StartMs, not both", ErrInvalidOptions)
		}
		cur, err := protocol.DecodeDanmakuCursor(opts.Cursor)
		if err != nil {
			return nil, err
		}
		if opts.Cursor != "" && cur.EpisodeID != episodeID {
			return nil, fmt.Errorf("the cursor belongs to another episode")
		}
		if opts.Cursor == "" {
			if cur.Duration, err = c.episodeDuration(ctx, seriesID, episodeID); err != nil {
				return nil, err
			}
			cur.Start = opts.StartMs
			if cur.Start >= cur.Duration {
				// Past the end: nothing plays from here.
				return &model.DanmakuPage{Items: []model.Danmaku{}, StartMS: cur.Start}, nil
			}
		}
		body, err := protocol.DanmakuBody(seriesID, episodeID, cur.Start, cur.Duration, opts.PageSize, cur.Next)
		if err != nil {
			return nil, err
		}
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		path := fmt.Sprintf(protocol.PathDanmaku, episodeID)
		ticket := c.clock().UnixMilli()
		rawQuery, headers := sign.SignPOST(path, protocol.DeviceQueryPairs(c.identity(p)), ticket, p.DeviceID, protocol.VersionName, body, nil)
		raw, _, err := c.doRaw(ctx, "POST", c.apiOrigin+path+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), body)
		if err != nil {
			return nil, err
		}
		res, err := protocol.ParseDanmaku(raw)
		if err != nil {
			return nil, err
		}
		page := &model.DanmakuPage{Items: res.Items, StartMS: res.Start, EndMS: res.End, Total: res.Total}
		if page.StartMS == 0 && page.EndMS == 0 {
			page.StartMS = cur.Start
		}
		// A next window that does not move forward, or starts past the end,
		// would page forever.
		if res.HasMore && res.Next > cur.Start && res.Next < cur.Duration {
			page.HasMore = true
			next := protocol.DanmakuCursor{EpisodeID: episodeID, Duration: cur.Duration, Start: res.Next, Next: res.Cursor}
			if page.Cursor, err = protocol.EncodeDanmakuCursor(next); err != nil {
				return nil, err
			}
		}
		return page, nil
	})
}

// episodeDuration 返回一集的时长（毫秒），多留一秒，免得漏掉片尾的弹幕。
// 详情没给时长时，返回 protocol.DanmakuFallbackDuration。
func (c *Client) episodeDuration(ctx context.Context, seriesID, episodeID string) (int64, error) {
	if d, ok := c.durations.get(episodeID); ok {
		return d, nil
	}
	_, episodes, err := c.seriesDetail(ctx, seriesID)
	if err != nil {
		return 0, err
	}
	c.durations.put(episodes)
	for _, e := range episodes {
		if e.ID != episodeID {
			continue
		}
		if d, ok := c.durations.get(episodeID); ok {
			return d, nil
		}
		return protocol.DanmakuFallbackDuration, nil
	}
	return 0, fmt.Errorf("episode %s is not in series %s", episodeID, seriesID)
}

// maxDurations bounds durationCache; past it the cache starts over.
const maxDurations = 4096

// durationCache 记住各集的时长（毫秒），拖动进度后重新读弹幕时不必再读详情。
// 时长不会变，所以不设过期。零值可以直接使用。
type durationCache struct {
	mu sync.Mutex
	ms map[string]int64
}

func (d *durationCache) get(episodeID string) (int64, bool) {
	d.mu.Lock()
	defer d.mu.Unlock()
	v, ok := d.ms[episodeID]
	return v, ok
}

// put 记住 episodes 里给出了时长的各集。
func (d *durationCache) put(episodes []model.Episode) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.ms == nil || len(d.ms)+len(episodes) > maxDurations {
		d.ms = make(map[string]int64, len(episodes))
	}
	for _, e := range episodes {
		if e.ID != "" && e.Duration > 0 {
			d.ms[e.ID] = e.Duration.Milliseconds() + 1000
		}
	}
}
