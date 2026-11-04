package sdk

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/beilunyang/openhongguo/internal/protocol"

	"github.com/beilunyang/openhongguo/sdk/model"
)

const (
	defaultAPIOrigin      = "https://" + protocol.APIHost
	defaultRegisterOrigin = "https://" + protocol.RegisterHost
	// defaultDistributionOrigin 承载 /reading/distribution/ 和 activity_api。
	defaultDistributionOrigin = "https://" + protocol.DistributionHost
)

// Client 执行匿名片库读取。只要注入的 transport 和 profile store 可以并发使用，
// Client 也可以并发使用。
type Client struct {
	guestMu        sync.Mutex
	transport      Transport
	profileStore   ProfileStore
	signer         Signer
	authProvider   AuthProvider
	logger         Logger
	clock          func() time.Time
	apiOrigin      string
	registerOrigin string
	distOrigin     string
	// device 是新档案注册时使用的设备（Options.Device）。
	device model.Device
	// retryAttempts 是单个请求最多执行的次数（1 表示不重试）。
	retryAttempts int
	// deviceResets 是一次调用最多换几台新设备（0 表示不换）。
	deviceResets int
	resetter     deviceResetter
}

// NewClient 根据 opts 创建客户端。Transport 和 ProfileStore 必填；
// 其他依赖和 origin 取默认值（见 Options）。
// 缺少必填依赖或 Options.Device 不可用时，返回 ErrInvalidOptions。
// 创建时不发送请求：匿名设备在首次读取时注册。
func NewClient(opts Options) (*Client, error) {
	if opts.Transport == nil || opts.ProfileStore == nil {
		return nil, ErrInvalidOptions
	}
	if opts.AuthProvider == nil {
		opts.AuthProvider = AnonymousProvider{}
	}
	if opts.Signer == nil {
		opts.Signer = UnavailableSigner{}
	}
	if opts.Clock == nil {
		opts.Clock = time.Now
	}
	api := strings.TrimRight(opts.APIOrigin, "/")
	if api == "" {
		api = defaultAPIOrigin
	}
	reg := strings.TrimRight(opts.RegisterOrigin, "/")
	if reg == "" {
		reg = defaultRegisterOrigin
	}
	dist := strings.TrimRight(opts.DistributionOrigin, "/")
	if dist == "" {
		dist = defaultDistributionOrigin
	}
	attempts := opts.RetryAttempts
	if attempts <= 0 {
		attempts = 1
	}
	resets := defaultDeviceResets
	if opts.DeviceResets != nil {
		resets = max(*opts.DeviceResets, 0)
	}
	if err := opts.Device.Validate(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidOptions, err)
	}
	return &Client{
		device:        opts.Device.OrDefault(),
		retryAttempts: attempts,
		deviceResets:  resets,
		transport:     opts.Transport, profileStore: opts.ProfileStore,
		signer: opts.Signer, authProvider: opts.AuthProvider,
		logger: opts.Logger, clock: opts.Clock,
		apiOrigin: api, registerOrigin: reg, distOrigin: dist,
	}, nil
}

func (c *Client) do(ctx context.Context, method, origin, path string, query url.Values, headers map[string]string, body []byte) ([]byte, map[string]string, error) {
	u, err := url.Parse(origin + path)
	if err != nil {
		return nil, nil, err
	}
	if query != nil {
		u.RawQuery = query.Encode()
	}
	return c.doRaw(ctx, method, u.String(), headers, body)
}

// doRaw 发送查询参数已签名的 URL，不会重新编码该查询参数。
func (c *Client) doRaw(ctx context.Context, method, rawURL string, headers map[string]string, body []byte) ([]byte, map[string]string, error) {
	for attempt := 0; ; attempt++ {
		raw, cookies, err := c.doOnce(ctx, method, rawURL, headers, body, attempt)
		if err == nil || !retryableInLoop(err) || attempt+1 >= c.retryAttempts {
			return raw, cookies, err
		}
		wait := retryWait(err, attempt)
		if wait > maxRetryWait {
			// 封禁窗口这么长，说明服务端要求调用方退避，
			// 而不是值得睡眠等过去的短暂抖动。
			return nil, nil, fmt.Errorf("%w; the service asks to wait about %s before retrying", err, wait.Round(time.Second))
		}
		if waitErr := sleepContext(ctx, wait); waitErr != nil {
			return nil, nil, waitErr
		}
	}
}

// maxRetryWait 限定单次原地重试的等待时长。重试会重发同一个已签名请求，
// 所以这个上限也让签名时间保持新鲜。
const maxRetryWait = 10 * time.Second

// retryableInLoop 标出值得原样重发的失败。
func retryableInLoop(err error) bool { return Retryable(err) }

// retryWait 返回本次尝试的退避时长；失败带有服务端封禁窗口时，返回该窗口。
func retryWait(err error, attempt int) time.Duration {
	backoff := time.Duration(300*(1<<uint(attempt))) * time.Millisecond
	if backoff > 5*time.Second {
		backoff = 5 * time.Second
	}
	// 把并发调用方分散到窗口内（0.8x .. 1.2x）。
	if crand, e := randInt64(); e == nil {
		backoff = time.Duration(float64(backoff) * (0.8 + 0.4*float64(crand%100)/100))
	}
	return RetryAfter(err, backoff)
}

// sleepContext 等待 d，或等到 ctx 结束。
func sleepContext(ctx context.Context, wait time.Duration) error {
	if wait <= 0 {
		return nil
	}
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func (c *Client) doOnce(ctx context.Context, method, rawURL string, headers map[string]string, body []byte, attempt int) ([]byte, map[string]string, error) {
	if err := refuseCredentialHeaders(headers); err != nil {
		return nil, nil, err
	}
	if headers == nil {
		headers = map[string]string{}
	}
	if _, ok := headers["Accept"]; !ok {
		headers["Accept"] = "application/json"
	}
	req := &Request{Method: method, URL: rawURL, Headers: headers, Body: body}
	if err := c.authProvider.Apply(ctx, req); err != nil {
		return nil, nil, err
	}
	if err := refuseCredentialHeaders(req.Headers); err != nil {
		return nil, nil, err
	}
	resp, err := c.transport.Do(ctx, req)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %s", ErrTransport, protocol.RedactedURL(req.URL))
	}
	if c.logger != nil {
		c.logger.Printf("HTTP %s %s -> %d", method, protocol.RedactedURL(req.URL), statusOf(resp))
	}
	raw, err := readAnonymous(resp)
	if err != nil {
		return nil, nil, err
	}
	return raw, resp.Cookies, nil
}

func statusOf(resp *Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

func readAnonymous(resp *Response) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("%w: nil response", ErrTransport)
	}
	switch resp.StatusCode {
	case 401, 403:
		return nil, ErrAuthenticationRequired
	case 429:
		return nil, ErrRateLimited
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIError{HTTPStatus: resp.StatusCode, Message: "HTTP request failed"}
	}
	body, err := decodeBody(resp)
	if err != nil {
		return nil, err
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil, ErrAnonymousLimited
	}
	if code, message, ok := protocol.BusinessCode(body); ok && code != "0" {
		n, convErr := strconv.Atoi(code)
		if convErr != nil {
			if protocol.AuthCode(code, message) {
				return nil, ErrAuthenticationRequired
			}
			return nil, &CodeError{Message: message, Kind: protocol.Classify(0, message)}
		}
		kind := protocol.Classify(n, message)
		return nil, &CodeError{
			Code: n, Name: protocol.CodeName(n), Message: message, Kind: kind,
		}
	}
	if status := protocol.BaseStatus(body); protocol.DeviceRejected(status) {
		return nil, &CodeError{Code: status, Name: "DEVICE_REJECTED", Kind: protocol.KindDeviceRejected}
	}
	return body, nil
}

func decodeBody(resp *Response) ([]byte, error) {
	encoding := ""
	for k, v := range resp.Headers {
		if strings.EqualFold(k, "Content-Encoding") {
			encoding = strings.ToLower(v)
		}
	}
	var r io.Reader = bytes.NewReader(resp.Body)
	switch encoding {
	case "gzip":
		zr, err := gzip.NewReader(r)
		if err != nil {
			return nil, err
		}
		defer zr.Close()
		r = zr
	case "":
	default:
		return nil, fmt.Errorf("%w: unsupported content encoding", ErrTransport)
	}
	return io.ReadAll(io.LimitReader(r, 8<<20))
}

func refuseCredentialHeaders(headers map[string]string) error {
	for name := range headers {
		for _, banned := range protocol.CredentialHeaderNames {
			if strings.EqualFold(name, banned) {
				return fmt.Errorf("%w: credential header refused", ErrAuthenticationRequired)
			}
		}
	}
	return nil
}

func (c *Client) ensureGuest(ctx context.Context) (*DeviceProfile, error) {
	return c.loadOrRegister(ctx, false)
}

// ensureSearchGuest 在已保存的游客没有 ttreq 时重新注册。
// 注册响应设置了 cookie 却缺少 ttreq，视为错误。注册响应不带 cookie 时，
// ticket 留空，这样 fixture 调用无需游客 cookie 也能搜索。
func (c *Client) ensureSearchGuest(ctx context.Context) (*DeviceProfile, error) {
	return c.loadOrRegister(ctx, true)
}

func (c *Client) loadOrRegister(ctx context.Context, needTicket bool) (*DeviceProfile, error) {
	c.guestMu.Lock()
	defer c.guestMu.Unlock()
	p, err := c.profileStore.Load(ctx)
	if err != nil && !errors.Is(err, ErrProfileNotFound) {
		return nil, err
	}
	if p != nil && p.Device == nil {
		// 在保存设备之前创建的档案，从此固定使用客户端的设备，
		// 这样它的机型不会在不同运行之间变化。
		d := c.device
		p.Device = &d
		if err = c.profileStore.Save(ctx, p); err != nil {
			return nil, err
		}
	}
	if guestReady(p) && (!needTicket || p.TTReq != "") {
		return p, nil
	}
	if p == nil {
		p, err = NewAnonymousDeviceProfile(nil, c.clock)
		if err != nil {
			return nil, err
		}
		d := c.device
		p.Device = &d
		if err = c.profileStore.Save(ctx, p); err != nil {
			return nil, err
		}
	}
	body, err := protocol.BuildRegisterBody(protocol.RegisterIdentity{
		CDID: p.CDID, OpenUDID: p.OpenUDID, ClientUDID: p.ClientUDID, OAID: p.OAID, ReqID: p.ReqID,
		Device: *p.Device,
	}, c.clock())
	if err != nil {
		return nil, err
	}
	if len(body) < 6 || body[0] != 0x74 || body[1] != 0x63 {
		return nil, ErrSignerUnavailable
	}
	q := url.Values{}
	// bdinstall.a1.k 先加上 req_id，加密的注册请求体再追加 tt_data=a。
	q.Set("req_id", p.ReqID)
	q.Set("tt_data", "a")
	headers := map[string]string{
		"User-Agent":                protocol.UserAgent(*p.Device),
		"Content-Type":              "application/octet-stream;tt-data=a",
		"log-encode-type":           "gzip",
		"sdk-version":               "2",
		"passport-sdk-version":      "5051452",
		"x-vc-bdturing-sdk-version": "4.0.3.cn",
		"x-ss-dp":                   protocol.AID,
	}
	// X-SS-STUB 是请求体的可移植 MD5。它会被发送，但从不写入日志。
	headers["X-SS-STUB"] = protocol.ContentMD5(body)
	raw, cookies, err := c.do(ctx, "POST", c.registerOrigin, protocol.PathRegister, q, headers, body)
	if err != nil {
		return nil, err
	}
	registered, err := protocol.ParseRegister(raw)
	if err != nil {
		return nil, err
	}
	p.DeviceID = registered.DeviceID
	p.InstallID = registered.InstallID
	p.DeviceToken = ""
	applyGuestCookies(p, cookies)
	if needTicket && len(cookies) > 0 && p.TTReq == "" {
		return nil, fmt.Errorf("%w: guest ticket missing", ErrAnonymousLimited)
	}
	if err := c.profileStore.Save(ctx, p); err != nil {
		return nil, err
	}
	return p, nil
}

func guestReady(p *DeviceProfile) bool {
	return p != nil && p.DeviceID != "" && p.InstallID != "" && p.DeviceID != "0" && p.InstallID != "0"
}

func applyGuestCookies(p *DeviceProfile, cookies map[string]string) {
	if p == nil || len(cookies) == 0 {
		return
	}
	if v := cookies["ttreq"]; v != "" {
		p.TTReq = v
	}
	if v := cookies["store-region"]; v != "" {
		p.StoreRegion = v
	}
	if v := cookies["store-region-src"]; v != "" {
		p.StoreRegionSrc = v
	}
}

func guestCookie(p *DeviceProfile) string {
	return "passport_csrf_token=; passport_csrf_token_default=; store-region=" + p.StoreRegion +
		"; store-region-src=" + p.StoreRegionSrc + "; install_id=" + p.InstallID + "; ttreq=" + p.TTReq
}

func (c *Client) randomInt31() int32 {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 1
	}
	return int32(binary.BigEndian.Uint32(b[:]) & 0x7fffffff)
}

// randomUUID 返回 v4 UUID 字符串，用作 session_uuid 查询参数。
func randomUUID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "00000000-0000-4000-8000-000000000000"
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

// randInt64 返回一个非负随机数，用于抖动。
func randInt64() (int64, error) {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return 0, err
	}
	v := int64(binary.BigEndian.Uint64(b[:]) & 0x7fffffffffffffff)
	return v, nil
}

// identity 返回 p 的设备查询身份：注册所得的 id 和注册时使用的设备。
func (c *Client) identity(p *DeviceProfile) protocol.SearchIdentity {
	return protocol.SearchIdentity{
		DeviceID: p.DeviceID, InstallID: p.InstallID, CDID: p.CDID, OpenUDID: p.OpenUDID,
		Device: c.deviceOf(p),
	}
}

// deviceOf 返回 p 注册时使用的设备；档案未指定设备时，返回客户端的设备。
func (c *Client) deviceOf(p *DeviceProfile) model.Device {
	if p != nil && p.Device != nil {
		return p.Device.OrDefault()
	}
	return c.device
}
