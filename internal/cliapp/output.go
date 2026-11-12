package cliapp

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"unicode"

	"github.com/beilunyang/openhongguo/internal/history"
	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// choices lists ranking contents as id=name, marking the selected one.
func choices(contents []model.RankContent, selected string) string {
	parts := make([]string, 0, len(contents))
	for _, c := range contents {
		mark := ""
		if c.ID == selected {
			mark = "*"
		}
		parts = append(parts, c.ID+"="+c.Name+mark)
	}
	return strings.Join(parts, " ")
}

func boardChoices(boards []model.RankBoard, selected string) string {
	parts := make([]string, 0, len(boards))
	for _, b := range boards {
		mark := ""
		if b.ID == selected {
			mark = "*"
		}
		parts = append(parts, b.ID+"="+b.Name+mark)
	}
	return strings.Join(parts, " ")
}

// oneLine folds s onto one line for text output. Control and bidi characters
// become spaces first: 弹幕 are written by anyone, and an escape sequence in
// one must not reach the terminal.
func oneLine(s string) string {
	s = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
			return ' '
		}
		return r
	}, s)
	return strings.Join(strings.Fields(s), " ")
}

// streamName labels one representation for text output.
func streamName(s model.Stream) string {
	for _, name := range []string{s.Definition, s.Quality, s.Codec} {
		if name != "" {
			return name
		}
	}
	return "stream"
}

func printValue(w io.Writer, v any, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return enc.Encode(jsonView(v))
	}
	switch x := v.(type) {
	case *sdk.Page[model.SeriesSummary]:
		for _, s := range x.Items {
			fmt.Fprintf(w, "%s\t%s\t%d\n", s.ID, s.Title, s.EpisodeCount)
		}
		if x.Cursor != "" {
			fmt.Fprintf(w, "cursor\t%s\n", x.Cursor)
		}
	case *sdk.Page[model.Episode]:
		for _, e := range x.Items {
			state := ""
			if !e.Playable {
				state = "\tlocked"
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%d%s\n", e.Number, e.ID, e.Title, int(e.Duration.Seconds()), state)
		}
	case *model.ShortRanking:
		for _, c := range x.Contents {
			if c.ID == x.Content {
				fmt.Fprintf(w, "content\t%s\n", choices(x.Contents, c.ID))
				fmt.Fprintf(w, "board\t%s\n", boardChoices(c.Boards, x.Board))
			}
		}
		for _, item := range x.Items {
			if item.ActorID != "" {
				// An actor row: name, 人气, the details line, and the move.
				move := ""
				if item.Change != nil {
					move = "\t" + item.Change.Kind.String()
					if item.Change.Places > 0 {
						move += fmt.Sprintf(" %d", item.Change.Places)
					}
				}
				// The user ID is what ohg person takes; the actor ID only
				// stands in when the row has none.
				id := item.UserID
				if id == "" {
					id = item.ActorID
				}
				fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s%s\n", item.Position, id, item.Actor, item.Score, strings.Join(item.Details, " · "), move)
				continue
			}
			// Position is the order on the page; the service's rank repeats
			// around promoted rows.
			mark := ""
			if item.Promoted {
				mark = "\tpromoted"
			}
			fmt.Fprintf(w, "%d\t%s\t%s\t%d\t%s%s\n", item.Position, item.SeriesID, item.Title, item.EpisodeCount, item.Score, mark)
		}
		if x.Cursor != "" {
			fmt.Fprintf(w, "cursor\t%s\n", x.Cursor)
		}
	case *model.Ranking:
		if x.ListName != "" {
			fmt.Fprintf(w, "list\t%s\tstage=%d\tcan_vote=%t\n", x.ListName, x.ActivityStage, x.CanVote)
		}
		for _, l := range x.Lists {
			fmt.Fprintf(w, "rank_list\t%s\t%s\ttype=%d\tsize=%d\n", l.ID, l.Name, l.Type, l.Size)
		}
		for _, item := range x.Items {
			fmt.Fprintf(w, "%d\t%s\t%d\t%s\n", item.Rank, item.Title, item.Votes, item.Actor)
		}
	case []model.FilterRow:
		for _, r := range x {
			parts := make([]string, 0, len(r.Options))
			for _, o := range r.Options {
				parts = append(parts, o.Name+"="+o.Value)
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", r.Key, r.Name, strings.Join(parts, " "))
		}
	case *model.ShortFilterPage:
		for _, item := range x.Items {
			fmt.Fprintf(w, "%s\t%s\t%d\n", item.ID, item.Title, item.EpisodeCount)
		}
		if x.Cursor != "" {
			fmt.Fprintf(w, "cursor\t%s\n", x.Cursor)
		}
	case *model.Recommend:
		if x.CellTitle != "" {
			fmt.Fprintf(w, "cell\t%s\n", x.CellTitle)
		}
		for _, item := range x.Items {
			fmt.Fprintf(w, "%s\t%s\t%d\n", item.ID, item.Title, item.EpisodeCount)
		}
	case *model.CategoryPage:
		if x.CategoryName != "" {
			fmt.Fprintf(w, "category\t%s\t%s\n", x.CategoryID, x.CategoryName)
		}
		if len(x.Tabs) > 0 {
			parts := make([]string, 0, len(x.Tabs))
			for _, t := range x.Tabs {
				parts = append(parts, t.Name+"="+t.Value)
			}
			fmt.Fprintf(w, "tabs\t%s\tdefault=%d\n", strings.Join(parts, " "), x.DefaultTab)
		}
		for _, group := range x.Filters {
			parts := make([]string, 0, len(group.Options))
			for _, o := range group.Options {
				parts = append(parts, o.Name+"="+o.Value)
			}
			fmt.Fprintf(w, "filter\t%s\t%s\n", group.Key, strings.Join(parts, " "))
		}
		for _, item := range x.Items {
			fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", item.ID, item.Title, item.Author, item.Words)
		}
		if x.HasMore {
			fmt.Fprintf(w, "cursor\t%d\n", x.NextOffset)
		}
	case *model.Album:
		fmt.Fprintf(w, "%s\t%s\t%d seasons\n", x.ID, x.Title, x.SeasonCount)
		for _, season := range x.Seasons {
			fmt.Fprintf(w, "season\t%d\t%s\t%s\t%d\t%s\n",
				season.Index, season.SeriesID, season.Title, season.EpisodeCount, season.Status)
		}
	case *model.Series:
		fmt.Fprintf(w, "%s\t%s\t%d\n%s\n", x.ID, x.Title, x.EpisodeCount, oneLine(x.Description))
		if x.CoverURL != "" {
			fmt.Fprintf(w, "cover\t%s\n", x.CoverURL)
		}
		fmt.Fprintf(w, "kind\t%s\t%s\n", x.ContentType, x.Platform)
		fmt.Fprintf(w, "status\t%s\tpay\t%s\n", x.Status, x.PayType)
		if x.RightText != "" {
			fmt.Fprintf(w, "badge\t%s\n", x.RightText)
		}
		if x.Heat > 0 || x.PlayCount > 0 || x.FollowCount > 0 || x.DiggCount > 0 {
			fmt.Fprintf(w, "counters\theat=%d\tplay=%d\tfollow=%d\tdigg=%d\n",
				x.Heat, x.PlayCount, x.FollowCount, x.DiggCount)
		}
		if !x.CreatedAt.IsZero() {
			fmt.Fprintf(w, "created\t%s\n", x.CreatedAt.Format("2006-01-02"))
		}
		for _, row := range x.Rows {
			fmt.Fprintf(w, "row\t%s\t%s\n", row.Kind, row.Text)
		}
		if n := x.Novel; n != nil {
			fmt.Fprintf(w, "novel\t%s\t%s\t%s\t%s\n", n.BookID, n.Title, n.Status, strings.Join(n.Tags, ","))
		}
	case *model.Playback:
		fmt.Fprintf(w, "episode\t%s\n", x.EpisodeID)
		if x.VideoID != "" {
			fmt.Fprintf(w, "video\t%s\n", x.VideoID)
		}
		if x.DurationSeconds > 0 {
			fmt.Fprintf(w, "duration\t%d\n", x.DurationSeconds)
		}
		fmt.Fprintf(w, "protected\t%t\n", x.Protected)
		for _, s := range x.Streams {
			fmt.Fprintf(w, "%s\t%s\t%d\t%t\t%s\n", streamName(s), s.Codec, s.Size, s.Protected, s.URL)
		}
	case *sdk.DeviceProfile:
		fmt.Fprintf(w, "device_id=%s\ninstall_id=%s\n", x.DeviceID, x.InstallID)
	case []history.Entry:
		for _, e := range x {
			state := "watching"
			if e.Finished {
				state = "finished"
			}
			fmt.Fprintf(w, "%s\t%s\t%d/%d\t%s\t%s\n", e.SeriesID, e.Title, e.Episode, e.Episodes, state,
				e.UpdatedAt.Local().Format("2006-01-02 15:04"))
		}
	default:
		return fmt.Errorf("unsupported output type %T", v)
	}
	return nil
}

func jsonView(v any) any {
	switch x := v.(type) {
	case *sdk.Page[model.Episode]:
		items := make([]map[string]any, 0, len(x.Items))
		for _, e := range x.Items {
			items = append(items, map[string]any{
				"id": e.ID, "series_id": e.SeriesID, "number": e.Number, "title": e.Title,
				"duration_seconds": int(e.Duration.Seconds()), "playable": e.Playable,
				"vertical": e.Vertical, "is_new": e.IsNew, "cover_url": e.CoverURL,
				"content_type": e.ContentType.String(), "comment_count": e.CommentCount,
				"digg_count": e.DiggCount,
			})
		}
		return map[string]any{"items": items, "has_more": x.HasMore}
	case *sdk.DeviceProfile:
		return map[string]any{"device_id": x.DeviceID, "install_id": x.InstallID}
	default:
		return v
	}
}

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
