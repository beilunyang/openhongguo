package sign

import (
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// PlaybackV5Signer 签名 V5 播放请求，可并发使用。
type PlaybackV5Signer struct{}

// NewPlaybackV5Signer 返回播放签名器。x-medusa 里的进程信息由本包统一维护，
// nowMillis 不再使用，保留参数是为了兼容调用方。
func NewPlaybackV5Signer(nowMillis int64) *PlaybackV5Signer {
	return &PlaybackV5Signer{}
}

// Sign 对实际发送的 gzip 请求体签名。查询串不添加 ts，选择分支时只调整 _rticket。
// pairs 里已有 _rticket 时就地填写（保持客户端的参数顺序），否则追加在末尾。
func (s *PlaybackV5Signer) Sign(pairs [][2]string, nowMillis int64, body []byte, entropy *Entropy) (string, map[string]string, error) {
	ts := uint32(nowMillis / 1000)
	bodyMD5 := [16]byte{}
	if body != nil {
		bodyMD5 = md5Raw(body)
	}
	withTicket := append([][2]string(nil), pairs...)
	slot := -1
	for i, kv := range withTicket {
		if kv[0] == "_rticket" {
			slot = i
			break
		}
	}
	if slot < 0 {
		withTicket = append(withTicket, [2]string{"_rticket", ""})
		slot = len(withTicket) - 1
	}
	for offset := int64(0); offset < 1024; offset++ {
		withTicket[slot][1] = strconv.FormatInt(nowMillis+offset, 10)
		query := encodeQuery(withTicket)
		medusaIn, ok := newMedusaV5Input(query, body, ts)
		if !ok {
			continue
		}
		ent := Entropy{}
		if entropy == nil {
			ent = drawEntropy(nowMillis)
		} else {
			ent = *entropy
		}
		headers := map[string]string{
			"x-khronos": strconv.FormatUint(uint64(ts), 10),
			"x-helios":  heliosV5(ts, ent.HeliosRandom),
			"x-medusa":  assembleMedusaV5(medusaIn, queryValue(query, "device_id"), requestMedusaEnv(query, queryValue(query, "device_id")), ent.medusaRandom()),
		}
		if body != nil {
			headers["Content-Type"] = "application/json; charset=utf-8"
			headers["x-ss-stub"] = strings.ToUpper(hex.EncodeToString(bodyMD5[:]))
		}
		return query, headers, nil
	}
	return "", nil, fmt.Errorf("playback V5 signing branch unavailable")
}
