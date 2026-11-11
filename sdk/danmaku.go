package sdk

import (
	"context"
	"fmt"

	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/internal/protocol/sign"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// DanmakuOptions 选择一页弹幕。Cursor 是上一页给出的令牌，为空时从这一集的
// 开头读起。PageSize 是一页最多几条，0 取官方播放器的 90；服务端每页给出的
// 条数有上限，再大也不会更多。
type DanmakuOptions struct {
	Cursor   string
	PageSize int
}

// ListDanmaku 加载一集的弹幕，一次一页。seriesID 是短剧 id，episodeID 是
// 这一集的 id（Episode.ID）。
//
// 弹幕按播放时间分页：一页是服务端划出的一段，条目按出现时间排序。
// 一段里没有弹幕时，这一页是空的，但 HasMore 仍为真，所以要一直翻到
// HasMore 为假。第一页会先读一次短剧详情，取这一集的时长，翻页到片尾为止。
//
// 服务端只下发精选的一部分弹幕，而且每台设备拿到的不一样：2026-11-11 实测，
// 一集统计有 8093 条（DanmakuPage.Total），不同的匿名设备翻完全部页取到
// 131 到 294 条。
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
		if page.StartMS == 0 {
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
	_, episodes, err := c.seriesDetail(ctx, seriesID)
	if err != nil {
		return 0, err
	}
	for _, e := range episodes {
		if e.ID != episodeID {
			continue
		}
		if e.Duration <= 0 {
			return protocol.DanmakuFallbackDuration, nil
		}
		return e.Duration.Milliseconds() + 1000, nil
	}
	return 0, fmt.Errorf("episode %s is not in series %s", episodeID, seriesID)
}
