package cliapp

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/beilunyang/openhongguo/adapters/filestore"
	"github.com/beilunyang/openhongguo/adapters/httptransport"
	"github.com/beilunyang/openhongguo/internal/history"
	mediapkg "github.com/beilunyang/openhongguo/internal/media"
	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
)

func fixture(t *testing.T, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "testdata", "protocol", name))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func TestCLIAnonymousReadsUseShippedClient(t *testing.T) {
	files := map[string]string{
		protocol.PathDiscover: "discover-response.json",
		protocol.PathSearch:   "search-response.json",
		protocol.PathSeries:   "series-response.json",
		protocol.PathRegister: "register-response.json",
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Cookie") != "" || r.Header.Get("Authorization") != "" {
			t.Errorf("credential header on %s", r.URL.Path)
		}
		name, ok := files[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(fixture(t, name))
	}))
	t.Cleanup(srv.Close)

	dir := t.TempDir()
	store := filestore.New(filepath.Join(dir, "anonymous.json"))
	transport := httptransport.New()
	client, err := sdk.NewClient(sdk.Options{
		Transport: transport, ProfileStore: store,
		APIOrigin: srv.URL, RegisterOrigin: srv.URL,
		Clock: func() time.Time { return time.UnixMilli(1_700_000_000_123).UTC() },
	})
	if err != nil {
		t.Fatal(err)
	}
	deps := Dependencies{Service: client, Profiles: store}

	var out bytes.Buffer
	deps.Stdout = &out
	deps.Stderr = io.Discard
	if code := Run(context.Background(), []string{"discover", "--json"}, deps); code != ExitOK {
		t.Fatalf("discover exit %d", code)
	}
	first := out.String()
	if !strings.Contains(first, "7100000000000101") || !strings.Contains(first, "Fixture Lantern Court") {
		t.Fatalf("discover output = %s", first)
	}
	out.Reset()
	if code := Run(context.Background(), []string{"discover", "--json"}, deps); code != ExitOK {
		t.Fatalf("second discover exit %d", code)
	}
	if out.String() != first {
		t.Fatalf("discover runs differ\n%s\n%s", first, out.String())
	}

	out.Reset()
	if code := Run(context.Background(), []string{"search", "短剧", "--json"}, deps); code != ExitOK {
		t.Fatalf("search exit %d", code)
	}
	if !strings.Contains(out.String(), "7100000000000101") || !strings.Contains(out.String(), "Fixture Lantern Court") {
		t.Fatalf("search output = %s", out.String())
	}

	out.Reset()
	if code := Run(context.Background(), []string{"info", "7100000000000101", "--json"}, deps); code != ExitOK {
		t.Fatalf("info exit %d", code)
	}
	if !strings.Contains(out.String(), "Fixture Lantern Court") || !strings.Contains(out.String(), "A public synopsis.") || !strings.Contains(out.String(), "episode_count") {
		t.Fatalf("info output = %s", out.String())
	}

	out.Reset()
	if code := Run(context.Background(), []string{"episodes", "7100000000000101"}, deps); code != ExitOK {
		t.Fatalf("episodes exit %d", code)
	}
	if !strings.HasPrefix(out.String(), "1\t7100000000000201\tEpisode One\t90\n") {
		t.Fatalf("episodes output = %s", out.String())
	}

	info, err := os.Stat(filepath.Join(dir, "anonymous.json"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("profile mode = %o", info.Mode().Perm())
	}
	saved, err := os.ReadFile(filepath.Join(dir, "anonymous.json"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(saved, []byte("device_token")) || bytes.Contains(saved, []byte("Cookie")) {
		t.Fatalf("profile stored a credential: %s", saved)
	}

	var show bytes.Buffer
	deps.Stdout = &show
	if code := Run(context.Background(), []string{"device", "show"}, deps); code != ExitOK {
		t.Fatalf("show exit %d", code)
	}
	if !strings.Contains(show.String(), "device_id=7100000000000001") || strings.Contains(show.String(), "device_token") {
		t.Fatalf("show = %s", show.String())
	}
	if code := Run(context.Background(), []string{"device", "reset"}, deps); code != ExitOK {
		t.Fatalf("reset exit %d", code)
	}
	if _, err := os.Stat(filepath.Join(dir, "anonymous.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("profile remains: %v", err)
	}
}

type captureService struct {
	channel string
	tab     string
	filters []string
	batch   []string
	// newQuery and worksCursor record the 上新 and works reads.
	newQuery    sdk.NewReleaseQuery
	worksCursor string
	// rankQuery, filterQuery, and panelGenre record the short-drama reads.
	rankQuery   sdk.ShortRankQuery
	filterQuery sdk.ShortFilterQuery
	panelGenre  string
}

func (c *captureService) Discover(_ context.Context, opts sdk.DiscoverOptions) (*sdk.Page[model.SeriesSummary], error) {
	c.channel = opts.Channel
	return &sdk.Page[model.SeriesSummary]{Items: []model.SeriesSummary{{ID: "7100000000000101", Title: "Fixture"}}}, nil
}

func (c *captureService) Search(_ context.Context, _ string, opts sdk.SearchOptions) (*sdk.Page[model.SeriesSummary], error) {
	c.tab = opts.Tab
	c.filters = opts.Filters
	if _, err := protocol.ResolveSearchTab(opts.Tab); err != nil {
		return nil, err
	}
	return &sdk.Page[model.SeriesSummary]{Items: []model.SeriesSummary{{ID: "7100000000000101", Title: "Fixture"}}}, nil
}

func (c *captureService) GetSeriesBatch(_ context.Context, ids []string) ([]model.SeriesDetail, error) {
	c.batch = ids
	return []model.SeriesDetail{
		{Series: &model.Series{SeriesSummary: model.SeriesSummary{ID: ids[0], Title: "One", EpisodeCount: 2},
			Novel: &model.Novel{BookID: "7100000000002001", Title: "Novel", Status: model.NovelOngoing, Tags: []string{"悬疑", "穿越"}}}},
	}, nil
}

func (c *captureService) GetSeries(context.Context, string) (*model.Series, error) {
	return nil, errors.New("unused")
}

func (c *captureService) GetAlbum(context.Context, string, sdk.AlbumOptions) (*model.Album, error) {
	return nil, errors.New("unused")
}

func (c *captureService) Seasons(context.Context, string) (*model.Album, error) {
	return nil, errors.New("unused")
}

func (c *captureService) Suggest(context.Context, string) ([]model.Suggestion, error) {
	return nil, errors.New("unused")
}

func (c *captureService) SearchCues(context.Context) ([]string, error) {
	return []string{"书名", "主角"}, nil
}

func (c *captureService) NewReleases(_ context.Context, q sdk.NewReleaseQuery) (*model.NewReleasePage, error) {
	c.newQuery = q
	return &model.NewReleasePage{Date: "20261024", Dates: []string{"20261023", "20261024"}, Cursor: "s1.next", Items: []model.NewRelease{
		{SeriesID: "7100000000001501", Title: "Out", Online: true, Heat: "378万热度", EpisodeCount: 60},
		{SeriesID: "7100000000001502", Title: "Soon", Heat: "159.7万人预约"},
	}}, nil
}

func (c *captureService) Person(_ context.Context, id string) (*model.Person, error) {
	return &model.Person{UserID: id, Name: "Actor", Description: "Bio", Tags: []string{"女演员"}, Labels: []string{"水瓶座"}, Followers: 3, Likes: 4}, nil
}

func (c *captureService) PersonWorks(_ context.Context, id, cursor string) (*model.WorkPage, error) {
	c.worksCursor = cursor
	return &model.WorkPage{Cursor: "w1.next", Items: []model.Work{{SeriesID: "7100000000001801", Title: "Work", EpisodeCount: 70, Tags: []string{"爱情", "70集"}}}}, nil
}

func (c *captureService) HotSearch(context.Context) (*model.HotSearch, error) {
	return &model.HotSearch{
		Guesses: []model.HotItem{{Position: 1, SeriesID: "7100000000000101", Title: "Fixture", Heat: "12万热度"}},
		Boards: []model.HotBoard{
			{Name: "短剧热搜榜", Items: []model.HotItem{{Position: 1, SeriesID: "7100000000000102", Title: "Hot", SearchScore: "355万热搜值", Heat: "4102万热度"}}},
			{Name: "热点话题榜", Topics: []model.HotTopic{{ID: "7100000000001001", Title: "Topic", Query: "Topic query", Heat: "4.6万热度"}}},
		},
	}, nil
}

func (c *captureService) SeriesRecommend(context.Context, string) (*model.Recommend, error) {
	return nil, errors.New("unused")
}

func (c *captureService) ShortFilterPanel(_ context.Context, genre string) ([]model.FilterRow, error) {
	c.panelGenre = genre
	return []model.FilterRow{{Key: "category_dim_theme", Name: "全部主题", Options: []model.FilterOption{{Name: "悬疑", Value: "cate_165"}, {Name: "玄幻", Value: "cate_7"}}}}, nil
}

func (c *captureService) ShortFilter(_ context.Context, q sdk.ShortFilterQuery) (*model.ShortFilterPage, error) {
	c.filterQuery = q
	return &model.ShortFilterPage{Items: []model.SeriesSummary{{ID: "7100000000000101", Title: "Fixture", EpisodeCount: 12}}, HasMore: true, Cursor: "s1.next"}, nil
}

func (c *captureService) Ranking(context.Context, string, string) (*model.Ranking, error) {
	return nil, errors.New("unused")
}

func (c *captureService) RankRows(context.Context, string, int) (*model.Ranking, error) {
	return nil, errors.New("unused")
}

func (c *captureService) RankGroups(context.Context) ([]model.RankList, error) {
	return nil, errors.New("unused")
}

func (c *captureService) ShortRank(_ context.Context, q sdk.ShortRankQuery) (*model.ShortRanking, error) {
	c.rankQuery = q
	if q.Content == "ranklist_celebrity" {
		return &model.ShortRanking{Content: q.Content, Items: []model.RankItem{
			{Position: 1, ActorID: "7100000000001401", UserID: "7100000000001301", Actor: "Actor A", Score: "759万人气", Details: []string{"67部作品", "双周追剧403万人"},
				Change: &model.RankChange{Kind: model.RankUp, Places: 1}},
			{Position: 2, ActorID: "7100000000001402", Actor: "Actor B", Score: "556万人气", Change: &model.RankChange{Kind: model.RankNew}},
		}}, nil
	}
	return &model.ShortRanking{
		Contents: []model.RankContent{
			{ID: "all", Name: "全部"},
			{ID: "human", Name: "真人剧", Selected: true, Boards: []model.RankBoard{{ID: "human_hot_sc", Name: "推荐榜"}, {ID: "human_hot_play", Name: "热播榜", Selected: true}}},
		},
		Content: "human", Board: "human_hot_play", Cursor: "s1.next",
		Items: []model.RankItem{
			{Position: 1, Rank: 1, SeriesID: "7100000000000101", Title: "First", EpisodeCount: 60, Score: "7921万热度"},
			{Position: 2, Rank: 2, SeriesID: "7100000000000102", Title: "Inserted", EpisodeCount: 30, Score: "平台推荐", Promoted: true},
		},
	}, nil
}

func (c *captureService) Categories(context.Context, string) (*model.CategoryPage, error) {
	return nil, errors.New("unused")
}

func (c *captureService) CategoryLanding(context.Context, string, sdk.LandingOptions) (*model.CategoryPage, error) {
	return nil, errors.New("unused")
}

func (c *captureService) ListEpisodes(context.Context, string, sdk.EpisodeOptions) (*sdk.Page[model.Episode], error) {
	return nil, errors.New("unused")
}

func (c *captureService) ResolvePlayback(context.Context, string, sdk.PlaybackOptions) (*model.Playback, error) {
	return nil, errors.New("unused")
}

func TestCLIPassesChannelAndSearchTab(t *testing.T) {
	svc := &captureService{}
	deps := Dependencies{Service: svc, Stdout: io.Discard, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"discover", "--channel", "电影"}, deps); code != ExitOK {
		t.Fatalf("discover exit %d", code)
	}
	if svc.channel != "电影" {
		t.Fatalf("channel = %s", svc.channel)
	}
	if code := Run(context.Background(), []string{"search", "短剧", "--tab", "用户"}, deps); code != ExitOK {
		t.Fatalf("search exit %d", code)
	}
	if svc.tab != "用户" {
		t.Fatalf("tab = %s", svc.tab)
	}
	var errOut bytes.Buffer
	deps.Stderr = &errOut
	if code := Run(context.Background(), []string{"search", "短剧", "--tab", "漫画"}, deps); code != ExitUsage {
		t.Fatalf("comic tab exit %d", code)
	}
	if code := Run(context.Background(), []string{"search", "奇幻", "--filter", "真人剧,影视剧", "--filter", "未看过"}, deps); code != ExitOK {
		t.Fatalf("filtered search exit %d", code)
	}
	if fmt.Sprint(svc.filters) != "[content_type_short_play content_type_long_video search_range_unread]" {
		t.Fatalf("filters = %v", svc.filters)
	}
	if code := Run(context.Background(), []string{"search", "奇幻", "--filter", "动漫"}, deps); code != ExitUsage {
		t.Fatalf("unknown filter exit %d", code)
	}
}

func TestDeviceShowOmitsGuestTicket(t *testing.T) {
	path := filepath.Join(t.TempDir(), "anonymous.json")
	store := filestore.New(path)
	profile, err := sdk.NewAnonymousDeviceProfile(strings.NewReader(strings.Repeat("a", 72)), func() time.Time { return time.Unix(1, 0).UTC() })
	if err != nil {
		t.Fatal(err)
	}
	profile.DeviceID = "7100000000000001"
	profile.InstallID = "7100000000000002"
	profile.TTReq = "fixture-ticket-value"
	profile.StoreRegion = "cn"
	profile.StoreRegionSrc = "did"
	if err := store.Save(context.Background(), profile); err != nil {
		t.Fatal(err)
	}
	saved, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(saved, []byte("Cookie")) || bytes.Contains(saved, []byte("device_token")) {
		t.Fatal("profile stored an account credential")
	}
	if !bytes.Contains(saved, []byte("fixture-ticket-value")) {
		t.Fatal("profile dropped the guest ticket")
	}
	var show bytes.Buffer
	code := Run(context.Background(), []string{"device", "show"}, Dependencies{Profiles: store, Stdout: &show, Stderr: io.Discard})
	if code != ExitOK {
		t.Fatalf("show exit %d", code)
	}
	if strings.Contains(show.String(), "fixture-ticket-value") || strings.Contains(show.String(), "ttreq") || strings.Contains(show.String(), "store_region") {
		t.Fatalf("show leaked the guest ticket: %s", show.String())
	}
	if !strings.Contains(show.String(), "device_id=7100000000000001") || !strings.Contains(show.String(), "install_id=7100000000000002") {
		t.Fatalf("show = %s", show.String())
	}
}

func TestCLIRejectsLoginAndMapsAccountError(t *testing.T) {
	var errOut bytes.Buffer
	code := Run(context.Background(), []string{"discover", "--login"}, Dependencies{Stderr: &errOut})
	if code != ExitUsage || !strings.Contains(errOut.String(), "login") {
		t.Fatalf("login flag code=%d err=%s", code, errOut.String())
	}
	errOut.Reset()
	code = Run(context.Background(), []string{"--token", "abc"}, Dependencies{Stderr: &errOut})
	if code != ExitUsage {
		t.Fatalf("token flag code=%d", code)
	}

	reg := fixture(t, "register-response.json")
	account := fixture(t, "account-required.json")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == protocol.PathRegister {
			_, _ = w.Write(reg)
			return
		}
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write(account)
	}))
	t.Cleanup(srv.Close)
	store := filestore.New(filepath.Join(t.TempDir(), "anonymous.json"))
	transport := httptransport.New()
	client, err := sdk.NewClient(sdk.Options{Transport: transport, ProfileStore: store, APIOrigin: srv.URL, RegisterOrigin: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	errOut.Reset()
	code = Run(context.Background(), []string{"discover"}, Dependencies{Service: client, Profiles: store, Stdout: io.Discard, Stderr: &errOut})
	if code != ExitUnavailable || !strings.Contains(errOut.String(), "authentication required") {
		t.Fatalf("auth code=%d err=%s", code, errOut.String())
	}
	if strings.Contains(errOut.String(), "DRM") {
		t.Fatalf("auth stderr mentioned DRM: %s", errOut.String())
	}
}

type playService struct {
	episodes  []model.Episode
	playback  *model.Playback
	playErr   error
	episodeID string
}

func (s *playService) Discover(context.Context, sdk.DiscoverOptions) (*sdk.Page[model.SeriesSummary], error) {
	return nil, errors.New("unused")
}

func (s *playService) Search(context.Context, string, sdk.SearchOptions) (*sdk.Page[model.SeriesSummary], error) {
	return nil, errors.New("unused")
}

func (s *playService) GetSeriesBatch(context.Context, []string) ([]model.SeriesDetail, error) {
	return nil, errors.New("unused")
}

func (s *playService) GetSeries(context.Context, string) (*model.Series, error) {
	return nil, errors.New("unused")
}

func (s *playService) GetAlbum(context.Context, string, sdk.AlbumOptions) (*model.Album, error) {
	return nil, errors.New("unused")
}

func (s *playService) Seasons(context.Context, string) (*model.Album, error) {
	return nil, errors.New("unused")
}

func (s *playService) Suggest(context.Context, string) ([]model.Suggestion, error) {
	return nil, errors.New("unused")
}

func (s *playService) SearchCues(context.Context) ([]string, error) {
	return nil, errors.New("unused")
}

func (s *playService) NewReleases(context.Context, sdk.NewReleaseQuery) (*model.NewReleasePage, error) {
	return nil, errors.New("unused")
}

func (s *playService) Person(context.Context, string) (*model.Person, error) {
	return nil, errors.New("unused")
}

func (s *playService) PersonWorks(context.Context, string, string) (*model.WorkPage, error) {
	return nil, errors.New("unused")
}

func (s *playService) HotSearch(context.Context) (*model.HotSearch, error) {
	return nil, errors.New("unused")
}

func (s *playService) SeriesRecommend(context.Context, string) (*model.Recommend, error) {
	return nil, errors.New("unused")
}

func (s *playService) ShortFilterPanel(context.Context, string) ([]model.FilterRow, error) {
	return nil, errors.New("unused")
}

func (s *playService) ShortFilter(context.Context, sdk.ShortFilterQuery) (*model.ShortFilterPage, error) {
	return nil, errors.New("unused")
}

func (s *playService) Ranking(context.Context, string, string) (*model.Ranking, error) {
	return nil, errors.New("unused")
}

func (s *playService) RankRows(context.Context, string, int) (*model.Ranking, error) {
	return nil, errors.New("unused")
}

func (s *playService) RankGroups(context.Context) ([]model.RankList, error) {
	return nil, errors.New("unused")
}

func (s *playService) ShortRank(context.Context, sdk.ShortRankQuery) (*model.ShortRanking, error) {
	return nil, errors.New("unused")
}

func (s *playService) Categories(context.Context, string) (*model.CategoryPage, error) {
	return nil, errors.New("unused")
}

func (s *playService) CategoryLanding(context.Context, string, sdk.LandingOptions) (*model.CategoryPage, error) {
	return nil, errors.New("unused")
}

func (s *playService) ListEpisodes(context.Context, string, sdk.EpisodeOptions) (*sdk.Page[model.Episode], error) {
	return &sdk.Page[model.Episode]{Items: s.episodes}, nil
}

func (s *playService) ResolvePlayback(_ context.Context, episodeID string, _ sdk.PlaybackOptions) (*model.Playback, error) {
	s.episodeID = episodeID
	return s.playback, s.playErr
}

type fakeMedia struct {
	mu        sync.Mutex
	opened    string
	player    string
	savedPath string
	savedURL  string
	written   int64
	err       error

	decryptURL   string
	decryptAudio string
	decryptLimit int
	decryptKey   string
	decryptPath  string
	decrypted    int64
	decryptErr   error

	streamURL      string
	streamAudio    string
	streamKey      string
	streamPlayer   string
	streamLimit    int
	streamDuration int
	streamed       []string
	streamStart    int
	// streamErrs and streamResults are returned call by call; a missing
	// entry means success.
	streamErrs    []error
	streamResults []mediapkg.Result
	// writeFiles makes Save and Decrypt create their output file.
	writeFiles bool
	saved      []string
}

func (m *fakeMedia) Open(_ context.Context, url, player string) error {
	m.opened = url
	m.player = player
	return m.err
}

func (m *fakeMedia) Save(_ context.Context, url, path string) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.savedURL = url
	m.savedPath = path
	return m.written, m.err
}

func (m *fakeMedia) Decrypt(_ context.Context, url, audio, key, path string, limit int) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.saved = append(m.saved, url)
	if m.writeFiles && m.decryptErr == nil {
		if err := os.WriteFile(path, []byte("mp4"), 0o644); err != nil {
			return 0, err
		}
	}
	m.decryptLimit = limit
	m.decryptURL = url
	m.decryptAudio = audio
	m.decryptKey = key
	m.decryptPath = path
	if m.decryptErr != nil {
		return 0, m.decryptErr
	}
	if m.decrypted == 0 {
		return 4096, nil
	}
	return m.decrypted, nil
}

func (m *fakeMedia) Play(_ context.Context, req mediapkg.Request) (mediapkg.Result, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.streamURL, m.streamAudio, m.streamKey, m.streamPlayer, m.streamLimit = req.URL, req.Audio, req.Key, req.Player, req.LimitSeconds
	m.streamDuration, m.streamStart = req.DurationSeconds, req.StartSeconds
	m.streamed = append(m.streamed, req.URL)
	i := len(m.streamed) - 1
	var res mediapkg.Result
	if i < len(m.streamResults) {
		res = m.streamResults[i]
	}
	if i < len(m.streamErrs) {
		return res, m.streamErrs[i]
	}
	return res, nil
}

func clearPlayback() *model.Playback {
	return &model.Playback{EpisodeID: "7100000000000201", VideoID: "v-fixture", DurationSeconds: 42, Streams: []model.Stream{
		{Definition: "360p", Codec: "h264", URL: "https://media.invalid/360.mp4", Size: 1000},
		{Definition: "1080p", Codec: "bytevc1", URL: "https://media.invalid/1080.mp4", Size: 9000},
	}}
}

func protectedPlayback() *model.Playback {
	return &model.Playback{EpisodeID: "7100000000000201", Protected: true, Streams: []model.Stream{
		{Definition: "720p", Codec: "bytevc1", URL: "https://media.invalid/drm.mp4", Protected: true, Encryption: "cenc-aes-ctr"},
	}}
}

func playFixtures() []model.Episode {
	return []model.Episode{
		{ID: "7100000000000201", SeriesID: "7100000000000101", Number: 1},
		{ID: "7100000000000202", SeriesID: "7100000000000101", Number: 2},
	}
}

func TestPlayHandsClearStreamToPlayer(t *testing.T) {
	svc := &playService{episodes: playFixtures(), playback: clearPlayback()}
	media := &fakeMedia{}
	var out bytes.Buffer
	deps := Dependencies{Service: svc, Media: media, Stdout: &out, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101"}, deps); code != ExitOK {
		t.Fatalf("play exit %d", code)
	}
	if media.streamURL != "https://media.invalid/1080.mp4" || media.streamPlayer != "" || media.streamKey != "" {
		t.Fatalf("played = %q player = %q key = %q", media.streamURL, media.streamPlayer, media.streamKey)
	}
	if media.streamDuration != 42 {
		t.Fatalf("duration = %d", media.streamDuration)
	}
	if svc.episodeID != "7100000000000201" {
		t.Fatalf("episode = %s", svc.episodeID)
	}

	if code := Run(context.Background(), []string{"play", "7100000000000101", "--episode", "2", "--quality", "360p", "--player", "mpv"}, deps); code != ExitOK {
		t.Fatalf("play episode 2 exit %d", code)
	}
	if svc.episodeID != "7100000000000202" || media.streamURL != "https://media.invalid/360.mp4" || media.streamPlayer != "mpv" {
		t.Fatalf("episode = %s played = %q player = %q", svc.episodeID, media.streamURL, media.streamPlayer)
	}
}

func TestPlayURLDownloadAndJSON(t *testing.T) {
	svc := &playService{episodes: playFixtures(), playback: clearPlayback()}
	media := &fakeMedia{written: 9000}
	var out bytes.Buffer
	deps := Dependencies{Service: svc, Media: media, Stdout: &out, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--url"}, deps); code != ExitOK {
		t.Fatalf("url exit %d", code)
	}
	if strings.TrimSpace(out.String()) != "https://media.invalid/1080.mp4" || media.opened != "" {
		t.Fatalf("url output = %q opened = %q", out.String(), media.opened)
	}
	out.Reset()
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--download", filepath.Join(t.TempDir(), "ep1.mp4")}, deps); code != ExitOK {
		t.Fatalf("download exit %d", code)
	}
	if media.savedURL != "https://media.invalid/1080.mp4" || !strings.Contains(out.String(), "\t9000\t1080p") {
		t.Fatalf("saved = %q out = %q", media.savedURL, out.String())
	}
	out.Reset()
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--json"}, deps); code != ExitOK {
		t.Fatalf("json exit %d", code)
	}
	if !strings.Contains(out.String(), `"definition": "1080p"`) || !strings.Contains(out.String(), `"protected": false`) {
		t.Fatalf("json = %s", out.String())
	}
}

func TestPlayReportsDRMWithoutOpening(t *testing.T) {
	svc := &playService{episodes: playFixtures(), playback: protectedPlayback(), playErr: sdk.ErrDRMProtected}
	media := &fakeMedia{}
	var errOut bytes.Buffer
	deps := Dependencies{Service: svc, Media: media, Stdout: io.Discard, Stderr: &errOut}
	if code := Run(context.Background(), []string{"play", "7100000000000101"}, deps); code != ExitUnavailable {
		t.Fatalf("drm exit %d err %s", code, errOut.String())
	}
	if media.opened != "" || media.savedPath != "" {
		t.Fatal("DRM stream reached the media launcher")
	}
	if !strings.Contains(errOut.String(), "cenc-aes-ctr") || !strings.Contains(errOut.String(), "DRM") {
		t.Fatalf("drm stderr = %s", errOut.String())
	}
	var jsonOut bytes.Buffer
	deps.Stdout = &jsonOut
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--json"}, deps); code != ExitUnavailable {
		t.Fatalf("drm json exit %d", code)
	}
	if !strings.Contains(jsonOut.String(), `"protected": true`) {
		t.Fatalf("drm json = %s", jsonOut.String())
	}
}

func TestPlayRejectsBadArguments(t *testing.T) {
	svc := &playService{episodes: playFixtures(), playback: clearPlayback()}
	deps := Dependencies{Service: svc, Media: &fakeMedia{}, Stdout: io.Discard, Stderr: io.Discard}
	cases := [][]string{
		{"play"},
		{"play", "https://share.invalid/abc"},
		{"play", "7100000000000101", "--pick"},
		{"play", "7100000000000101", "--episode", "0"},
		{"play", "7100000000000101", "--episode", "9"},
		{"play", "7100000000000101", "--quality", "4k"},
		{"play", "7100000000000101", "--url", "--download", "x.mp4"},
		{"play", "7100000000000101", "--key", "nothex"},
		{"play", "7100000000000101", "--url", "--key", strings.Repeat("aa", 16)},
	}
	for _, args := range cases {
		if code := Run(context.Background(), args, deps); code != ExitUsage {
			t.Fatalf("%v exit %d", args, code)
		}
	}
}

func TestPlayDecryptsProtectedStreamWithKey(t *testing.T) {
	key := strings.Repeat("0011", 8)
	svc := &playService{episodes: playFixtures(), playback: protectedPlayback(), playErr: sdk.ErrDRMProtected}
	out := filepath.Join(t.TempDir(), "ep1.mp4")
	media := &fakeMedia{}
	var buf bytes.Buffer
	deps := Dependencies{Service: svc, Media: media, Stdout: &buf, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--key", key, "--download", out}, deps); code != ExitOK {
		t.Fatalf("decrypt exit %d", code)
	}
	if media.decryptURL != "https://media.invalid/drm.mp4" || media.decryptKey != key || media.decryptPath != out {
		t.Fatalf("decrypt url=%q key=%q path=%q", media.decryptURL, media.decryptKey, media.decryptPath)
	}
	if !strings.Contains(buf.String(), "\t4096\t720p") {
		t.Fatalf("stdout = %q", buf.String())
	}
	if media.opened != "" {
		t.Fatal("decrypted download also opened a player")
	}
}

func TestPlayStreamsProtectedStreamIntoPlayer(t *testing.T) {
	key := strings.Repeat("ab", 16)
	svc := &playService{episodes: playFixtures(), playback: protectedPlayback(), playErr: sdk.ErrDRMProtected}
	media := &fakeMedia{streamErrs: []error{mediapkg.ErrStopped}}
	deps := Dependencies{Service: svc, Media: media, Stdout: io.Discard, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--key", key, "--player", "mpv"}, deps); code != ExitOK {
		t.Fatalf("stream play exit %d", code)
	}
	if media.streamURL != "https://media.invalid/drm.mp4" || media.streamKey != key || media.streamPlayer != "mpv" {
		t.Fatalf("stream url=%q key=%q player=%q", media.streamURL, media.streamKey, media.streamPlayer)
	}
	if media.decryptPath != "" || media.opened != "" {
		t.Fatal("streamed playback also wrote or opened a file")
	}
}

func TestPlayFallsBackToTempFileForOpener(t *testing.T) {
	key := strings.Repeat("ab", 16)
	svc := &playService{episodes: playFixtures(), playback: protectedPlayback(), playErr: sdk.ErrDRMProtected}
	media := &fakeMedia{decrypted: 1234, streamErrs: []error{mediapkg.ErrNeedsFile}}
	deps := Dependencies{Service: svc, Media: media, Stdout: io.Discard, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--key", key, "--player", "mpv"}, deps); code != ExitOK {
		t.Fatalf("decrypt play exit %d", code)
	}
	if media.decryptPath == "" || !strings.HasSuffix(media.decryptPath, ".mp4") {
		t.Fatalf("decrypt path = %q", media.decryptPath)
	}
	if media.opened != media.decryptPath || media.player != "mpv" {
		t.Fatalf("opened = %q player = %q (want %q)", media.opened, media.player, media.decryptPath)
	}
}

func TestPlayRejectsKeyOnClearStream(t *testing.T) {
	svc := &playService{episodes: playFixtures(), playback: clearPlayback()}
	deps := Dependencies{Service: svc, Media: &fakeMedia{}, Stdout: io.Discard, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--key", strings.Repeat("aa", 16)}, deps); code != ExitUsage {
		t.Fatalf("clear+key exit %d", code)
	}
}

func TestPlayReadsKeyFromEnvironment(t *testing.T) {
	key := strings.Repeat("cd", 16)
	t.Setenv("OHG_CENC_KEY", key)
	svc := &playService{episodes: playFixtures(), playback: protectedPlayback(), playErr: sdk.ErrDRMProtected}
	media := &fakeMedia{}
	deps := Dependencies{Service: svc, Media: media, Stdout: io.Discard, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--download", filepath.Join(t.TempDir(), "e.mp4")}, deps); code != ExitOK {
		t.Fatalf("env key exit %d", code)
	}
	if media.decryptKey != key {
		t.Fatalf("key = %q", media.decryptKey)
	}
}

func TestPlayDerivesKeyFromModelMaterial(t *testing.T) {
	pb := protectedPlayback()
	pb.KeyMaterial = "nLwSxG6JI85zhT//c7AK4HauO+lzrg7TdZwO0XWrP+J0pDypqQ=="
	svc := &playService{episodes: playFixtures(), playback: pb, playErr: sdk.ErrDRMProtected}
	media := &fakeMedia{}
	deps := Dependencies{Service: svc, Media: media, Stdout: io.Discard, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--download", filepath.Join(t.TempDir(), "e.mp4")}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if media.decryptKey != "1be760855c48a8c86000eeb8c5bb1010" {
		t.Fatalf("derived key = %q", media.decryptKey)
	}
}

func TestPlayNoDecryptReportsProtection(t *testing.T) {
	pb := protectedPlayback()
	pb.KeyMaterial = "nLwSxG6JI85zhT//c7AK4HauO+lzrg7TdZwO0XWrP+J0pDypqQ=="
	svc := &playService{episodes: playFixtures(), playback: pb, playErr: sdk.ErrDRMProtected}
	media := &fakeMedia{}
	var errOut bytes.Buffer
	deps := Dependencies{Service: svc, Media: media, Stdout: io.Discard, Stderr: &errOut}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--no-decrypt"}, deps); code != ExitUnavailable {
		t.Fatalf("exit %d (%s)", code, errOut.String())
	}
	if media.decryptPath != "" || media.opened != "" {
		t.Fatal("--no-decrypt still touched media")
	}
}

func TestPlayWithoutKeyMaterialStillReportsProtection(t *testing.T) {
	svc := &playService{episodes: playFixtures(), playback: protectedPlayback(), playErr: sdk.ErrDRMProtected}
	deps := Dependencies{Service: svc, Media: &fakeMedia{}, Stdout: io.Discard, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101"}, deps); code != ExitUnavailable {
		t.Fatalf("exit %d", code)
	}
}

func moviePlayback() *model.Playback {
	return &model.Playback{
		EpisodeID:   "7273805203457969211",
		VideoID:     "v-movie-fixture",
		Protected:   true,
		AudioURL:    "https://media.invalid/movie-audio.m4a",
		KeyMaterial: "nLwSxG6JI85zhT//c7AK4HauO+lzrg7TdZwO0XWrP+J0pDypqQ==",
		Streams: []model.Stream{
			{Definition: "720p", Codec: "bytevc1", URL: "https://media.invalid/movie-720.mp4", Protected: true, Encryption: "cenc-aes-ctr", Dash: true},
			{Definition: "1080p", Codec: "bytevc1", URL: "https://media.invalid/movie-1080.mp4", Protected: true, Encryption: "cenc-aes-ctr", Dash: true},
		},
	}
}

func TestPlaySegmentedTitleUsesAudioTrack(t *testing.T) {
	svc := &playService{episodes: playFixtures(), playback: moviePlayback(), playErr: sdk.ErrDRMProtected}
	media := &fakeMedia{}
	deps := Dependencies{Service: svc, Media: media, Stdout: io.Discard, Stderr: io.Discard}
	out := filepath.Join(t.TempDir(), "movie.mp4")
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--download", out}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if media.decryptURL != "https://media.invalid/movie-1080.mp4" {
		t.Fatalf("video = %q", media.decryptURL)
	}
	if media.decryptAudio != "https://media.invalid/movie-audio.m4a" {
		t.Fatalf("audio = %q", media.decryptAudio)
	}
	if media.decryptKey != "1be760855c48a8c86000eeb8c5bb1010" {
		t.Fatalf("key = %q", media.decryptKey)
	}
}

func TestPlayPreviewLimitsDecryption(t *testing.T) {
	svc := &playService{episodes: playFixtures(), playback: moviePlayback(), playErr: sdk.ErrDRMProtected}
	media := &fakeMedia{}
	deps := Dependencies{Service: svc, Media: media, Stdout: io.Discard, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--preview", "45"}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if media.streamLimit != 45 || media.streamAudio != "https://media.invalid/movie-audio.m4a" {
		t.Fatalf("limit = %d audio = %q", media.streamLimit, media.streamAudio)
	}
	out := filepath.Join(t.TempDir(), "x.mp4")
	deps.Stdout = io.Discard
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--preview", "45", "--download", out}, deps); code != ExitOK {
		t.Fatalf("preview+download exit %d", code)
	}
	if media.decryptPath != out || media.decryptLimit != 45 {
		t.Fatalf("path = %q limit = %d", media.decryptPath, media.decryptLimit)
	}
}

func TestPlayURLReportsSegmentedSourcesWithoutDecrypting(t *testing.T) {
	svc := &playService{episodes: playFixtures(), playback: moviePlayback(), playErr: sdk.ErrDRMProtected}
	media := &fakeMedia{}
	var out bytes.Buffer
	deps := Dependencies{Service: svc, Media: media, Stdout: &out, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--url"}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	lines := strings.Fields(out.String())
	if len(lines) != 2 || lines[0] != "https://media.invalid/movie-1080.mp4" || lines[1] != "https://media.invalid/movie-audio.m4a" {
		t.Fatalf("urls = %q", lines)
	}
	if media.decryptPath != "" || media.opened != "" || media.savedPath != "" {
		t.Fatal("--url touched the media launcher")
	}
}

// seqService resolves every episode to its own protected stream.
type seqService struct {
	playService
	mu       sync.Mutex
	resolved []string
	// fail makes ResolvePlayback of an episode ID return that error.
	fail map[string]error
	// results is what Search returns.
	results []model.SeriesSummary
	query   string
	// book makes the series a book or audiobook (no video platform).
	book bool
}

func (s *seqService) ResolvePlayback(_ context.Context, episodeID string, _ sdk.PlaybackOptions) (*model.Playback, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.resolved = append(s.resolved, episodeID)
	if err := s.fail[episodeID]; err != nil {
		return nil, err
	}
	pb := protectedPlayback()
	pb.KeyMaterial = "nLwSxG6JI85zhT//c7AK4HauO+lzrg7TdZwO0XWrP+J0pDypqQ=="
	pb.Streams[0].URL = "https://media.invalid/" + episodeID + ".mp4"
	pb.Streams[0].BackupURL = "https://backup.invalid/" + episodeID + ".mp4"
	return pb, sdk.ErrDRMProtected
}

func (s *seqService) Search(_ context.Context, query string, _ sdk.SearchOptions) (*sdk.Page[model.SeriesSummary], error) {
	s.query = query
	return &sdk.Page[model.SeriesSummary]{Items: s.results}, nil
}

func (s *seqService) GetSeries(context.Context, string) (*model.Series, error) {
	platform := model.PlatformOuter
	if s.book {
		platform = model.PlatformUnknown
	}
	return &model.Series{SeriesSummary: model.SeriesSummary{ID: "7100000000000101", Title: "剧/名"}, Platform: platform}, nil
}

func threeEpisodes() []model.Episode {
	return append(playFixtures(), model.Episode{ID: "7100000000000203", SeriesID: "7100000000000101", Number: 3})
}

func TestPlayAutoplayContinuesToLastEpisode(t *testing.T) {
	svc := &seqService{playService: playService{episodes: threeEpisodes()}}
	media := &fakeMedia{}
	var errOut bytes.Buffer
	deps := Dependencies{Service: svc, Media: media, Stdout: io.Discard, Stderr: &errOut}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--episode", "2", "--autoplay"}, deps); code != ExitOK {
		t.Fatalf("exit %d (%s)", code, errOut.String())
	}
	want := []string{"https://media.invalid/7100000000000202.mp4", "https://media.invalid/7100000000000203.mp4"}
	if strings.Join(media.streamed, " ") != strings.Join(want, " ") {
		t.Fatalf("streamed = %q", media.streamed)
	}
	if media.streamKey != "1be760855c48a8c86000eeb8c5bb1010" {
		t.Fatalf("key = %q", media.streamKey)
	}
	if !strings.Contains(errOut.String(), "episode 3 of 3") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestPlayAutoplayStopsWhenPlayerClosed(t *testing.T) {
	svc := &seqService{playService: playService{episodes: threeEpisodes()}}
	media := &fakeMedia{streamErrs: []error{nil, mediapkg.ErrStopped}}
	deps := Dependencies{Service: svc, Media: media, Stdout: io.Discard, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--autoplay"}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if len(media.streamed) != 2 {
		t.Fatalf("streamed = %q", media.streamed)
	}
}

func TestPlayAutoplayStopsOnError(t *testing.T) {
	svc := &seqService{playService: playService{episodes: threeEpisodes()}}
	boom := errors.New("stream failed: boom")
	media := &fakeMedia{streamErrs: []error{boom, boom}}
	var errOut bytes.Buffer
	deps := Dependencies{Service: svc, Media: media, Stdout: io.Discard, Stderr: &errOut}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--autoplay"}, deps); code != ExitFailure {
		t.Fatalf("exit %d", code)
	}
	// The backup CDN URL is tried once before giving up.
	want := "https://media.invalid/7100000000000201.mp4 https://backup.invalid/7100000000000201.mp4"
	if strings.Join(media.streamed, " ") != want {
		t.Fatalf("streamed = %q", media.streamed)
	}
	if !strings.Contains(errOut.String(), "episode 1: stream failed: boom") {
		t.Fatalf("stderr = %q", errOut.String())
	}
}

func TestPlayFallsBackToBackupURL(t *testing.T) {
	svc := &seqService{playService: playService{episodes: threeEpisodes()}}
	media := &fakeMedia{streamErrs: []error{errors.New("player ffplay could not play the stream")}}
	deps := Dependencies{Service: svc, Media: media, Stdout: io.Discard, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101"}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if len(media.streamed) != 2 || media.streamed[1] != "https://backup.invalid/7100000000000201.mp4" {
		t.Fatalf("streamed = %q", media.streamed)
	}
	// Closing the player is not a CDN failure.
	media.streamed, media.streamErrs = nil, []error{mediapkg.ErrStopped}
	if code := Run(context.Background(), []string{"play", "7100000000000101"}, deps); code != ExitOK || len(media.streamed) != 1 {
		t.Fatalf("exit %d streamed = %q", code, media.streamed)
	}
}

func TestPlayAutoplayRejectsConflictingFlags(t *testing.T) {
	deps := Dependencies{Service: &seqService{playService: playService{episodes: threeEpisodes()}}, Media: &fakeMedia{}, Stdout: io.Discard, Stderr: io.Discard}
	for _, extra := range [][]string{{"--url"}, {"--json"}, {"--download", "x.mp4"}, {"--key", strings.Repeat("aa", 16)}, {"--player", "open"}} {
		args := append([]string{"play", "7100000000000101", "--autoplay"}, extra...)
		if code := Run(context.Background(), args, deps); code != ExitUsage {
			t.Fatalf("%v exit %d", extra, code)
		}
	}
}

func historyDeps(t *testing.T, media *fakeMedia) (Dependencies, *history.Store, *bytes.Buffer) {
	t.Helper()
	store := history.NewStore(filepath.Join(t.TempDir(), "history.json"))
	var out bytes.Buffer
	svc := &seqService{playService: playService{episodes: threeEpisodes()}}
	return Dependencies{Service: svc, Media: media, History: store, Stdout: &out, Stderr: io.Discard}, store, &out
}

func TestPlayRemembersUnfinishedEpisodeAndResumes(t *testing.T) {
	media := &fakeMedia{streamErrs: []error{nil, mediapkg.ErrStopped}}
	deps, store, _ := historyDeps(t, media)
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--autoplay"}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	e, ok, err := store.Get(context.Background(), "7100000000000101")
	if err != nil || !ok || e.Episode != 2 || e.Finished || e.Episodes != 3 {
		t.Fatalf("entry = %+v ok=%v err=%v", e, ok, err)
	}
	media.streamed, media.streamErrs = nil, nil
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--resume"}, deps); code != ExitOK {
		t.Fatalf("resume exit %d", code)
	}
	if len(media.streamed) != 1 || media.streamed[0] != "https://media.invalid/7100000000000202.mp4" {
		t.Fatalf("resumed = %q", media.streamed)
	}
	if e, _, _ := store.Get(context.Background(), "7100000000000101"); e.Episode != 2 || !e.Finished {
		t.Fatalf("after resume = %+v", e)
	}
}

func TestPlayResumeAfterFinishedEpisode(t *testing.T) {
	media := &fakeMedia{}
	deps, store, _ := historyDeps(t, media)
	ctx := context.Background()
	if err := store.Record(ctx, history.Entry{SeriesID: "7100000000000101", Episode: 1, Finished: true}); err != nil {
		t.Fatal(err)
	}
	if code := Run(ctx, []string{"play", "7100000000000101", "--resume"}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if media.streamURL != "https://media.invalid/7100000000000202.mp4" {
		t.Fatalf("resumed = %q", media.streamURL)
	}
	if err := store.Record(ctx, history.Entry{SeriesID: "7100000000000101", Episode: 3, Finished: true}); err != nil {
		t.Fatal(err)
	}
	if code := Run(ctx, []string{"play", "7100000000000101", "--resume"}, deps); code != ExitUsage {
		t.Fatalf("all watched exit %d", code)
	}
}

func TestPlayResumeWithoutHistoryStartsAtFirstEpisode(t *testing.T) {
	media := &fakeMedia{}
	deps, _, _ := historyDeps(t, media)
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--resume"}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if media.streamURL != "https://media.invalid/7100000000000201.mp4" {
		t.Fatalf("played = %q", media.streamURL)
	}
	for _, extra := range [][]string{{"--episode", "2"}, {"--url"}, {"--download", "x.mp4"}} {
		args := append([]string{"play", "7100000000000101", "--resume"}, extra...)
		if code := Run(context.Background(), args, deps); code != ExitUsage {
			t.Fatalf("%v exit %d", extra, code)
		}
	}
}

func TestPlayDownloadAndPreviewAreNotRemembered(t *testing.T) {
	deps, store, _ := historyDeps(t, &fakeMedia{})
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--download", filepath.Join(t.TempDir(), "e.mp4")}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--preview", "5"}, deps); code != ExitOK {
		t.Fatalf("preview exit %d", code)
	}
	if list, _ := store.List(context.Background()); len(list) != 0 {
		t.Fatalf("history = %+v", list)
	}
}

func TestHistoryListsAndClears(t *testing.T) {
	deps, store, out := historyDeps(t, &fakeMedia{})
	ctx := context.Background()
	if err := store.Record(ctx, history.Entry{SeriesID: "7100000000000101", Title: "剧", Episode: 2, Episodes: 3}); err != nil {
		t.Fatal(err)
	}
	if code := Run(ctx, []string{"history"}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if !strings.HasPrefix(out.String(), "7100000000000101\t剧\t2/3\twatching\t") {
		t.Fatalf("history = %q", out.String())
	}
	if code := Run(ctx, []string{"history", "clear"}, deps); code != ExitOK {
		t.Fatalf("clear exit %d", code)
	}
	if list, _ := store.List(ctx); len(list) != 0 {
		t.Fatalf("after clear = %+v", list)
	}
}

// pagingService serves a fixed list of pages keyed by cursor.
type pagingService struct {
	captureService
	pages map[string]*sdk.Page[model.SeriesSummary]
	calls []string
}

func (s *pagingService) Discover(_ context.Context, opts sdk.DiscoverOptions) (*sdk.Page[model.SeriesSummary], error) {
	s.calls = append(s.calls, opts.Cursor)
	if page, ok := s.pages[opts.Cursor]; ok {
		return page, nil
	}
	return &sdk.Page[model.SeriesSummary]{}, nil
}

func TestDiscoverAllFollowsCursorsAndDropsRepeats(t *testing.T) {
	svc := &pagingService{pages: map[string]*sdk.Page[model.SeriesSummary]{
		"":   {Items: []model.SeriesSummary{{ID: "1"}, {ID: "2"}}, Cursor: "c1", HasMore: true},
		"c1": {Items: []model.SeriesSummary{{ID: "2"}, {ID: "3"}}, Cursor: "c2", HasMore: true},
		"c2": {Items: []model.SeriesSummary{{ID: "3"}, {ID: "4"}}, Cursor: "", HasMore: false},
	}}
	var out bytes.Buffer
	deps := Dependencies{Service: svc, Stdout: &out, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"discover", "--all", "--pages", "5"}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	var ids []string
	for _, line := range lines {
		ids = append(ids, strings.SplitN(line, "\t", 2)[0])
	}
	if strings.Join(ids, ",") != "1,2,3,4" {
		t.Fatalf("ids = %v (output %q)", ids, out.String())
	}
	if len(svc.calls) != 3 {
		t.Fatalf("calls = %v", svc.calls)
	}
}

// rejectingService fails the first call with a device rejection.
type rejectingService struct {
	captureService
	calls  int
	resets *int
	inner  Service
}

func (s *rejectingService) GetSeries(ctx context.Context, id string) (*model.Series, error) {
	s.calls++
	if s.calls == 1 {
		return nil, &sdk.CodeError{Code: 263175, Kind: protocol.KindDeviceRejected}
	}
	return &model.Series{SeriesSummary: model.SeriesSummary{ID: id, Title: "Recovered"}}, nil
}

type fakeResetter struct{ calls *int }

func (f fakeResetter) ResetDevice(context.Context) error { *f.calls++; return nil }

func TestDeviceRejectionTriggersResetAndReplay(t *testing.T) {
	resets := 0
	svc := &rejectingService{resets: &resets}
	var out bytes.Buffer
	deps := Dependencies{Service: svc, DeviceReset: fakeResetter{calls: &resets}, Stdout: &out, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"info", "7100000000000101"}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if resets != 1 || svc.calls != 2 {
		t.Fatalf("resets = %d calls = %d", resets, svc.calls)
	}
	if !strings.Contains(out.String(), "Recovered") {
		t.Fatalf("output = %q", out.String())
	}
}

func TestDeviceRejectionDuringPlaybackIsNotReplayed(t *testing.T) {
	resets := 0
	svc := &seqService{playService: playService{episodes: threeEpisodes()},
		fail: map[string]error{"7100000000000202": sdk.ErrDeviceRejected}}
	media := &fakeMedia{}
	var errOut bytes.Buffer
	deps := Dependencies{Service: svc, Media: media, DeviceReset: fakeResetter{calls: &resets}, Stdout: io.Discard, Stderr: &errOut}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--autoplay"}, deps); code == ExitOK {
		t.Fatal("rejected playback reported success")
	}
	if resets != 1 {
		t.Fatalf("resets = %d", resets)
	}
	// Episode 1 played once; replaying the command would play it again.
	if len(media.streamed) != 1 {
		t.Fatalf("streamed = %q", media.streamed)
	}
	for _, want := range []string{"registering a new one", "ohg continue"} {
		if !strings.Contains(errOut.String(), want) {
			t.Fatalf("stderr %q lacks %q", errOut.String(), want)
		}
	}
}

func TestShortRankFilterAndCuesCommands(t *testing.T) {
	var out bytes.Buffer
	svc := &captureService{}
	deps := Dependencies{Service: svc, Stdout: &out, Stderr: io.Discard}
	run := func(args ...string) (int, string) {
		out.Reset()
		code := Run(context.Background(), args, deps)
		return code, out.String()
	}
	code, got := run("rank", "--shorts", "--content", "human", "--board", "human_hot_play", "--cursor", "s1.prev")
	if code != ExitOK {
		t.Fatalf("rank --shorts exit %d:\n%s", code, got)
	}
	if svc.rankQuery != (sdk.ShortRankQuery{Content: "human", Board: "human_hot_play", Cursor: "s1.prev"}) {
		t.Fatalf("rank query = %+v", svc.rankQuery)
	}
	for _, want := range []string{
		"content\tall=全部 human=真人剧*\n",
		"board\thuman_hot_sc=推荐榜 human_hot_play=热播榜*\n",
		"1\t7100000000000101\tFirst\t60\t7921万热度\n",
		"2\t7100000000000102\tInserted\t30\t平台推荐\tpromoted\n",
		"cursor\ts1.next\n",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rank output lacks %q:\n%s", want, got)
		}
	}
	if code, got := run("rank", "--shorts", "--content", "ranklist_celebrity"); code != ExitOK ||
		got != "1\t7100000000001301\tActor A\t759万人气\t67部作品 · 双周追剧403万人\tup 1\n2\t7100000000001402\tActor B\t556万人气\t\tnew\n" {
		t.Fatalf("actor rank exit %d:\n%q", code, got)
	}
	if code, got := run("info", "7100000000000101", "7100000000000199"); code != ExitOK || fmt.Sprint(svc.batch) != "[7100000000000101 7100000000000199]" ||
		!strings.Contains(got, "7100000000000101\tOne\t2\n") || !strings.Contains(got, "novel\t7100000000002001\tNovel\tongoing\t悬疑,穿越\n") {
		t.Fatalf("info batch exit %d:\n%q", code, got)
	}
	if code, got := run("new", "--date", "20261024"); code != ExitOK || svc.newQuery.Date != "20261024" ||
		got != "date\t20261024\ndates\t20261023 20261024\n7100000000001501\tOut\t\tonline\t378万热度\t60\n7100000000001502\tSoon\t\tupcoming\t159.7万人预约\t0\ncursor\ts1.next\n" {
		t.Fatalf("new exit %d:\n%q", code, got)
	}
	if code, got := run("person", "7100000000001701"); code != ExitOK ||
		got != "7100000000001701\tActor\t女演员 · 水瓶座\nBio\nfollowers\t3\tlikes\t4\n# 作品\n7100000000001801\tWork\t70\t爱情 · 70集\ncursor\tw1.next\n" {
		t.Fatalf("person exit %d:\n%q", code, got)
	}
	if code, got := run("person", "7100000000001701", "--cursor", "w1.prev"); code != ExitOK || svc.worksCursor != "w1.prev" ||
		got != "7100000000001801\tWork\t70\t爱情 · 70集\ncursor\tw1.next\n" {
		t.Fatalf("person --cursor exit %d:\n%q", code, got)
	}
	if code, got := run("filter", "--panel", "--genre", "comic_series"); code != ExitOK || svc.panelGenre != "comic_series" ||
		got != "category_dim_theme\t全部主题\t悬疑=cate_165 玄幻=cate_7\n" {
		t.Fatalf("filter --panel exit %d: %q", code, got)
	}
	code, got = run("filter", "--genre", "short_play", "--filter", "category_dim_theme=cate_165", "--filter", "category_dim_theme=cate_7", "--cursor", "s1.prev")
	if code != ExitOK || got != "7100000000000101\tFixture\t12\ncursor\ts1.next\n" {
		t.Fatalf("filter exit %d: %q", code, got)
	}
	q := svc.filterQuery
	if fmt.Sprint(q.Selections) != "map[category_dim_theme:[cate_165 cate_7] genre:[short_play]]" || q.Cursor != "s1.prev" {
		t.Fatalf("filter query = %+v", q)
	}
	if code, got := run("cues"); code != ExitOK || got != "书名\n主角\n" {
		t.Fatalf("cues exit %d: %q", code, got)
	}
	if code, got := run("hot"); code != ExitOK || got != "# 猜你想搜\n1\t7100000000000101\tFixture\t12万热度\n# 短剧热搜榜\n1\t7100000000000102\tHot\t355万热搜值\n# 热点话题榜\n1\tTopic query\t4.6万热度\n" {
		t.Fatalf("hot exit %d: %q", code, got)
	}
	for _, args := range [][]string{
		{"rank", "--shorts", "--offset", "15"},
		{"rank", "--content", "human"},
		{"filter", "--panel", "--filter", "sort=hot_score"},
		{"filter", "--filter", "sort"},
		{"filter", "extra"},
		{"new", "extra"},
		{"person"},
		{"browse"},
	} {
		if code, _ := run(args...); code != ExitUsage {
			t.Fatalf("%v exit %d", args, code)
		}
	}
}

// limitedService answers the first empty playback requests with an empty
// model, the way the service treats an anonymous device it has served once.
type limitedService struct {
	seqService
	empty int
}

func (s *limitedService) ResolvePlayback(ctx context.Context, episodeID string, opts sdk.PlaybackOptions) (*model.Playback, error) {
	s.mu.Lock()
	if s.empty > 0 {
		s.empty--
		s.mu.Unlock()
		return nil, sdk.ErrAnonymousLimited
	}
	s.mu.Unlock()
	return s.seqService.ResolvePlayback(ctx, episodeID, opts)
}

func TestEmptyPlaybackRetriesOnFreshDevice(t *testing.T) {
	resets := 0
	svc := &limitedService{seqService: seqService{playService: playService{episodes: threeEpisodes()}}, empty: 2}
	media := &fakeMedia{}
	deps := Dependencies{Service: svc, Media: media, DeviceReset: fakeResetter{calls: &resets}, Stdout: io.Discard, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--episode", "1"}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	if resets != 2 || len(media.streamed) != 1 {
		t.Fatalf("resets = %d streamed = %q", resets, media.streamed)
	}
}

func TestEmptyPlaybackGivesUpAfterFreshDevices(t *testing.T) {
	resets := 0
	svc := &limitedService{seqService: seqService{playService: playService{episodes: threeEpisodes()}}, empty: 99}
	deps := Dependencies{Service: svc, Media: &fakeMedia{}, DeviceReset: fakeResetter{calls: &resets}, Stdout: io.Discard, Stderr: io.Discard}
	if code := Run(context.Background(), []string{"play", "7100000000000101", "--episode", "1"}, deps); code != ExitUnavailable {
		t.Fatalf("exit %d", code)
	}
	if resets != playbackFreshDevices {
		t.Fatalf("resets = %d", resets)
	}
}
