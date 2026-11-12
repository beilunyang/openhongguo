package cliapp

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// maxDanmakuPages stops a list that never ends. An hour of 30-second windows
// is 120 pages.
const maxDanmakuPages = 500

// danmakuList is the --json output of danmaku: every page of one episode.
type danmakuList struct {
	SeriesID  string          `json:"series_id"`
	EpisodeID string          `json:"episode_id"`
	Episode   int             `json:"episode"`
	Total     int64           `json:"total,omitempty"`
	Items     []model.Danmaku `json:"items"`
}

// cmdDanmaku prints every 弹幕 of one episode in play order.
func cmdDanmaku(ctx context.Context, args []string, d Dependencies) error {
	if err := requireService(d); err != nil {
		return err
	}
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(f.rest) != 1 {
		return usage("danmaku needs one series ID or title")
	}
	id, _, err := resolveSeries(ctx, d, f.rest[0], f.pick)
	if err != nil {
		return err
	}
	episodes, err := d.Service.ListEpisodes(ctx, id, sdk.EpisodeOptions{})
	if err != nil {
		return err
	}
	episode, ok := findEpisode(episodes.Items, f.episode)
	if !ok {
		return usage("series %s has no episode %d", id, f.episode)
	}
	out := danmakuList{SeriesID: id, EpisodeID: episode.ID, Episode: episode.Number, Items: []model.Danmaku{}}
	seen := map[string]bool{}
	cursor := ""
	for pages := 0; ; pages++ {
		if pages == maxDanmakuPages {
			return errors.New("danmaku did not end after 500 pages")
		}
		page, err := d.Service.ListDanmaku(ctx, id, episode.ID, sdk.DanmakuOptions{Cursor: cursor})
		if err != nil {
			return err
		}
		if page.Total > out.Total {
			out.Total = page.Total
		}
		for _, m := range page.Items {
			// Adjacent windows can share their boundary millisecond.
			if m.ID != "" && seen[m.ID] {
				continue
			}
			seen[m.ID] = true
			out.Items = append(out.Items, m)
		}
		if !page.HasMore || page.Cursor == "" {
			break
		}
		cursor = page.Cursor
	}
	sort.SliceStable(out.Items, func(i, j int) bool { return out.Items[i].OffsetMS < out.Items[j].OffsetMS })
	if f.json {
		return printJSON(d.Stdout, out)
	}
	for _, m := range out.Items {
		fmt.Fprintf(d.Stdout, "%s\t%s\n", clock(m.OffsetMS), oneLine(m.Text))
	}
	if out.Total > 0 {
		fmt.Fprintf(d.Stderr, "ohg: %d danmaku (the service counts %d and serves a selection)\n", len(out.Items), out.Total)
	} else {
		fmt.Fprintf(d.Stderr, "ohg: %d danmaku\n", len(out.Items))
	}
	return nil
}

// clock formats milliseconds of play time as m:ss.
func clock(ms int64) string {
	s := ms / 1000
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
