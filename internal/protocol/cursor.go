package protocol

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// EncodeFeedCursor packs a cell/change continuation. The session stays inside
// the token so a later page can send it without being written to the profile.
func EncodeFeedCursor(next FeedContinuation) (string, error) {
	offset, err := strconv.Atoi(next.Offset)
	if err != nil || offset <= 0 || next.CellID == "" || next.SessionID == "" || next.TabType == "" || next.ShowType == "" {
		return "", fmt.Errorf("discover cursor is not verified")
	}
	limit := CellChangeLimit(next.Limit)
	raw, err := json.Marshal(feedCursor{
		Cell: next.CellID, Session: next.SessionID, Template: next.ClientTemplate,
		Show: next.ShowType, Tab: next.TabType, Offset: offset, Limit: limit,
	})
	if err != nil {
		return "", err
	}
	return "d1." + base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeFeedCursor unpacks a cursor from EncodeFeedCursor.
func DecodeFeedCursor(token string) (FeedContinuation, error) {
	payload, ok := strings.CutPrefix(token, "d1.")
	if !ok || payload == "" {
		return FeedContinuation{}, fmt.Errorf("discover cursor is not verified")
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return FeedContinuation{}, fmt.Errorf("discover cursor is not verified")
	}
	var cur feedCursor
	if err := json.Unmarshal(raw, &cur); err != nil || cur.Offset <= 0 {
		return FeedContinuation{}, fmt.Errorf("discover cursor is not verified")
	}
	if !digits(cur.Cell) || !digits(cur.Tab) || !digits(cur.Show) || cur.Session == "" || strings.ContainsAny(cur.Session, "\r\n") {
		return FeedContinuation{}, fmt.Errorf("discover cursor is not verified")
	}
	next := FeedContinuation{
		CellID: cur.Cell, SessionID: cur.Session, ClientTemplate: cur.Template,
		ShowType: cur.Show, TabType: cur.Tab, Offset: strconv.Itoa(cur.Offset),
		Limit: cur.Limit,
	}
	return next, nil
}

type feedCursor struct {
	Cell     string `json:"cell"`
	Session  string `json:"session"`
	Template string `json:"template,omitempty"`
	Show     string `json:"show"`
	Tab      string `json:"tab"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit,omitempty"`
}

// ShortCursor continues a short-drama filter, ranking, or 上新 list: the next
// offset, the series already shown (sent back as filter_ids so the service
// skips them), the service's session, and the query the list answers, so a
// cursor alone resumes the same list.
type ShortCursor struct {
	Offset  int      `json:"o"`
	Seen    []string `json:"s,omitempty"`
	Session string   `json:"x,omitempty"`
	// Selections is a filter list's query; Content and Board a ranking's.
	Selections map[string][]string `json:"q,omitempty"`
	Content    string              `json:"c,omitempty"`
	Board      string              `json:"b,omitempty"`
	// Date is the 上新 calendar day (YYYYMMDD) the list answers.
	Date string `json:"d,omitempty"`
}

// EncodeShortCursor packs a ShortCursor into an opaque token.
func EncodeShortCursor(c ShortCursor) (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return "s1." + base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeShortCursor unpacks a token from EncodeShortCursor. An empty token is
// the first page.
func DecodeShortCursor(token string) (ShortCursor, error) {
	if token == "" {
		return ShortCursor{}, nil
	}
	bad := fmt.Errorf("short-drama cursor is not valid")
	payload, ok := strings.CutPrefix(token, "s1.")
	if !ok {
		return ShortCursor{}, bad
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return ShortCursor{}, bad
	}
	var c ShortCursor
	if err := json.Unmarshal(raw, &c); err != nil || c.Offset < 0 || strings.ContainsAny(c.Session, "\r\n") {
		return ShortCursor{}, bad
	}
	if c.Date != "" && !ValidDate(c.Date) {
		return ShortCursor{}, bad
	}
	for _, id := range c.Seen {
		if !digits(id) {
			return ShortCursor{}, bad
		}
	}
	return c, nil
}

func digits(s string) bool {
	if s == "" || len(s) > 25 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// WorksCursor continues a person's works: the person and the service's own
// cursor string.
type WorksCursor struct {
	UserID string `json:"u"`
	Next   string `json:"n"`
}

// EncodeWorksCursor packs a WorksCursor into an opaque token.
func EncodeWorksCursor(c WorksCursor) (string, error) {
	raw, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	return "w1." + base64.RawURLEncoding.EncodeToString(raw), nil
}

// DecodeWorksCursor unpacks a token from EncodeWorksCursor. An empty token is
// the first page.
func DecodeWorksCursor(token string) (WorksCursor, error) {
	if token == "" {
		return WorksCursor{}, nil
	}
	bad := fmt.Errorf("works cursor is not valid")
	payload, ok := strings.CutPrefix(token, "w1.")
	if !ok {
		return WorksCursor{}, bad
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return WorksCursor{}, bad
	}
	var c WorksCursor
	if err := json.Unmarshal(raw, &c); err != nil || !digits(c.UserID) || c.Next == "" {
		return WorksCursor{}, bad
	}
	return c, nil
}

// ValidDate reports whether s is a YYYYMMDD calendar day.
func ValidDate(s string) bool {
	if len(s) != 8 || !digits(s) {
		return false
	}
	_, err := time.Parse("20060102", s)
	return err == nil
}
