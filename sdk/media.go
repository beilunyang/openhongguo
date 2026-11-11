package sdk

import (
	"context"
	"fmt"

	"github.com/beilunyang/openhongguo/internal/drm"
	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/internal/protocol/sign"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// PlaybackOptions 预留给码流规格或接口的选择。
// 第二阶段按服务端下发的原样解析模型，由调用方挑选流。
type PlaybackOptions struct{}

// ResolvePlayback 加载一集的播放模型：各清晰度的地址、编码和码率。
//
// 请求就是官方客户端发出的、带签名的 video_model 调用。
// 响应是客户端的 TTVideoEngine 模型。受保护的模型会和 ErrDRMProtected
// 一起返回；把它传给 ContentKey 取得密钥。
// 模型携带的是包装后的密钥材料（Playback.KeyMaterial），从不包含内容密钥本身、
// 分片校验数据或解密后的媒体。
func (c *Client) ResolvePlayback(ctx context.Context, episodeID string, _ PlaybackOptions) (*model.Playback, error) {
	return withFreshDevice(ctx, c, true, func(ctx context.Context) (*model.Playback, error) {
		if !validID(episodeID) {
			return nil, fmt.Errorf("invalid episode ID")
		}
		p, err := c.ensureSearchGuest(ctx)
		if err != nil {
			return nil, err
		}
		body, err := protocol.PlaybackBody(episodeID)
		if err != nil {
			return nil, err
		}
		ticket := c.clock().UnixMilli()
		// Playback is signed with the full Argus/Ladon envelope (and without
		// Medusa); the other endpoints keep using SignPOST / SignGET. See
		// internal/protocol/sign/playback.go.
		rawQuery, headers := sign.SignPlaybackPOST(protocol.PathPlayback, protocol.DeviceQueryPairs(c.identity(p)), ticket, p.DeviceID, protocol.VersionName, body, nil)
		raw, _, err := c.doRaw(ctx, "POST", c.apiOrigin+protocol.PathPlayback+"?"+rawQuery, anonymousHeaders(c, p, ticket, headers), body)
		if err != nil {
			return nil, err
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
