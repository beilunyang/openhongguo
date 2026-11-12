package cliapp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/beilunyang/openhongguo/sdk/model"
)

func TestCLIDanmakuReadsEveryPage(t *testing.T) {
	svc := &playService{
		episodes: []model.Episode{{ID: "7100000000000201", Number: 1}, {ID: "7100000000000202", Number: 2}},
		danmaku: []*model.DanmakuPage{
			{Items: []model.Danmaku{{ID: "1", Text: "second", OffsetMS: 2500}, {ID: "2", Text: "boundary", OffsetMS: 30000}}, Total: 99, HasMore: true, Cursor: "m1.a"},
			{Items: []model.Danmaku{}, HasMore: true, Cursor: "m1.b"},
			{Items: []model.Danmaku{{ID: "2", Text: "boundary", OffsetMS: 30000}, {ID: "3", Text: "line\nbreak", OffsetMS: 61000}}},
		},
	}
	var stdout, stderr bytes.Buffer
	deps := Dependencies{Service: svc, Stdout: &stdout, Stderr: &stderr}
	if code := Run(context.Background(), []string{"danmaku", "7100000000000101", "--episode", "2"}, deps); code != ExitOK {
		t.Fatalf("exit %d: %s", code, stderr.String())
	}
	want := []string{"7100000000000202 ", "7100000000000202 m1.a", "7100000000000202 m1.b"}
	if strings.Join(svc.danmakuCalls, "|") != strings.Join(want, "|") {
		t.Fatalf("calls = %q", svc.danmakuCalls)
	}
	if got := stdout.String(); got != "0:02\tsecond\n0:30\tboundary\n1:01\tline break\n" {
		t.Fatalf("stdout = %q", got)
	}
	if !strings.Contains(stderr.String(), "3 danmaku (the service counts 99") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestCLIDanmakuJSON(t *testing.T) {
	svc := &playService{
		episodes: []model.Episode{{ID: "7100000000000201", Number: 1}},
		danmaku:  []*model.DanmakuPage{{Items: []model.Danmaku{{ID: "1", Text: "hi", OffsetMS: 1200, Likes: 3}}}},
	}
	var stdout bytes.Buffer
	deps := Dependencies{Service: svc, Stdout: &stdout}
	if code := Run(context.Background(), []string{"danmaku", "7100000000000101", "--json"}, deps); code != ExitOK {
		t.Fatalf("exit %d", code)
	}
	var out struct {
		SeriesID  string `json:"series_id"`
		EpisodeID string `json:"episode_id"`
		Episode   int    `json:"episode"`
		Items     []struct {
			Text     string `json:"text"`
			OffsetMS int64  `json:"offset_ms"`
			Likes    int64  `json:"likes"`
		} `json:"items"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out.SeriesID != "7100000000000101" || out.EpisodeID != "7100000000000201" || out.Episode != 1 ||
		len(out.Items) != 1 || out.Items[0].Text != "hi" || out.Items[0].OffsetMS != 1200 || out.Items[0].Likes != 3 {
		t.Fatalf("json = %+v", out)
	}
}

func TestCLIDanmakuRejectsMissingEpisode(t *testing.T) {
	svc := &playService{episodes: []model.Episode{{ID: "7100000000000201", Number: 1}}}
	var stderr bytes.Buffer
	deps := Dependencies{Service: svc, Stderr: &stderr}
	if code := Run(context.Background(), []string{"danmaku", "7100000000000101", "--episode", "5"}, deps); code != ExitUsage {
		t.Fatalf("exit %d", code)
	}
	if len(svc.danmakuCalls) != 0 {
		t.Fatalf("danmaku read for a missing episode: %q", svc.danmakuCalls)
	}
}

func TestOneLineDropsTerminalControls(t *testing.T) {
	in := "a\x1b]52;c;cHduZWQ=\x07b\u009b2Jc\u202ed\n\te"
	if got := oneLine(in); strings.ContainsAny(got, "\x1b\x07\u009b\u202e\n\t") || got != "a ]52;c;cHduZWQ= b 2Jc d e" {
		t.Fatalf("oneLine = %q", got)
	}
}
