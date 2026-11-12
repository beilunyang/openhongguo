# openhongguo-cli

openhongguo-cli 是 openhongguo 的命令行工具，命令名是 `ohg`。它可以搜索、浏览、播放和下载红果短剧。

所有命令都使用匿名设备。命令不需要登录。

> **免责声明**：本仓库的所有内容仅供学习和参考之用，禁止用于商业用途。使用前请阅读[完整的免责声明](../README.md#免责声明)。使用本仓库的内容即表示您同意其所有条款和条件。

术语的含义见[根目录 README 的术语表](../README.md#术语)。

## 安装

如果要安装 `ohg`，执行以下步骤：

1. 取得 `ohg`。任选一种方式：

   | 方式 | 说明 |
   |---|---|
   | 下载二进制 | 从 [GitHub Releases](https://github.com/beilunyang/openhongguo/releases) 下载对应平台的 `openhongguo-cli_*` 压缩包。macOS 上解压后执行一次 `xattr -d com.apple.quarantine ohg` |
   | `go install` | `go install github.com/beilunyang/openhongguo/cmd/ohg@latest`。需要 Go 1.22 或更高版本 |
   | 从源码构建 | `go build -o bin/ohg ./cmd/ohg`。需要 Go 1.22 或更高版本 |

2. 安装外部程序。下表列出各用途需要的程序。

   | 用途 | 需要的程序 |
   |---|---|
   | 播放 | `mpv`（推荐）、`ffplay` 或 `vlc` 中的一个。在 macOS 上，如果三个都没有，`ohg` 使用系统的 `open` |
   | 解密加密分集 | `mpv` 和 `ffplay` 可以自己解密。用 `vlc` 或 `open` 播放时，以及下载时，需要 `ffmpeg` |
   | HTTPS | 系统的 CA 证书。精简版 Linux 镜像需要安装 `ca-certificates` |

   在 macOS 上，用 Homebrew 安装：

   ```sh
   brew install mpv ffmpeg
   ```

   在 Debian 和 Ubuntu 上，用 apt 安装：

   ```sh
   apt install mpv ffmpeg ca-certificates
   ```

## 用法

下文假设 `ohg` 在 `PATH` 中。完整的参数见 `ohg --help`。

### 搜索与播放

| 命令 | 作用 |
|---|---|
| `ohg search 全球死考` | 搜索短剧。输出短剧 ID、剧名和集数 |
| `ohg play 全球死考第一季` | 按剧名播放第 1 集。使用最匹配的结果 |
| `ohg play 全球死考 --pick` | 列出搜索结果。选择一部短剧后播放 |
| `ohg play 7675400266065136665 --episode 5 --autoplay` | 从第 5 集开始连续播放 |
| `ohg continue` | 续播最近看的短剧 |
| `ohg download 全球死考第一季 --episodes 1-20 --jobs 4` | 用 4 个并发下载第 1～20 集 |
| `ohg history` | 显示观看记录 |

播放时，`ohg` 不把分集写入磁盘，一边下载一边播放。各播放器的行为如下：

| 播放器 | 解密方式 | 拖动进度 |
|---|---|---|
| `mpv`、`ffplay` | 播放器自己解密 | 可以 |
| `vlc` | 读取 `ffmpeg` 解密后的数据流 | 不可以 |
| `open`、`xdg-open` | 先把分集解密到临时文件。播放结束后删除临时文件 | 取决于系统的播放器 |

`--autoplay` 在一集播放结束后播放下一集。`ohg` 提前取得下一集的地址。

以下任一情况发生时，连续播放停止：

- 最后一集播放结束。
- 你关闭播放器，而分集没有播放结束。
- 你按 Ctrl-C。
- 连续 3 集不能用匿名设备播放。

如果一集不能用匿名设备播放，`ohg` 跳过这一集。

在 mpv 中，按 `>` 播放下一集，按 `<` 播放上一集。

如果主 CDN 地址播放失败，`ohg` 使用备用地址再试一次。

`ohg` 用以下方法判断一集是否播放结束：

| 播放器 | 判断方法 |
|---|---|
| mpv | 脚本报告结束原因 |
| ffplay | 最后的播放位置是否接近结尾 |
| 其他播放器 | 播放器退出即为播放结束 |

### 观看记录与续播

每次播放，`ohg` 保存一条记录：剧名、集数、是否播放结束、停止的位置（秒）。记录保存在配置目录的 `history.json` 中，文件权限是 0600。

| 命令或参数 | 作用 |
|---|---|
| `play <短剧> --resume` | 如果上次的分集没有播放结束，从停止位置前 3 秒继续播放。如果已经播放结束，播放下一集 |
| `continue [play 的参数]` | 续播最近看的短剧。与 `play <最近的短剧> --resume` 相同 |

> **注意**：使用 `--url`、`--json`、`--download` 或 `--preview` 时，`ohg` 不保存记录。

### 下载

`download` 把每一集保存为 `<目录>/第NNN集.mp4`。目录名的默认值是剧名。

下载时，`ohg` 先写入 `.part` 文件。下载完成后，`ohg` 把文件改名为 `.mp4`。所以目录中的 `.mp4` 文件都是完整的。

`ohg` 跳过已经存在的分集。如果下载中断，再次运行同一条命令，下载从中断处继续。

如果只下载一集，也可以用 `play <短剧> --episode N --download PATH`。

### 弹幕

| 命令 | 作用 |
|---|---|
| `ohg danmaku 全球死考第一季` | 列出第 1 集的弹幕。每行是出现时间和内容，按时间排序 |
| `ohg danmaku 7675400266065136665 --episode 5 --json` | 以 JSON 输出第 5 集的弹幕，带毫秒时间、点赞数和发送时间 |

`danmaku` 翻完一集的全部弹幕再输出，最后在标准错误输出条数。弹幕不带发送者的身份。

> **注意**：服务端只下发一部分弹幕，不同的匿名设备拿到的条数也不一样。服务端给出总数时，`ohg` 一并输出，它通常比列出的条数多得多。

### 浏览与发现

| 命令 | 作用 |
|---|---|
| `ohg discover --channel 电影` | 显示频道首页。默认频道是“推荐” |
| `ohg discover --channel 漫剧 --all` | 按游标翻页，删除跨页重复的短剧 |
| `ohg search 末世 --all --pages 3` | 搜索结果翻页 |
| `ohg search 奇幻 --filter 真人剧,影视剧` | 只显示指定的类型。只用于“综合”分类 |
| `ohg info 7690184933037329432` | 显示短剧的元数据 |
| `ohg info 7675400266065136665 7683373426689461273` | 一次读取多部短剧 |
| `ohg episodes 7690184933037329432` | 显示分集列表。不能播放的分集标为 locked |
| `ohg seasons 7675400266065136665` | 显示多季短剧的所有季 |
| `ohg related 7690184933037329432` | 显示推荐的短剧 |
| `ohg suggest 末世` | 显示搜索联想词 |
| `ohg hot` | 显示热搜 |
| `ohg new` | 显示上新日历 |
| `ohg new --date 20261026` | 显示指定日期的上新 |
| `ohg person 3358342267023107` | 显示演员主页和作品 |
| `ohg cues` | 显示搜索框中的提示词 |

各命令的补充说明如下：

| 命令 | 说明 |
|---|---|
| `search --filter` | 可选的类型：真人剧、漫剧、AI 剧、影视剧、未看过 |
| `info` | 输出类型、状态、角标、热度和标签。如果短剧改编自小说，输出多一行 `novel`。如果给出多个 ID，每 10 部短剧发送一个请求 |
| `seasons` | 列出所有季，包括已经预告但还没有分集的季 |
| `related` | 匿名设备得到通用的“更多好剧”。结果与这部短剧无关 |
| `hot` | 输出“猜你想搜”、短剧热搜榜、漫剧热搜榜和热点话题榜。每个榜只包含页面上显示的条目 |
| `new` | 输出当天已经上线和即将上线的短剧，以及上线时间、热度或预约人数。输出中的 `dates` 是前后 7 天中可以选择的日期。用 `--cursor` 翻页 |
| `person` | ID 来自演员榜的第二列。用 `--cursor` 显示更多作品 |
| `cues` | 搜索框中滚动的提示词，例如“书名 / 主角 / 作者”。它不是热搜 |

### 排行榜与筛选

| 命令 | 作用 |
|---|---|
| `ohg rank --shorts` | 显示“全部”内容的“推荐榜”（默认） |
| `ohg rank --shorts --content human --board human_new_rank` | 显示真人剧的新剧榜 |
| `ohg rank --shorts --cursor <上一页的 cursor>` | 显示下一页 |
| `ohg filter --panel --genre comic_series` | 显示筛选面板 |
| `ohg filter --genre comic_series --filter category_dim_theme=cate_7 --filter sort=hot_score` | 筛选“漫剧 · 玄幻 · 最高热度” |
| `ohg filter --cursor <上一页的 cursor>` | 显示下一页 |

`rank --shorts` 先输出内容类型和榜单目录，然后按页面顺序列出短剧。榜单目录列出每种内容可以选择的榜单。

| 游标 | 包含的信息 |
|---|---|
| `rank --shorts` 的游标 | 内容和榜单 |
| `filter` 的游标 | 筛选条件 |

`filter --panel` 列出每个筛选维度的 KEY 和可选值。维度与体裁有关。

排行榜的名次使用页面顺序。

> **注意**：
>
> - 服务端插入“平台推荐”的推广位，标为 `promoted`。推广位使用下一行的名次。
> - “推荐榜”按个人兴趣排序，所以名次会重复。
> - 短剧榜单翻页时，游标包含已经显示的短剧。服务端删除这些短剧，所以各页之间不重复。

演员榜（`--content ranklist_celebrity`，包括剧星榜和星光榜）列出演员，不列出短剧。每一行包含以下内容：

1. 用户 ID。
2. 名字。
3. 人气或星光值。
4. 作品数等副标题。
5. 名次变化（up、down、same 或 new）。

用户 ID 可以直接传给 `ohg person`。演员榜按 offset 和 session 翻页。

### 小说与听书

下面的命令返回书（小说和听书），不返回短剧。

> **注意**：`play` 和 `download` 不接受书的 ID。

| 命令 | 作用 |
|---|---|
| `ohg category 68` | 显示分类的标签页：全部、男生、女生 |
| `ohg landing 68 --filter sort_by=76 --filter book_status=0` | 按排序方式和完结状态筛选分类中的书 |
| `ohg landing 68 --tab 1 --filter sub_category=515 --filter word_type=1 --cursor 10` | 按频道、子题材和字数筛选，并翻页 |
| `ohg rank` | 显示书城榜单 |
| `ohg rank --groups` | 显示书城榜单的类型 |
| `ohg rank --activity <id> --list <id>` | 显示活动榜单 |

| 命令 | 说明 |
|---|---|
| `landing` | 列出分类中的书。可以按排序方式、完结状态、频道、子题材和字数筛选。用 `--cursor` 翻页 |
| `rank` | 每个请求返回一批不同的书。名次只在当前页中有效 |
| `rank --activity` | 如果活动没有开放，活动榜单没有条目 |

## 配置

用环境变量配置 `ohg`。通常不需要设置这些变量。

| 环境变量 | 作用 |
|---|---|
| `OHG_CONFIG_DIR` | 配置目录。目录中保存设备档案 `anonymous.json` 和观看记录 `history.json` |
| `OHG_CENC_KEY` | 手动指定一集的内容密钥。通常不需要 |
| `OHG_API_ORIGIN` | 读取类接口的地址 |
| `OHG_REGISTER_ORIGIN` | 设备注册接口的地址 |
| `OHG_DISTRIBUTION_ORIGIN` | 短剧筛选（`/reading/distribution/`）和活动榜接口的地址 |

配置目录的默认值是 `<用户配置目录>/openhongguo`。在 macOS 上，它是 `~/Library/Application Support/openhongguo`。

第一次运行时，如果 `ohg` 找到旧名称的目录 `…/hgdrama`，它把目录中的两个文件复制到新目录。

接口地址的默认值与 App 相同：

| 环境变量 | 默认值 | 其他可用的值 |
|---|---|---|
| `OHG_API_ORIGIN` | `api5-normal-sinfonlinea.fqnovel.com` | `https://reading.snssdk.com`。这个地址不提供短剧筛选和活动榜 |
| `OHG_REGISTER_ORIGIN` | `log5-applog-lq.fqnovel.com` | `https://log.snssdk.com` |
| `OHG_DISTRIBUTION_ORIGIN` | 与 `OHG_API_ORIGIN` 的默认值相同 | — |

如果把 `OHG_API_ORIGIN` 改为 `reading.snssdk.com`，短剧筛选和活动榜仍然使用 `OHG_DISTRIBUTION_ORIGIN` 的地址。

`ohg` 自动处理以下失败：

| 失败 | `ohg` 的处理 |
|---|---|
| 限流或网络错误 | 等待一段时间后重试。默认最多尝试 3 次 |
| 服务端要求等待超过 10 秒 | 停止，并报告需要等待的时间 |
| 服务端拒绝匿名设备 | 报告这个情况，然后注册一台新的匿名设备。查询类命令和 `download` 再发送一次请求。`play` 不再发送请求，用 `ohg continue` 继续播放 |

## 匿名设备

| 命令 | 作用 |
|---|---|
| `ohg device show` | 显示当前的匿名设备 |
| `ohg device reset` | 注册一台新的匿名设备 |

匿名设备在以下位置使用同一个手机型号：注册请求、每个请求的设备参数和 User-Agent。

| 项目 | 默认值 |
|---|---|
| 机型 | 小米 14（`23127PN0CC`，Android 14） |
| 渠道 | 小米商店（`xiaomi_8662_64`） |

机型保存在 `anonymous.json` 的 `device` 字段中。所以一个设备 ID 始终对应同一个机型。

## 平台

| 平台 | 测试情况 |
|---|---|
| macOS | 实际测试了 mpv 和 ffplay 播放 |
| Linux | 单元测试通过。实际测试了下载和解密 |
| Windows | 只做了交叉编译。`internal/media` 的测试用 `/bin/sh` 脚本模拟播放器，不能在 Windows 上运行 |
