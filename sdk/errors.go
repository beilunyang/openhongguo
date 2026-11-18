package sdk

import (
	"errors"
	"fmt"
	"time"

	"github.com/beilunyang/openhongguo/internal/protocol"
)

// 以下是哨兵错误，调用方可用 errors.Is 匹配。认证与 DRM 分开处理，
// 受保护的媒体响应不会被当作登录失败；两条路径都不会解密或解锁内容。
var (
	// ErrInvalidOptions 表示客户端的设置或调用参数无效。
	ErrInvalidOptions = errors.New("openhongguo: invalid client options")
	// ErrProfileNotFound 表示还没有保存过设备档案。ProfileStore 首次读取时返回它。
	ErrProfileNotFound = errors.New("openhongguo: profile not found")
	// ErrSignerUnavailable 表示请求需要签名，而客户端没有可用的签名器，或者签名服务不可用。
	ErrSignerUnavailable = errors.New("openhongguo: signer unavailable")
	// ErrAuthenticationRequired 表示内容需要登录才能访问，匿名客户端无法继续。
	ErrAuthenticationRequired = errors.New("openhongguo: authentication required")
	// ErrDRMProtected 表示分集是加密的，需要内容密钥才能解密。
	// ResolvePlayback 返回它时，播放模型照常返回，交给 ContentKey 取得密钥。
	ErrDRMProtected = errors.New("openhongguo: DRM protected")
	// ErrNotProtected 表示为未加密的播放模型请求了内容密钥。
	ErrNotProtected = errors.New("openhongguo: playback is not encrypted")
	// ErrAnonymousLimited 表示服务端给匿名设备返回了空结果。
	ErrAnonymousLimited = errors.New("openhongguo: anonymous response was empty")
	// ErrRateLimited 表示请求太频繁，被服务端限流。
	ErrRateLimited = errors.New("openhongguo: rate limited")
	// ErrTransport 表示网络请求失败。
	ErrTransport = errors.New("openhongguo: transport failure")
	// ErrSignatureRejected 表示服务端拒绝了请求的签名。
	ErrSignatureRejected = errors.New("openhongguo: signature rejected")
	// ErrContentUnavailable 表示内容已下架、不存在，或当前调用方无权访问。
	// 服务端的错误码有 101001、101104、101109、101115 等。
	ErrContentUnavailable = errors.New("openhongguo: content unavailable")
	// ErrParamInvalid 表示服务端认为请求参数无效。
	// 服务端的错误码有 100103、4000 和 101142。
	ErrParamInvalid = errors.New("openhongguo: invalid request")
	// ErrChallengeRequired 表示服务端要求验证码或其他验证，匿名客户端无法继续。
	// 对应 111 HIT_VERIFY_CODE。
	ErrChallengeRequired = errors.New("openhongguo: verification required")
	// ErrDeviceRejected 表示匿名设备被拒绝，需要重置设备后重新注册。
	// 重置用 ResetDevice。对应 BaseResp 263175 和 263176。
	ErrDeviceRejected = errors.New("openhongguo: device rejected")
	// ErrEndOfList 表示列表已经没有更多内容。对应 101 ALL_CONTENT_SHOWN。
	ErrEndOfList = errors.New("openhongguo: no more content")
	// ErrAlbumUnavailable 表示匿名读取不到这个合集，或者这部短剧没有其他季。
	// 所有抽样的合集都返回 101001。
	ErrAlbumUnavailable = errors.New("openhongguo: album is not available anonymously")
)

// APIError 表示一次服务端失败，其文本从不包含请求头或 URL 查询参数。
type APIError struct {
	// HTTPStatus 是响应状态码。
	HTTPStatus int
	// Code 是失败所带的服务端错误码；若只是普通 HTTP 失败，客户端将其留空。
	Code    string
	Message string
}

// Error 给出状态码和错误码，但不含消息，因为消息可能回显请求细节。
func (e *APIError) Error() string {
	if e == nil {
		return "openhongguo: API error"
	}
	return fmt.Sprintf("openhongguo: API error (HTTP %d, code %s)", e.HTTPStatus, e.Code)
}

// Is 匹配 ErrRateLimited、ErrAuthenticationRequired 和 ErrSignatureRejected。
func (e *APIError) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	switch target {
	case ErrRateLimited:
		return e.HTTPStatus == 429 || e.Code == "rate_limited"
	case ErrAuthenticationRequired:
		return e.HTTPStatus == 401 || e.HTTPStatus == 403 || e.Code == "auth_required" || e.Code == "401" || e.Code == "403"
	case ErrSignatureRejected:
		return e.Code == "signature_rejected"
	default:
		return false
	}
}

// CodeError 表示一次携带业务码的服务端失败。
//
// 业务码并非全局统一：100103 在书籍 API 中表示“invalid parameter”，
// 在交易 API 中却另有含义。因此 Code 和 Kind 分开保存，调用方应按 Kind 分支。
type CodeError struct {
	// Code 是响应体中的业务码，服务端只给出消息时为 0；
	// Name 是已知的业务码符号名。
	Code    int
	Name    string
	Message string
	// Kind 是该业务码对返回它的接口而言的含义。
	Kind protocol.ErrorKind
	// HTTPStatus 和 Path 描述失败的请求；客户端目前尚未填充它们。
	HTTPStatus int
	Path       string
}

// Error 给出服务端的消息及其业务码。
func (e *CodeError) Error() string {
	if e == nil {
		return "openhongguo: service error"
	}
	name := e.Name
	if name == "" {
		name = e.Kind.String()
	}
	return fmt.Sprintf("openhongguo: %s (code %d, %s)", e.Message, e.Code, name)
}

// Is 让调用方可以用 errors.Is 匹配上面的哨兵错误。
func (e *CodeError) Is(target error) bool {
	if e == nil || target == nil {
		return false
	}
	switch target {
	case ErrAuthenticationRequired:
		return e.Kind == protocol.KindAuth
	case ErrRateLimited:
		return e.Kind == protocol.KindRateLimited
	case ErrContentUnavailable:
		return e.Kind == protocol.KindContentGone
	case ErrParamInvalid:
		return e.Kind == protocol.KindParam
	case ErrChallengeRequired:
		return e.Kind == protocol.KindChallenge
	case ErrDeviceRejected:
		return e.Kind == protocol.KindDeviceRejected
	case ErrEndOfList:
		return e.Kind == protocol.KindEndOfList
	}
	return false
}

// Retryable 报告原样重发同一请求是否可能成功。限流、
// 传输失败和服务端错误值得重试；认证、参数、内容或验证类失败则不值得。
func Retryable(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrRateLimited) || errors.Is(err, ErrTransport) {
		return true
	}
	var api *APIError
	if errors.As(err, &api) {
		return api.HTTPStatus == 0 || api.HTTPStatus >= 500
	}
	var code *CodeError
	if errors.As(err, &code) {
		return code.Kind == protocol.KindRateLimited
	}
	return false
}

// RetryAfter 建议重试 err 前应等待多久。
//
// 限流响应的消息里可能带有服务端自己的封禁窗口（SvrBanApiModel）；
// 否则使用调用方给的默认值。
func RetryAfter(err error, fallback time.Duration) time.Duration {
	var code *CodeError
	if errors.As(err, &code) {
		if minWait, maxWait, ok := protocol.BanWindow(code.Message); ok {
			if maxWait > minWait && minWait > 0 {
				return minWait + (maxWait-minWait)/2
			}
			return maxWait
		}
	}
	return fallback
}
