package protocol

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/beilunyang/openhongguo/sdk/model"
)

// DanmakuPageSize is the count the shipped player asks for in one window.
const DanmakuPageSize = 90

// DanmakuFallbackDuration stands in for an episode whose length is unknown.
const DanmakuFallbackDuration = 60 * 60 * 1000

// DanmakuBody asks for an episode's 弹幕 from startMS on; times are
// milliseconds of play time. comment_type 20 with comment_source 601 and
// server_channel 1000 selects the timed 弹幕 list (observed 2026-11-11).
//
// durationMS is the episode's length, sent as playlet_item_duration. The
// service pages in fixed windows up to it and only sometimes skips the empty
// tail itself (2026-11-11: on some devices a 135 s episode sent with an hour
// kept paging empty 30 s windows past its end), so it should be the real
// length. cursor is the service's own cursor string from the previous window;
// empty is the first.
func DanmakuBody(seriesID, episodeID string, startMS, durationMS int64, count int, cursor string) ([]byte, error) {
	if count <= 0 {
		count = DanmakuPageSize
	}
	if durationMS <= 0 {
		durationMS = DanmakuFallbackDuration
	}
	return jsonMarshal(map[string]any{
		"aid":               json.Number(AID),
		"comment_source":    601,
		"server_channel":    1000,
		"comment_type":      20,
		"compliance_status": 0,
		"count":             count,
		"cursor":            cursor,
		"group_id":          episodeID,
		"group_type":        30,
		"sort":              1,
		"business_param": map[string]any{
			"book_id":                 seriesID,
			"start_offset_time":       startMS,
			"playlet_item_duration":   durationMS,
			"need_danmaku_guide_type": []int{1, 3, 4, 2},
		},
	})
}

// DanmakuResult is one window of an episode's 弹幕.
type DanmakuResult struct {
	// Items are sorted by play time.
	Items []model.Danmaku
	// Start and End bound the window, in milliseconds of play time.
	Start, End int64
	// Total is the service's count of the episode's 弹幕; it serves only
	// some of them.
	Total int64
	// HasMore reports a later window. Next is where it starts and Cursor is
	// the service's cursor string to send with it.
	HasMore bool
	Next    int64
	Cursor  string
}

// ParseDanmaku reads one window of the 弹幕 list. Rows without text or a play
// time are skipped. The sender's identity is not read.
func ParseDanmaku(raw []byte) (*DanmakuResult, error) {
	var env struct {
		Code flexString `json:"code"`
		Data struct {
			List []struct {
				Comment *struct {
					ID     flexString `json:"comment_id"`
					Common struct {
						Content struct {
							Text string `json:"text"`
						} `json:"content"`
						Created flexInt64 `json:"create_timestamp"`
					} `json:"common"`
					Expand struct {
						Offset *flexInt64 `json:"offset_time"`
					} `json:"expand"`
					Stat struct {
						Diggs    flexInt64 `json:"digg_count"`
						PlusOnes flexInt64 `json:"add_one_count"`
					} `json:"stat"`
				} `json:"comment"`
			} `json:"data_list"`
			Info struct {
				Cursor  string `json:"cursor"`
				HasMore bool   `json:"has_more"`
			} `json:"common_list_info"`
			Extra struct {
				Next flexInt64 `json:"next_query_danmaku_list_time"`
			} `json:"extra"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, err
	}
	if env.Code.String() != "" && env.Code.String() != "0" {
		return nil, fmt.Errorf("danmaku code %s", env.Code.String())
	}
	out := &DanmakuResult{
		Items:   []model.Danmaku{},
		HasMore: env.Data.Info.HasMore,
		Next:    int64(env.Data.Extra.Next),
		Cursor:  env.Data.Info.Cursor,
	}
	// The service's cursor is a JSON object that also describes this window.
	var window struct {
		Start flexInt64 `json:"start_offset_time"`
		End   flexInt64 `json:"end_offset_time"`
		Total flexInt64 `json:"danmaku_count"`
	}
	if json.Unmarshal([]byte(out.Cursor), &window) == nil {
		out.Start, out.End, out.Total = int64(window.Start), int64(window.End), int64(window.Total)
	}
	for _, row := range env.Data.List {
		c := row.Comment
		if c == nil || c.Expand.Offset == nil || *c.Expand.Offset < 0 {
			continue
		}
		text := strings.TrimSpace(c.Common.Content.Text)
		if text == "" {
			continue
		}
		d := model.Danmaku{
			ID: c.ID.String(), Text: text, OffsetMS: int64(*c.Expand.Offset),
			Likes: int64(c.Stat.Diggs), PlusOnes: int64(c.Stat.PlusOnes),
		}
		if c.Common.Created > 0 {
			d.CreatedAt = time.Unix(int64(c.Common.Created), 0).UTC()
		}
		out.Items = append(out.Items, d)
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].OffsetMS < out.Items[j].OffsetMS })
	return out, nil
}

// DanmakuCursor continues an episode's 弹幕: the episode and its length,
// where the next window starts, and the service's cursor string.
type DanmakuCursor struct {
	EpisodeID string `json:"e"`
	Duration  int64  `json:"d"`
	Start     int64  `json:"t"`
	Next      string `json:"n,omitempty"`
}

// EncodeDanmakuCursor packs a DanmakuCursor into an opaque token.
func EncodeDanmakuCursor(c DanmakuCursor) (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return "m1." + base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeDanmakuCursor unpacks a token from EncodeDanmakuCursor. An empty token
// is the first window.
func DecodeDanmakuCursor(token string) (DanmakuCursor, error) {
	if token == "" {
		return DanmakuCursor{}, nil
	}
	bad := fmt.Errorf("danmaku cursor is not valid")
	payload, ok := strings.CutPrefix(token, "m1.")
	if !ok {
		return DanmakuCursor{}, bad
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return DanmakuCursor{}, bad
	}
	var c DanmakuCursor
	if err := json.Unmarshal(raw, &c); err != nil || !digits(c.EpisodeID) || c.Start <= 0 || c.Duration <= 0 {
		return DanmakuCursor{}, bad
	}
	return c, nil
}
