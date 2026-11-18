package sdk

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// signatureHeaders 是签名服务负责生成的请求头，全部小写，和 SDK 一致。
var signatureHeaders = []string{"x-gorgon", "x-medusa", "x-helios", "x-khronos", "x-argus", "x-ladon"}

// remoteSigner 包住 Transport：带签名头的请求发出前，交给 openhongguo-signer
// 重新签名（见 Options.SignerURL）。签名服务的请求也经过下层 Transport，
// 所以自定义的 Transport（例如 Node.js 绑定的 JS 回调）同样适用。
type remoteSigner struct {
	next     Transport
	endpoint string
	token    string
}

func newRemoteSigner(next Transport, endpoint, token string) *remoteSigner {
	return &remoteSigner{next: next, endpoint: strings.TrimRight(endpoint, "/"), token: token}
}

// Do 为带签名头的请求重新签名，再交给下层 Transport 发送。签名失败时
// 不发送请求，返回的错误包含 ErrSignerUnavailable 和签名服务给出的原因。
func (t *remoteSigner) Do(ctx context.Context, req *Request) (*Response, error) {
	if req != nil && hasHeader(req.Headers, "x-medusa") {
		if err := t.sign(ctx, req); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrSignerUnavailable, err)
		}
	}
	return t.next.Do(ctx, req)
}

func (t *remoteSigner) sign(ctx context.Context, req *Request) error {
	var payload bytes.Buffer
	enc := json.NewEncoder(&payload)
	// 保留 URL 里的 &，不转成 &。
	enc.SetEscapeHTML(false)
	in := map[string]string{"url": req.URL}
	if len(req.Body) > 0 {
		in["body_hex"] = hex.EncodeToString(req.Body)
	}
	if err := enc.Encode(in); err != nil {
		return err
	}
	headers := map[string]string{
		"Content-Type": "application/json",
		// Transport 不解压响应；要求签名服务（和它前面的 caddy）不压缩。
		"Accept-Encoding": "identity",
	}
	if t.token != "" {
		// Transport 拒绝发送 Authorization 头，签名服务也认这个头。
		headers["X-Signer-Token"] = t.token
	}
	resp, err := t.next.Do(ctx, &Request{
		Method: "POST", URL: t.endpoint + "/feature-hash", Headers: headers, Body: payload.Bytes(),
	})
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return fmt.Errorf("openhongguo-signer: %v", err)
	}
	if resp == nil {
		return errors.New("openhongguo-signer: no response")
	}
	if resp.StatusCode != 200 {
		var e struct {
			Error   string `json:"error"`
			Neptune string `json:"neptune"`
		}
		_ = json.Unmarshal(resp.Body, &e)
		switch {
		case e.Error == "":
			return fmt.Errorf("openhongguo-signer: HTTP %d", resp.StatusCode)
		case e.Neptune != "":
			return fmt.Errorf("%s (X-Neptune %s)", e.Error, e.Neptune)
		default:
			return errors.New(e.Error)
		}
	}
	var out struct {
		Headers map[string]string `json:"headers"`
	}
	if err := json.Unmarshal(resp.Body, &out); err != nil {
		return fmt.Errorf("openhongguo-signer: bad response: %v", err)
	}
	got := make(map[string]string, len(out.Headers))
	for k, v := range out.Headers {
		got[strings.ToLower(k)] = v
	}
	if got["x-medusa"] == "" {
		return errors.New("openhongguo-signer: no signature returned")
	}
	for name := range req.Headers {
		for _, h := range signatureHeaders {
			if strings.EqualFold(name, h) {
				delete(req.Headers, name)
			}
		}
	}
	for _, h := range signatureHeaders {
		if v := got[h]; v != "" {
			req.Headers[h] = v
		}
	}
	return nil
}

func hasHeader(headers map[string]string, name string) bool {
	for k := range headers {
		if strings.EqualFold(k, name) {
			return true
		}
	}
	return false
}
