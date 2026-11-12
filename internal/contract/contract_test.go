package contract

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beilunyang/openhongguo/adapters/memstore"
	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/sdk"
)

var fixtureDir = filepath.Join("..", "..", "testdata", "protocol")

func TestShapeWalksArraysAndEmbeddedJSON(t *testing.T) {
	raw := []byte(`{"code":0,"data":{"list":[{"id":"1"},{"id":2,"x":null}],"model":"{\"v\":[true]}","empty":[]}}`)
	got, err := Shape(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := []Field{
		{"code", "number"},
		{"data.empty", "array"},
		{"data.list[].id", "number"},
		{"data.list[].id", "string"},
		{"data.list[].x", "null"},
		{"data.model{}.v[]", "bool"},
	}
	if len(got) != len(want) {
		t.Fatalf("shape = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("shape[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestCompareReportsMissingAndRetypedFields(t *testing.T) {
	fixture, _ := Shape([]byte(`{"a":{"id":"1","n":2,"tags":[],"label":"x","list":[1]},"gone":true}`))
	live, _ := Shape([]byte(`{"a":{"id":1,"n":3,"tags":["x"],"label":null,"list":null,"extra":"ignored"}}`))
	drifts := Compare(fixture, live)
	got := make([]string, len(drifts))
	for i, d := range drifts {
		got[i] = d.String()
	}
	// A null label is an unset optional field; a null list loses the list.
	want := "a.id: type number, fixture has string; a.list[]: missing (fixture has number); gone: missing (fixture has bool)"
	if strings.Join(got, "; ") != want {
		t.Fatalf("drifts = %q", got)
	}
}

// fixtureTransport answers each endpoint with a fixture file, as the live
// service would if it still matched the contract.
type fixtureTransport struct {
	t        *testing.T
	override map[string][]byte
}

func (f *fixtureTransport) Do(_ context.Context, req *sdk.Request) (*sdk.Response, error) {
	u, err := url.Parse(req.URL)
	if err != nil {
		return nil, err
	}
	if body, ok := f.override[u.Path]; ok {
		return &sdk.Response{StatusCode: 200, Body: body}, nil
	}
	// One path serves several pages; the query or body tells them apart.
	switch {
	case u.Path == protocol.PathPlan && u.Query().Get("from") == "search_input_page":
		return f.file("hot-search.json"), nil
	case u.Path == protocol.PathPlan && u.Query().Get("from") == protocol.PlanFromSeriesSeasons:
		return f.file("series-seasons.json"), nil
	case u.Path == protocol.PathPlan:
		return f.file("plan-response.json"), nil
	case u.Path == protocol.PathShortFilter && strings.Contains(string(req.Body), `"only_panel"`):
		return f.file("short-filter-panel.json"), nil
	case u.Path == protocol.PathShortFilter:
		return f.file("short-filter-content.json"), nil
	case u.Path == protocol.PathRankCells && u.Query().Get("selected_items") == protocol.ShortRankActors:
		return f.file("short-rank-actors.json"), nil
	case u.Path == protocol.PathRankCells && u.Query().Get("cell_id") == protocol.ShortRankCell:
		return f.file("short-rank.json"), nil
	}
	if strings.HasPrefix(u.Path, "/novel/commentapi/user/profile/") {
		return f.file("person-works.json"), nil
	}
	if strings.HasPrefix(u.Path, "/novel/commentapi/comment/list/") {
		return f.file("danmaku-response.json"), nil
	}
	if u.Path == protocol.PathSeriesBatch {
		return f.file("series-batch.json"), nil
	}
	name := map[string]string{
		protocol.PathRegister:        "register-response.json",
		protocol.PathDiscover:        "discover-response.json",
		protocol.PathSearch:          "search-response.json",
		protocol.PathSeries:          "series-response.json",
		protocol.PathPlayback:        "clear-playback.json",
		protocol.PathSuggest:         "suggest-response.json",
		protocol.PathCue:             "cue-response.json",
		protocol.PathCategory:        "category-front.json",
		protocol.PathCategoryLanding: "category-landing.json",
		protocol.PathRankCells:       "rank-rows.json",
		protocol.PathProfile:         "person.json",
		protocol.PathSubscribeList:   "new-releases.json",
	}[u.Path]
	if name == "" {
		f.t.Fatalf("unexpected request to %s", u.Path)
	}
	body, err := os.ReadFile(filepath.Join(fixtureDir, name))
	if err != nil {
		f.t.Fatal(err)
	}
	return &sdk.Response{StatusCode: 200, Body: body}, nil
}

func (f *fixtureTransport) file(name string) *sdk.Response {
	body, err := os.ReadFile(filepath.Join(fixtureDir, name))
	if err != nil {
		f.t.Fatal(err)
	}
	return &sdk.Response{StatusCode: 200, Body: body}
}

func runFixtures(t *testing.T, override map[string][]byte) []Check {
	t.Helper()
	return Run(context.Background(), Config{
		Transport:  &fixtureTransport{t: t, override: override},
		Profiles:   memstore.New(),
		FixtureDir: fixtureDir,
		Query:      "Fixture",
	})
}

func TestRunPassesWhenServiceMatchesFixtures(t *testing.T) {
	checks := runFixtures(t, nil)
	names := []string{}
	for _, c := range checks {
		names = append(names, c.Name)
		if !c.OK() {
			t.Errorf("%s: err=%v drifts=%v", c.Name, c.Err, c.Drifts)
		}
	}
	if strings.Join(names, ",") != "discover,register,search,series detail,series playback,danmaku,suggest,related,seasons,series batch,hot search,category,landing,rank rows,filter panel,short filter,short rank,actor rank,person,person works,new releases,cues" {
		t.Fatalf("checks = %v", names)
	}
}

func TestRunReportsDriftInSeriesDetail(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(fixtureDir, "series-response.json"))
	if err != nil {
		t.Fatal(err)
	}
	// The service renames the episode title field.
	changed := []byte(strings.ReplaceAll(string(raw), `"title"`, `"name"`))
	checks := runFixtures(t, map[string][]byte{protocol.PathSeries: changed})
	for _, c := range checks {
		if c.Name != "series detail" {
			continue
		}
		if len(c.Drifts) != 1 || c.Drifts[0].Path != "data.video_data.video_list[].title" {
			t.Fatalf("drifts = %v err = %v", c.Drifts, c.Err)
		}
		return
	}
	t.Fatal("no series detail check")
}
