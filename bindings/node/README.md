# openhongguo（Node.js）

openhongguo 是红果短剧（Hongguo）的非官方 Node.js SDK。SDK 可以浏览短剧目录、搜索短剧，并取得分集的播放地址。

SDK 用匿名设备发送所有请求。SDK 不需要登录。

SDK 的核心是 Go SDK 编译成的 WebAssembly，所以 Node.js SDK 与 Go SDK 的行为相同。

> **免责声明**：本仓库的所有内容仅供学习和参考之用，禁止用于商业用途。使用前请阅读[完整的免责声明](https://github.com/beilunyang/openhongguo#免责声明)。使用本仓库的内容即表示您同意其所有条款和条件。

术语的含义见[根目录 README 的术语表](https://github.com/beilunyang/openhongguo#术语)。

## 快速上手

下面的代码搜索一部短剧，然后取得第 1 集的播放地址。

```js
import { createClient } from 'openhongguo';

const client = await createClient();
const page = await client.search('全球死考');
const episodes = await client.listEpisodes(page.items[0].id);
const { playback } = await client.resolvePlayback(episodes.items[0].id);
console.log(playback.streams.map((s) => `${s.definition} ${s.url}`));
```

第一次请求时，SDK 注册一台匿名设备，并保存设备档案。以后的请求都使用这台匿名设备。

## 安装

需要 Node.js 20 或更高版本。

```sh
npm install openhongguo
```

> **注意**：这个包只支持 ES 模块（`import`）。

## 接口

所有方法都返回 Promise。如果方法失败，它抛出 `OpenhongguoError`。

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

参数和返回值的类型见 `index.d.ts` 和 `types.d.ts`。类型定义由 Go 源码生成，字段说明与 Go SDK 相同。

### 翻页

`discoverAll` 和 `searchAll` 逐部遍历带游标的列表。服务端会在不同的页中返回同一部短剧。这两个方法删除重复的短剧。

```js
for await (const series of client.searchAll('末世', {}, { maxPages: 3 })) {
  console.log(series.id, series.title);
}
```

### 搜索筛选

`SearchFilter` 提供“综合”分类的筛选项。

```js
import { SearchFilter } from 'openhongguo';

await client.search('奇幻', { filters: [SearchFilter.ShortPlay, SearchFilter.Film] });
```

| 筛选项 | 含义 |
|---|---|
| `ShortPlay` | 真人剧 |
| `Comic` | 漫剧 |
| `AIComic` | AI 剧 |
| `Film` | 影视剧 |
| `Unwatched` | 未看过 |

### 加密分集

`resolvePlayback` 返回 `{ playback, contentKey }`。如果分集是加密的，`playback.protected` 是 `true`，`contentKey.key` 是内容密钥。加密方式是 cenc-aes-ctr。

```js
const { playback, contentKey } = await client.resolvePlayback(episodeId);
if (playback.protected) {
  // 例如：ffmpeg -decryption_key <key> -i <url> -c copy out.mp4
  console.log(contentKey.key);
}
```

如果 SDK 不能推导内容密钥，结果中没有 `contentKey`。`contentKeyError` 给出原因。播放模型仍然正常返回。

## 字段命名

| 方向 | 风格 | 示例 |
|---|---|---|
| 返回值 | 下划线风格，与 Go SDK 的 JSON 相同 | `has_more`、`cover_url`、`episode_count` |
| 查询参数 | 驼峰风格 | `pageSize`、`cursor` |

返回值的字段名也与 CLI 的 `--json` 输出相同。

## 设置

`createClient(options)` 的设置都是可选的。

| 设置 | 作用 |
|---|---|
| `profile` | 设备档案的位置：文件路径，或自定义的存储对象 |
| `transport` | 自定义的请求发送函数 |
| `timeoutMs` | 默认发送函数的超时，单位是毫秒。默认值是 30000 |
| `retryAttempts` | 可重试的失败的最多尝试次数。默认值是 3 |
| `deviceResets` | 设备被拒或播放返回空结果时，一次调用最多更换几台匿名设备。默认值是 2，0 表示关闭 |
| `device` | 新匿名设备的机型。空字段使用默认值（小米 14） |
| `log` | 接收诊断日志的函数。日志中的 URL 已经脱敏 |
| `apiOrigin`、`registerOrigin`、`distributionOrigin` | 接口地址。默认值与 App 相同 |

### 设备档案

设备档案的默认位置是 `<用户配置目录>/openhongguo/anonymous.json`。CLI 使用同一个设备档案。

| 系统 | 默认位置 |
|---|---|
| macOS | `~/Library/Application Support/openhongguo/anonymous.json` |
| 其他系统 | `<用户配置目录>/openhongguo/anonymous.json` |

如果要更改目录，设置环境变量 `OHG_CONFIG_DIR`。

如果只把设备档案保存在内存中，使用 `memoryProfileStore`。进程退出后，设备档案丢失。

```js
import { createClient, memoryProfileStore } from 'openhongguo';

const client = await createClient({ profile: memoryProfileStore() });
```

自定义的存储对象必须实现三个异步方法：

| 方法 | 作用 |
|---|---|
| `load()` | 读取设备档案。如果没有设备档案，返回 `null` |
| `save(json)` | 保存设备档案 |
| `reset()` | 删除设备档案 |

### 自定义请求

`transport` 接收一个请求对象，返回一个响应对象。

```js
const client = await createClient({
  transport: async ({ method, url, headers, body }) => {
    // ……发送请求……
    return { status, headers, cookies, body }; // body 是 Uint8Array
  },
});
```

> **注意**：响应体必须是原始字节。不要自动解压响应体。SDK 根据 `Content-Encoding` 自己解压。

> **注意**：只在没有收到响应时，`transport` 才抛出错误。如果收到 HTTP 错误状态码，正常返回响应。SDK 判断状态码。

## 错误处理

所有错误都是 `OpenhongguoError`。`OpenhongguoError` 有三个属性：

| 属性 | 含义 |
|---|---|
| `code` | 错误码，例如 `RATE_LIMITED`、`CONTENT_UNAVAILABLE`、`DEVICE_REJECTED` |
| `retryable` | 重试同一请求能否成功 |
| `retryAfterMs` | 服务端建议的等待时间，单位是毫秒。0 表示没有建议 |

```js
import { OpenhongguoError } from 'openhongguo';

try {
  await client.getSeries(seriesId);
} catch (e) {
  if (e instanceof OpenhongguoError && e.code === 'CONTENT_UNAVAILABLE') {
    console.log('这部短剧已下架');
  }
}
```

下表列出所有错误码。

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

`dist/openhongguo.wasm` 是 Go SDK 编译成的 WebAssembly。桥接代码在 `bindings/node/wasm` 中。方法表在 `bindings/internal/dispatch` 中，Android SDK 和 iOS SDK 也使用它。

| 工作 | 执行者 |
|---|---|
| 发送请求 | JS。Go 把请求传给 `transport`。默认的 `transport` 使用 `node:https` |
| 读写设备档案 | JS。Go 通过 `profile` 的三个方法读写 JSON |
| 执行 SDK 方法 | Go。每个方法在一个 goroutine 中运行，结果以 JSON 返回给 JS |

第一次调用 `createClient` 时，SDK 加载 wasm。加载需要几百毫秒。同一进程中的所有客户端共用一个 wasm 实例。

## 开发

从源码构建需要 Go 1.22 或更高版本。在本目录中运行以下命令。

| 命令 | 作用 |
|---|---|
| `npm run build` | 编译 wasm，复制 `wasm_exec.js`，重新生成 `types.d.ts` |
| `npm test` | 用 `testdata/protocol` 的样本回放测试。测试不访问网络 |

> **注意**：不要手工修改 `types.d.ts`。`go run ./bindings/node/tsgen` 生成这个文件。仓库根目录的 `go test ./...` 检查它是否过时。

## 许可证

本包使用 Apache License 2.0 许可证。`dist/wasm_exec.js` 来自 Go 项目，使用 BSD 许可证，全文见 `GO_LICENSE`。
