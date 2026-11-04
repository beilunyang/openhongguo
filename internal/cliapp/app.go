package cliapp

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/beilunyang/openhongguo/internal/history"
	"github.com/beilunyang/openhongguo/internal/media"
	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
)

const (
	ExitOK          = 0
	ExitUsage       = 2
	ExitUnavailable = 3
	ExitFailure     = 1
)

// Service is the anonymous catalog and playback surface the CLI calls.
type Service interface {
	Discover(context.Context, sdk.DiscoverOptions) (*sdk.Page[model.SeriesSummary], error)
	Search(context.Context, string, sdk.SearchOptions) (*sdk.Page[model.SeriesSummary], error)
	GetSeries(context.Context, string) (*model.Series, error)
	GetSeriesBatch(context.Context, []string) ([]model.SeriesDetail, error)
	GetAlbum(context.Context, string, sdk.AlbumOptions) (*model.Album, error)
	Seasons(context.Context, string) (*model.Album, error)
	Suggest(context.Context, string) ([]model.Suggestion, error)
	SearchCues(context.Context) ([]string, error)
	HotSearch(context.Context) (*model.HotSearch, error)
	NewReleases(context.Context, sdk.NewReleaseQuery) (*model.NewReleasePage, error)
	Person(context.Context, string) (*model.Person, error)
	PersonWorks(context.Context, string, string) (*model.WorkPage, error)
	SeriesRecommend(context.Context, string) (*model.Recommend, error)
	ShortFilterPanel(context.Context, string) ([]model.FilterRow, error)
	ShortFilter(context.Context, sdk.ShortFilterQuery) (*model.ShortFilterPage, error)
	Ranking(context.Context, string, string) (*model.Ranking, error)
	RankRows(context.Context, string, int) (*model.Ranking, error)
	RankGroups(context.Context) ([]model.RankList, error)
	ShortRank(context.Context, sdk.ShortRankQuery) (*model.ShortRanking, error)
	Categories(context.Context, string) (*model.CategoryPage, error)
	CategoryLanding(context.Context, string, sdk.LandingOptions) (*model.CategoryPage, error)
	ListEpisodes(context.Context, string, sdk.EpisodeOptions) (*sdk.Page[model.Episode], error)
	ResolvePlayback(context.Context, string, sdk.PlaybackOptions) (*model.Playback, error)
}

// Media plays a resolved stream with an external player, saves it, or
// rewrites a protected stream into a clear file with a caller-supplied key.
type Media interface {
	Open(ctx context.Context, url, player string) error
	Save(ctx context.Context, url, path string) (int64, error)
	Decrypt(ctx context.Context, url, audio, key, path string, limitSeconds int) (int64, error)
	// Play plays a stream without writing a file, decrypting it with
	// req.Key when set, and reports where playback stopped and whether the
	// viewer asked for another episode. It returns media.ErrStopped when the
	// player is closed before the end and media.ErrNeedsFile when the player
	// can only open a decrypted file.
	Play(ctx context.Context, req media.Request) (media.Result, error)
}

// ProfileStore is the local anonymous profile.
type ProfileStore interface {
	Load(context.Context) (*sdk.DeviceProfile, error)
	Reset(context.Context) error
}

// WatchHistory remembers the last episode played per series.
type WatchHistory interface {
	Get(ctx context.Context, seriesID string) (history.Entry, bool, error)
	Record(ctx context.Context, e history.Entry) error
	List(ctx context.Context) ([]history.Entry, error)
	Clear(ctx context.Context) error
}

// Dependencies wires the CLI to the shipped SDK. History is optional; without
// it playback is not remembered and --resume is unavailable.
type Dependencies struct {
	Service  Service
	Profiles ProfileStore
	Media    Media
	History  WatchHistory
	// Stdin answers the --pick prompt.
	Stdin          io.Reader
	Stdout, Stderr io.Writer
}

// Run executes one anonymous CLI invocation.
func Run(ctx context.Context, args []string, d Dependencies) int {
	if d.Stdout == nil {
		d.Stdout = io.Discard
	}
	if d.Stderr == nil {
		d.Stderr = io.Discard
	}
	if err := rejectAccountFlags(args); err != nil {
		fmt.Fprintln(d.Stderr, "ohg:", err)
		return ExitUsage
	}
	err := dispatch(ctx, args, d)
	if err == nil {
		return ExitOK
	}
	fmt.Fprintln(d.Stderr, "ohg:", strings.TrimPrefix(err.Error(), "openhongguo: "))
	if errors.Is(err, errUsage) {
		return ExitUsage
	}
	if errors.Is(err, sdk.ErrSignerUnavailable) || errors.Is(err, sdk.ErrDRMProtected) || errors.Is(err, sdk.ErrAuthenticationRequired) || errors.Is(err, sdk.ErrAnonymousLimited) {
		return ExitUnavailable
	}
	return ExitFailure
}

var errUsage = errors.New("invalid arguments")

func usage(format string, args ...any) error {
	return fmt.Errorf("%w: %s", errUsage, fmt.Sprintf(format, args...))
}

func rejectAccountFlags(args []string) error {
	for _, arg := range args {
		name := strings.TrimLeft(arg, "-")
		name = strings.SplitN(name, "=", 2)[0]
		switch strings.ToLower(name) {
		case "login", "password", "token", "cookie", "session", "username":
			return usage("account login is not available")
		}
	}
	return nil
}

func dispatch(ctx context.Context, args []string, d Dependencies) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" || args[0] == "help" {
		fmt.Fprintln(d.Stdout, helpText)
		return nil
	}
	switch args[0] {
	case "discover":
		return cmdDiscover(ctx, args[1:], d)
	case "search":
		return cmdSearch(ctx, args[1:], d)
	case "info":
		return cmdInfo(ctx, args[1:], d)
	case "episodes":
		return cmdEpisodes(ctx, args[1:], d)
	case "play":
		return cmdPlay(ctx, args[1:], d)
	case "landing":
		return cmdLanding(ctx, args[1:], d)
	case "category":
		return cmdCategory(ctx, args[1:], d)
	case "rank":
		return cmdRank(ctx, args[1:], d)
	case "filter":
		return cmdFilter(ctx, args[1:], d)
	case "related":
		return cmdRelated(ctx, args[1:], d)
	case "hot":
		return cmdHot(ctx, args[1:], d)
	case "new":
		return cmdNew(ctx, args[1:], d)
	case "person":
		return cmdPerson(ctx, args[1:], d)
	case "cues":
		return cmdCues(ctx, args[1:], d)
	case "suggest":
		return cmdSuggest(ctx, args[1:], d)
	case "seasons":
		return cmdSeasons(ctx, args[1:], d)
	case "album":
		return cmdAlbum(ctx, args[1:], d)
	case "device":
		return cmdDevice(ctx, args[1:], d)
	case "history":
		return cmdHistory(ctx, args[1:], d)
	case "continue":
		return cmdContinue(ctx, args[1:], d)
	case "download":
		return cmdDownload(ctx, args[1:], d)
	default:
		return usage("unknown command %q", args[0])
	}
}

const helpText = `ohg — anonymous short-drama catalog and playback

  ohg discover [--channel 推荐|找剧|首页真人剧|首页漫剧|电影|听书|小说|社区|漫画|大视听|经典|知识|漫剧|真人剧]
               [--cursor TOKEN] [--all [--pages N]] [--json]
  ohg search <keyword> [--tab 综合|剧集|视频|讨论|小说|用户|听书] [--page-size N] [--cursor OFFSET]
              [--filter 真人剧|漫剧|AI剧|影视剧|未看过]... [--all [--pages N]] [--json]
  ohg info <series-id>... [--json]
  ohg episodes <series-id> [--json]
  ohg album <album-id> [--json]
  ohg seasons <series-id> [--json]
  ohg rank --shorts [--content ID] [--board ID] [--cursor C] [--json]
  ohg rank [--groups] [--cell ID] [--offset N] [--json]
  ohg rank --activity ID [--list ID] [--json]
  ohg filter --panel [--genre short_play|comic_series|ai_series] [--json]
  ohg filter [--genre G] [--filter KEY=VALUE]... [--cursor C] [--json]
  ohg related <series-id> [--json]
  ohg suggest <prefix> [--json]
  ohg hot [--json]
  ohg new [--date YYYYMMDD] [--cursor C] [--json]
  ohg person <user-id> [--cursor C] [--json]
  ohg cues [--json]
  ohg category <category-id> [--json]
  ohg landing <category-id> [--tab 8|1|0] [--filter KEY=VALUE]... [--cursor OFFSET] [--json]
  ohg play <series-id|title> [--pick] [--episode N | --resume] [--autoplay]
                          [--quality best|worst|360p|480p|540p|720p|1080p]
                          [--player auto|mpv|ffplay|vlc|open|xdg-open] [--url] [--json]
                          [--download PATH] [--key HEX] [--preview SECONDS] [--no-decrypt]
  ohg continue [play flags]
  ohg download <series-id|title> [--pick] [--episodes 1-10,12] [--dir PATH]
                          [--jobs N] [--quality Q]
  ohg history [--json]
  ohg history clear
  ohg device show
  ohg device reset

Every command is anonymous. There is no login.

play and download take a series ID or a title. A title is searched and the
best match is used; --pick lists the matches to choose from.

play resolves the service media model and streams it without writing a
file: ffplay and mpv decrypt a cenc-aes-ctr stream themselves, so seeking
works; vlc reads an ffmpeg-decrypted pipe; with open/xdg-open the episode is
decrypted to a temp file first. The content key is recovered from the
model's wrapped key material, or taken from --key HEX / OHG_CENC_KEY.
ffmpeg is required for decryption outside ffplay and mpv.
--download PATH saves one episode; --preview SECONDS plays or saves only the
leading seconds; --no-decrypt reports the protected model and stops.

--autoplay continues with the next episode when one ends, until the last
episode, until the player is closed early, or until Ctrl-C. In mpv, > and <
switch to the next and previous episode at any time.
Playback is remembered per series, including where an episode was left.
--resume continues from there, or from the episode after the last one
watched to the end; continue resumes the most recent series.

download saves every listed episode (default: all) as 第NNN集.mp4 under
--dir (default: the series title), --jobs at a time (default 3). Episodes
already saved are skipped, so an interrupted download can be run again.`

func requireService(d Dependencies) error {
	if d.Service == nil {
		return errors.New("service is not configured")
	}
	return nil
}

func usageFrom(err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()
	if strings.HasPrefix(msg, "unknown discover channel") || strings.HasPrefix(msg, "unknown search tab") || strings.HasPrefix(msg, "search filters apply") || strings.HasSuffix(msg, "cursor is not verified") {
		return usage("%s", msg)
	}
	return err
}

func cmdDevice(ctx context.Context, args []string, d Dependencies) error {
	if len(args) != 1 {
		return usage("device needs show or reset")
	}
	if d.Profiles == nil {
		return errors.New("profile store is not configured")
	}
	switch args[0] {
	case "show":
		p, err := d.Profiles.Load(ctx)
		if err != nil {
			return err
		}
		return printValue(d.Stdout, p, false)
	case "reset":
		return d.Profiles.Reset(ctx)
	default:
		return usage("unknown device command %q", args[0])
	}
}

func cmdHistory(ctx context.Context, args []string, d Dependencies) error {
	if d.History == nil {
		return errors.New("watch history is not configured")
	}
	if len(args) == 1 && args[0] == "clear" {
		return d.History.Clear(ctx)
	}
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(f.rest) != 0 {
		return usage("history takes no arguments other than clear")
	}
	entries, err := d.History.List(ctx)
	if err != nil {
		return err
	}
	return printValue(d.Stdout, entries, f.json)
}

func validateID(raw string) (string, error) {
	if raw == "" {
		return "", usage("missing series ID")
	}
	if strings.Contains(raw, "://") {
		return "", usage("share links are not supported; use a series ID")
	}
	for _, r := range raw {
		if r < '0' || r > '9' {
			return "", usage("series ID must be numeric")
		}
	}
	if len(raw) < 4 || len(raw) > 25 {
		return "", usage("series ID must be 4-25 digits")
	}
	return raw, nil
}

// usageFromID turns a malformed ID or cursor into a usage error.
func usageFromID(err error) error {
	if msg := err.Error(); strings.HasPrefix(msg, "invalid user ID") || strings.HasPrefix(msg, "the cursor belongs") || strings.HasSuffix(msg, "cursor is not valid") {
		return usage("%s", msg)
	}
	return err
}
