# 发布

本文说明怎样发布一个新版本。所有平台使用同一个版本号，构建和发布由 GitHub Actions 完成。

## 发布的包

| 平台 | 包 | 发布位置 |
|---|---|---|
| Go | `github.com/beilunyang/openhongguo` | Git tag，proxy.golang.org 收录 |
| Swift（iOS、macOS） | 仓库根目录的 `Package.swift` | Git tag，xcframework 在 GitHub Release |
| Android | `io.github.beilunyang:openhongguo` | Maven Central |
| Node.js | `openhongguo` | npm |
| React Native | `react-native-openhongguo` | npm |
| Flutter | `openhongguo` | pub.dev |
| 命令行 | openhongguo-cli（命令 `ohg`） | GitHub Release 的二进制 |

> **注意**：不发布到 CocoaPods trunk。CocoaPods trunk 在 2026 年 12 月 2 日转为只读。React Native 和 Flutter 的 podspec 随各自的包分发，不使用 trunk。

## 流程

| workflow | 触发方式 | 作用 |
|---|---|---|
| `prepare-release` | 手动运行，输入版本号 | 改版本号，构建原生库，运行测试，写入 `Package.swift` 的校验和。审批通过后提交、打 tag、推送，并创建 GitHub Release |
| `publish` | 推送 `vX.Y.Z` tag | 从 Release 取得原生库，发布到 npm、pub.dev、Maven Central，并让 proxy.golang.org 收录 |
| `release` | 创建 Release | 构建 openhongguo-cli 的 macOS、Linux 和 Windows 二进制，上传到 Release |

原生库只在 `prepare-release` 中构建一次。Swift 下载的 xcframework 和其他平台的包使用同一份构建。

版本号的位置都记在 `scripts/set-version.sh` 中。`scripts/set-version.sh --check X.Y.Z` 检查所有位置是否都是 `X.Y.Z`。

## 发布一个版本

> **警告**：npm、Maven Central 和 pub.dev 的发布不能撤回。批准前，确认版本号正确。

如果要发布版本 `X.Y.Z`，执行以下步骤：

1. 如果要写更新说明，在 `bindings/flutter/CHANGELOG.md` 顶部加上 `## X.Y.Z` 一节，提交到 `main`。没有写时，workflow 加一节，链接到 GitHub Release。
2. 在 Actions 页面运行 **prepare release**。分支选 `main`，版本号填 `X.Y.Z`（不带 `v`）。`dry_run` 默认勾选：只构建和测试，不提交、不打 tag。改动的 diff、xcframework zip 和原生库保存为 artifact。
3. 试运行通过后，取消 `dry_run` 再运行一次。
4. 构建和测试通过后，`tag` 任务等待审批。在运行页面上批准。
5. 等待 `publish` 和 `release` 完成。各平台的发布并行进行，大约需要 20～30 分钟。

如果 `publish` 的某个任务失败，修复原因后重新运行这次 workflow。已经发布了这个版本的平台会被跳过。

> **注意**：Maven Central 在发布后 10～30 分钟才显示新版本。在这段时间内重新运行，Maven Central 任务会因为版本已存在而失败，可以忽略。

## 一次性设置

以下设置只需要做一次。

### GitHub 环境

在仓库的 **Settings > Environments** 中创建两个环境：

| 环境 | 设置 | Secrets |
|---|---|---|
| `release` | **Required reviewers** 加入发布负责人 | `RELEASE_TOKEN` |
| `publish` | **Deployment branches and tags** 只允许 tag `v*` | `MAVEN_CENTRAL_USERNAME`、`MAVEN_CENTRAL_PASSWORD`、`SIGNING_KEY`、`SIGNING_KEY_PASSWORD` |

`RELEASE_TOKEN` 是 fine-grained personal access token：

| 项目 | 值 |
|---|---|
| Repository access | 只选 `beilunyang/openhongguo` |
| Permissions | Contents：Read and write |
| Expiration | 设置过期时间，到期前更换 |

> **注意**：用 workflow 自带的 `GITHUB_TOKEN` 推送的 tag 不会触发 `publish`，所以需要 `RELEASE_TOKEN`。

Maven Central 的 Secrets：

| Secret | 内容 |
|---|---|
| `MAVEN_CENTRAL_USERNAME`、`MAVEN_CENTRAL_PASSWORD` | 在 [Central Portal](https://central.sonatype.com/account) 生成的 user token |
| `SIGNING_KEY` | ASCII 格式的 GPG 私钥：`gpg --armor --export-secret-keys <KEY_ID>` 的输出 |
| `SIGNING_KEY_PASSWORD` | 这个私钥的密码 |

### npm

在 npmjs.com 中，分别打开 `openhongguo` 和 `react-native-openhongguo` 的 **Settings > Trusted publishing**，添加 GitHub Actions：

| 项目 | 值 |
|---|---|
| Organization or user | `beilunyang` |
| Repository | `openhongguo` |
| Workflow filename | `publish.yml` |
| Environment | `publish` |

### pub.dev

在 pub.dev 打开 `openhongguo` 的 **Admin** 页面，启用 **Automated publishing** 中的 GitHub Actions：

| 项目 | 值 |
|---|---|
| Repository | `beilunyang/openhongguo` |
| Tag pattern | `v{{version}}` |
| Environment | `publish` |

## 检查

| 平台 | 检查方法 |
|---|---|
| Go | `go get github.com/beilunyang/openhongguo@vX.Y.Z` |
| Swift | 在新的 Swift 包中加入依赖，运行 `swift build` |
| Android | 在 Maven Central 搜索 `io.github.beilunyang:openhongguo`。新版本需要 10～30 分钟才能出现 |
| npm | `npm view openhongguo version` |
| pub.dev | 打开 `https://pub.dev/packages/openhongguo` |
| 命令行 | Release 页面有 `openhongguo-cli_vX.Y.Z_*` 文件 |

## 不用 Actions 发布

GitHub Actions 不可用时，可以在本机发布：

1. 运行 `scripts/set-version.sh X.Y.Z`。
2. 运行 `make mobile`，然后运行 `make test test-mobile`。
3. 打包 xcframework，计算校验和，把校验和写入 `Package.swift`：

   ```sh
   mkdir -p dist/vX.Y.Z
   (cd bindings/mobile/ios && ditto -c -k --sequesterRsrc --keepParent \
     OpenhongguoCore.xcframework ../../../dist/vX.Y.Z/OpenhongguoCore.xcframework.zip)
   swift package compute-checksum dist/vX.Y.Z/OpenhongguoCore.xcframework.zip
   ```

4. 运行 `scripts/set-version.sh --check X.Y.Z`，提交，然后推送。
5. 创建 tag 并推送，然后创建 GitHub Release，上传第 3 步的 zip。

   > **警告**：必须上传第 3 步生成的 zip。重新打包会得到不同的校验和，SPM 会拒绝下载。

6. 发布到 Maven Central、npm 和 pub.dev。Maven Central 的凭据放在 `~/.gradle/gradle.properties` 的 `mavenCentralUsername`、`mavenCentralPassword`、`signingInMemoryKey` 和 `signingInMemoryKeyPassword` 中。用 JDK 17 或 21 运行 Gradle：AGP 内置的 Dokka 不能识别 JDK 25 的版本号。

   ```sh
   (cd bindings/mobile/android && ./gradlew publishAndReleaseToMavenCentral)
   (cd bindings/node && npm run build && npm publish)
   (cd bindings/react-native && npm publish)
   (cd bindings/flutter && flutter pub publish)
   GOPROXY=https://proxy.golang.org go list -m github.com/beilunyang/openhongguo@vX.Y.Z
   ```
