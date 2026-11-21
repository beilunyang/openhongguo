package cliapp

import (
	"github.com/beilunyang/openhongguo/sdk/model"
	"testing"
)

func TestCodecSelectionMixedRepresentations(t *testing.T) {
	pb := &model.Playback{Streams: []model.Stream{{Definition: "720p", Height: 720, Codec: "bytevc2", URL: "https://media.invalid/720.mp4"}, {Definition: "1080p", Height: 960, Codec: "bytevc1", URL: "https://media.invalid/1080.mp4"}}}
	s, e := selectCodec(pb, "", "bytevc1")
	if e != nil || s.Codec != "bytevc1" {
		t.Fatalf("%+v %v", s, e)
	}
	if _, e = selectCodec(pb, "720p", "bytevc1"); e == nil {
		t.Fatal("must not silently override explicit quality")
	}
	if _, e = selectCodec(pb, "", "h264"); e == nil {
		t.Fatal("missing codec accepted")
	}
	s, e = selectCodec(pb, "720p", "")
	if e != nil || s.Codec != "bytevc2" {
		t.Fatal("default behavior changed")
	}
	s, e = selectCodec(pb, "", "BYTEVC1")
	if e != nil || s.Codec != "bytevc1" {
		t.Fatal("case insensitive selection failed")
	}
	if len(pb.Streams) != 2 || pb.Streams[0].Codec != "bytevc2" {
		t.Fatal("input mutated")
	}
}
func TestCodecFlag(t *testing.T) {
	for _, args := range [][]string{{"--codec", "bytevc1"}, {"--codec=bytevc1"}} {
		f, e := parseFlags(args)
		if e != nil || f.codec != "bytevc1" {
			t.Fatalf("%v %v", f, e)
		}
	}
	for _, args := range [][]string{{"--codec"}, {"--codec="}} {
		if _, e := parseFlags(args); e == nil {
			t.Fatal("empty codec accepted")
		}
	}
}
