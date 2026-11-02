package cliapp

import (
	"context"
	"fmt"
	"strings"

	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
)

func cmdDiscover(ctx context.Context, args []string, d Dependencies) error {
	if err := requireService(d); err != nil {
		return err
	}
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(f.rest) != 0 {
		return usage("discover takes no arguments")
	}
	if f.all {
		return runAllPages(ctx, d, f, func(ctx context.Context, cursor string) (*sdk.Page[model.SeriesSummary], error) {
			return d.Service.Discover(ctx, sdk.DiscoverOptions{Cursor: cursor, Channel: f.channel, PageSize: f.pageSize})
		})
	}
	page, err := d.Service.Discover(ctx, sdk.DiscoverOptions{Cursor: f.cursor, Channel: f.channel, PageSize: f.pageSize})
	if err != nil {
		return usageFrom(err)
	}
	return printValue(d.Stdout, page, f.json)
}

func cmdSearch(ctx context.Context, args []string, d Dependencies) error {
	if err := requireService(d); err != nil {
		return err
	}
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(f.rest) != 1 || strings.TrimSpace(f.rest[0]) == "" {
		return usage("search needs one keyword")
	}
	filters, err := searchFilters(f.filters)
	if err != nil {
		return err
	}
	if f.all {
		return runAllPages(ctx, d, f, func(ctx context.Context, cursor string) (*sdk.Page[model.SeriesSummary], error) {
			return d.Service.Search(ctx, f.rest[0], sdk.SearchOptions{Cursor: cursor, PageSize: f.pageSize, Tab: f.tab, Filters: filters})
		})
	}
	page, err := d.Service.Search(ctx, f.rest[0], sdk.SearchOptions{Cursor: f.cursor, PageSize: f.pageSize, Tab: f.tab, Filters: filters})
	if err != nil {
		return usageFrom(err)
	}
	return printValue(d.Stdout, page, f.json)
}

// searchFilterNames are the labels of the app's search filter panel.
var searchFilterNames = map[string]string{
	"真人剧": sdk.SearchFilterShortPlay,
	"漫剧":  sdk.SearchFilterComic,
	"AI剧": sdk.SearchFilterAIComic,
	"影视剧": sdk.SearchFilterFilm,
	"未看过": sdk.SearchFilterUnwatched,
}

// searchFilters reads --filter labels; each may hold several, comma-separated.
func searchFilters(values []string) ([]string, error) {
	var out []string
	for _, v := range values {
		for _, name := range strings.Split(v, ",") {
			name = strings.TrimSpace(name)
			filter, ok := searchFilterNames[name]
			if !ok {
				return nil, usage("unknown search filter %q (真人剧|漫剧|AI剧|影视剧|未看过)", name)
			}
			out = append(out, filter)
		}
	}
	return out, nil
}

// runAllPages follows cursors and prints every distinct row, in order.
func runAllPages(ctx context.Context, d Dependencies, f flags, fetch func(context.Context, string) (*sdk.Page[model.SeriesSummary], error)) error {
	pager := sdk.NewPager(f.cursor, fetch)
	limit := f.pages
	if limit == 0 {
		limit = 10
	}
	items := make([]model.SeriesSummary, 0, 32)
	more := false
	for i := 0; i < limit; i++ {
		fresh, hasMore, err := pager.Next(ctx)
		if err != nil {
			return usageFrom(err)
		}
		items = append(items, fresh...)
		more = hasMore
		if !hasMore {
			break
		}
	}
	page := &sdk.Page[model.SeriesSummary]{Items: items, HasMore: more}
	return printValue(d.Stdout, page, f.json)
}

func cmdSuggest(ctx context.Context, args []string, d Dependencies) error {
	if err := requireService(d); err != nil {
		return err
	}
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(f.rest) != 1 || strings.TrimSpace(f.rest[0]) == "" {
		return usage("suggest needs one prefix")
	}
	words, err := d.Service.Suggest(ctx, f.rest[0])
	if err != nil {
		return err
	}
	if f.json {
		return printJSON(d.Stdout, words)
	}
	for _, w := range words {
		if w.Keyword != "" && w.Keyword != w.Text {
			fmt.Fprintf(d.Stdout, "%s\t%s\n", w.Text, w.Keyword)
			continue
		}
		fmt.Fprintln(d.Stdout, w.Text)
	}
	return nil
}

func cmdCategory(ctx context.Context, args []string, d Dependencies) error {
	if err := requireService(d); err != nil {
		return err
	}
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(f.rest) != 1 {
		return usage("category needs one category ID")
	}
	page, err := d.Service.Categories(ctx, f.rest[0])
	if err != nil {
		return err
	}
	return printValue(d.Stdout, page, f.json)
}

func cmdLanding(ctx context.Context, args []string, d Dependencies) error {
	if err := requireService(d); err != nil {
		return err
	}
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(f.rest) != 1 {
		return usage("landing needs one category ID")
	}
	filters := map[string]string{}
	for _, kv := range f.filters {
		key, value, ok := strings.Cut(kv, "=")
		if !ok || key == "" || value == "" {
			return usage("--filter takes key=value, got %q", kv)
		}
		filters[key] = value
	}
	page, err := d.Service.CategoryLanding(ctx, f.rest[0], sdk.LandingOptions{Tab: f.tab, Filters: filters, Cursor: f.cursor})
	if err != nil {
		return err
	}
	return printValue(d.Stdout, page, f.json)
}

func cmdCues(ctx context.Context, args []string, d Dependencies) error {
	if err := requireService(d); err != nil {
		return err
	}
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(f.rest) != 0 {
		return usage("cues takes no arguments")
	}
	words, err := d.Service.SearchCues(ctx)
	if err != nil {
		return err
	}
	if f.json {
		return printJSON(d.Stdout, words)
	}
	for _, w := range words {
		fmt.Fprintln(d.Stdout, w)
	}
	return nil
}

// cmdNew prints one day of the 上新 calendar: the day and the days on offer,
// then series ID, title, local publish time, online or upcoming, heat or
// reservations, and episodes.
func cmdNew(ctx context.Context, args []string, d Dependencies) error {
	if err := requireService(d); err != nil {
		return err
	}
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(f.rest) != 0 {
		return usage("new takes no arguments")
	}
	page, err := d.Service.NewReleases(ctx, sdk.NewReleaseQuery{Date: f.date, Cursor: f.cursor})
	if err != nil {
		if msg := err.Error(); strings.HasPrefix(msg, "date must be") || strings.HasPrefix(msg, "the cursor belongs") || strings.HasSuffix(msg, "cursor is not valid") {
			return usage("%s", msg)
		}
		return err
	}
	if f.json {
		return printJSON(d.Stdout, page)
	}
	fmt.Fprintf(d.Stdout, "date\t%s\n", page.Date)
	if len(page.Dates) > 0 {
		fmt.Fprintf(d.Stdout, "dates\t%s\n", strings.Join(page.Dates, " "))
	}
	for _, r := range page.Items {
		state, when := "upcoming", ""
		if r.Online {
			state = "online"
		}
		if !r.PublishAt.IsZero() {
			when = r.PublishAt.Local().Format("01-02 15:04")
		}
		fmt.Fprintf(d.Stdout, "%s\t%s\t%s\t%s\t%s\t%d\n", r.SeriesID, r.Title, when, state, r.Heat, r.EpisodeCount)
	}
	if page.Cursor != "" {
		fmt.Fprintf(d.Stdout, "cursor\t%s\n", page.Cursor)
	}
	return nil
}

// cmdHot prints the hot lists: each list's name, then position, series ID,
// title, and the list's figure; topics print their search query.
func cmdHot(ctx context.Context, args []string, d Dependencies) error {
	if err := requireService(d); err != nil {
		return err
	}
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(f.rest) != 0 {
		return usage("hot takes no arguments")
	}
	hot, err := d.Service.HotSearch(ctx)
	if err != nil {
		return err
	}
	if f.json {
		return printJSON(d.Stdout, hot)
	}
	printItems := func(items []model.HotItem) {
		for _, it := range items {
			score := it.SearchScore
			if score == "" {
				score = it.Heat
			}
			fmt.Fprintf(d.Stdout, "%d\t%s\t%s\t%s\n", it.Position, it.SeriesID, it.Title, score)
		}
	}
	if len(hot.Guesses) > 0 {
		fmt.Fprintln(d.Stdout, "# 猜你想搜")
		printItems(hot.Guesses)
	}
	for _, b := range hot.Boards {
		fmt.Fprintf(d.Stdout, "# %s\n", b.Name)
		printItems(b.Items)
		for i, t := range b.Topics {
			fmt.Fprintf(d.Stdout, "%d\t%s\t%s\n", i+1, t.Query, t.Heat)
		}
	}
	return nil
}
