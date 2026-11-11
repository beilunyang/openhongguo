// Package sdk 是红果短剧（Hongguo）片库的非官方客户端。
//
// 所有请求都以匿名设备身份发出：客户端首次使用时注册一台设备，
// 把它存进 ProfileStore，之后每次调用都出示它。全程不涉及账号。
//
// # 快速上手
//
// NewClient 需要一个 Transport 和一个 ProfileStore。Transport 负责发送请求，
// ProfileStore 负责保存匿名设备。最简单的办法是用 adapters.NewClient，
// 它用现成的实现补齐这两项，把设备档案存在指定的文件里：
//
//	client, err := adapters.NewClient("anonymous.json", sdk.Options{})
//
// 也可以自己组合 adapters 的子包，或者换成自己的实现：
//
//	client, err := sdk.NewClient(sdk.Options{
//		Transport:    httptransport.New(),
//		ProfileStore: memstore.New(),
//	})
//
// # 客户端能读取什么
//
//   - 片库：[Client.Discover]、[Client.Search]、[Client.Suggest]、
//     [Client.HotSearch]、[Client.SearchCues] 和 [Client.NewReleases]。
//   - 短剧与分集：[Client.GetSeries]、[Client.GetSeriesBatch]、
//     [Client.ListEpisodes]、[Client.Seasons] 和 [Client.SeriesRecommend]。
//   - 弹幕：[Client.ListDanmaku]。
//   - 榜单与筛选：[Client.ShortRank]、[Client.ShortFilterPanel]、
//     [Client.ShortFilter]、[Client.Ranking]、[Client.RankRows] 和
//     [Client.RankGroups]。
//   - 人物：[Client.Person] 和 [Client.PersonWorks]。
//   - 书籍（小说和有声书，不是短剧）：[Client.Categories]
//     和 [Client.CategoryLanding]。
//   - 播放：[Client.ResolvePlayback]；加密分集另用 [ContentKey]。
//
// 按游标分页的列表会在不同页里重复出现同一部作品。[Pager] 会跟随游标翻页，
// 并去掉重复项。
//
// # 错误处理
//
// 服务端失败可以用 errors.Is 匹配 Err* 系列哨兵错误。这样调用方按原因分支，
// 而不必依赖服务端的错误码。[Retryable] 和 [RetryAfter] 告诉你能否重试、
// 何时重试；设置 [Options.RetryAttempts] 后，客户端会自动重试。
package sdk
