# openhongguo（Flutter）

本目录包含红果短剧（Hongguo）非官方 SDK 的 Flutter 版。SDK 可以浏览短剧目录、搜索短剧，并取得分集的播放地址。

SDK 用匿名设备发送所有请求。SDK 不需要登录。

原生层是 gomobile 编译的 Go SDK。Dart 代码通过平台通道调用原生层，所以行为与 Go SDK 相同。

> **免责声明**：本仓库的所有内容仅供学习和参考之用，禁止用于商业用途。使用前请阅读[完整的免责声明](https://github.com/beilunyang/openhongguo#免责声明)。使用本仓库的内容即表示您同意其所有条款和条件。

术语的含义见[根目录 README 的术语表](https://github.com/beilunyang/openhongguo#术语)。

## 快速上手

下面的代码搜索一部短剧，然后取得第 1 集的播放地址。

```dart
import 'package:openhongguo/openhongguo.dart';

final client = await OpenhongguoClient.create();
final page = await client.search('全球死考');
final episodes = await client.listEpisodes(page.items.first.id);
final result = await client.resolvePlayback(episodes.items.first.id);
for (final s in result.playback.streams) {
  print('${s.definition} ${s.url}');
}
```

第一次请求时，SDK 注册一台匿名设备，并保存设备档案。以后的请求都使用这台匿名设备。

## 要求

| 平台 | 最低版本 |
|---|---|
| Android | 7.0（API 24） |
| iOS | 15 |

最低版本与 Flutter 的要求相同。

> **注意**：这个包不支持 Web 和桌面平台。

## 安装

```sh
flutter pub add openhongguo
```

这个包已经包含编译好的原生库，不需要 Go 和 gomobile。

包的大小主要来自 Go 运行时：

| 平台 | 大小 |
|---|---|
| Android | 每种 CPU 架构约 8 MB |
| iOS | 真机部分约 9 MB |

如果用 `flutter build apk --split-per-abi` 按架构拆分 APK，每个 APK 只包含一份。

## 接口

方法与其他语言的 SDK 一一对应。所有方法都返回 `Future`。如果方法失败，它抛出 `OpenhongguoException`。

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

`release()` 释放原生层的客户端。调用 `release()` 后，其他方法都会失败。

### 类型

返回值的类型由 Go 源码生成。字段说明与 Go SDK 相同。

| 项目 | 规则 |
|---|---|
| 属性名 | JSON 字段名转为驼峰，例如 `cover_url` 转为 `coverUrl`。Kotlin SDK 和 Swift SDK 使用相同的规则 |
| 缺省值 | 缺省的字段使用零值（空字符串、0、空列表）。只有 Go 中的指针字段可以为 null，例如 `Series.novel` |
| 枚举 | `ContentType`、`SeriesStatus` 等枚举是包含整数的类。已知的取值是静态常量，例如 `ContentType.shortPlay`。SDK 保留服务端新增的取值。如果常量名与 Dart 关键字相同，常量名加上 `Value` 后缀，例如 `RankChangeKind.newValue` |
| 查询参数 | `SearchOptions` 等查询参数的字段都可以为空。空字段不发送。用 `copyWith` 修改部分字段 |

有两个类型使用了不同的名称，以免与 Dart 和 Flutter 的类型同名：

| Go 中的名称 | Dart 中的名称 | 原因 |
|---|---|---|
| `Page` | `PagedResult` | Flutter widgets 有 `Page` |
| `Stream` | `VideoStream` | `dart:core` 有 `Stream` |

### 翻页

`discoverAll` 和 `searchAll` 逐部遍历带游标的列表，返回 `Stream`。服务端会在不同的页中返回同一部短剧。这两个方法删除重复的短剧。

```dart
await for (final series in client.searchAll('末世', const SearchOptions(), 3)) {
  print('${series.id} ${series.title}');
}
```

### 加密分集

`resolvePlayback` 返回播放模型和内容密钥。如果分集是加密的，`playback.protected` 是 true，`contentKey.key` 是内容密钥。加密方式是 cenc-aes-ctr。

如果 SDK 不能推导内容密钥，`contentKey` 是 null。`contentKeyError` 给出原因。播放模型仍然正常返回。

## 设置

`OpenhongguoClient.create` 的参数是 `ClientOptions`。所有设置都是可选的。

| 设置 | 作用 |
|---|---|
| `profilePath` | 设备档案的路径。默认目录不参加系统备份：Android 是 `noBackupFilesDir`，iOS 是标记为不备份的 Application Support |
| `apiOrigin`、`registerOrigin`、`distributionOrigin` | 接口地址。默认值与 App 相同 |
| `retryAttempts` | 可重试的失败的最多尝试次数。默认值是 3 |
| `deviceResets` | 设备被拒或播放返回空结果时，一次调用最多更换几台匿名设备。默认值是 2，0 表示关闭 |
| `device` | 新匿名设备的机型。空字段使用默认值（小米 14） |
| `log` | 接收诊断日志的函数。日志中的 URL 已经脱敏 |

## 错误处理

所有错误都是 `OpenhongguoException`。`OpenhongguoException` 有三个属性：

| 属性 | 含义 |
|---|---|
| `code` | 错误码，例如 `ErrorCode.rateLimited` |
| `retryable` | 重试同一请求能否成功 |
| `retryAfterMs` | 服务端建议的等待时间，单位是毫秒。0 表示没有建议 |

```dart
try {
  await client.getSeries(seriesId);
} on OpenhongguoException catch (e) {
  if (e.code == ErrorCode.contentUnavailable) print('这部短剧已下架');
}
```

下表列出所有错误码。Dart 的常量名是驼峰，例如 `RATE_LIMITED` 是 `ErrorCode.rateLimited`。

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

## 开发

如果要从源码使用这个包，执行以下步骤：

1. 编译 Go core，并把它复制到 `android/` 和 `ios/`。构建需要 Go 1.26 或更高版本和 gomobile，环境要求见 [bindings/mobile/README.md](https://github.com/beilunyang/openhongguo/tree/main/bindings/mobile#构建)：

   ```sh
   ../mobile/build.sh flutter
   ```

2. 在 App 的 `pubspec.yaml` 中用路径引用这个包：

   ```yaml
   dependencies:
     openhongguo:
       path: ../openhongguo/bindings/flutter
   ```

> **注意**：不要手工修改 `lib/src/types.dart`。`go run ./bindings/mobile/gen` 生成这个文件。仓库根目录的 `go test ./...` 检查它是否过时。

| 命令 | 作用 |
|---|---|
| `flutter test` | 用模拟的平台通道测试客户端 |

## 许可证

本包使用 Apache License 2.0 许可证，见仓库根目录的 `LICENSE`。gomobile 生成的代码来自 Go 项目，使用 BSD 许可证。
