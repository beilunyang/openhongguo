package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestParseDanmakuReadsWindow(t *testing.T) {
	res, err := ParseDanmaku(fixture(t, "danmaku-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	// The blank row is skipped and the rest are put in play order.
	if len(res.Items) != 2 {
		t.Fatalf("items = %+v", res.Items)
	}
	first, second := res.Items[0], res.Items[1]
	if first.ID != "7100000000002101" || first.Text != "Fixture danmaku one" || first.OffsetMS != 329 ||
		first.Likes != 15 || first.PlusOnes != 7 || !first.CreatedAt.Equal(time.Unix(1787839345, 0)) {
		t.Fatalf("first = %+v", first)
	}
	if second.OffsetMS != 12500 || second.Text != "Fixture danmaku two" {
		t.Fatalf("second = %+v", second)
	}
	if res.Start != 0 || res.End != 30000 || res.Total != 0 || !res.HasMore || res.Next != 30000 {
		t.Fatalf("window = %+v", res)
	}
	var cur map[string]any
	if err := json.Unmarshal([]byte(res.Cursor), &cur); err != nil || cur["end_offset_time"] != float64(30000) {
		t.Fatalf("cursor = %q", res.Cursor)
	}
}

func TestParseDanmakuReadsTotalFromCursor(t *testing.T) {
	// Some devices get the episode's count and per-window counts in the cursor.
	raw := `{"code":0,"data":{"data_list":[],"common_list_info":{"has_more":true,
		"cursor":"{\"start_offset_time\":30001,\"end_offset_time\":60000,\"danmaku_window_count\":{\"0\":53},\"danmaku_count\":8093,\"session_id\":\"S\"}"},
		"extra":{"next_query_danmaku_list_time":60000}}}`
	res, err := ParseDanmaku([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if res.Start != 30001 || res.End != 60000 || res.Total != 8093 || res.Next != 60000 || len(res.Items) != 0 {
		t.Fatalf("res = %+v", res)
	}
}

func TestParseDanmakuSkipsRowsWithoutTime(t *testing.T) {
	raw := `{"code":0,"data":{"data_list":[{"comment":{"comment_id":"1","common":{"content":{"text":"no time"}}}},{"data_type":9}],
		"common_list_info":{"cursor":"not json","has_more":false},"extra":{}}}`
	res, err := ParseDanmaku([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Items) != 0 || res.HasMore || res.Total != 0 || res.Cursor != "not json" {
		t.Fatalf("res = %+v", res)
	}
	if _, err := ParseDanmaku([]byte(`{"code":103001,"data":{}}`)); err == nil {
		t.Fatal("error code accepted")
	}
}

func TestDanmakuBodyMatchesPlayer(t *testing.T) {
	raw, err := DanmakuBody("7100000000000101", "7100000000000201", 30000, 91000, 0, `{"session_id":"x"}`)
	if err != nil {
		t.Fatal(err)
	}
	var b struct {
		AID           int    `json:"aid"`
		CommentSource int    `json:"comment_source"`
		ServerChannel int    `json:"server_channel"`
		CommentType   int    `json:"comment_type"`
		GroupID       string `json:"group_id"`
		GroupType     int    `json:"group_type"`
		Count         int    `json:"count"`
		Cursor        string `json:"cursor"`
		Param         struct {
			BookID   string `json:"book_id"`
			Start    int64  `json:"start_offset_time"`
			Duration int64  `json:"playlet_item_duration"`
		} `json:"business_param"`
	}
	if err := json.Unmarshal(raw, &b); err != nil {
		t.Fatal(err)
	}
	if b.AID != 8662 || b.CommentSource != 601 || b.ServerChannel != 1000 || b.CommentType != 20 || b.GroupType != 30 ||
		b.GroupID != "7100000000000201" || b.Count != DanmakuPageSize || b.Cursor != `{"session_id":"x"}` {
		t.Fatalf("body = %s", raw)
	}
	if b.Param.BookID != "7100000000000101" || b.Param.Start != 30000 || b.Param.Duration != 91000 {
		t.Fatalf("business_param = %s", raw)
	}
	raw, _ = DanmakuBody("7100000000000101", "7100000000000201", 0, 0, 150, "")
	if err := json.Unmarshal(raw, &b); err != nil || b.Param.Duration != DanmakuFallbackDuration || b.Count != 150 {
		t.Fatalf("defaults = %s", raw)
	}
}

func TestDanmakuCursorRoundTrip(t *testing.T) {
	in := DanmakuCursor{EpisodeID: "7100000000000201", Duration: 91000, Start: 30000, Next: `{"session_id":"x"}`}
	token, err := EncodeDanmakuCursor(in)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := DecodeDanmakuCursor(token); err != nil || out != in {
		t.Fatalf("decode = %+v, %v", out, err)
	}
	if c, err := DecodeDanmakuCursor(""); err != nil || c != (DanmakuCursor{}) {
		t.Fatalf("empty = %+v, %v", c, err)
	}
	works, _ := EncodeWorksCursor(WorksCursor{UserID: "7100000000001701", Next: "x"})
	for _, bad := range []DanmakuCursor{
		{EpisodeID: "../x", Duration: 91000, Start: 30000},
		{EpisodeID: "7100000000000201", Duration: 91000},
		{EpisodeID: "7100000000000201", Start: 30000},
	} {
		token, _ := EncodeDanmakuCursor(bad)
		if _, err := DecodeDanmakuCursor(token); err == nil {
			t.Fatalf("bad cursor %+v accepted", bad)
		}
	}
	if _, err := DecodeDanmakuCursor(works); err == nil {
		t.Fatal("works cursor accepted")
	}
}
