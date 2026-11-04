// Package dispatch 把 SDK 的方法按名字和 JSON 参数调用，供各语言的绑定共用。
//
// Node.js（wasm）、Android 和 iOS 的绑定都只调用 Call：方法名沿用 JS 的
// 驼峰写法，参数是一个 JSON 数组，结果再编码成 JSON。错误用 Error 转成
// 带错误码的对象，各语言据此构造自己的异常类型。
package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// Config 是绑定传来的客户端设置（JSON），字段名与 Node.js 的 ClientOptions 一致。
type Config struct {
	APIOrigin          string `json:"apiOrigin"`
	RegisterOrigin     string `json:"registerOrigin"`
	DistributionOrigin string `json:"distributionOrigin"`
	RetryAttempts      int    `json:"retryAttempts"`
	// DeviceResets 是一次调用最多换几台新设备。缺省时用 SDK 的默认值，0 表示关闭。
	DeviceResets *int         `json:"deviceResets"`
	Device       model.Device `json:"device"`
}

// ParseConfig 解码 raw；raw 为空时返回零值。
func ParseConfig(raw string) (Config, error) {
	var cfg Config
	if raw == "" {
		return cfg, nil
	}
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return cfg, fmt.Errorf("%w: %v", sdk.ErrInvalidOptions, err)
	}
	return cfg, nil
}

// Apply 把 cfg 填进 opts。
func (cfg Config) Apply(opts *sdk.Options) {
	opts.APIOrigin = cfg.APIOrigin
	opts.RegisterOrigin = cfg.RegisterOrigin
	opts.DistributionOrigin = cfg.DistributionOrigin
	opts.RetryAttempts = cfg.RetryAttempts
	opts.DeviceResets = cfg.DeviceResets
	opts.Device = cfg.Device
}

// method 是一个可按名字调用的 SDK 方法：args 是参数数组的各项。
type method func(ctx context.Context, c *sdk.Client, args []json.RawMessage) (any, error)

// playbackResult 是 resolvePlayback 的结果：加密剧集同时给出内容密钥，
// 因为 Playback 里的密钥材料不会序列化，绑定的另一端拿不到。密钥推导失败时，
// 仍然返回播放模型，并在 ContentKeyError 里说明原因。
type playbackResult struct {
	Playback        *model.Playback   `json:"playback"`
	ContentKey      *model.ContentKey `json:"contentKey,omitempty"`
	ContentKeyError map[string]any    `json:"contentKeyError,omitempty"`
}

var methods = map[string]method{
	"discover": func(ctx context.Context, c *sdk.Client, a []json.RawMessage) (any, error) {
		var o sdk.DiscoverOptions
		if err := arg(a, 0, &o); err != nil {
			return nil, err
		}
		return c.Discover(ctx, o)
	},
	"search": func(ctx context.Context, c *sdk.Client, a []json.RawMessage) (any, error) {
		var q string
		var o sdk.SearchOptions
		if err := args2(a, &q, &o); err != nil {
			return nil, err
		}
		return c.Search(ctx, q, o)
	},
	"suggest": oneString(func(ctx context.Context, c *sdk.Client, s string) (any, error) { return c.Suggest(ctx, s) }),
	"hotSearch": func(ctx context.Context, c *sdk.Client, _ []json.RawMessage) (any, error) {
		return c.HotSearch(ctx)
	},
	"searchCues": func(ctx context.Context, c *sdk.Client, _ []json.RawMessage) (any, error) {
		return c.SearchCues(ctx)
	},
	"newReleases": func(ctx context.Context, c *sdk.Client, a []json.RawMessage) (any, error) {
		var q sdk.NewReleaseQuery
		if err := arg(a, 0, &q); err != nil {
			return nil, err
		}
		return c.NewReleases(ctx, q)
	},
	"getSeries": oneString(func(ctx context.Context, c *sdk.Client, id string) (any, error) { return c.GetSeries(ctx, id) }),
	"getSeriesBatch": func(ctx context.Context, c *sdk.Client, a []json.RawMessage) (any, error) {
		var ids []string
		if err := arg(a, 0, &ids); err != nil {
			return nil, err
		}
		return c.GetSeriesBatch(ctx, ids)
	},
	"listEpisodes": func(ctx context.Context, c *sdk.Client, a []json.RawMessage) (any, error) {
		var id string
		var o sdk.EpisodeOptions
		if err := args2(a, &id, &o); err != nil {
			return nil, err
		}
		return c.ListEpisodes(ctx, id, o)
	},
	"seasons":         oneString(func(ctx context.Context, c *sdk.Client, id string) (any, error) { return c.Seasons(ctx, id) }),
	"seriesRecommend": oneString(func(ctx context.Context, c *sdk.Client, id string) (any, error) { return c.SeriesRecommend(ctx, id) }),
	"getAlbum": oneString(func(ctx context.Context, c *sdk.Client, id string) (any, error) {
		return c.GetAlbum(ctx, id, sdk.AlbumOptions{})
	}),
	"shortRank": func(ctx context.Context, c *sdk.Client, a []json.RawMessage) (any, error) {
		var q sdk.ShortRankQuery
		if err := arg(a, 0, &q); err != nil {
			return nil, err
		}
		return c.ShortRank(ctx, q)
	},
	"shortFilterPanel": oneString(func(ctx context.Context, c *sdk.Client, genre string) (any, error) {
		return c.ShortFilterPanel(ctx, genre)
	}),
	"shortFilter": func(ctx context.Context, c *sdk.Client, a []json.RawMessage) (any, error) {
		var q sdk.ShortFilterQuery
		if err := arg(a, 0, &q); err != nil {
			return nil, err
		}
		return c.ShortFilter(ctx, q)
	},
	"ranking": func(ctx context.Context, c *sdk.Client, a []json.RawMessage) (any, error) {
		var activity, list string
		if err := args2(a, &activity, &list); err != nil {
			return nil, err
		}
		return c.Ranking(ctx, activity, list)
	},
	"rankRows": func(ctx context.Context, c *sdk.Client, a []json.RawMessage) (any, error) {
		var cell string
		var offset int
		if err := args2(a, &cell, &offset); err != nil {
			return nil, err
		}
		return c.RankRows(ctx, cell, offset)
	},
	"rankGroups": func(ctx context.Context, c *sdk.Client, _ []json.RawMessage) (any, error) {
		return c.RankGroups(ctx)
	},
	"person": oneString(func(ctx context.Context, c *sdk.Client, id string) (any, error) { return c.Person(ctx, id) }),
	"personWorks": func(ctx context.Context, c *sdk.Client, a []json.RawMessage) (any, error) {
		var id, cursor string
		if err := args2(a, &id, &cursor); err != nil {
			return nil, err
		}
		return c.PersonWorks(ctx, id, cursor)
	},
	"categories": oneString(func(ctx context.Context, c *sdk.Client, id string) (any, error) { return c.Categories(ctx, id) }),
	"categoryLanding": func(ctx context.Context, c *sdk.Client, a []json.RawMessage) (any, error) {
		var id string
		var o sdk.LandingOptions
		if err := args2(a, &id, &o); err != nil {
			return nil, err
		}
		return c.CategoryLanding(ctx, id, o)
	},
	"resolvePlayback": oneString(func(ctx context.Context, c *sdk.Client, id string) (any, error) {
		pb, err := c.ResolvePlayback(ctx, id, sdk.PlaybackOptions{})
		if errors.Is(err, sdk.ErrDRMProtected) && pb != nil {
			key, kerr := sdk.ContentKey(pb)
			if kerr != nil {
				return playbackResult{Playback: pb, ContentKeyError: Error(kerr)}, nil
			}
			return playbackResult{Playback: pb, ContentKey: &key}, nil
		}
		if err != nil {
			return nil, err
		}
		return playbackResult{Playback: pb}, nil
	}),
	"resetDevice": func(ctx context.Context, c *sdk.Client, _ []json.RawMessage) (any, error) {
		return nil, c.ResetDevice(ctx)
	},
}

// Methods 返回全部方法名，按字母排序。
func Methods() []string {
	names := make([]string, 0, len(methods))
	for n := range methods {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Call 调用名为 name 的方法，raw 是参数数组的 JSON（可以为空）。
// 结果可以直接编码成 JSON；ctx 取消时，进行中的请求随之取消。
func Call(ctx context.Context, c *sdk.Client, name, raw string) (any, error) {
	m, ok := methods[name]
	if !ok {
		return nil, fmt.Errorf("%w: unknown method %q", sdk.ErrInvalidOptions, name)
	}
	var a []json.RawMessage
	if raw != "" {
		if err := json.Unmarshal([]byte(raw), &a); err != nil {
			return nil, fmt.Errorf("%w: arguments: %v", sdk.ErrInvalidOptions, err)
		}
	}
	v, err := m(ctx, c, a)
	if err != nil && ctx.Err() != nil && !errors.Is(err, ctx.Err()) {
		// transport 不一定用 %w 包装取消的原因；ctx 已结束时以它为准。
		err = fmt.Errorf("%w: %v", ctx.Err(), err)
	}
	return v, err
}

// arg 把第 i 个参数解码到 v；参数缺省或为 null 时保留 v 的零值。
func arg(a []json.RawMessage, i int, v any) error {
	if i >= len(a) || string(a[i]) == "null" {
		return nil
	}
	if err := json.Unmarshal(a[i], v); err != nil {
		return fmt.Errorf("%w: argument %d: %v", sdk.ErrInvalidOptions, i+1, err)
	}
	return nil
}

func args2(a []json.RawMessage, v0, v1 any) error {
	if err := arg(a, 0, v0); err != nil {
		return err
	}
	return arg(a, 1, v1)
}

func oneString(f func(context.Context, *sdk.Client, string) (any, error)) method {
	return func(ctx context.Context, c *sdk.Client, a []json.RawMessage) (any, error) {
		var s string
		if err := arg(a, 0, &s); err != nil {
			return nil, err
		}
		return f(ctx, c, s)
	}
}

// ErrorCode 是绑定使用的一个错误码，以及它对应的错误。
type ErrorCode struct {
	Err  error
	Code string
}

// ErrorCodes 把 SDK 的哨兵错误映射成绑定使用的错误码，按匹配的先后排列。
// 文档生成器也读取它。
var ErrorCodes = []ErrorCode{
	// 取消排在最前：取消导致的失败可能同时包装了其他哨兵错误。
	{context.Canceled, "CANCELLED"},
	{sdk.ErrInvalidOptions, "INVALID_OPTIONS"},
	{sdk.ErrProfileNotFound, "PROFILE_NOT_FOUND"},
	{sdk.ErrSignerUnavailable, "SIGNER_UNAVAILABLE"},
	{sdk.ErrAuthenticationRequired, "AUTHENTICATION_REQUIRED"},
	{sdk.ErrNotProtected, "NOT_PROTECTED"},
	{sdk.ErrDRMProtected, "DRM_PROTECTED"},
	{sdk.ErrAnonymousLimited, "ANONYMOUS_LIMITED"},
	{sdk.ErrRateLimited, "RATE_LIMITED"},
	{sdk.ErrTransport, "TRANSPORT"},
	{sdk.ErrSignatureRejected, "SIGNATURE_REJECTED"},
	{sdk.ErrContentUnavailable, "CONTENT_UNAVAILABLE"},
	{sdk.ErrParamInvalid, "PARAM_INVALID"},
	{sdk.ErrChallengeRequired, "CHALLENGE_REQUIRED"},
	{sdk.ErrDeviceRejected, "DEVICE_REJECTED"},
	{sdk.ErrEndOfList, "END_OF_LIST"},
	{sdk.ErrAlbumUnavailable, "ALBUM_UNAVAILABLE"},
}

// Error 把 err 转成 {code, message, retryable, retryAfterMs}。
// 不对应任何哨兵错误时，code 是 "UNKNOWN"。
func Error(err error) map[string]any {
	code := "UNKNOWN"
	for _, e := range ErrorCodes {
		if errors.Is(err, e.Err) {
			code = e.Code
			break
		}
	}
	return map[string]any{
		"code":         code,
		"message":      err.Error(),
		"retryable":    sdk.Retryable(err),
		"retryAfterMs": sdk.RetryAfter(err, 0).Milliseconds(),
	}
}
