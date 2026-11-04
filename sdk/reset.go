package sdk

import (
	"context"
	"errors"
	"sync"
)

// defaultDeviceResets 是 Options.DeviceResets 为 nil 时，一次调用最多换几台新设备。
const defaultDeviceResets = 2

// deviceResetter 让并发调用在同一台设备失效时只注册一台新设备。
type deviceResetter struct {
	mu sync.Mutex
	// gen 在每次换设备后加一。调用开始前记下它，失败后若它已经变了，
	// 说明别的调用刚换过设备，直接用新设备重试即可。
	gen uint64
}

func (r *deviceResetter) generation() uint64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.gen
}

// freshDeviceKey 标记已经在自动换设备的调用里，内层的公开方法不再重复换设备。
type freshDeviceKey struct{}

// withFreshDevice 执行 call；call 因设备失效失败时，换一台新的匿名设备重新执行，
// 最多 c.deviceResets 次。设备被拒（ErrDeviceRejected）在任何读取上都换设备；
// 空结果（ErrAnonymousLimited）只在 playback 为 true 时换设备，
// 因为只有播放接口确定会对用过的设备返回空结果（2026-10-29 起），
// 其他读取返回空结果可能只是没有内容。
func withFreshDevice[T any](ctx context.Context, c *Client, playback bool, call func(context.Context) (T, error)) (T, error) {
	if c.deviceResets <= 0 || ctx.Value(freshDeviceKey{}) != nil {
		return call(ctx)
	}
	ctx = context.WithValue(ctx, freshDeviceKey{}, true)
	for i := 0; ; i++ {
		gen := c.resetter.generation()
		v, err := call(ctx)
		if i >= c.deviceResets || !needsFreshDevice(err, playback) {
			return v, err
		}
		if resetErr := c.resetFrom(ctx, gen, err); resetErr != nil {
			return v, err
		}
	}
}

func needsFreshDevice(err error, playback bool) bool {
	if errors.Is(err, ErrDeviceRejected) {
		return true
	}
	return playback && errors.Is(err, ErrAnonymousLimited)
}

// resetFrom 在设备仍是第 gen 代时换一台新设备；已经换过就什么都不做。
func (c *Client) resetFrom(ctx context.Context, gen uint64, cause error) error {
	c.resetter.mu.Lock()
	defer c.resetter.mu.Unlock()
	if c.resetter.gen != gen {
		return nil
	}
	if c.logger != nil {
		c.logger.Printf("anonymous device unusable (%v); registering a new one", cause)
	}
	if err := c.resetDevice(ctx); err != nil {
		if c.logger != nil {
			c.logger.Printf("registering a new anonymous device failed: %v", err)
		}
		return err
	}
	c.resetter.gen++
	return nil
}

// ResetDevice 丢弃当前匿名设备，并注册一台新设备。
// Options.DeviceResets 开启时（默认），客户端遇到设备被拒或播放返回空结果会自己调用它，
// 调用方通常不需要再手动调用。
func (c *Client) ResetDevice(ctx context.Context) error {
	c.resetter.mu.Lock()
	defer c.resetter.mu.Unlock()
	if err := c.resetDevice(ctx); err != nil {
		return err
	}
	c.resetter.gen++
	return nil
}

func (c *Client) resetDevice(ctx context.Context) error {
	if c.profileStore == nil {
		return ErrInvalidOptions
	}
	if err := c.profileStore.Reset(ctx); err != nil {
		return err
	}
	_, err := c.ensureGuest(ctx)
	return err
}
