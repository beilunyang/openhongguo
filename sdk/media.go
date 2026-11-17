package sdk

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/beilunyang/openhongguo/internal/drm"
	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/internal/protocol/sign"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// PlaybackOptions 预留给码流规格或接口的选择。
type PlaybackOptions struct{}

// ResolvePlayback 加载一集的各清晰度地址、编码和码率。
// 优先使用 V5 手机端协议，空响应或接口不支持时兼容旧播放接口。
// 受保护的模型与 ErrDRMProtected 一起返回，可继续交给 ContentKey。
func (c *Client) ResolvePlayback(ctx context.Context, episodeID string, _ PlaybackOptions) (*model.Playback, error) {
	if !validID(episodeID) {
		return nil, fmt.Errorf("invalid episode ID")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	body, err := protocol.PlaybackV5Body(episodeID)
	if err != nil {
		return nil, err
	}
	return withFreshDevice(ctx, c, true, func(ctx context.Context) (*model.Playback, error) {
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		var raw []byte
		for attempt := 0; attempt < 2; attempt++ {
			if err = ctx.Err(); err != nil {
				return nil, err
			}
			var query string
			var headers map[string]string
			query, headers, err = c.playbackSigner.Sign(protocol.PlaybackV5Pairs(c.identity(p)), c.clock().UnixMilli(), body, nil)
			if err != nil {
				return nil, err
			}
			headers = anonymousHeaders(c, p, c.clock().UnixMilli(), headers)
			headers["User-Agent"] = protocol.PlaybackUserAgent(c.deviceOf(p))
			headers["Accept"] = "application/json; charset=utf-8,application/x-protobuf"
			headers["x-xs-from-web"] = "0"
			headers["Content-Encoding"] = "gzip"
			raw, _, err = c.doRaw(ctx, "POST", c.apiOrigin+protocol.PathPlaybackV5+"?"+query, headers, body)
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			if !errors.Is(err, ErrAnonymousLimited) || attempt == 1 {
				break
			}
			if err = sleepContext(ctx, 300*time.Millisecond); err != nil {
				return nil, err
			}
		}
		if err != nil {
			if !errors.Is(err, ErrAnonymousLimited) && !playbackRouteUnavailable(err) {
				return nil, err
			}
			v5Err := err
			legacyBody, bodyErr := protocol.PlaybackBody(episodeID)
			if bodyErr != nil {
				return nil, bodyErr
			}
			ticket := c.clock().UnixMilli()
			query, headers := sign.SignPlaybackPOST(protocol.PathPlayback, protocol.DeviceQueryPairs(c.identity(p)), ticket, p.DeviceID, protocol.VersionName, legacyBody, nil)
			raw, _, err = c.doRaw(ctx, "POST", c.apiOrigin+protocol.PathPlayback+"?"+query, anonymousHeaders(c, p, ticket, headers), legacyBody)
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			if err != nil {
				if playbackRouteUnavailable(err) {
					return nil, v5Err
				}
				return nil, err
			}
		}
		playback, err := protocol.ParsePlayback(raw, episodeID)
		if err != nil {
			return nil, err
		}
		if playback.Protected {
			return playback, ErrDRMProtected
		}
		return playback, nil
	})
}

func playbackRouteUnavailable(err error) bool {
	var api *APIError
	return errors.As(err, &api) && (api.HTTPStatus == 404 || api.HTTPStatus == 405)
}

// ContentKey 从播放模型算出加密分集的内容密钥。
// 播放模型就是 ResolvePlayback 连同 ErrDRMProtected 一起返回的那个。这是纯本地计算：
// 按官方播放器的方式从模型的 KeyMaterial 解包出密钥，不再发出任何请求。
// 模型的所有流共用这个密钥（同一集的每种清晰度都带有相同的 spade_a 和 kid，
// 2026-10-24 观察）。
//
// 未加密的模型返回 ErrNotProtected；当前版本无法解包的密钥材料返回
// ErrDRMProtected。
func ContentKey(pb *model.Playback) (model.ContentKey, error) {
	if pb == nil {
		return model.ContentKey{}, fmt.Errorf("playback model is missing")
	}
	if !pb.Protected {
		return model.ContentKey{}, ErrNotProtected
	}
	k, err := drm.Derive(pb.KeyMaterial)
	if err != nil {
		return model.ContentKey{}, fmt.Errorf("%w: %v", ErrDRMProtected, err)
	}
	return model.ContentKey{Hex: k.Hex, Version: k.Version}, nil
}
