# openhongguo

[![Go Reference](https://pkg.go.dev/badge/github.com/beilunyang/openhongguo/sdk.svg)](https://pkg.go.dev/github.com/beilunyang/openhongguo/sdk)
[![npm](https://img.shields.io/npm/v/openhongguo)](https://www.npmjs.com/package/openhongguo)
[![npm react-native](https://img.shields.io/npm/v/react-native-openhongguo?label=npm%20react-native)](https://www.npmjs.com/package/react-native-openhongguo)
[![Maven Central](https://img.shields.io/maven-central/v/io.github.beilunyang/openhongguo)](https://central.sonatype.com/artifact/io.github.beilunyang/openhongguo)
[![pub.dev](https://img.shields.io/pub/v/openhongguo)](https://pub.dev/packages/openhongguo)
[![CI](https://github.com/beilunyang/openhongguo/actions/workflows/ci.yml/badge.svg)](https://github.com/beilunyang/openhongguo/actions/workflows/ci.yml)

openhongguo 是红果短剧（Hongguo）的非官方 SDK，提供 Go、Node.js、Android、iOS、React Native 和 Flutter 版本。SDK 可以浏览短剧目录、搜索短剧、读取短剧和分集的详情，并取得分集的播放地址和内容密钥。

SDK 用匿名设备发送所有请求，不需要登录。Go 以外的 SDK 都由 Go SDK 编译而成，所以各语言的行为相同。

> **免责声明**
>
> 请以学习为目的使用本仓库。
>
> 本仓库的所有内容仅供学习和参考之用，禁止用于商业用途。任何人或组织不得将本仓库的内容用于非法用途或侵犯他人合法权益。
>
> 本仓库所涉及的接口分析与调用技术仅用于学习和研究，不得用于对其他平台进行大规模数据采集或其他非法行为。
>
> 对于因使用本仓库内容而引起的任何法律责任，本仓库不承担任何责任。使用本仓库的内容即表示您同意本免责声明的所有条款和条件。
>
> 详细的免责声明见[文末](#免责声明)。

## 文档

| 文档 | 内容 |
|---|---|
| [类型参考](docs/types.md) | 所有返回值和查询参数的字段 |
| [Go 包文档](https://pkg.go.dev/github.com/beilunyang/openhongguo/sdk) | Go SDK 的完整 API |
| [命令行文档](docs/cli.md) | 演示程序 `ohg` 的用法 |
| [开发说明](docs/development.md) | 接口契约检查和持续集成 |
| [发布手册](docs/release.md) | 发布新版本的步骤 |

## 安装

| 语言 | 包 | 安装 | 文档 |
|---|---|---|---|
| Go | [`github.com/beilunyang/openhongguo`](https://pkg.go.dev/github.com/beilunyang/openhongguo/sdk) | `go get github.com/beilunyang/openhongguo@latest` | 本文 |
| Node.js | [`openhongguo`](https://www.npmjs.com/package/openhongguo) | `npm install openhongguo` | [bindings/node](bindings/node/README.md) |
| Android（Kotlin） | [`io.github.beilunyang:openhongguo`](https://central.sonatype.com/artifact/io.github.beilunyang/openhongguo) | `implementation("io.github.beilunyang:openhongguo:0.1.0")` | [bindings/mobile](bindings/mobile/README.md) |
| iOS / macOS（Swift） | Swift Package | `.package(url: "https://github.com/beilunyang/openhongguo.git", from: "0.1.1")` | [bindings/mobile](bindings/mobile/README.md) |
| React Native | [`react-native-openhongguo`](https://www.npmjs.com/package/react-native-openhongguo) | `npx expo install react-native-openhongguo` | [bindings/react-native](bindings/react-native/README.md) |
| Flutter | [`openhongguo`](https://pub.dev/packages/openhongguo) | `flutter pub add openhongguo` | [bindings/flutter](bindings/flutter/README.md) |

## 要求

| 语言 | 最低版本 |
|---|---|
| Go | 1.22 |
| Node.js | 20 |
| Android | 5.0（API 21）；Flutter 是 7.0（API 24） |
| iOS | 13；React Native 是 15.1，Flutter 是 15 |
| macOS | 11 |

## 快速上手

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/beilunyang/openhongguo/adapters"
	"github.com/beilunyang/openhongguo/sdk"
)

func main() {
	ctx := context.Background()
	client, err := adapters.NewClient("anonymous.json", sdk.Options{})
	if err != nil {
		log.Fatal(err)
	}

	page, err := client.Search(ctx, "全球死考", sdk.SearchOptions{})
	if err != nil {
		log.Fatal(err)
	}
	episodes, err := client.ListEpisodes(ctx, page.Items[0].ID, sdk.EpisodeOptions{})
	if err != nil {
		log.Fatal(err)
	}
	pb, err := client.ResolvePlayback(ctx, episodes.Items[0].ID, sdk.PlaybackOptions{})
	if pb == nil {
		log.Fatal(err)
	}
	for _, s := range pb.Streams {
		fmt.Println(s.Definition, s.URL)
	}
}
```

第一次请求时，SDK 注册一台匿名设备，并把设备档案保存到 `anonymous.json`。以后的请求都使用这台匿名设备。

<details>
<summary>其他语言的写法</summary>

| 语言 | 创建客户端并搜索 |
|---|---|
| Node.js | `const client = await createClient();`<br>`const page = await client.search('全球死考');` |
| Kotlin | `val client = OpenhongguoClient.create(context)`<br>`val page = client.search("全球死考")` |
| Swift | `let client = try OpenhongguoClient.create()`<br>`let page = try await client.search("全球死考")` |
| React Native | `const client = await createClient();`<br>`const page = await client.search('全球死考');` |
| Dart | `final client = await OpenhongguoClient.create();`<br>`final page = await client.search('全球死考');` |

</details>

## 客户端选项

`adapters.NewClient` 用默认的实现创建客户端。`sdk.Options` 的字段都是可选的。

```go
client, err := adapters.NewClient("anonymous.json", sdk.Options{
	RetryAttempts: 5,
	Logger:        log.Default(),
})
```

| 字段 | 作用 | 默认值 |
|---|---|---|
| `Transport` | 发送 HTTP 请求 | `httptransport.New()` |
| `ProfileStore` | 保存设备档案 | `filestore.New(profilePath)` |
| `Device` | 新匿名设备的机型。空字段使用默认值 | 小米 14 |
| `RetryAttempts` | 可重试的失败的最多尝试次数 | 1，即不重试（其他语言的 SDK 默认是 3） |
| `Logger` | 接收诊断日志 | 不记录 |
| `APIOrigin`、`RegisterOrigin`、`DistributionOrigin` | 接口地址 | 与 App 相同 |

如果不用 `adapters`，直接调用 `sdk.NewClient`。这时 `Transport` 和 `ProfileStore` 是必填的。

## 分页

带游标的列表返回 `sdk.Page`：`Items` 是本页的条目，`Cursor` 是下一页的游标，`HasMore` 表示是否还有下一页。

如果要遍历所有页，使用 `DiscoverPager` 或 `SearchPager`。服务端会在不同的页中返回同一部短剧，Pager 删除重复的短剧。

```go
pager := client.SearchPager("末世", sdk.SearchOptions{})
for {
	items, more, err := pager.Next(ctx)
	if err != nil {
		log.Fatal(err)
	}
	for _, s := range items {
		fmt.Println(s.ID, s.Title)
	}
	if !more {
		break
	}
}
```

## 错误处理

所有错误都可以用 `errors.Is` 与 `sdk` 包的哨兵错误比较。

```go
_, err := client.GetSeries(ctx, seriesID)
switch {
case errors.Is(err, sdk.ErrContentUnavailable):
	fmt.Println("这部短剧已下架")
case errors.Is(err, sdk.ErrRateLimited):
	time.Sleep(sdk.RetryAfter(err, 10*time.Second))
case err != nil:
	log.Fatal(err)
}
```

其他语言的异常有同样的错误码（`code`）、是否可重试（`retryable`）和建议的等待时间（`retryAfterMs`）。

<!-- gen:errors-go -->
<!-- 由 go run ./bindings/docgen 生成，请修改 sdk 的 Go 注释，不要手工修改。 -->

| Go | 错误码 | 含义 | 可重试 |
|---|---|---|---|
| `ErrInvalidOptions` | `INVALID_OPTIONS` | 客户端的设置或调用参数无效 | 否 |
| `ErrProfileNotFound` | `PROFILE_NOT_FOUND` | 还没有保存过设备档案 | 否 |
| `ErrSignerUnavailable` | `SIGNER_UNAVAILABLE` | 请求需要签名，而客户端没有可用的签名器 | 否 |
| `ErrAuthenticationRequired` | `AUTHENTICATION_REQUIRED` | 内容需要登录才能访问，匿名客户端无法继续 | 否 |
| `ErrNotProtected` | `NOT_PROTECTED` | 为未加密的播放模型请求了内容密钥 | 否 |
| `ErrDRMProtected` | `DRM_PROTECTED` | 分集是加密的，需要内容密钥才能解密 | 否 |
| `ErrAnonymousLimited` | `ANONYMOUS_LIMITED` | 服务端给匿名设备返回了空结果 | 否 |
| `ErrRateLimited` | `RATE_LIMITED` | 请求太频繁，被服务端限流 | 是 |
| `ErrTransport` | `TRANSPORT` | 网络请求失败 | 是 |
| `ErrSignatureRejected` | `SIGNATURE_REJECTED` | 服务端拒绝了请求的签名 | 否 |
| `ErrContentUnavailable` | `CONTENT_UNAVAILABLE` | 内容已下架、不存在，或当前调用方无权访问 | 否 |
| `ErrParamInvalid` | `PARAM_INVALID` | 服务端认为请求参数无效 | 否 |
| `ErrChallengeRequired` | `CHALLENGE_REQUIRED` | 服务端要求验证码或其他验证，匿名客户端无法继续 | 否 |
| `ErrDeviceRejected` | `DEVICE_REJECTED` | 匿名设备被拒绝，需要重置设备后重新注册 | 否 |
| `ErrEndOfList` | `END_OF_LIST` | 列表已经没有更多内容 | 否 |
| `ErrAlbumUnavailable` | `ALBUM_UNAVAILABLE` | 匿名读取不到这个合集，或者这部短剧没有其他季 | 否 |
<!-- /gen:errors-go -->

## 重试与超时

| 机制 | 说明 |
|---|---|
| 自动重试 | 限流、网络失败和服务端 5xx 错误可以重试。`RetryAttempts` 设置每个请求的最多尝试次数，SDK 在两次尝试之间等待 |
| 等待上限 | 如果服务端要求等待超过 10 秒，SDK 不等待，直接返回 `ErrRateLimited`。错误信息包含需要等待的时间 |
| 手动重试 | `sdk.Retryable(err)` 判断重试能否成功，`sdk.RetryAfter(err, fallback)` 返回建议的等待时间 |
| 超时 | 用 `context.WithTimeout` 限制一次调用的总时间。`httptransport` 对每个请求另有 30 秒的超时 |

```go
ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
defer cancel()
page, err := client.Search(ctx, "全球死考", sdk.SearchOptions{})
```

## 加密分集

如果分集是加密的，`ResolvePlayback` 返回播放模型和 `ErrDRMProtected`。把播放模型传给 `sdk.ContentKey`，取得内容密钥。

```go
pb, err := client.ResolvePlayback(ctx, episodeID, sdk.PlaybackOptions{})
if errors.Is(err, sdk.ErrDRMProtected) {
	key, err := sdk.ContentKey(pb) // 把 key.Hex 传给 ffmpeg 的 -decryption_key
	...
}
```

| 项目 | 说明 |
|---|---|
| 加密方式 | cenc-aes-ctr |
| 适用范围 | 一个分集的所有清晰度使用同一个内容密钥 |
| 计算方式 | 只在本地计算。不发送请求，不需要 ffmpeg |

## 匿名设备

SDK 第一次请求时注册匿名设备，然后把设备档案交给 `ProfileStore` 保存。

| 操作 | 方法 |
|---|---|
| 保存到文件 | `filestore.New(path)`，或 `adapters.NewClient(path, opts)` |
| 只保存在内存 | `memstore.New()`。进程退出后丢失 |
| 自定义存储 | 实现 `sdk.ProfileStore` 的 `Load`、`Save` 和 `Reset` |
| 更换匿名设备 | 调用 `client.ResetDevice(ctx)` |
| 更换机型 | 设置 `Options.Device`，然后调用 `ResetDevice`。机型只对新注册的匿名设备有效 |

如果服务端拒绝匿名设备，请求返回 `ErrDeviceRejected`。这时调用 `ResetDevice`，然后重试。

## 自定义 HTTP

实现 `sdk.Transport`，可以使用代理、记录请求或在测试中模拟服务端。

```go
type Transport interface {
	Do(ctx context.Context, req *sdk.Request) (*sdk.Response, error)
}
```

> **注意**：
>
> - 只在没有收到响应时返回错误。HTTP 错误状态码要作为 `Response` 正常返回，SDK 自己判断。
> - `Response.Body` 必须是原始字节，不要解压。SDK 根据 `Content-Encoding` 自己解压。
> - 请求 URL 的查询参数包含设备 ID。记录日志前先脱敏。

`httptransport.New()` 是默认实现。`SetLogger` 把每个请求的方法、脱敏后的 URL 和状态码写入日志。

## 接口

各语言的方法一一对应，参数的顺序相同。Go 的方法名以大写字母开头，其他语言以小写字母开头。返回值的字段见[类型参考](docs/types.md)。

<!-- gen:methods-go -->
<!-- 由 go run ./bindings/docgen 生成，请修改 sdk 的 Go 注释，不要手工修改。 -->

### 发现与搜索

| Go | 其他语言 | 作用 |
|---|---|---|
| `Discover` | `discover(options)` | 加载书城的一个频道，默认是“推荐” |
| `DiscoverPager` | `discoverAll(options)` | 逐页遍历书城的一个频道，并去掉跨页重复的短剧 |
| `Search` | `search(query, options)` | 搜索短剧，可以按真人剧、漫剧等筛选 |
| `SearchPager` | `searchAll(query, options)` | 逐页遍历搜索结果，并去掉跨页重复的短剧 |
| `Suggest` | `suggest(prefix)` | 返回某个前缀的搜索联想词 |
| `HotSearch` | `hotSearch()` | 加载搜索页的“猜你想搜”和各个热搜榜 |
| `SearchCues` | `searchCues()` | 加载搜索框里轮播的提示词 |
| `NewReleases` | `newReleases(query)` | 加载上新日历的一页 |

### 短剧与分集

| Go | 其他语言 | 作用 |
|---|---|---|
| `GetSeries` | `getSeries(seriesId)` | 加载一部短剧的详情 |
| `GetSeriesBatch` | `getSeriesBatch(seriesIds)` | 加载多部短剧及其分集，按请求顺序返回 |
| `ListEpisodes` | `listEpisodes(seriesId, options)` | 加载一部短剧的分集列表 |
| `Seasons` | `seasons(seriesId)` | 列出多季短剧的全部季 |
| `SeriesRecommend` | `seriesRecommend(seriesId)` | 加载一部短剧的相关推荐 |
| `GetAlbum` | `getAlbum(albumId)` | 加载一个合集（多季合集）及其各季 |

### 榜单与分类

| Go | 其他语言 | 作用 |
|---|---|---|
| `ShortRank` | `shortRank(query)` | 加载短剧排行榜的一页，包括演员榜 |
| `ShortFilterPanel` | `shortFilterPanel(genre)` | 加载短剧筛选面板的各个选择行 |
| `ShortFilter` | `shortFilter(query)` | 按条件筛选短剧，一次一页 |
| `RankRows` | `rankRows(cellId, offset)` | 加载书城榜单的一页（经典频道的榜单入口） |
| `RankGroups` | `rankGroups()` | 列出书城榜单的类型（经典频道的各个榜单） |
| `Ranking` | `ranking(activityId, rankListId)` | 加载一个活动榜单 |
| `Categories` | `categories(categoryId)` | 加载书籍分类的标签页：全部、男生、女生和默认标签页 |
| `CategoryLanding` | `categoryLanding(categoryId, options)` | 加载书籍分类下列表的一页，以及它的筛选面板 |

### 演员

| Go | 其他语言 | 作用 |
|---|---|---|
| `Person` | `person(userId)` | 加载一位演员或一个角色的主页 |
| `PersonWorks` | `personWorks(userId, cursor)` | 加载一位演员参演的短剧，一次一页（主页上的作品 › 剧作） |

### 播放与设备

| Go | 其他语言 | 作用 |
|---|---|---|
| `ResolvePlayback` | `resolvePlayback(episodeId)` | 加载一集的播放模型：各清晰度的地址、编码和码率 |
| `ContentKey` | 并入 `resolvePlayback` 的结果 | 从播放模型算出加密分集的内容密钥 |
| `ResetDevice` | `resetDevice()` | 丢弃当前匿名设备，并注册一台新设备 |
<!-- /gen:methods-go -->

## 命令行演示

`ohg` 是用 SDK 写的命令行工具，演示了 SDK 的典型用法。

```sh
go build -o bin/ohg ./cmd/ohg
bin/ohg search 全球死考
bin/ohg play 全球死考第一季
```

播放需要 `mpv`、`ffplay` 或 `vlc`，下载需要 `ffmpeg`。完整用法见 [docs/cli.md](docs/cli.md)。

## 版本

SDK 还没有发布 1.0 版本。在 1.0 之前，次版本号可能包含不兼容的变更。

服务端的接口没有公开文档，随时可能改版。每日的[接口契约检查](docs/development.md#接口契约检查)用于尽早发现改版。

## 贡献

根目录的 `Makefile` 是构建和测试的入口。

| 命令 | 作用 |
|---|---|
| `make test` | 运行不需要移动端工具链的所有测试。测试不访问网络 |
| `make test-mobile` | 运行 Android 和 iOS 的测试 |
| `make gen` | 修改 SDK 的类型或注释后，重新生成各语言的类型和文档 |
| `make build` | 构建 CLI、Node.js 包和所有移动端包 |

> **注意**：README 中的方法表和错误表由 `make gen` 生成。不要手工修改，修改 `sdk` 的 Go 注释。

## 术语

| 术语 | 含义 |
|---|---|
| 短剧 | 一部剧，有一个短剧 ID。电影也按短剧处理 |
| 分集 | 短剧中的一集，有一个分集 ID。序数写作“第 1 集” |
| 季、合集 | 多季短剧的每一季是一部短剧；各季组成一个合集 |
| 匿名设备 | SDK 注册的虚拟手机。SDK 用它发送请求 |
| 设备档案 | 保存匿名设备信息的 JSON 文件 |
| 播放模型 | 一个分集的播放信息：各清晰度的地址、编码和码率 |
| 内容密钥 | 解密加密分集的 AES 密钥 |

## 免责声明

### 1. 项目目的与性质

本项目（以下简称“本项目”）是作为一个技术研究与学习工具而创建的，旨在探索和学习移动应用接口协议的分析与调用技术。

本项目专注于短剧平台公开内容接口的技术研究，旨在提供给学习者和研究者作为技术交流之用。

### 2. 法律合规性声明

本项目开发者（以下简称“开发者”）郑重提醒用户在下载、安装和使用本项目时，严格遵守中华人民共和国相关法律法规，包括但不限于《中华人民共和国网络安全法》、《中华人民共和国反间谍法》等所有适用的国家法律和政策。

用户应自行承担一切因使用本项目而可能引起的法律责任。

### 3. 使用目的限制

本项目严禁用于任何非法目的或非学习、非研究的商业行为。

本项目不得用于任何形式的非法侵入他人计算机系统，不得用于任何侵犯他人知识产权或其他合法权益的行为。

用户应保证其使用本项目的目的纯属个人学习和技术研究，不得用于任何形式的非法活动。

### 4. 免责声明

开发者已尽最大努力确保本项目的正当性及安全性，但不对用户使用本项目可能引起的任何形式的直接或间接损失承担责任，包括但不限于由于使用本项目而导致的任何数据丢失、设备损坏、法律诉讼等。

### 5. 内容版权声明

通过本项目访问的短剧、图片、文字等内容，其著作权及其他权利归原权利人所有。

用户不得复制、传播、公开展示或以其他方式使用这些内容，侵犯权利人的合法权益。

本项目提供的下载和解密功能仅用于个人学习和技术研究。用户应在研究完成后及时删除相关内容。

### 6. 知识产权声明

本项目的知识产权归开发者所有。本项目受到著作权法和国际著作权条约以及其他知识产权法律和条约的保护。

用户在遵守本声明及相关法律法规的前提下，可以下载和使用本项目。

### 7. 最终解释权

关于本项目的最终解释权归开发者所有。开发者保留随时更改或更新本免责声明的权利，恕不另行通知。

## 交流群

<table>
  <tr>
    <td>
      <img src="https://pic.otaku.ren/20250309/AQADAcQxGxQjaVZ-.jpg" width="360" />
    </td>
    <td>
      <img src="https://pic.otaku.ren/20250309/AQADCMQxGxQjaVZ-.jpg" width="360" />
    </td>
  </tr>
  <tr>
    <td>
      关注公众号，获取更多项目动态，以及 AI、区块链和独立开发资讯。
    </td>
    <td>
      添加微信，备注「openhongguo」，拉你进微信交流群。
    </td>
  </tr>
</table>

也可以在社交平台上找到我：

- X (Twitter)：[@0xdinglv](https://x.com/0xdinglv)
- 小红书：[主页](https://xhslink.com/m/19FI1djnItu)
- Telegram：[@beilunyang](https://t.me/beilunyang)

## 友链

感谢 [LINUXDO](https://linux.do/) 社区提供的帮助与支持。

## 许可证

本项目使用 Apache License 2.0 许可证。全文见 [LICENSE](LICENSE)。Copyright 2026 beilunyang。
