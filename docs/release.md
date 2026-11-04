# 发布

本文说明怎样发布一个新版本。所有平台使用同一个版本号。

## 发布的包

| 平台 | 包 | 发布位置 |
|---|---|---|
| Go | `github.com/beilunyang/openhongguo` | Git tag，proxy.golang.org 自动收录 |
| Swift（iOS、macOS） | 仓库根目录的 `Package.swift` | Git tag，xcframework 在 GitHub Release |
| Android | `io.github.beilunyang:openhongguo` | Maven Central |
| Node.js | `openhongguo` | npm |
| React Native | `react-native-openhongguo` | npm |
| Flutter | `openhongguo` | pub.dev |

> **注意**：不发布到 CocoaPods trunk。CocoaPods trunk 在 2026 年 12 月 2 日转为只读。React Native 和 Flutter 的 podspec 随各自的包分发，不使用 trunk。

## 版本号的位置

| 文件 | 内容 |
|---|---|
| `bindings/node/package.json` | `version` |
| `bindings/react-native/package.json` | `version` |
| `bindings/flutter/pubspec.yaml` | `version` |
| `bindings/flutter/ios/openhongguo.podspec` | `s.version` |
| `bindings/flutter/CHANGELOG.md` | 新版本的一节 |
| `bindings/mobile/android/build.gradle.kts` | `coordinates(...)` 的第三个参数 |
| `Package.swift` | xcframework 的 URL（包含版本号）和 `checksum` |

## 准备

如果要准备版本 `X.Y.Z`，执行以下步骤：

1. 修改上表中的所有版本号。
2. 构建所有原生库：

   ```sh
   make mobile
   ```

3. 打包 xcframework，然后计算校验和：

   ```sh
   mkdir -p dist/vX.Y.Z
   (cd bindings/mobile/ios && ditto -c -k --sequesterRsrc --keepParent \
     OpenhongguoCore.xcframework ../../../dist/vX.Y.Z/OpenhongguoCore.xcframework.zip)
   swift package compute-checksum dist/vX.Y.Z/OpenhongguoCore.xcframework.zip
   ```

4. 把校验和和新的 URL 写入 `Package.swift`。
5. 运行测试：

   ```sh
   make test test-mobile
   ```

6. 提交，然后推送。

> **警告**：发布时必须上传第 3 步生成的 zip。重新打包会得到不同的校验和，SPM 会拒绝下载。

## 发布

> **警告**：npm、Maven Central 和 pub.dev 的发布不能撤回。发布前，确认版本号正确。

1. 创建 tag，然后推送：

   ```sh
   git tag vX.Y.Z
   git push origin vX.Y.Z
   ```

2. 创建 GitHub Release，上传 zip：

   ```sh
   gh release create vX.Y.Z dist/vX.Y.Z/OpenhongguoCore.xcframework.zip --title vX.Y.Z --notes "…"
   ```

   发布 Release 后，`release` workflow 自动构建 openhongguo-cli（命令 `ohg`）的 macOS、Linux 和 Windows 二进制，连同校验和文件上传到这个 Release。给已有的 Release 补上二进制：

   ```sh
   gh workflow run release.yml -f tag=vX.Y.Z
   ```

3. 发布到 Maven Central。`~/.gradle/gradle.properties` 中必须有 `mavenCentralUsername`、`mavenCentralPassword`、`signingInMemoryKey` 和 `signingInMemoryKeyPassword`：

   ```sh
   (cd bindings/mobile/android && ./gradlew publishToMavenCentral)
   ```

   然后在 [Central Portal](https://central.sonatype.com/publishing) 中确认并发布这次部署。

   > **注意**：用 JDK 17 或 21 运行 Gradle。AGP 内置的 Dokka 不能识别 JDK 25 的版本号，生成 javadoc 时失败。Android Studio 自带的 JBR 是 JDK 25。

4. 发布到 npm：

   ```sh
   (cd bindings/node && npm run build && npm publish)
   (cd bindings/react-native && npm publish)
   ```

5. 发布到 pub.dev：

   ```sh
   (cd bindings/flutter && flutter pub publish)
   ```

6. 让 proxy.golang.org 收录新版本：

   ```sh
   GOPROXY=https://proxy.golang.org go list -m github.com/beilunyang/openhongguo@vX.Y.Z
   ```

## 检查

| 平台 | 检查方法 |
|---|---|
| Go | `go get github.com/beilunyang/openhongguo@vX.Y.Z` |
| Swift | 在新的 Swift 包中加入依赖，运行 `swift build` |
| Android | 在 Maven Central 搜索 `io.github.beilunyang:openhongguo`。新版本需要 10～30 分钟才能出现 |
| npm | `npm view openhongguo version` |
| pub.dev | 打开 `https://pub.dev/packages/openhongguo` |
