package sdk

import (
	"time"

	"github.com/beilunyang/openhongguo/sdk/model"
)

// Logger 接收诊断日志行。实现只会收到脱敏后的 URL，必须能处理这种情况。
type Logger interface {
	Printf(format string, args ...any)
}

// Options 提供所有平台依赖。origin 为空时，使用匿名读取时观察到的生产主机。
// 测试用本地服务器覆盖它们。
type Options struct {
	// Transport 负责发送请求（必填）。
	Transport Transport
	// ProfileStore 跨运行保存匿名设备（必填）。
	ProfileStore ProfileStore
	// Signer 为需要签名的请求签名；为 nil 时使用 UnavailableSigner。
	Signer Signer
	// AuthProvider 附加账号状态；为 nil 时使用 AnonymousProvider。
	AuthProvider AuthProvider
	// Logger 接收诊断信息；为 nil 时不记录日志。
	Logger Logger
	// Clock 提供当前时间；为 nil 时使用 time.Now。
	Clock func() time.Time
	// APIOrigin 承载片库读取，格式为 "https://host"；
	// 为空时使用 App 的 API 主机。RegisterOrigin 承载设备注册；
	// 为空时使用 App 的注册主机。
	APIOrigin      string
	RegisterOrigin string
	// DistributionOrigin 承载 /reading/distribution/ 和 activity_api
	// 读取（短剧筛选和活动榜单），APIOrigin 的默认主机不提供这些接口。
	// 为空时使用 App 自己的 API 主机。
	DistributionOrigin string
	// Device 是新匿名档案注册时使用的手机，之后每次请求都出示它。
	// 零值字段取 model.DefaultDevice（Android 14 上的 Xiaomi 14，
	// 小米应用商店渠道）。档案会保留注册时的设备，
	// 所以修改此项只影响之后注册的档案，包括设备重置之后。
	Device model.Device
	// RetryAttempts 限定可重试失败（限流、传输、5xx）时单个请求最多执行几次。
	// 0 或 1 表示不重试。
	RetryAttempts int
	// DeviceResets 限定一次调用最多换几台新的匿名设备。服务端拒绝当前设备
	// （ErrDeviceRejected），或播放接口返回空结果（ErrAnonymousLimited；
	// 2026-10-29 起，服务端对每台匿名设备大约只给一次播放模型）时，
	// 客户端丢弃当前设备、注册一台新设备，再重新执行这次调用。
	// nil 表示默认的 2 次；sdk.Int(0) 表示关闭，交给调用方自己调用 ResetDevice。
	// 换设备会覆盖 ProfileStore 里保存的档案，并通过 Logger 记一行日志。
	DeviceResets *int
}

// Int 返回指向 v 的指针，用来设置 Options.DeviceResets 这类可选的整数字段，
// 例如 sdk.Int(0)。Go 1.26 起也可以直接写 new(0)。
func Int(v int) *int { return &v }
