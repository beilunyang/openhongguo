package contract

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// Config selects what a live run exercises.
type Config struct {
	// Transport and Profiles are the anonymous client's dependencies.
	Transport sdk.Transport
	Profiles  sdk.ProfileStore
	// FixtureDir holds the contract fixtures (testdata/protocol).
	FixtureDir string
	// Query is searched to find a series to play.
	Query string
	// MovieSeriesID, when set, also checks a segmented (DASH) title.
	MovieSeriesID string
	// CategoryID is the category whose frame and landing list are checked;
	// empty uses 68 (末世), observed to answer anonymously.
	CategoryID string
	// FFmpeg is the ffmpeg binary used to prove the derived key decrypts a
	// frame; empty skips that check.
	FFmpeg string
	// APIOrigin and DistributionOrigin override the client's hosts, as the
	// CLI's OHG_API_ORIGIN and OHG_DISTRIBUTION_ORIGIN do; empty keeps the
	// defaults.
	APIOrigin          string
	DistributionOrigin string
}

// Check is one step of a live run.
type Check struct {
	Name string
	// Fixture is the contract file the live response was compared with.
	Fixture string
	Drifts  []Drift
	Err     error
	// Live is the shape of the live response, for a report or a diff.
	Live []Field
}

// OK reports a step with neither drift nor error.
func (c Check) OK() bool { return c.Err == nil && len(c.Drifts) == 0 }

// Run calls the live service the way the CLI does and checks every
// response against its fixture, then checks that the results are usable:
// a search finds the series, the series lists episodes, the playback model
// has streams, and a protected stream's derived key decrypts a frame.
func Run(ctx context.Context, cfg Config) []Check {
	rec := &recorder{next: cfg.Transport, bodies: map[string][]byte{}}
	client, err := sdk.NewClient(sdk.Options{
		Transport: rec, ProfileStore: cfg.Profiles, Signer: sdk.UnavailableSigner{},
		APIOrigin: cfg.APIOrigin, DistributionOrigin: cfg.DistributionOrigin,
	})
	if err != nil {
		return []Check{{Name: "client", Err: err}}
	}
	var checks []Check
	_, err = client.Discover(ctx, sdk.DiscoverOptions{})
	checks = append(checks, rec.check(cfg, "discover", "discover-response.json", protocol.PathDiscover, err))
	if rec.has(protocol.PathRegister) {
		// A fresh profile registers a device first.
		checks = append(checks, rec.check(cfg, "register", "register-response.json", protocol.PathRegister, nil))
	}

	seriesID, err := searchSeries(ctx, client, cfg.Query)
	checks = append(checks, rec.check(cfg, "search", "search-response.json", protocol.PathSearch, err))
	if err != nil {
		return checks
	}
	checks = append(checks, playSeries(ctx, client, rec, cfg, "series", seriesID, "")...)
	checks = append(checks, browse(ctx, client, rec, cfg, seriesID)...)
	if cfg.MovieSeriesID != "" {
		checks = append(checks, playSeries(ctx, client, rec, cfg, "movie", cfg.MovieSeriesID, "dynamic-playback.json")...)
	}
	return checks
}

// browse checks the endpoints around a series that a client screen relies
// on: search suggestions, the related list, the seasons, the hot lists, the category pages, and the
// bookmall ranking rows, the short-drama filter and rankings (actors included), a person's page
// and works, the 上新 calendar, and the search
// box cues. The 星粉榜 activity is left out: it answers with no
// rows while the activity is closed.
func browse(ctx context.Context, client *sdk.Client, rec *recorder, cfg Config, seriesID string) []Check {
	nonEmpty := func(n int, what string, err error) error {
		if err == nil && n == 0 {
			return fmt.Errorf("%s is empty", what)
		}
		return err
	}
	category := cfg.CategoryID
	if category == "" {
		category = "68"
	}
	var checks []Check

	sug, err := client.Suggest(ctx, cfg.Query)
	checks = append(checks, rec.check(cfg, "suggest", "suggest-response.json", protocol.PathSuggest, nonEmpty(len(sug), "suggestion list", err)))

	n := 0
	rel, err := client.SeriesRecommend(ctx, seriesID)
	if err == nil {
		n = len(rel.Items)
	}
	checks = append(checks, rec.check(cfg, "related", "plan-response.json", protocol.PathPlan, nonEmpty(n, "related list", err)))

	// searchSeries prefers a multi-season title; a single-season one answers
	// 101116 here and shows up as drift.
	n = 0
	rec.forget(protocol.PathPlan)
	album, err := client.Seasons(ctx, seriesID)
	if err == nil {
		n = len(album.Seasons)
	}
	checks = append(checks, rec.check(cfg, "seasons", "series-seasons.json", protocol.PathPlan, nonEmpty(n, "season list", err)))

	// The batch read keys its answer by series ID, so its shape is not
	// compared; every season asked for must come back with episodes.
	ids := []string{seriesID}
	if album != nil {
		for _, season := range album.Seasons {
			ids = append(ids, season.SeriesID)
		}
	}
	batch := Check{Name: "series batch"}
	details, err := client.GetSeriesBatch(ctx, ids)
	switch {
	case err != nil:
		batch.Err = err
	case len(details) == 0 || details[0].Series.ID != seriesID:
		batch.Err = fmt.Errorf("the batch left out %s", seriesID)
	default:
		for _, d := range details {
			// An announced season has no episodes yet.
			if d.Series.EpisodeCount > 0 && len(d.Episodes) == 0 {
				batch.Err = fmt.Errorf("series %s came back without episodes", d.Series.ID)
				break
			}
		}
	}
	checks = append(checks, batch)

	n = 0
	rec.forget(protocol.PathPlan)
	hot, err := client.HotSearch(ctx)
	if err == nil {
		for _, b := range hot.Boards {
			if len(b.Items) > 0 {
				n++
			}
		}
	}
	checks = append(checks, rec.check(cfg, "hot search", "hot-search.json", protocol.PathPlan, nonEmpty(n, "hot title list", err)))

	n = 0
	front, err := client.Categories(ctx, category)
	if err == nil {
		n = len(front.Tabs)
	}
	checks = append(checks, rec.check(cfg, "category", "category-front.json", protocol.PathCategory, nonEmpty(n, "category tab list", err)))

	n = 0
	landing, err := client.CategoryLanding(ctx, category, sdk.LandingOptions{})
	if err == nil {
		n = len(landing.Items)
	}
	checks = append(checks, rec.check(cfg, "landing", "category-landing.json", protocol.PathCategoryLanding, nonEmpty(n, "category list", err)))

	n = 0
	rank, err := client.RankRows(ctx, "", 0)
	if err == nil {
		n = len(rank.Items)
	}
	checks = append(checks, rec.check(cfg, "rank rows", "rank-rows.json", protocol.PathRankCells, nonEmpty(n, "ranking rows", err)))

	rows, err := client.ShortFilterPanel(ctx, "")
	checks = append(checks, rec.check(cfg, "filter panel", "short-filter-panel.json", protocol.PathShortFilter, nonEmpty(len(rows), "short filter rows", err)))

	n = 0
	filtered, err := client.ShortFilter(ctx, sdk.ShortFilterQuery{Selections: map[string][]string{"genre": {"comic_series"}}})
	if err == nil {
		n = len(filtered.Items)
	}
	checks = append(checks, rec.check(cfg, "short filter", "short-filter-content.json", protocol.PathShortFilter, nonEmpty(n, "short filter rows", err)))

	n = 0
	short, err := client.ShortRank(ctx, sdk.ShortRankQuery{Content: "human", Board: "human_hot_play"})
	if err == nil {
		n = len(short.Items)
	}
	c := rec.check(cfg, "short rank", "short-rank.json", protocol.PathRankCells, nonEmpty(n, "short-drama ranking", err))
	if c.Err == nil {
		// The ranked rows must be videos, not books (platform 0).
		c.Err = rankedVideos(ctx, client, short.Items, 3)
	}
	checks = append(checks, c)

	// The actor ranking shares the path; its rows are people.
	n = 0
	rec.forget(protocol.PathRankCells)
	actors, err := client.ShortRank(ctx, sdk.ShortRankQuery{Content: protocol.ShortRankActors})
	if err == nil {
		for _, it := range actors.Items {
			if it.ActorID != "" && it.Actor != "" {
				n++
			}
		}
	}
	checks = append(checks, rec.check(cfg, "actor rank", "short-rank-actors.json", protocol.PathRankCells, nonEmpty(n, "actor ranking", err)))

	// The ranking's first actor stands in for a person's page and works.
	person := ""
	if actors != nil {
		for _, it := range actors.Items {
			if it.UserID != "" {
				person = it.UserID
				break
			}
		}
	}
	if person == "" {
		checks = append(checks, Check{Name: "person", Err: fmt.Errorf("the actor ranking named no user ID")})
	} else {
		_, err := client.Person(ctx, person)
		n = 0
		if err == nil {
			n = 1
		}
		checks = append(checks, rec.check(cfg, "person", "person.json", protocol.PathProfile, nonEmpty(n, "person page", err)))
		n = 0
		works, err := client.PersonWorks(ctx, person, "")
		if err == nil {
			n = len(works.Items)
		}
		checks = append(checks, rec.check(cfg, "person works", "person-works.json", fmt.Sprintf(protocol.PathUserWorks, person), nonEmpty(n, "work list", err)))
	}

	n = 0
	// Yesterday is all titles that are out; today mixes in announced ones
	// as the day goes on, which would read as drift.
	yesterday := time.Now().In(time.FixedZone("CST", 8*3600)).AddDate(0, 0, -1).Format("20060102")
	releases, err := client.NewReleases(ctx, sdk.NewReleaseQuery{Date: yesterday})
	if err == nil {
		n = len(releases.Items)
	}
	checks = append(checks, rec.check(cfg, "new releases", "new-releases.json", protocol.PathSubscribeList, nonEmpty(n, "上新 list", err)))

	cues, err := client.SearchCues(ctx)
	checks = append(checks, rec.check(cfg, "cues", "cue-response.json", protocol.PathCue, nonEmpty(len(cues), "search cues", err)))
	return checks
}

// rankedVideos checks that the first want non-promoted ranking rows are
// videos according to their series detail.
func rankedVideos(ctx context.Context, client *sdk.Client, items []model.RankItem, want int) error {
	for _, item := range items {
		if want == 0 {
			break
		}
		if item.Promoted {
			continue
		}
		series, err := client.GetSeries(ctx, item.SeriesID)
		if err != nil {
			return fmt.Errorf("ranked series %s: %w", item.SeriesID, err)
		}
		if series.Platform == model.PlatformUnknown {
			return fmt.Errorf("ranked series %s (%s) is not a video", item.SeriesID, series.Title)
		}
		want--
	}
	return nil
}

func searchSeries(ctx context.Context, client *sdk.Client, query string) (string, error) {
	page, err := client.Search(ctx, query, sdk.SearchOptions{PageSize: 10})
	if err != nil {
		return "", err
	}
	if len(page.Items) == 0 {
		return "", fmt.Errorf("search %q found nothing", query)
	}
	// The seasons check needs a multi-season title, and the search order
	// shifts from day to day; prefer the first of the top hits that is one.
	for i, it := range page.Items {
		if i == 5 {
			break
		}
		if s, err := client.GetSeries(ctx, it.ID); err == nil && s.IsMultiSeason {
			return it.ID, nil
		}
	}
	return page.Items[0].ID, nil
}

// playSeries checks the series detail and the first episode's playback.
// playbackFixture is chosen from the model when empty.
func playSeries(ctx context.Context, client *sdk.Client, rec *recorder, cfg Config, name, seriesID, playbackFixture string) []Check {
	var checks []Check
	rec.forget(protocol.PathSeries)
	page, err := client.ListEpisodes(ctx, seriesID, sdk.EpisodeOptions{})
	if err == nil {
		err = checkEpisodes(page.Items)
	}
	checks = append(checks, rec.check(cfg, name+" detail", "series-response.json", protocol.PathSeries, err))
	if err != nil {
		return checks
	}

	rec.forget(protocol.PathPlayback)
	pb, err := client.ResolvePlayback(ctx, page.Items[0].ID, sdk.PlaybackOptions{})
	// Since 2026-10-29 the service answers each anonymous device with a model
	// about once, and even a new device's first request is not certain, so
	// an empty answer is retried on new devices the way the CLI does.
	for i := 0; i < 2 && errors.Is(err, sdk.ErrAnonymousLimited); i++ {
		if err = client.ResetDevice(ctx); err != nil {
			break
		}
		pb, err = client.ResolvePlayback(ctx, page.Items[0].ID, sdk.PlaybackOptions{})
	}
	if errors.Is(err, sdk.ErrDRMProtected) && pb != nil {
		err = nil
	}
	if err == nil && len(pb.Streams) == 0 {
		err = errors.New("playback model has no streams")
	}
	if playbackFixture == "" {
		playbackFixture = "clear-playback.json"
		if pb != nil && pb.Protected {
			playbackFixture = "drm-playback.json"
		}
	}
	checks = append(checks, rec.check(cfg, name+" playback", playbackFixture, protocol.PathPlayback, err))
	if err != nil || !pb.Protected {
		return checks
	}

	key := Check{Name: name + " key"}
	k, err := sdk.ContentKey(pb)
	if err != nil {
		key.Err = fmt.Errorf("key material no longer unwraps: %w", err)
		return append(checks, key)
	}
	if cfg.FFmpeg != "" {
		stream, _ := pb.Select("worst")
		key.Err = decryptsFrame(ctx, cfg.FFmpeg, stream.URL, k.Hex)
	}
	return append(checks, key)
}

func checkEpisodes(items []model.Episode) error {
	if len(items) == 0 {
		return errors.New("series lists no episodes")
	}
	for i, e := range items {
		if e.ID == "" || e.Number != i+1 {
			return fmt.Errorf("episode %d is malformed (number %d)", i+1, e.Number)
		}
	}
	return nil
}

// decryptsFrame proves the key by decoding one video frame. With a wrong key
// the decoder fails on the first sample.
func decryptsFrame(ctx context.Context, ffmpeg, src, key string) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, ffmpeg, "-hide_banner", "-v", "error", "-nostdin",
		"-decryption_key", key, "-i", src, "-frames:v", "1", "-xerror", "-f", "null", "-")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if i := strings.IndexByte(msg, '\n'); i >= 0 {
			msg = msg[:i]
		}
		// The message can name the stream URL; keep only its host and path.
		msg = strings.ReplaceAll(msg, src, protocol.RedactedURL(src))
		return fmt.Errorf("derived key does not decrypt a frame: %v %s", err, msg)
	}
	return nil
}

// check compares the response recorded for path with fixture. err is the
// step's own error and wins over a comparison error.
func (r *recorder) check(cfg Config, name, fixture, path string, err error) Check {
	c := Check{Name: name, Fixture: fixture, Err: err}
	body, ok := r.body(path)
	if !ok {
		if c.Err == nil {
			c.Err = fmt.Errorf("no response recorded for %s", path)
		}
		return c
	}
	var cmpErr error
	c.Live, c.Drifts, cmpErr = compareFixture(filepath.Join(cfg.FixtureDir, fixture), body)
	if c.Err == nil {
		c.Err = cmpErr
	}
	return c
}

func compareFixture(path string, live []byte) ([]Field, []Drift, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	want, err := Shape(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("fixture %s: %w", filepath.Base(path), err)
	}
	got, err := Shape(live)
	if err != nil {
		return nil, nil, fmt.Errorf("live response is not JSON: %w", err)
	}
	return got, Compare(want, got), nil
}

// recorder keeps the last decoded response body per URL path.
type recorder struct {
	next   sdk.Transport
	mu     sync.Mutex
	bodies map[string][]byte
}

func (r *recorder) Do(ctx context.Context, req *sdk.Request) (*sdk.Response, error) {
	resp, err := r.next.Do(ctx, req)
	if err != nil || resp == nil {
		return resp, err
	}
	u, perr := url.Parse(req.URL)
	if perr != nil {
		return resp, err
	}
	if body, derr := decode(resp); derr == nil {
		r.mu.Lock()
		r.bodies[u.Path] = body
		r.mu.Unlock()
	}
	return resp, err
}

func (r *recorder) body(path string) ([]byte, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	b, ok := r.bodies[path]
	return b, ok
}

func (r *recorder) has(path string) bool {
	_, ok := r.body(path)
	return ok
}

func (r *recorder) forget(path string) {
	r.mu.Lock()
	delete(r.bodies, path)
	r.mu.Unlock()
}

// decode undoes the gzip content encoding the SDK also accepts.
func decode(resp *sdk.Response) ([]byte, error) {
	for k, v := range resp.Headers {
		if strings.EqualFold(k, "Content-Encoding") && strings.EqualFold(v, "gzip") {
			zr, err := gzip.NewReader(bytes.NewReader(resp.Body))
			if err != nil {
				return nil, err
			}
			defer zr.Close()
			return io.ReadAll(io.LimitReader(zr, 8<<20))
		}
	}
	return resp.Body, nil
}
