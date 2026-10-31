package cliapp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/beilunyang/openhongguo/internal/history"
	"github.com/beilunyang/openhongguo/internal/media"
	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// maxUnavailable is how many episodes in a row autoplay skips before it
// gives up: one locked episode is skipped, a locked series is not walked.
const maxUnavailable = 3

// resumeBackoff rewinds a resumed episode a little so the viewer sees where
// they left off.
const resumeBackoff = 3

func cmdPlay(ctx context.Context, args []string, d Dependencies) error {
	if err := requireService(d); err != nil {
		return err
	}
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(f.rest) != 1 {
		return usage("play needs one series ID or title")
	}
	if f.url && f.download != "" {
		return usage("--url and --download are mutually exclusive")
	}
	if f.download != "" && f.player != "" && f.player != "auto" {
		return usage("--download does not use --player")
	}
	key := strings.TrimSpace(f.key)
	if key == "" {
		key = strings.TrimSpace(os.Getenv("OHG_CENC_KEY"))
	}
	if key != "" && f.url {
		return usage("--key cannot be combined with --url")
	}
	if key != "" && f.noDecrypt {
		return usage("--key and --no-decrypt are mutually exclusive")
	}
	if f.autoplay && (f.url || f.json || f.download != "") {
		return usage("--autoplay cannot be combined with --url, --json, or --download")
	}
	if f.autoplay && (f.player == "open" || f.player == "xdg-open") {
		return usage("--autoplay needs mpv, ffplay, or vlc to know when an episode ends")
	}
	if f.autoplay && key != "" {
		return usage("--autoplay derives each episode's key; drop --key and OHG_CENC_KEY")
	}
	if f.resume && f.episodeSet {
		return usage("--resume and --episode are mutually exclusive")
	}
	if f.resume && (f.url || f.json || f.download != "") {
		return usage("--resume cannot be combined with --url, --json, or --download")
	}
	if f.resume && d.History == nil {
		return errors.New("watch history is not configured")
	}
	id, title, err := resolveSeries(ctx, d, f.rest[0], f.pick)
	if err != nil {
		return err
	}
	page, err := d.Service.ListEpisodes(ctx, id, sdk.EpisodeOptions{})
	if err != nil {
		return err
	}
	start := 0
	if f.resume {
		n, pos, err := resumePoint(ctx, d.History, id, len(page.Items))
		if err != nil {
			return err
		}
		f.episode, start = n, pos
	}
	episode, ok := findEpisode(page.Items, f.episode)
	if !ok {
		return usage("series %s has no episode %d", id, f.episode)
	}
	// Only a real playback is remembered, not --url, --json, --download, or
	// a --preview that never reaches the end.
	remember := d.History != nil && !f.url && !f.json && f.download == "" && f.preview == 0
	if series, err := d.Service.GetSeries(ctx, id); err == nil {
		if err := requireVideo(series); err != nil {
			return err
		}
		if title == "" {
			title = series.Title
		}
	}
	s := &session{ctx: ctx, d: d, f: f, key: key}
	return s.run(page.Items, episode, start, func(e history.Entry) {
		if remember {
			e.SeriesID, e.Title, e.Episodes = id, title, len(page.Items)
			recordHistory(ctx, d, e)
		}
	})
}

// session plays one or more episodes of a series.
type session struct {
	ctx context.Context
	d   Dependencies
	f   flags
	// key is the caller-supplied content key of the first episode.
	key string
	// prefetch resolves the episode that will most likely play next while
	// the current one plays.
	prefetch struct {
		number int
		ch     <-chan resolved
	}
}

type resolved struct {
	playback *model.Playback
	err      error
}

func (s *session) run(items []model.Episode, episode model.Episode, start int, record func(history.Entry)) error {
	total := len(items)
	unavailable := 0
	for {
		if s.f.autoplay || s.f.resume || episode.Number != s.f.episode {
			fmt.Fprintf(s.d.Stderr, "ohg: playing episode %d of %d\n", episode.Number, total)
		}
		record(history.Entry{Episode: episode.Number, Position: float64(start)})
		pending := s.resolve(episode)
		if next, ok := findEpisode(items, episode.Number+1); ok && s.f.autoplay {
			s.startPrefetch(next)
		}
		res, err := s.play(episode, pending, start)
		start = 0
		s.key = "" // a caller key belongs to the first episode only
		switch {
		case res.Jump != 0:
			record(history.Entry{Episode: episode.Number, Position: res.Position})
			target, ok := findEpisode(items, episode.Number+res.Jump)
			if !ok {
				return nil
			}
			episode = target
			continue
		case errors.Is(err, media.ErrStopped) || errors.Is(err, context.Canceled):
			// Closing the player or pressing Ctrl-C ends playback quietly.
			record(history.Entry{Episode: episode.Number, Position: res.Position})
			return nil
		case err != nil && s.f.autoplay && unavailableEpisode(err):
			unavailable++
			if unavailable >= maxUnavailable {
				return fmt.Errorf("episode %d: %w (%d episodes in a row unavailable; stopping)", episode.Number, err, unavailable)
			}
			fmt.Fprintf(s.d.Stderr, "ohg: episode %d is not available anonymously; skipping\n", episode.Number)
		case err != nil:
			if s.f.autoplay {
				return fmt.Errorf("episode %d: %w", episode.Number, err)
			}
			return err
		default:
			unavailable = 0
			record(history.Entry{Episode: episode.Number, Finished: true})
		}
		if !s.f.autoplay {
			return nil
		}
		next, ok := findEpisode(items, episode.Number+1)
		if !ok {
			return nil
		}
		episode = next
	}
}

// unavailableEpisode reports an episode the anonymous surface will not play.
func unavailableEpisode(err error) bool {
	return errors.Is(err, sdk.ErrAnonymousLimited) || errors.Is(err, sdk.ErrAuthenticationRequired) || errors.Is(err, sdk.ErrDRMProtected)
}

// resolve returns the prefetched model of episode when there is one, and
// starts resolving it otherwise.
func (s *session) resolve(episode model.Episode) <-chan resolved {
	if s.prefetch.ch != nil && s.prefetch.number == episode.Number {
		ch := s.prefetch.ch
		s.prefetch.ch = nil
		return ch
	}
	return s.fetch(episode)
}

func (s *session) startPrefetch(episode model.Episode) {
	s.prefetch.number, s.prefetch.ch = episode.Number, s.fetch(episode)
}

func (s *session) fetch(episode model.Episode) <-chan resolved {
	ch := make(chan resolved, 1)
	go func() {
		pb, err := resolvePlayback(s.ctx, s.d, episode.ID)
		if errors.Is(err, sdk.ErrDRMProtected) && pb != nil {
			err = nil
		}
		ch <- resolved{pb, err}
	}()
	return ch
}

// play waits for the episode's model and plays, saves, or reports it.
func (s *session) play(episode model.Episode, pending <-chan resolved, start int) (media.Result, error) {
	r := <-pending
	if r.err == nil && r.playback != nil && expired(r.playback) {
		// A prefetched model can outlive its CDN URLs if playback paused.
		r = <-s.fetch(episode)
	}
	if r.err != nil {
		return media.Result{}, r.err
	}
	if r.playback == nil {
		return media.Result{}, errors.New("empty playback model")
	}
	return playResolved(s.ctx, s.d, episode, r.playback, s.key, s.f, start)
}

// expired reports a model whose URLs expire within a minute.
func expired(pb *model.Playback) bool {
	return pb.ExpiresAt > 0 && time.Unix(pb.ExpiresAt, 0).Before(time.Now().Add(time.Minute))
}

// playResolved plays, saves, or reports one resolved episode.
func playResolved(ctx context.Context, d Dependencies, episode model.Episode, playback *model.Playback, key string, f flags, start int) (media.Result, error) {
	stream, err := playback.Select(f.quality)
	if err != nil {
		return media.Result{}, usage("%s", err.Error())
	}
	if playback.Protected && key == "" && !f.noDecrypt {
		if k, err := sdk.ContentKey(playback); err == nil {
			key = k.Hex
		}
	}
	if key != "" && !playback.Protected {
		return media.Result{}, usage("episode %d is not encrypted; drop --key", episode.Number)
	}
	if f.json {
		if err := printValue(d.Stdout, playback, true); err != nil {
			return media.Result{}, err
		}
		if playback.Protected && key == "" {
			return media.Result{}, drmError(playback)
		}
		return media.Result{}, nil
	}
	if f.url {
		// The CDN URLs are public and expire on their own, so --url reports
		// them without touching the decryptor.
		fmt.Fprintln(d.Stdout, stream.URL)
		if playback.AudioURL != "" {
			// A segmented title keeps its audio in a separate track.
			fmt.Fprintln(d.Stdout, playback.AudioURL)
		}
		return media.Result{}, nil
	}
	if playback.Protected && key == "" {
		return media.Result{}, drmError(playback)
	}
	if d.Media == nil {
		return media.Result{}, errors.New("media launcher is not configured")
	}
	if f.download != "" {
		written, err := withBackup(ctx, d, stream, func(url string) (int64, error) {
			return saveStream(ctx, d, url, playback.AudioURL, key, playback.Protected, f.download, f.preview)
		})
		if err != nil {
			return media.Result{}, err
		}
		fmt.Fprintf(d.Stdout, "%s\t%d\t%s\n", f.download, written, streamName(stream))
		return media.Result{}, nil
	}
	req := media.Request{
		URL: stream.URL, Audio: playback.AudioURL, Key: key, Player: f.player,
		StartSeconds: start, LimitSeconds: f.preview, DurationSeconds: playback.DurationSeconds,
	}
	var res media.Result
	_, err = withBackup(ctx, d, stream, func(url string) (int64, error) {
		req.URL = url
		var playErr error
		res, playErr = d.Media.Play(ctx, req)
		return 0, playErr
	})
	if !errors.Is(err, media.ErrNeedsFile) || f.autoplay {
		return res, err
	}
	// open and xdg-open return at once, so autoplay never gets here.
	return media.Result{}, playFromTempFile(ctx, d, stream, playback.AudioURL, key, f)
}

// withBackup runs op on the stream URL and, when that fails for a reason
// other than the viewer, once more on the backup CDN URL.
func withBackup(ctx context.Context, d Dependencies, stream model.Stream, op func(url string) (int64, error)) (int64, error) {
	n, err := op(stream.URL)
	if err == nil || stream.BackupURL == "" || stream.BackupURL == stream.URL || ctx.Err() != nil ||
		errors.Is(err, media.ErrStopped) || errors.Is(err, media.ErrNeedsFile) || errors.Is(err, errUsage) {
		return n, err
	}
	fmt.Fprintf(d.Stderr, "ohg: %v; retrying the backup URL\n", err)
	return op(stream.BackupURL)
}

// saveStream writes one stream to path, decrypting a protected one.
func saveStream(ctx context.Context, d Dependencies, url, audio, key string, protected bool, path string, limit int) (int64, error) {
	if protected {
		return d.Media.Decrypt(ctx, url, audio, key, path, limit)
	}
	return d.Media.Save(ctx, url, path)
}

// playFromTempFile decrypts into a temp file for a player that can only
// open a file, and removes the file once the player exits.
func playFromTempFile(ctx context.Context, d Dependencies, stream model.Stream, audio, key string, f flags) error {
	tmp, err := os.CreateTemp("", "ohg-*.mp4")
	if err != nil {
		return err
	}
	target := tmp.Name()
	_ = tmp.Close()
	// Decrypt never overwrites, so drop the probe file first.
	if err := os.Remove(target); err != nil {
		return err
	}
	if _, err := withBackup(ctx, d, stream, func(url string) (int64, error) {
		return d.Media.Decrypt(ctx, url, audio, key, target, f.preview)
	}); err != nil {
		return err
	}
	defer os.Remove(target)
	return d.Media.Open(ctx, target, f.player)
}

// resolveSeries turns a series ID or a title into a series ID. A title is
// searched: an exact title match wins, then the first result; with pick the
// viewer chooses. The title is returned when the search found it.
func resolveSeries(ctx context.Context, d Dependencies, arg string, pick bool) (id, title string, err error) {
	if isNumeric(arg) {
		if pick {
			return "", "", usage("--pick needs a title, not a series ID")
		}
		id, err := validateID(arg)
		return id, "", err
	}
	if strings.Contains(arg, "://") {
		_, err := validateID(arg)
		return "", "", err
	}
	query := strings.TrimSpace(arg)
	if query == "" {
		return "", "", usage("missing series ID or title")
	}
	page, err := d.Service.Search(ctx, query, sdk.SearchOptions{PageSize: 10})
	if err != nil {
		return "", "", usageFrom(err)
	}
	if len(page.Items) == 0 {
		return "", "", fmt.Errorf("no series matches %q", query)
	}
	choice := page.Items[0]
	if pick {
		choice, err = pickSeries(d, page.Items)
		if err != nil {
			return "", "", err
		}
	} else {
		for _, s := range page.Items {
			if s.Title == query {
				choice = s
				break
			}
		}
		fmt.Fprintf(d.Stderr, "ohg: %s (%s, %d episodes)\n", choice.Title, choice.ID, choice.EpisodeCount)
	}
	return choice.ID, choice.Title, nil
}

// pickSeries lists items on stderr and reads the chosen number from stdin.
func pickSeries(d Dependencies, items []model.SeriesSummary) (model.SeriesSummary, error) {
	for i, s := range items {
		fmt.Fprintf(d.Stderr, "%2d) %s (%d episodes)\n", i+1, s.Title, s.EpisodeCount)
	}
	fmt.Fprintf(d.Stderr, "choose 1-%d: ", len(items))
	if d.Stdin == nil {
		return model.SeriesSummary{}, errors.New("--pick needs an interactive stdin")
	}
	line, err := bufio.NewReader(d.Stdin).ReadString('\n')
	if err != nil && strings.TrimSpace(line) == "" {
		return model.SeriesSummary{}, usage("no series chosen")
	}
	n, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || n < 1 || n > len(items) {
		return model.SeriesSummary{}, usage("choose a number from 1 to %d", len(items))
	}
	return items[n-1], nil
}

func isNumeric(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// resumePoint picks where to resume: inside the episode left unfinished, or
// at the start of the one after the last finished episode. A series never
// played starts at episode 1.
func resumePoint(ctx context.Context, h WatchHistory, seriesID string, total int) (episode, start int, err error) {
	e, ok, err := h.Get(ctx, seriesID)
	if err != nil {
		return 0, 0, err
	}
	if !ok {
		return 1, 0, nil
	}
	if !e.Finished {
		start = int(e.Position) - resumeBackoff
		if start < 0 {
			start = 0
		}
		return e.Episode, start, nil
	}
	if e.Episode >= total {
		return 0, 0, usage("all %d episodes of %s were watched; pass --episode to replay one", total, seriesID)
	}
	return e.Episode + 1, 0, nil
}

// recordHistory saves the watch position. A failure only warns, because it
// must not interrupt playback.
func recordHistory(ctx context.Context, d Dependencies, e history.Entry) {
	if err := d.History.Record(ctx, e); err != nil && ctx.Err() == nil {
		fmt.Fprintln(d.Stderr, "ohg: warning: history not saved:", err)
	}
}

// requireVideo refuses a title with no video platform: books and audiobooks
// share the series detail endpoint and list their chapters as episodes, but
// their playback model answers 101000 (observed 2026-10-24 on 25 videos with
// platform 3 or 4 and 7 books and audiobooks with platform 0).
func requireVideo(series *model.Series) error {
	if series.Platform != model.PlatformUnknown {
		return nil
	}
	return usage("%s (%s) is a book or audiobook, not a video; play and download take short dramas and films", series.Title, series.ID)
}

// cmdContinue resumes the series played most recently.
func cmdContinue(ctx context.Context, args []string, d Dependencies) error {
	if d.History == nil {
		return errors.New("watch history is not configured")
	}
	entries, err := d.History.List(ctx)
	if err != nil {
		return err
	}
	if len(entries) == 0 {
		return usage("nothing to continue; play a series first")
	}
	return cmdPlay(ctx, append([]string{entries[0].SeriesID, "--resume"}, args...), d)
}
