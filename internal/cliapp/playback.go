package cliapp

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/beilunyang/openhongguo/sdk"
	"github.com/beilunyang/openhongguo/sdk/model"
)

// playbackFreshDevices is how many new anonymous devices one playback request
// may move to when the service answers with an empty model.
const playbackFreshDevices = 2

// resetMu keeps concurrent fetches (prefetch, download --jobs) from
// re-registering over each other between a reset and its retry.
var resetMu sync.Mutex

// resolvePlayback loads an episode's model. Since 2026-10-29 the service
// answers each anonymous device with a model about once and with empty
// bodies after that, so an empty answer is retried on a new device. The SDK
// leaves this to its callers; this is the CLI doing it as one.
func resolvePlayback(ctx context.Context, d Dependencies, episodeID string) (*model.Playback, error) {
	pb, err := d.Service.ResolvePlayback(ctx, episodeID, sdk.PlaybackOptions{})
	for i := 0; i < playbackFreshDevices && errors.Is(err, sdk.ErrAnonymousLimited) && d.DeviceReset != nil; i++ {
		resetMu.Lock()
		resetErr := d.DeviceReset.ResetDevice(ctx)
		if resetErr == nil {
			pb, err = d.Service.ResolvePlayback(ctx, episodeID, sdk.PlaybackOptions{})
		}
		resetMu.Unlock()
		if resetErr != nil {
			return nil, fmt.Errorf("%w (registering a new device failed: %v)", err, resetErr)
		}
	}
	return pb, err
}
