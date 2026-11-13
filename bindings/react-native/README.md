# react-native-openhongguo

本目录包含红果短剧（Hongguo）非官方 SDK 的 React Native 版。SDK 可以浏览短剧目录、搜索短剧，并取得分集的播放地址。

SDK 用匿名设备发送所有请求。SDK 不需要登录。

原生层是 gomobile 编译的 Go SDK，通过 Expo Modules 接入。所以 SDK 的行为与 Go SDK 相同，接口与 Node.js SDK 相同。

> **免责声明**：本仓库的所有内容仅供学习和参考之用，禁止用于商业用途。使用前请阅读[完整的免责声明](https://github.com/beilunyang/openhongguo#免责声明)。使用本仓库的内容即表示您同意其所有条款和条件。

术语的含义见[根目录 README 的术语表](https://github.com/beilunyang/openhongguo#术语)。

## 快速上手

下面的代码搜索一部短剧，然后取得第 1 集的播放地址。

```ts
import { createClient } from 'react-native-openhongguo';

const client = await createClient();
const page = await client.search('全球死考');
const episodes = await client.listEpisodes(page.items[0].id);
const { playback } = await client.resolvePlayback(episodes.items[0].id);
console.log(playback.streams.map((s) => `${s.definition} ${s.url}`));
```

第一次请求时，SDK 注册一台匿名设备，并保存设备档案。以后的请求都使用这台匿名设备。

## 要求

| 项目 | 要求 |
|---|---|
| Expo Modules | Expo 应用可以直接使用。不使用 Expo 的 React Native 应用，先运行 `npx install-expo-modules` |
| 原生代码 | 包含原生代码，不能在 Expo Go 中运行。用 `npx expo run:android` 或开发构建（development build）运行 |
| 系统版本 | Android 5.0（API 21）或更高，iOS 15.1 或更高 |

## 安装

如果要在 App 中安装这个包，执行以下步骤：

1. 安装包：

   ```sh
   npx expo install react-native-openhongguo
   ```

2. 重新生成原生工程（`npx expo prebuild`），或者运行 `npx expo run:android`。

这个包已经包含编译好的原生库，不需要 Go 和 gomobile。

包的大小主要来自 Go 运行时：

| 平台 | 大小 |
|---|---|
| Android | 每种 CPU 架构约 8 MB |
| iOS | 真机部分约 9 MB |

如果 Android 应用按架构拆分 APK（例如 `-PreactNativeArchitectures=arm64-v8a`），每个 APK 只包含一份。

## 接口

接口与 Node.js SDK `openhongguo` 相同，见 [bindings/node/README.md](https://github.com/beilunyang/openhongguo/tree/main/bindings/node#接口)。

| 项目 | 说明 |
|---|---|
| 返回值 | 所有方法都返回 Promise |
| 错误 | 如果方法失败，它抛出 `OpenhongguoError` |
| 翻页 | 用 `discoverAll` 和 `searchAll`（`for await`） |
| 字段名 | 下划线风格，例如 `has_more`、`cover_url`。与 Go SDK 的 JSON 相同 |

下表列出与 Node.js SDK 不同的设置和方法。

| 设置或方法 | 说明 |
|---|---|
| `profilePath` | 设备档案的路径，代替 Node.js SDK 的 `profile`。默认目录不参加系统备份：Android 是 `noBackupFilesDir`，iOS 是标记为不备份的 Application Support。所以设备档案不会通过备份复制到其他手机 |
| `log` | 接收诊断日志的函数 |
| `client.release()` | 释放原生层的客户端。调用后，其他方法都会失败 |

> **注意**：这个包不支持 Node.js SDK 的 `transport` 和 `timeoutMs`。Go 的 `net/http` 直接发送请求。每个请求的超时是 30 秒。

## 开发

如果要从源码使用这个包，执行以下步骤：

1. 编译 Go core，并把它复制到 `android/` 和 `ios/`。构建需要 Go 1.26 或更高版本和 gomobile，环境要求见 [bindings/mobile/README.md](https://github.com/beilunyang/openhongguo/tree/main/bindings/mobile#构建)：

   ```sh
   ../mobile/build.sh react-native
   ```

2. 打包，然后在 App 中安装生成的文件：

   ```sh
   npm install
   npm pack
   npm install /path/to/react-native-openhongguo-0.3.0.tgz
   ```

| 命令 | 作用 |
|---|---|
| `npm install` | 安装依赖。只安装 typescript |
| `npm test` | 编译到 `build/`，然后用假的原生模块测试客户端 |

> **注意**：不要手工修改 `src/types.ts`。`go run ./bindings/node/tsgen` 生成这个文件，内容与 Node.js SDK 的 `types.d.ts` 相同。仓库根目录的 `go test ./...` 检查它是否过时。

## 许可证

本包使用 Apache License 2.0 许可证，见仓库根目录的 `LICENSE`。gomobile 生成的代码来自 Go 项目，使用 BSD 许可证。
