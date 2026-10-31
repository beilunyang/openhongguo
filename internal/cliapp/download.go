package cliapp

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// cmdDownload saves several episodes of a series, f.jobs at a time.
func cmdDownload(ctx context.Context, args []string, d Dependencies) error {
	if err := requireService(d); err != nil {
		return err
	}
	if d.Media == nil {
		return errors.New("media launcher is not configured")
	}
	f, err := parseFlags(args)
	if err != nil {
		return err
	}
	if len(f.rest) != 1 {
		return usage("download needs one series ID or title")
	}
	id, title, err := resolveSeries(ctx, d, f.rest[0], f.pick)
	if err != nil {
		return err
	}
	if series, err := d.Service.GetSeries(ctx, id); err == nil {
		if err := requireVideo(series); err != nil {
			return err
		}
		if title == "" {
			title = series.Title
		}
	}
	page, err := d.Service.ListEpisodes(ctx, id, sdk.EpisodeOptions{})
	if err != nil {
		return err
	}
	numbers, err := parseEpisodeRange(f.episodes, len(page.Items))
	if err != nil {
		return err
	}
	dir := f.dir
	if dir == "" {
		dir = safeFileName(title)
		if dir == "" {
			dir = id
		}
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	width := len(strconv.Itoa(len(page.Items)))
	if width < 3 {
		width = 3
	}

	jobs := make(chan model.Episode)
	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		failed int
	)
	for i := 0; i < f.jobs; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for episode := range jobs {
				path := filepath.Join(dir, fmt.Sprintf("第%0*d集.mp4", width, episode.Number))
				written, skipped, err := downloadEpisode(ctx, d, episode, path, f.quality)
				mu.Lock()
				switch {
				case err != nil:
					if ctx.Err() == nil {
						failed++
						fmt.Fprintf(d.Stderr, "ohg: episode %d: %v\n", episode.Number, err)
					}
				case skipped:
					fmt.Fprintf(d.Stdout, "%d\t%s\texists\n", episode.Number, path)
				default:
					fmt.Fprintf(d.Stdout, "%d\t%s\t%d\n", episode.Number, path, written)
				}
				mu.Unlock()
			}
		}()
	}
feed:
	for _, n := range numbers {
		episode, ok := findEpisode(page.Items, n)
		if !ok {
			continue
		}
		select {
		case jobs <- episode:
		case <-ctx.Done():
			break feed
		}
	}
	close(jobs)
	wg.Wait()
	if err := ctx.Err(); err != nil {
		return err
	}
	if failed > 0 {
		return fmt.Errorf("%d of %d episodes failed; run the same command again to retry them", failed, len(numbers))
	}
	return nil
}

// downloadEpisode saves one episode to path through a .part file, so path
// only ever holds a complete episode. An existing path is skipped.
func downloadEpisode(ctx context.Context, d Dependencies, episode model.Episode, path, quality string) (written int64, skipped bool, err error) {
	if _, err := os.Stat(path); err == nil {
		return 0, true, nil
	}
	pb, err := resolvePlayback(ctx, d, episode.ID)
	if err != nil && !(errors.Is(err, sdk.ErrDRMProtected) && pb != nil) {
		return 0, false, err
	}
	stream, err := pb.Select(quality)
	if err != nil {
		return 0, false, usage("%s", err.Error())
	}
	key := ""
	if pb.Protected {
		k, err := sdk.ContentKey(pb)
		if err != nil {
			return 0, false, drmError(pb)
		}
		key = k.Hex
	}
	part := path + ".part"
	// A .part file is what an interrupted run left behind.
	_ = os.Remove(part)
	written, err = withBackup(ctx, d, stream, func(url string) (int64, error) {
		return saveStream(ctx, d, url, pb.AudioURL, key, pb.Protected, part, 0)
	})
	if err != nil {
		_ = os.Remove(part)
		return 0, false, err
	}
	if err := os.Rename(part, path); err != nil {
		return 0, false, err
	}
	return written, false, nil
}

// parseEpisodeRange reads "1-10,12" into sorted, distinct episode numbers.
// An empty spec means every episode.
func parseEpisodeRange(spec string, total int) ([]int, error) {
	if strings.TrimSpace(spec) == "" {
		all := make([]int, total)
		for i := range all {
			all[i] = i + 1
		}
		return all, nil
	}
	seen := map[int]bool{}
	for _, part := range strings.Split(spec, ",") {
		part = strings.TrimSpace(part)
		lo, hi, isRange := strings.Cut(part, "-")
		from, err1 := strconv.Atoi(strings.TrimSpace(lo))
		to := from
		var err2 error
		if isRange {
			to, err2 = strconv.Atoi(strings.TrimSpace(hi))
		}
		if err1 != nil || err2 != nil || from < 1 || to < from {
			return nil, usage("episodes must look like 1-10,12, got %q", part)
		}
		if to > total {
			return nil, usage("the series has %d episodes, got %q", total, part)
		}
		for n := from; n <= to; n++ {
			seen[n] = true
		}
	}
	out := make([]int, 0, len(seen))
	for n := range seen {
		out = append(out, n)
	}
	sort.Ints(out)
	return out, nil
}

// safeFileName drops characters that are not allowed in a file name on
// common file systems.
func safeFileName(s string) string {
	s = strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		}
		if r < 0x20 {
			return -1
		}
		return r
	}, s)
	return strings.Trim(strings.TrimSpace(s), ".")
}
