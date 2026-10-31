# 类型参考

<!-- 由 go run ./bindings/docgen 生成，请修改 sdk 的 Go 注释，不要手工修改。 -->

各语言 SDK 的返回值和查询参数都用下面这些类型，类型名相同。只有 Flutter 例外：分页结果 `Page` 叫 `PagedResult`，流 `Stream` 叫 `VideoStream`。

字段名有两种写法。Go 的 JSON、Node.js 和 React Native 用下划线风格（`cover_url`）；Kotlin、Swift 和 Dart 用驼峰（`coverUrl`）。

字段缺省时取零值（空字符串、0、空列表）。类型标有“或 null”的字段例外，它们在 Go 里是指针，可以为空。

## 返回值

### Album

Album 是多季的合集，来自 /novel/player/album_detail/v1/。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `id` | `id` | string |  |
| `title` | `title` | string |  |
| `description` | `description` | string |  |
| `cover_url` | `coverUrl` | string |  |
| `cover_expires_at` | `coverExpiresAt` | 时间（RFC 3339 字符串） |  |
| `season_count` | `seasonCount` | int |  |
| `seasons` | `seasons` | [Season](#season) 列表 |  |

### CategoryPage

CategoryPage 是一个分类落地页及其筛选项。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `category_id` | `categoryId` | string |  |
| `category_name` | `categoryName` | string |  |
| `description` | `description` | string |  |
| `filters` | `filters` | [FilterGroup](#filtergroup) 列表 |  |
| `tabs` | `tabs` | [FilterOption](#filteroption) 列表 | Tabs 是受众标签页（全部、男生、女生），每个 Value 是标签页类型编号；DefaultTab 是页面打开时所在的标签页类型。 |
| `default_tab` | `defaultTab` | int |  |
| `items` | `items` | [SeriesSummary](#seriessummary) 列表 |  |
| `has_more` | `hasMore` | bool |  |
| `next_offset` | `nextOffset` | int | NextOffset 是下一页的偏移量，即 LandingOptions.Cursor 的值。 |

### ContentKey

ContentKey 是受保护播放模型的 CENC（cenc-aes-ctr）内容密钥，由 sdk.ContentKey 还原。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `key` | `key` | string | Hex 是 16 字节 AES 密钥的小写十六进制形式，即 ffmpeg 的 -decryption_key 和播放器的 ClearKey 配置接受的形式。 |
| `version` | `version` | string | Version 是播放器在密钥旁读取的方案标记。 |

### ContentType

ContentType 是视频的 content_type，取自 com.dragon.read.rpc.model.VideoContentType。官方 App 的枚举混用小数值和从 1000 起的一段数值；这里两者都复现。

取值是 int，已知的取值如下。服务端新增的取值也会原样保留。

| 常量 | 值 | 说明 |
|---|---|---|
| `ContentUnknown` | `0` |  |
| `ContentShortSeries` | `1` | 真人短剧 |
| `ContentScenePlay` | `2` | 场景剧 |
| `ContentExpertRecommend` | `3` | 专家荐书 |
| `ContentPPTRecommend` | `4` | PPT 荐书 |
| `ContentMovie` | `5` | 电影 |
| `ContentTelePlay` | `6` | 电视剧 |
| `ContentShortPlay` | `7` | 短剧 |
| `ContentDropMaterial` | `1000` | 投放素材 |
| `ContentPUGC` | `1001` | PUGC |
| `ContentUnrealShortPlay` | `1002` | 虚拍短剧 |
| `ContentAlbum` | `1003` | 合集 |
| `ContentMotionComic` | `1004` | 漫剧（动态漫） |

### Device

Device 是匿名身份出示的手机。注册请求体、每个请求的设备参数和 User-Agent 都描述它，所以一个设备 id 始终对应一个机型。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `brand` | `brand` | string | Brand 是 device_brand，也是制造商（"Xiaomi"）；Model 是机型代号，作为 device_type、device_model 发送，也写进 User-Agent（"23127PN0CC"）。 |
| `model` | `model` | string |  |
| `os_version` | `osVersion` | string | OSVersion 是 Android 版本（"14"），OSAPI 是其 API 级别（34），Build 是 User-Agent 携带的构建 id（"UKQ1.230804.001"）。 |
| `os_api` | `osApi` | int |  |
| `build` | `build` | string |  |
| `resolution` | `resolution` | string | Resolution 是 "height*width"，单位是像素；DPI 是屏幕密度，单位是每英寸点数。 |
| `dpi` | `dpi` | int |  |
| `pad` | `pad` | bool | Pad 让设备以平板身份出现（dragon_device_type=pad）。 |
| `channel` | `channel` | string | Channel 是 App 构建上报的分发渠道（"xiaomi_8662_64" 来自小米应用商店）。 |

### Episode

Episode 是一个公开列出的分集。Number 对调用方从 1 开始。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `id` | `id` | string |  |
| `series_id` | `seriesId` | string |  |
| `number` | `number` | int |  |
| `title` | `title` | string |  |
| `playable` | `playable` | bool | 服务端禁止播放该分集时，Playable 为 false（video_list[].disable_play）。 |
| `vertical` | `vertical` | bool | Vertical 标记竖屏画幅的分集。 |
| `is_new` | `isNew` | bool | IsNew 标记本次更新中发布的分集。 |
| `cover_url` | `coverUrl` | string | CoverURL 是分集剧照；CoverExpiresAt 是它的签名过期时间。 |
| `cover_expires_at` | `coverExpiresAt` | 时间（RFC 3339 字符串） |  |
| `content_type` | `contentType` | [ContentType](#contenttype) | ContentType 与 Series.ContentType 是同一枚举。 |
| `comment_count` | `commentCount` | int64 | CommentCount 和 DiggCount 是分集的计数。 |
| `digg_count` | `diggCount` | int64 |  |

### FilterGroup

FilterGroup 是一个筛选维度（状态、排序、子分类、篇幅）。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `key` | `key` | string | Key 是该组设置的查询参数，例如 sort_by。 |
| `options` | `options` | [FilterOption](#filteroption) 列表 |  |

### FilterOption

FilterOption 是分类页上的一个可选值。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `name` | `name` | string | Name 是显示的标签；Value 是查询选中它时发送的值。 |
| `value` | `value` | string |  |

### FilterRow

FilterRow 是短剧筛选页的一个选择行。Key 是该行的 select_items 键（genre、category_dim_theme、category_dim_role、category_dim_epoch、sort、gender、online_time、creation_status）；选项的 Value 是查询所选的值。有哪些行取决于 genre。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `key` | `key` | string |  |
| `name` | `name` | string |  |
| `options` | `options` | [FilterOption](#filteroption) 列表 |  |

### HotBoard

HotBoard 是一个热榜。作品榜（短剧热搜榜、漫剧热搜榜、短剧新剧榜）填充 Items；热点话题榜填充 Topics。出现哪些榜单因调用而异。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `name` | `name` | string |  |
| `items` | `items` | [HotItem](#hotitem) 列表 |  |
| `topics` | `topics` | [HotTopic](#hottopic) 列表 |  |

### HotItem

HotItem 是热榜上的一部作品，按榜单顺序排列。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `position` | `position` | int | Position 是在榜单上的名次，从 1 开始。 |
| `series_id` | `seriesId` | string |  |
| `title` | `title` | string |  |
| `cover_url` | `coverUrl` | string |  |
| `cover_expires_at` | `coverExpiresAt` | 时间（RFC 3339 字符串） |  |
| `episode_count` | `episodeCount` | int |  |
| `content_type` | `contentType` | [ContentType](#contenttype) |  |
| `search_score` | `searchScore` | string | SearchScore 是榜单自己的数值（"355万热搜值"，有时带 "·第1季"）；Heat 是作品的总热度（"4102万热度"）。 |
| `heat` | `heat` | string |  |
| `tag` | `tag` | string | Tag 是角标，例如新剧。 |

### HotSearch

HotSearch 是查询前的搜索页：猜你想搜一行，以及其下的热榜。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `guesses` | `guesses` | [HotItem](#hotitem) 列表 |  |
| `boards` | `boards` | [HotBoard](#hotboard) 列表 |  |

### HotTopic

HotTopic 是热点话题榜的一项。Query 是它打开的搜索。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `id` | `id` | string |  |
| `title` | `title` | string |  |
| `query` | `query` | string |  |
| `heat` | `heat` | string |  |

### InfoKind

InfoKind 是 secondary_infos[].data_type，取自 com.dragon.read.rpc.model.SecondaryInfoDataType。

取值是 int，已知的取值如下。服务端新增的取值也会原样保留。

| 常量 | 值 | 说明 |
|---|---|---|
| `InfoSeasonOrder` | `0` | InfoSeasonOrder 是 "第 N 季" 那一行（实测值为 0）。 |
| `InfoRecommendReason` | `1` | InfoRecommendReason 是推荐理由。 |
| `InfoRankScore` | `2` | InfoRankScore 是榜单名次。 |
| `InfoCategory` | `3` | InfoCategory 是分类标签，例如 "末世"（实测值为 3）。 |
| `InfoScore` | `4` | InfoScore 是用户评分。 |
| `InfoAuthorName` | `8` | InfoAuthorName 是作者名。 |
| `InfoTags` | `9` | InfoTags 是标签列表。 |
| `InfoCreationStatus` | `11` | InfoCreationStatus 是连载中或已完结的说明。 |
| `InfoTwoActors` | `33` | InfoTwoActors 是演员行 "演员·A B"（实测值为 33）。 |
| `InfoPlotTag` | `40` | InfoPlotTag 是剧情标签。 |

### InfoRow

InfoRow 是 secondary_infos 的一项：详情页上带标签的一行。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `kind` | `kind` | [InfoKind](#infokind) |  |
| `text` | `text` | string |  |
| `schema` | `schema` | string | Schema 是该行打开的应用内链接（如果有）。 |

### NewRelease

NewRelease 是上新日历中的一部作品：已上线（Online），或已预告在 PublishAt 上线。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `series_id` | `seriesId` | string |  |
| `title` | `title` | string |  |
| `description` | `description` | string |  |
| `cover_url` | `coverUrl` | string |  |
| `cover_expires_at` | `coverExpiresAt` | 时间（RFC 3339 字符串） |  |
| `episode_count` | `episodeCount` | int |  |
| `online` | `online` | bool | 作品已上线时 Online 为 true；PublishAt 是预告作品的上线时间。 |
| `publish_at` | `publishAt` | 时间（RFC 3339 字符串） |  |
| `categories` | `categories` | string 列表 |  |
| `heat` | `heat` | string | 已上线的作品 Heat 形如 "378万热度"，预告作品形如 "159.7万人预约"；Note 是预告文案（"预计10月26日上线"）。 |
| `note` | `note` | string |  |

### NewReleasePage

NewReleasePage 是上新日历中的一天。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `date` | `date` | string | Date 是本次返回的日期（YYYYMMDD）；Dates 是日历提供的日期，即今天前后各一周。 |
| `dates` | `dates` | string 列表 |  |
| `items` | `items` | [NewRelease](#newrelease) 列表 |  |
| `has_more` | `hasMore` | bool |  |
| `cursor` | `cursor` | string |  |

### Novel

Novel 是短剧改编自的小说。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `book_id` | `bookId` | string |  |
| `title` | `title` | string | Title 是小说本身的名称；ShortTitle 是它与短剧共用的名称。 |
| `short_title` | `shortTitle` | string |  |
| `status` | `status` | [NovelStatus](#novelstatus) |  |
| `tags` | `tags` | string 列表 |  |
| `reads` | `reads` | int64 |  |
| `cover_url` | `coverUrl` | string |  |
| `cover_expires_at` | `coverExpiresAt` | 时间（RFC 3339 字符串） |  |
| `show_at_episode` | `showAtEpisode` | int | ShowAtEpisode 是播放器推荐该小说时所在的集数。 |

### NovelStatus

NovelStatus 是客户端的 NovelCreationStatus；未给出时为 -1。

取值是 int，已知的取值如下。服务端新增的取值也会原样保留。

| 常量 | 值 | 说明 |
|---|---|---|
| `NovelFinished` | `0` | 已完结 |
| `NovelOngoing` | `1` | 连载中 |
| `NovelUpdatedToday` | `3` | 今日更新 |

### PayType

PayType 是 pay_info.pay_type，取自 com.dragon.read.rpc.model.VideoPayType。

取值是 int，已知的取值如下。服务端新增的取值也会原样保留。

| 常量 | 值 | 说明 |
|---|---|---|
| `PayFree` | `0` | PayFree 表示无需账号即可观看。 |
| `PayVip` | `1` | PayVip 表示需要订阅。 |
| `PaySingle` | `2` | PaySingle 表示需要单次购买。 |

### Person

Person 是演员或角色的主页。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `user_id` | `userId` | string |  |
| `name` | `name` | string |  |
| `description` | `description` | string |  |
| `avatar_url` | `avatarUrl` | string |  |
| `tags` | `tags` | string 列表 | Tags 是主页上的头衔（"女演员"、"虚拟人物"）；Labels 是名字下方的小徽标（"水瓶座"）。 |
| `labels` | `labels` | string 列表 |  |
| `followers` | `followers` | int64 | Followers 是粉丝数；Likes 是获赞数。 |
| `likes` | `likes` | int64 |  |

### PlatformType

PlatformType 是 video_platform，取自 com.dragon.read.rpc.model.VideoPlatformType。

取值是 int，已知的取值如下。服务端新增的取值也会原样保留。

| 常量 | 值 | 说明 |
|---|---|---|
| `PlatformUnknown` | `0` | 未知；有声书带此值 |
| `PlatformDouyin` | `1` | 抖音 |
| `PlatformUGCVideo` | `2` | UGC 视频 |
| `PlatformOuter` | `3` | 站外与自制短剧 |
| `PlatformXigua` | `4` | 西瓜视频，电影来自这里 |
| `PlatformChameleon` | `5` | Chameleon |
| `PlatformMixed` | `1024` | 混合来源 |

### Playback

Playback 是一个已解析的匿名媒体模型。它描述服务端提供的内容，从不携带解密密钥或许可证材料。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `episode_id` | `episodeId` | string |  |
| `video_id` | `videoId` | string | VideoID 是分集背后的媒体 id。 |
| `media_type` | `mediaType` | string | MediaType 是服务端的媒体类型，例如 "video"。 |
| `duration_seconds` | `durationSeconds` | int |  |
| `expires_at` | `expiresAt` | int64 | ExpiresAt 是流 URL 失效的时间，单位是 unix 秒。 |
| `protected` | `protected` | bool | Protected 标记加密的分集；参见 Stream.Protected。 |
| `streams` | `streams` | [Stream](#stream) 列表 |  |
| `audio_url` | `audioUrl` | string | AudioURL 是分段（DASH）作品（如长片电影）的独立音轨。单文件的表示形式留空。 |

### RankBoard

RankBoard 是某个榜单内容下的一个榜（推荐榜、热播榜……）。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `id` | `id` | string |  |
| `name` | `name` | string |  |
| `selected` | `selected` | bool |  |

### RankChange

RankChange 是榜单行的变动（客户端中的 RankChangeType）。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `kind` | `kind` | [RankChangeKind](#rankchangekind) |  |
| `places` | `places` | int | Places 是该行变动的位数；新上榜或不变的行为 0。 |

### RankChangeKind

RankChangeKind 的取值即客户端的 RankChangeType。

取值是 int，已知的取值如下。服务端新增的取值也会原样保留。

| 常量 | 值 | 说明 |
|---|---|---|
| `RankNew` | `0` | 新上榜 |
| `RankUp` | `1` | 上升 RankChange.Places 位 |
| `RankDown` | `2` | 下降 RankChange.Places 位 |
| `RankUnchanged` | `3` | 名次不变 |

### RankContent

RankContent 是短剧榜单页的一个内容（全部、真人剧、漫剧、AI剧、系列剧、演员），带有自己的各个榜。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `id` | `id` | string |  |
| `name` | `name` | string |  |
| `selected` | `selected` | bool |  |
| `boards` | `boards` | [RankBoard](#rankboard) 列表 |  |

### RankItem

RankItem 是榜单的一行。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `rank` | `rank` | int | Rank 是服务端给该行的名次（参见 Position）。 |
| `votes` | `votes` | int64 | Votes 和 Kind 用于活动榜单行：分别是票数和服务端的条目类型，原样透传。 |
| `kind` | `kind` | int |  |
| `actor` | `actor` | string | Actor 在书籍榜单上是作者，在活动榜单或短剧演员榜上是演员名。 |
| `series_id` | `seriesId` | string |  |
| `title` | `title` | string |  |
| `cover_url` | `coverUrl` | string |  |
| `score` | `score` | string | Score 是显示的热度（"2550万热度"、"998万推荐"、"平台推荐"）。 |
| `episode_count` | `episodeCount` | int | EpisodeCount 由短剧榜单填充。 |
| `position` | `position` | int | Position 是该行在短剧榜单页上的名次，从 1 开始。Rank 是服务端自己的编号，在推广行前后会重复，在推荐榜上是个性化的。 |
| `promoted` | `promoted` | bool | Promoted 标记服务端插入短剧榜单的行：它显示 "平台推荐" 而非热度，名次照抄下一行（实测）。 |
| `actor_id` | `actorId` | string | ActorID、AvatarURL、Intro 和 Details 用于短剧榜单的演员行，该行的 Actor 是名字，Score 是人气。Intro 列出作品；Details 存放 "67部作品" 这样的行。 |
| `user_id` | `userId` | string | UserID 是演员的账号，即 ActorProfile 和 ActorWorks 接受的 id。 |
| `avatar_url` | `avatarUrl` | string |  |
| `intro` | `intro` | string |  |
| `details` | `details` | string 列表 |  |
| `change` | `change` | [RankChange](#rankchange) 或 null | Change 是该行相对上一期榜单的变动（如果有）。 |

### RankList

RankList 是活动的一个榜单频道。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `id` | `id` | string |  |
| `name` | `name` | string |  |
| `type` | `type` | int | Type 和 Size 是服务端的榜单类型和长度，原样透传。 |
| `size` | `size` | int |  |

### Ranking

Ranking 是一个活动榜单页。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `list_name` | `listName` | string |  |
| `lists` | `lists` | [RankList](#ranklist) 列表 | Lists 是该活动的其他榜单频道。 |
| `items` | `items` | [RankItem](#rankitem) 列表 |  |
| `has_more` | `hasMore` | bool |  |
| `next_offset` | `nextOffset` | int | NextOffset 用于续取列表（RankRows 把它当作 offset）；SessionID 是服务端上报的分页会话。 |
| `activity_stage` | `activityStage` | int | ActivityStage 和 CanVote 是服务端上报的活动状态。 |
| `can_vote` | `canVote` | bool |  |
| `session_id` | `sessionId` | string |  |

### Recommend

Recommend 是一部短剧的相关视频单元：包含各行、单元标题，以及客户端为后续请求保留的分页会话。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `cell_title` | `cellTitle` | string |  |
| `session_id` | `sessionId` | string |  |
| `items` | `items` | [SeriesSummary](#seriessummary) 列表 |  |

### Season

Season 是合集（多季合集）中的一项。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `series_id` | `seriesId` | string |  |
| `title` | `title` | string |  |
| `index` | `index` | int | Index 是季序号，从 1 开始。 |
| `cover_url` | `coverUrl` | string |  |
| `episode_count` | `episodeCount` | int |  |
| `status` | `status` | [SeriesStatus](#seriesstatus) |  |

### Series

Series 是一部短剧的详情视图。

包含 [SeriesSummary](#seriessummary) 的全部字段。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `status` | `status` | [SeriesStatus](#seriesstatus) | Status 是发布状态；PayType 是付费墙状态。 |
| `pay_type` | `payType` | [PayType](#paytype) |  |
| `content_type` | `contentType` | [ContentType](#contenttype) | ContentType 和 Platform 描述作品是什么、来自哪里（短剧、电影、漫剧……）。 |
| `platform` | `platform` | [PlatformType](#platformtype) |  |
| `heat` | `heat` | int64 | Heat 是原始热度分；App 将其显示为 "红果热度值N万"。 |
| `play_count` | `playCount` | int64 | PlayCount 是短剧的总播放量（series_play_cnt）。 |
| `follow_count` | `followCount` | int64 | FollowCount 和 DiggCount 是标题旁显示的计数。 |
| `digg_count` | `diggCount` | int64 |  |
| `created_at` | `createdAt` | 时间（RFC 3339 字符串） | CreatedAt 是发布时间（create_time，unix 秒）。 |
| `right_text` | `rightText` | string | RightText 是集数旁的角标（"更新至12集"）。 |
| `subtitles` | `subtitles` | string 列表 | Subtitles 是标题下方的简短状态行。 |
| `age_gate` | `ageGate` | int | AgeGate 对不受限的作品为 0。 |
| `is_multi_season` | `isMultiSeason` | bool | IsMultiSeason 标记属于多季合集的作品。 |
| `season_index` | `seasonIndex` | int | SeasonIndex 是该合集内的季序号，从 1 开始。 |
| `related_album_id` | `relatedAlbumId` | string | RelatedAlbumID 是各季共用的合集 id。 |
| `related_novel_book_id` | `relatedNovelBookId` | string | RelatedNovelBookID 是短剧改编的小说（如已公开）。 |
| `novel` | `novel` | [Novel](#novel) 或 null | Novel 是详情页链接的原著小说（video_relate_book），服务端给出时才有。 |
| `rows` | `rows` | [InfoRow](#inforow) 列表 | Rows 是详情页的次要信息行（季、分类、演员）。 |

### SeriesDetail

SeriesDetail 是一部短剧及其分集，由批量读取返回。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `series` | `series` | [Series](#series) 或 null |  |
| `episodes` | `episodes` | [Episode](#episode) 列表 |  |

### SeriesStatus

SeriesStatus 是短剧的发布状态，由 data.video_data.series_status 携带。

名称取自官方 App 的枚举 com.dragon.read.rpc.model.SeriesStatus（SeriesUpdating、SeriesEnd、SeriesUpdateTaday、SeriesUpdateStop）；值 0 和 1 也已在线上实测。

取值是 int，已知的取值如下。服务端新增的取值也会原样保留。

| 常量 | 值 | 说明 |
|---|---|---|
| `SeriesUpdating` | `0` | SeriesUpdating 是未完结的短剧（"更新至 N 集"）。 |
| `SeriesEnd` | `1` | SeriesEnd 是已完结的短剧（"全 N 集"）。 |
| `SeriesUpdatedToday` | `3` | SeriesUpdatedToday 是枚举中的“今日更新”状态（推断）。 |
| `SeriesUpdateStopped` | `4` | SeriesUpdateStopped 是枚举中的“停止更新”状态（推断）。 |

### SeriesSummary

SeriesSummary 是公开目录中的一行。ID 和标题是稳定字段。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `id` | `id` | string |  |
| `title` | `title` | string |  |
| `description` | `description` | string |  |
| `cover_url` | `coverUrl` | string |  |
| `cover_expires_at` | `coverExpiresAt` | 时间（RFC 3339 字符串） | CoverExpiresAt 是 CoverURL 的签名过期时间（x-expires）；过期后要从 API 重新获取封面。 |
| `episode_count` | `episodeCount` | int |  |
| `author` | `author` | string | Author、Category 和 Words 由分类页的行填充。 |
| `category` | `category` | string |  |
| `words` | `words` | int64 |  |

### ShortFilterPage

ShortFilterPage 是符合筛选条件的短剧的一页。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `items` | `items` | [SeriesSummary](#seriessummary) 列表 |  |
| `has_more` | `hasMore` | bool |  |
| `cursor` | `cursor` | string | Cursor 用于续取列表；它携带已展示的短剧，服务端会跳过它们。 |

### ShortRanking

ShortRanking 是短剧榜单的一页。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `contents` | `contents` | [RankContent](#rankcontent) 列表 | Contents 是该页的目录：每个内容及其各个榜。 |
| `content` | `content` | string | Content 和 Board 是本页对应的 ID。 |
| `board` | `board` | string |  |
| `version` | `version` | string | Version 是服务端上报的榜单版本。 |
| `items` | `items` | [RankItem](#rankitem) 列表 |  |
| `has_more` | `hasMore` | bool |  |
| `cursor` | `cursor` | string |  |

### Stream

Stream 是媒体模型中的一个表示形式。Protected 标记官方客户端在内部解密的表示形式；sdk.ContentKey 从模型的 KeyMaterial 还原它的密钥。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `definition` | `definition` | string | Definition 是分辨率标签（"720p"）；Quality 是服务端的画质标签（"normal"）。 |
| `quality` | `quality` | string |  |
| `codec` | `codec` | string |  |
| `width` | `width` | int |  |
| `height` | `height` | int |  |
| `bitrate` | `bitrate` | int64 | Bitrate 的单位是 bit/s；Size 是文件大小，单位是字节。 |
| `size` | `size` | int64 |  |
| `url` | `url` | string | BackupURL 是同一文件的第二个 CDN 地址。 |
| `backup_url` | `backupUrl` | string |  |
| `hls` | `hls` | bool | HLS 标记 m3u8 播放列表；Dash 标记分段（DASH）的表示形式。 |
| `dash` | `dash` | bool |  |
| `protected` | `protected` | bool |  |
| `encryption` | `encryption` | string | Encryption 是受保护流的加密方案，例如 "cenc-aes-ctr"。 |

### Suggestion

Suggestion 是一条搜索建议。Text 是搜索框显示的文本；Keyword 是它提交的内容（直达行为内容 ID）。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `text` | `text` | string |  |
| `keyword` | `keyword` | string |  |
| `kind` | `kind` | int | Kind 是服务端的行类型，原样透传。 |

### Work

Work 是某人出演的一部短剧。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `series_id` | `seriesId` | string |  |
| `title` | `title` | string |  |
| `description` | `description` | string |  |
| `cover_url` | `coverUrl` | string |  |
| `cover_expires_at` | `coverExpiresAt` | 时间（RFC 3339 字符串） |  |
| `episode_count` | `episodeCount` | int |  |
| `tags` | `tags` | string 列表 | Tags 是卡片上的各行，例如 "爱情" 和 "70集"。 |
| `heat` | `heat` | int64 | Heat 是短剧的原始热度分；Plays 是其播放量。 |
| `plays` | `plays` | int64 |  |

### WorkPage

WorkPage 是某人作品的一页（作品 › 剧作）。

| 字段 | 驼峰 | 类型 | 说明 |
|---|---|---|---|
| `items` | `items` | [Work](#work) 列表 |  |
| `has_more` | `hasMore` | bool |  |
| `cursor` | `cursor` | string |  |

## 查询参数

查询参数的字段全部可选，不填表示不传。字段名在各语言里都是驼峰。

### DiscoverOptions

DiscoverOptions 选择一个发现页。Channel 是书城频道名，空频道即推荐。Cursor 是上一页给出的令牌。PageSize 是 cell/change 的 limit（0 表示 20，上限 50）。

| 字段 | 类型 | 说明 |
|---|---|---|
| `cursor` | string |  |
| `channel` | string |  |
| `pageSize` | int |  |

### SearchOptions

SearchOptions 选择一个搜索页。Tab 取值为综合、剧集、视频、讨论、小说、用户或听书，空 tab 即综合。Cursor 是下一个偏移量。

| 字段 | 类型 | 说明 |
|---|---|---|
| `cursor` | string |  |
| `pageSize` | int |  |
| `tab` | string |  |
| `filters` | string 列表 | Filters 把综合 tab 收窄到 SearchFilter* 值；选多个时互相放宽。游标不携带这些筛选，所以要连同游标传入相同的筛选。 |

### EpisodeOptions

EpisodeOptions 选择一个分集页。游标不在第 1 阶段的范围内。

| 字段 | 类型 | 说明 |
|---|---|---|
| `cursor` | string |  |
| `pageSize` | int |  |

### LandingOptions

LandingOptions 选择一个分类落地页。

Tab 是 Categories 给出的 tab 值：8（全部，默认）、1（男生）或 0（女生）。Filters 把 CategoryPage.Filters 中的筛选组键（book_status、sort_by、sub_category、word_type）映射到该组的某个选项值。Cursor 是上一页的 NextOffset。

| 字段 | 类型 | 说明 |
|---|---|---|
| `tab` | string |  |
| `filters` | string 到 string 的映射 |  |
| `cursor` | string |  |

### NewReleaseQuery

NewReleaseQuery 选择上新日历中的一天：Date 格式为 YYYYMMDD（空即今天），Cursor 是上一页的 Cursor，它带有所在的日期。

| 字段 | 类型 | 说明 |
|---|---|---|
| `date` | string |  |
| `cursor` | string |  |

### ShortFilterQuery

ShortFilterQuery 在筛选页（筛选）上挑选短剧。

Selections 把 FilterRow.Key 映射到选中的选项 Values；缺省的行即全部。Cursor 是上一页的 Cursor。

| 字段 | 类型 | 说明 |
|---|---|---|
| `selections` | string 到 string 列表 的映射 |  |
| `cursor` | string |  |

### ShortRankQuery

ShortRankQuery 选择一个短剧榜单。Content 是 RankContent ID（all、human、comic_series_rank、ai_playlet、series_album；空即 all），Board 是它的某个 RankBoard ID（空即服务端对该 content 的默认值）。Cursor 是上一页的 Cursor。

| 字段 | 类型 | 说明 |
|---|---|---|
| `content` | string |  |
| `board` | string |  |
| `cursor` | string |  |
