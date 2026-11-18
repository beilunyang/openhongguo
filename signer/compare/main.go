// Command compare runs the same SDK calls signed three ways and prints a
// table: the Go signer, the unidbg emulator (openhongguo-signer), and a control
// whose x-medusa is corrupted.
//
// Each mode gets its own freshly registered anonymous device, so per-device
// playback limits do not leak between modes. The emulator mode swaps the
// six signature headers for the ones /feature-hash returns, signing the
// exact URL and body bytes that go on the wire.
//
// Expected result: go and emu agree on every step, and the control fails a
// step (search and the V5 playback route both validate medusa). Exit status 1 means go and emu
// disagree, or the control did not fail, so the check proved nothing.
//
// The output carries no device ids, cookies or signature values.
package main

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/beilunyang/openhongguo/adapters/httptransport"
	"github.com/beilunyang/openhongguo/adapters/memstore"
	"github.com/beilunyang/openhongguo/sdk"
)

var signatureHeaders = []string{"x-gorgon", "x-medusa", "x-helios", "x-khronos", "x-argus", "x-ladon"}

type mode string

const (
	modeGo      mode = "go"
	modeEmu     mode = "emu"
	modeControl mode = "control"
)

// signingTransport rewrites the signature headers of signed requests.
type signingTransport struct {
	next  sdk.Transport
	mode  mode
	relay string
	token string
	// swaps counts requests whose headers came from the emulator.
	swaps int
	// signErr is the last signing failure; the SDK hides it behind a
	// generic transport error.
	signErr error
	// lastPath is the last signed path, for error messages.
	lastPath string
}

func (t *signingTransport) Do(ctx context.Context, req *sdk.Request) (*sdk.Response, error) {
	if _, signed := req.Headers["x-medusa"]; signed {
		if u, err := url.Parse(req.URL); err == nil {
			t.lastPath = u.Path
		}
		switch t.mode {
		case modeEmu:
			if err := t.emulatorSign(ctx, req); err != nil {
				t.signErr = err
				return nil, err
			}
			t.swaps++
		case modeControl:
			// Same length, wrong bytes.
			m := []byte(req.Headers["x-medusa"])
			for i := len(m) / 4; i < len(m)/2; i++ {
				if m[i] == 'A' {
					m[i] = 'B'
				} else {
					m[i] = 'A'
				}
			}
			req.Headers["x-medusa"] = string(m)
		}
	}
	return t.next.Do(ctx, req)
}

func (t *signingTransport) emulatorSign(ctx context.Context, req *sdk.Request) error {
	var payload bytes.Buffer
	enc := json.NewEncoder(&payload)
	// SignerServer's JSON reader does not decode &.
	enc.SetEscapeHTML(false)
	if err := enc.Encode(map[string]string{"url": req.URL, "body_hex": hex.EncodeToString(req.Body)}); err != nil {
		return err
	}
	hreq, err := http.NewRequestWithContext(ctx, "POST", t.relay+"/feature-hash", &payload)
	if err != nil {
		return err
	}
	hreq.Header.Set("Content-Type", "application/json")
	if t.token != "" {
		hreq.Header.Set("Authorization", "Bearer "+t.token)
	}
	resp, err := http.DefaultClient.Do(hreq)
	if err != nil {
		return fmt.Errorf("openhongguo-signer: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var e struct {
			Error string `json:"error"`
		}
		json.NewDecoder(resp.Body).Decode(&e)
		if e.Error == "" {
			e.Error = "openhongguo-signer: HTTP " + strconv.Itoa(resp.StatusCode)
		}
		return errors.New(e.Error)
	}
	var out struct {
		Headers map[string]string `json:"headers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return fmt.Errorf("openhongguo-signer: %w", err)
	}
	got := map[string]string{}
	for k, v := range out.Headers {
		got[strings.ToLower(k)] = v
	}
	if neptune, ok := got["x-neptune"]; ok && got["x-medusa"] == "" {
		code, _, _ := strings.Cut(neptune, "|")
		return fmt.Errorf("openhongguo-signer: refused to sign (X-Neptune %s)", code)
	}
	if got["x-medusa"] == "" {
		return errors.New("openhongguo-signer: no signature returned")
	}
	for _, k := range signatureHeaders {
		delete(req.Headers, k)
		if v := got[k]; v != "" {
			req.Headers[k] = v
		}
	}
	return nil
}

// result is one step's outcome, compared across modes.
type result struct {
	ok     bool
	detail string
}

func (r result) String() string {
	if r.ok {
		return "ok " + r.detail
	}
	return "FAIL " + r.detail
}

func (t *signingTransport) classify(err error) string {
	if err != nil && t.signErr != nil {
		return t.signErr.Error()
	}
	return classify(err)
}

func classify(err error) string {
	switch {
	case err == nil:
		return ""
	case errors.Is(err, sdk.ErrAnonymousLimited):
		return "empty response"
	case errors.Is(err, sdk.ErrDeviceRejected):
		return "device rejected"
	default:
		msg := err.Error()
		if len(msg) > 60 {
			msg = msg[:60] + "…"
		}
		return msg
	}
}

type steps struct {
	search, episodes, playback result
}

func run(ctx context.Context, m mode, relay, token, query string) (steps, int) {
	tr := &signingTransport{next: httptransport.New(), mode: m, relay: relay, token: token}
	c, err := sdk.NewClient(sdk.Options{Transport: tr, ProfileStore: memstore.New(), DeviceResets: sdk.Int(0)})
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	var s steps
	skip := result{detail: "skipped"}
	s.episodes, s.playback = skip, skip

	page, err := c.Search(ctx, query, sdk.SearchOptions{})
	if err != nil || len(page.Items) == 0 {
		s.search = result{detail: or(tr.classify(err), "no results")}
		return s, tr.swaps
	}
	s.search = result{ok: true, detail: fmt.Sprintf("(%d series)", len(page.Items))}

	eps, err := c.ListEpisodes(ctx, page.Items[0].ID, sdk.EpisodeOptions{})
	if err != nil || len(eps.Items) == 0 {
		s.episodes = result{detail: or(tr.classify(err), "no episodes")}
		return s, tr.swaps
	}
	s.episodes = result{ok: true, detail: fmt.Sprintf("(%d episodes)", len(eps.Items))}

	pb, err := c.ResolvePlayback(ctx, eps.Items[0].ID, sdk.PlaybackOptions{})
	switch {
	case pb != nil && len(pb.Streams) > 0:
		// A protected model comes back with ErrDRMProtected; it is still a model.
		s.playback = result{ok: true, detail: fmt.Sprintf("(%d streams)", len(pb.Streams))}
	default:
		s.playback = result{detail: or(tr.classify(err), "no streams")}
	}
	return s, tr.swaps
}

func or(a, b string) string {
	if a != "" {
		return a
	}
	return b
}

func main() {
	relay := flag.String("relay", "http://127.0.0.1:19830", "openhongguo-signer address")
	query := flag.String("query", "重生", "search keyword")
	token := flag.String("token", os.Getenv("SIGNER_TOKEN"), "bearer token (default $SIGNER_TOKEN)")
	control := flag.Bool("control", true, "also run the corrupted-medusa control")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	if resp, err := http.Get(*relay + "/ready"); err != nil {
		fmt.Fprintf(os.Stderr, "openhongguo-signer not reachable at %s: %v\n", *relay, err)
		os.Exit(2)
	} else {
		resp.Body.Close()
	}

	modes := []mode{modeGo, modeEmu}
	if *control {
		modes = append(modes, modeControl)
	}
	all := map[mode]steps{}
	for _, m := range modes {
		s, swaps := run(ctx, m, *relay, *token, *query)
		all[m] = s
		if m == modeEmu {
			fmt.Printf("emulator signed %d requests\n", swaps)
		}
	}

	fmt.Printf("\n%-10s", "step")
	for _, m := range modes {
		fmt.Printf("  %-26s", m)
	}
	fmt.Println()
	rows := []struct {
		name string
		get  func(steps) result
	}{
		{"search", func(s steps) result { return s.search }},
		{"episodes", func(s steps) result { return s.episodes }},
		{"playback", func(s steps) result { return s.playback }},
	}
	failed := false
	for _, r := range rows {
		fmt.Printf("%-10s", r.name)
		for _, m := range modes {
			fmt.Printf("  %-26s", r.get(all[m]))
		}
		fmt.Println()
		if r.get(all[modeGo]).ok != r.get(all[modeEmu]).ok {
			failed = true
		}
	}
	fmt.Println()
	switch {
	case failed:
		fmt.Println("MISMATCH: the Go signer and the emulator disagree")
	case !all[modeGo].playback.ok:
		fmt.Println("INCONCLUSIVE: playback failed in both modes; retry later")
	default:
		fmt.Println("MATCH: the Go signer and the emulator agree")
	}
	if *control {
		c := all[modeControl]
		if c.search.ok && c.episodes.ok && c.playback.ok {
			fmt.Println("CONTROL DID NOT FAIL: every step accepted a corrupted x-medusa, so this run proves nothing about the signature")
			failed = true
		} else {
			fmt.Println("control: corrupted x-medusa was rejected, as expected")
		}
	}
	if failed {
		os.Exit(1)
	}
}
