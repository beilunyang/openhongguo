package sdk_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/beilunyang/openhongguo/adapters/httptransport"
	"github.com/beilunyang/openhongguo/adapters/memstore"
	"github.com/beilunyang/openhongguo/internal/protocol"
	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
)

func TestPlaybackFallbackProvider(t *testing.T) {
	const episodeID = "7100000000000201"
	const otherEpisodeID = "7100000000000202"
	const fallbackURL = "https://media.invalid/fallback.mp4"
	validModel := &model.Playback{EpisodeID: episodeID, Streams: []model.Stream{{URL: fallbackURL}}}
	providerErr := errors.New("provider unavailable")
	clear := string(fixture(t, "clear-playback.json"))
	protected := string(fixture(t, "drm-playback.json"))
	tests := []struct {
		name          string
		status        int
		body          string
		providerModel *model.Playback
		providerErr   error
		wantCalls     int
		wantLegacy    int32
		wantErr       error
		wantError     bool
	}{
		{name: "V5 success", status: 200, body: clear},
		{name: "protected V5 model", status: 200, body: protected, wantErr: sdk.ErrDRMProtected},
		{name: "empty V5 and legacy responses", status: 200, providerModel: validModel, wantCalls: 1, wantLegacy: 1},
		{name: "null model", status: 200, body: `{"code":0,"data":{"video_model":null}}`, providerModel: validModel, wantCalls: 1},
		{name: "empty stream map", status: 200, body: `{"code":0,"data":{"video_model":{"video_list":{}}}}`, providerModel: validModel, wantCalls: 1},
		{name: "unavailable routes", status: 404, providerModel: validModel, wantCalls: 1, wantLegacy: 1},
		{name: "upstream failure", status: 503, providerModel: validModel, wantCalls: 1},
		{name: "authentication required", status: 403, wantErr: sdk.ErrAuthenticationRequired},
		{name: "rate limited", status: 429, wantErr: sdk.ErrRateLimited},
		{name: "invalid JSON", status: 200, body: `{`, wantError: true},
		{name: "provider fails", status: 200, providerErr: providerErr, wantCalls: 1, wantLegacy: 1, wantErr: sdk.ErrAnonymousLimited},
		{name: "provider returns nil", status: 503, wantCalls: 1, wantError: true},
		{name: "provider returns wrong episode", status: 503, providerModel: &model.Playback{EpisodeID: otherEpisodeID, Streams: validModel.Streams}, wantCalls: 1, wantError: true},
		{name: "provider returns no URL", status: 503, providerModel: &model.Playback{EpisodeID: episodeID}, wantCalls: 1, wantError: true},
	}
	reg := fixture(t, "register-response.json")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var primaryCalls, legacyCalls atomic.Int32
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case protocol.PathRegister:
					_, _ = w.Write(reg)
				case protocol.PathPlaybackV5, protocol.PathPlayback:
					primaryCalls.Add(1)
					if r.URL.Path == protocol.PathPlayback {
						legacyCalls.Add(1)
					}
					w.WriteHeader(tt.status)
					_, _ = w.Write([]byte(tt.body))
				default:
					http.NotFound(w, r)
				}
			}))
			defer srv.Close()
			calls := 0
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			client, err := sdk.NewClient(sdk.Options{
				Transport: httptransport.New(), ProfileStore: memstore.New(),
				APIOrigin: srv.URL, RegisterOrigin: srv.URL, DeviceResets: sdk.Int(0),
				FallbackPlaybackProvider: func(gotCtx context.Context, id string) (*model.Playback, error) {
					calls++
					if gotCtx != ctx || id != episodeID {
						t.Fatalf("provider did not receive the original context and episode ID")
					}
					if primaryCalls.Load() == 0 || legacyCalls.Load() != tt.wantLegacy {
						t.Fatalf("provider called before the primary playback chain finished")
					}
					return tt.providerModel, tt.providerErr
				},
			})
			if err != nil {
				t.Fatal(err)
			}
			pb, err := client.ResolvePlayback(ctx, episodeID, sdk.PlaybackOptions{})
			if calls != tt.wantCalls || legacyCalls.Load() != tt.wantLegacy {
				t.Fatalf("provider calls = %d, legacy calls = %d", calls, legacyCalls.Load())
			}
			if tt.wantError {
				if err == nil || pb != nil {
					t.Fatalf("expected an error with no model, got %v, %v", pb, err)
				}
			} else if !errors.Is(err, tt.wantErr) {
				t.Fatalf("error = %v, want %v", err, tt.wantErr)
			}
			if tt.providerErr != nil && !errors.Is(err, tt.providerErr) {
				t.Fatalf("provider error was lost: %v", err)
			}
			if tt.providerModel == validModel && pb != validModel {
				t.Fatalf("provider model was not returned: %v", pb)
			}
			if tt.wantErr == sdk.ErrDRMProtected && (pb == nil || !pb.Protected) {
				t.Fatalf("protected model was lost: %v", pb)
			}
			cancel()
			if _, err := client.ResolvePlayback(ctx, episodeID, sdk.PlaybackOptions{}); !errors.Is(err, context.Canceled) || calls != tt.wantCalls {
				t.Fatalf("canceled request returned %v with %d provider calls", err, calls)
			}
		})
	}
}
