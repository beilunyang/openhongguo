# openhongguo（Android 与 iOS）

本目录包含红果短剧（Hongguo）非官方 SDK 的 Android 版和 iOS 版。SDK 可以浏览短剧目录、搜索短剧，并取得分集的播放地址。

SDK 用匿名设备发送所有请求。SDK 不需要登录。

两个平台的核心都是 gomobile 编译的 Go SDK，所以行为与 Go SDK 和 Node.js SDK 相同。

> **免责声明**：本仓库的所有内容仅供学习和参考之用，禁止用于商业用途。使用前请阅读[完整的免责声明](../../README.md#免责声明)。使用本仓库的内容即表示您同意其所有条款和条件。

术语的含义见[根目录 README 的术语表](../../README.md#术语)。

## 快速上手

下面的代码搜索一部短剧，然后取得第 1 集的播放地址。

Android（Kotlin）：

```kotlin
import dev.openhongguo.OpenhongguoClient

val client = OpenhongguoClient.create(context)
val page = client.search("全球死考")
val episodes = client.listEpisodes(page.items.first().id)
val result = client.resolvePlayback(episodes.items.first().id)
result.playback.streams.forEach { println("${it.definition} ${it.url}") }
```

iOS（Swift）：

```swift
import Openhongguo

let client = try OpenhongguoClient.create()
let page = try await client.search("全球死考")
let episodes = try await client.listEpisodes(page.items[0].id)
let result = try await client.resolvePlayback(episodes.items[0].id)
for s in result.playback.streams { print(s.definition, s.url) }
```

第一次请求时，SDK 注册一台匿名设备，并保存设备档案。以后的请求都使用这台匿名设备。

## 安装

### Android

在 App 模块的构建文件中加入依赖。依赖来自 Maven Central：

```kotlin
dependencies {
    implementation("io.github.beilunyang:openhongguo:0.2.0")
}
```

kotlinx-coroutines 会随依赖一起引入。最低系统版本是 Android 5.0（API 21）。aar 已经声明网络权限，也包含混淆规则。不需要其他配置。

### iOS 和 macOS

如果要在 Xcode 中加入 Swift 包，执行以下步骤：

1. 选择 File > Add Package Dependencies。
2. 输入 `https://github.com/beilunyang/openhongguo`。
3. 选择版本规则 Up to Next Major Version，从 0.2.0 开始。
4. 把 `Openhongguo` 加入 App 的 target。

也可以在 `Package.swift` 中加入依赖：

```swift
.package(url: "https://github.com/beilunyang/openhongguo.git", from: "0.2.0")
```

Swift 包从 GitHub Release 下载 Go 编译出的 xcframework。最低系统版本是 iOS 13 和 macOS 11。

## 构建

只有在修改 SDK，或者为 React Native 和 Flutter 准备原生库时，才需要从源码构建。

构建需要以下工具：

| 平台 | 工具 |
|---|---|
| 两个平台 | Go 1.26 或更高版本（gomobile 的要求）、gomobile |
| Android | Android SDK（设置环境变量 `ANDROID_HOME`）、Android NDK、JDK 17 或更高版本 |
| iOS | Xcode |

`build.sh` 使用 `$ANDROID_HOME/ndk` 中最新的 NDK。如果要使用其他 NDK，设置环境变量 `ANDROID_NDK_HOME`。Android Studio 自带的 JBR 可以用作 JDK。

如果要构建库，执行以下步骤：

1. 安装 gomobile：

   ```sh
   go install golang.org/x/mobile/cmd/gomobile@latest
   gomobile init
   ```

2. 在本目录中运行 `build.sh`，参数选择下表中的一个：

   | 命令 | 结果 |
   |---|---|
   | `./build.sh android` | 生成 `android/build/outputs/aar/openhongguo-release.aar` |
   | `./build.sh ios` | 生成 `ios/OpenhongguoCore.xcframework` |
   | `./build.sh react-native` | 把 core 复制到 `../react-native` |
   | `./build.sh flutter` | 把 core 复制到 `../flutter` |
   | `./build.sh` | 执行以上所有构建 |

## 接口

两个平台的方法与 Node.js SDK 一一对应。

| 平台 | 方法的形式 |
|---|---|
| Kotlin | 挂起函数 |
| Swift | `async throws` |

<!-- gen:methods -->
<!-- 由 go run ./bindings/docgen 生成，请修改 sdk 的 Go 注释，不要手工修改。 -->

### 发现与搜索

| 方法 | 作用 |
|---|---|
| `discover(options)` | 加载书城的一个频道，默认是“推荐” |
| `discoverAll(options)` | 逐页遍历书城的一个频道，并去掉跨页重复的短剧 |
| `search(query, options)` | 搜索短剧，可以按真人剧、漫剧等筛选 |
| `searchAll(query, options)` | 逐页遍历搜索结果，并去掉跨页重复的短剧 |
| `suggest(prefix)` | 返回某个前缀的搜索联想词 |
| `hotSearch()` | 加载搜索页的“猜你想搜”和各个热搜榜 |
| `searchCues()` | 加载搜索框里轮播的提示词 |
| `newReleases(query)` | 加载上新日历的一页 |

### 短剧与分集

| 方法 | 作用 |
|---|---|
| `getSeries(seriesId)` | 加载一部短剧的详情 |
| `getSeriesBatch(seriesIds)` | 加载多部短剧及其分集，按请求顺序返回 |
| `listEpisodes(seriesId, options)` | 加载一部短剧的分集列表 |
| `listDanmaku(seriesId, episodeId, options)` | 加载一集的弹幕，一次一页 |
| `seasons(seriesId)` | 列出多季短剧的全部季 |
| `seriesRecommend(seriesId)` | 加载一部短剧的相关推荐 |
| `getAlbum(albumId)` | 加载一个合集（多季合集）及其各季 |

### 榜单与分类

| 方法 | 作用 |
|---|---|
| `shortRank(query)` | 加载短剧排行榜的一页，包括演员榜 |
| `shortFilterPanel(genre)` | 加载短剧筛选面板的各个选择行 |
| `shortFilter(query)` | 按条件筛选短剧，一次一页 |
| `rankRows(cellId, offset)` | 加载书城榜单的一页（经典频道的榜单入口） |
| `rankGroups()` | 列出书城榜单的类型（经典频道的各个榜单） |
| `ranking(activityId, rankListId)` | 加载一个活动榜单 |
| `categories(categoryId)` | 加载书籍分类的标签页：全部、男生、女生和默认标签页 |
| `categoryLanding(categoryId, options)` | 加载书籍分类下列表的一页，以及它的筛选面板 |

### 演员

| 方法 | 作用 |
|---|---|
| `person(userId)` | 加载一位演员或一个角色的主页 |
| `personWorks(userId, cursor)` | 加载一位演员参演的短剧，一次一页（主页上的作品 › 剧作） |

### 播放与设备

| 方法 | 作用 |
|---|---|
| `resolvePlayback(episodeId)` | 加载一集的播放模型：各清晰度的地址、编码和码率 |
| `resetDevice()` | 丢弃当前匿名设备，并注册一台新设备 |
<!-- /gen:methods -->

多个线程可以同时使用同一个客户端。

### 类型

返回值的类型在 `Types.kt` 和 `Types.swift` 中。类型由 Go 源码生成，字段说明与 Go SDK 相同。

| 项目 | 规则 |
|---|---|
| 属性名 | JSON 字段名转为驼峰，例如 `cover_url` 转为 `coverUrl`。两个平台相同 |
| 缺省值 | 缺省的字段使用零值（空字符串、0、空列表）。只有 Go 中的指针字段可以为空，例如 `Series.novel` |
| 枚举 | `ContentType`、`SeriesStatus` 等枚举是包含整数的值类型。已知的取值是常量，例如 Kotlin 的 `ContentType.SHORT_PLAY` 和 Swift 的 `.shortPlay`。SDK 保留服务端新增的取值 |
| 查询参数 | `SearchOptions` 等查询参数的字段都可以为空。空字段不发送 |

> **注意**：Swift 的 `Stream` 类型与 Foundation 的 `Stream` 同名。如果同时导入两个模块，写 `Openhongguo.Stream`。

### 翻页

`discoverAll` 和 `searchAll` 逐部遍历带游标的列表。服务端会在不同的页中返回同一部短剧。这两个方法删除重复的短剧。

| 平台 | 返回类型 |
|---|---|
| Kotlin | `Flow` |
| Swift | `AsyncThrowingStream` |

```kotlin
client.searchAll("末世", maxPages = 3).collect { println("${it.id} ${it.title}") }
```

```swift
for try await series in client.searchAll("末世", maxPages: 3) {
    print(series.id, series.title)
}
```

### 加密分集

`resolvePlayback` 返回播放模型和内容密钥。如果分集是加密的，`playback.protected` 是真，`contentKey.key` 是内容密钥。加密方式是 cenc-aes-ctr。

如果 SDK 不能推导内容密钥，`contentKey` 为空。`contentKeyError` 给出原因。播放模型仍然正常返回。

### 取消

如果 Kotlin 的协程或 Swift 的 Task 取消，SDK 取消进行中的请求。

| 平台 | 抛出的异常 |
|---|---|
| Kotlin | `CancellationException` |
| Swift | `CancellationError` |

## 设置

`create` 的第二个参数是 `ClientOptions`。所有设置都是可选的。

| 设置 | 作用 |
|---|---|
| `apiOrigin`、`registerOrigin`、`distributionOrigin` | 接口地址。默认值与 App 相同 |
| `retryAttempts` | 可重试的失败的最多尝试次数。默认值是 3 |
| `deviceResets` | 设备被拒或播放返回空结果时，一次调用最多更换几台匿名设备。默认值是 2，0 表示关闭 |
| `device` | 新匿名设备的机型。空字段使用默认值（小米 14） |
| `logger` | 接收诊断日志的函数。日志中的 URL 已经脱敏 |

### 设备档案

设备档案的默认位置如下：

| 平台 | 默认位置 |
|---|---|
| Android | `noBackupFilesDir/openhongguo/anonymous.json` |
| iOS | `Application Support/openhongguo/anonymous.json`。目录标记为不备份 |

两个位置都不参加系统备份。所以设备档案不会通过备份复制到其他手机。

如果要指定设备档案的文件，使用以下方法：

| 平台 | 方法 |
|---|---|
| Kotlin | `OpenhongguoClient.create(file, options)` |
| Swift | `OpenhongguoClient.create(profile: url, options: options)` |

## 错误处理

| 平台 | 异常类型 |
|---|---|
| Kotlin | `OpenhongguoException` |
| Swift | `OpenhongguoError` |

两种异常都有三个属性：

| 属性 | 含义 |
|---|---|
| `code` | 错误码，例如 `RATE_LIMITED`（Swift 是 `.rateLimited`） |
| `retryable` | 重试同一请求能否成功 |
| `retryAfterMs` | 服务端建议的等待时间，单位是毫秒。0 表示没有建议 |

```kotlin
try {
    client.getSeries(seriesId)
} catch (e: OpenhongguoException) {
    if (e.code == ErrorCode.CONTENT_UNAVAILABLE) println("这部短剧已下架")
}
```

```swift
do {
    _ = try await client.getSeries(seriesId)
} catch let e as OpenhongguoError where e.code == .contentUnavailable {
    print("这部短剧已下架")
}
```

下表列出所有错误码。Kotlin 的常量名与错误码相同。Swift 的常量名是驼峰，例如 `RATE_LIMITED` 是 `.rateLimited`。

<!-- gen:errors -->
<!-- 由 go run ./bindings/docgen 生成，请修改 sdk 的 Go 注释，不要手工修改。 -->

| 错误码 | 含义 | 可重试 |
|---|---|---|
| `INVALID_OPTIONS` | 客户端的设置或调用参数无效 | 否 |
| `PROFILE_NOT_FOUND` | 还没有保存过设备档案 | 否 |
| `SIGNER_UNAVAILABLE` | 请求需要签名，而客户端没有可用的签名器 | 否 |
| `AUTHENTICATION_REQUIRED` | 内容需要登录才能访问，匿名客户端无法继续 | 否 |
| `NOT_PROTECTED` | 为未加密的播放模型请求了内容密钥 | 否 |
| `DRM_PROTECTED` | 分集是加密的，需要内容密钥才能解密 | 否 |
| `ANONYMOUS_LIMITED` | 服务端给匿名设备返回了空结果 | 否 |
| `RATE_LIMITED` | 请求太频繁，被服务端限流 | 是 |
| `TRANSPORT` | 网络请求失败 | 是 |
| `SIGNATURE_REJECTED` | 服务端拒绝了请求的签名 | 否 |
| `CONTENT_UNAVAILABLE` | 内容已下架、不存在，或当前调用方无权访问 | 否 |
| `PARAM_INVALID` | 服务端认为请求参数无效 | 否 |
| `CHALLENGE_REQUIRED` | 服务端要求验证码或其他验证，匿名客户端无法继续 | 否 |
| `DEVICE_REJECTED` | 匿名设备被拒绝，需要重置设备后重新注册 | 否 |
| `END_OF_LIST` | 列表已经没有更多内容 | 否 |
| `ALBUM_UNAVAILABLE` | 匿名读取不到这个合集，或者这部短剧没有其他季 | 否 |
| `CANCELLED` | 调用被取消 | 否 |
<!-- /gen:errors -->

## 实现原理

| 目录 | 内容 |
|---|---|
| `core/` | gomobile 编译的 Go 包。它是一个单独的 Go 模块，因为 gomobile 要求模块依赖 `golang.org/x/mobile`。它只导出一个按名称调用的入口。参数和结果都是 JSON。结果通过回调返回，不占用调用方的线程 |
| `android/` | Kotlin 封装，是一个 Gradle 库工程 |
| `ios/` | Swift 封装，是一个 Swift 包 |
| `gen/` | 根据 Go 源码生成 `Types.kt` 和 `Types.swift` |
| `fixtureserver/` | 在本机回放 `testdata/protocol` 的样本，供测试使用 |

Go 的 `net/http` 发送请求。Go 读写设备档案。

所有语言的 SDK 共用 `bindings/internal/dispatch` 的方法表。如果要增加 SDK 方法，执行以下步骤：

1. 在 `bindings/internal/dispatch` 中注册方法。
2. 在各平台的客户端中增加对应的方法。
3. 在 `bindings/docgen/methods.go` 的 `groups` 中加入方法，然后运行 `make gen`。

库的大小主要来自 Go 运行时：

| 平台 | 大小 |
|---|---|
| Android | 每种 CPU 架构约 8 MB，aar 共约 13 MB。如果 App 按架构拆分 APK，每个 APK 只包含一份 |
| iOS | 真机部分约 9 MB |

## 开发

如果修改了 `sdk` 或 `sdk/model` 的类型，在仓库根目录重新生成类型：

```sh
go run ./bindings/mobile/gen
```

仓库根目录的 `go test ./...` 检查生成的文件是否过时。

| 命令 | 作用 |
|---|---|
| `(cd core && go test ./...)` | 测试 Go 入口。测试回放样本 |
| `./build.sh android` | 构建 aar，同时运行 Kotlin 单元测试 |
| `(cd ios && swift test)` | 在 macOS 上运行 Swift 测试。测试回放样本 |

Kotlin 的设备测试需要模拟器和样本服务。如果要运行设备测试，执行以下步骤：

1. 启动一台 Android 模拟器。
2. 在仓库根目录启动样本服务：

   ```sh
   go run ./bindings/mobile/fixtureserver -addr 127.0.0.1:8790
   ```

3. 运行设备测试：

   ```sh
   (cd android && ./gradlew connectedDebugAndroidTest)
   ```

## 许可证

本库使用 Apache License 2.0 许可证，见仓库根目录的 `LICENSE`。gomobile 生成的代码来自 Go 项目，使用 BSD 许可证。
