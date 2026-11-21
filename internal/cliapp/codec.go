package cliapp

import (
	"fmt"
	"github.com/beilunyang/openhongguo/sdk/model"
	"strings"
)

// selectCodec is an explicit codec filter, not decoder capability detection.
// It preserves the caller's quality constraint and never mutates the model.
func selectCodec(pb *model.Playback, quality, codec string) (model.Stream, error) {
	if strings.TrimSpace(codec) == "" {
		return pb.Select(quality)
	}
	filtered := *pb
	filtered.Streams = nil
	available := []string{}
	for _, s := range pb.Streams {
		available = append(available, fmt.Sprintf("%s/%s", streamName(s), s.Codec))
		if strings.EqualFold(strings.TrimSpace(codec), s.Codec) {
			filtered.Streams = append(filtered.Streams, s)
		}
	}
	if len(filtered.Streams) == 0 {
		return model.Stream{}, fmt.Errorf("codec %q unavailable; streams: %s", codec, strings.Join(available, ", "))
	}
	stream, err := filtered.Select(quality)
	if err != nil {
		return model.Stream{}, fmt.Errorf("codec %q: %w", codec, err)
	}
	return stream, nil
}
