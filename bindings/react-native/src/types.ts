// 由 go run ./bindings/node/tsgen 生成，请勿手工修改。
// 返回值的字段名与 Go SDK 的 JSON 一致（如 has_more、cover_url）；
// 查询参数用驼峰（如 pageSize）。

/** Page 是一页匿名结果。 */
export interface Page<T> {
  items: T[];
  cursor?: string;
  has_more: boolean;
}

/**
 * Album 是多季的合集，来自 /novel/player/album_detail/v1/。
 */
export interface Album {
  id: string;
  title?: string;
  description?: string;
  cover_url?: string;
  cover_expires_at?: string;
  season_count?: number;
  seasons?: Season[];
}

/**
 * CategoryPage 是一个分类落地页及其筛选项。
 */
export interface CategoryPage {
  category_id?: string;
  category_name?: string;
  description?: string;
  filters?: FilterGroup[];
  /**
   * Tabs 是受众标签页（全部、男生、女生），每个 Value 是标签页类型编号；
   * DefaultTab 是页面打开时所在的标签页类型。
   */
  tabs?: FilterOption[];
  default_tab?: number;
  items: SeriesSummary[];
  has_more: boolean;
  /**
   * NextOffset 是下一页的偏移量，即 LandingOptions.Cursor 的值。
   */
  next_offset?: number;
}

/**
 * ContentKey 是受保护播放模型的 CENC（cenc-aes-ctr）内容密钥，
 * 由 sdk.ContentKey 还原。
 */
export interface ContentKey {
  /**
   * Hex 是 16 字节 AES 密钥的小写十六进制形式，
   * 即 ffmpeg 的 -decryption_key 和播放器的 ClearKey 配置接受的形式。
   */
  key: string;
  /**
   * Version 是播放器在密钥旁读取的方案标记。
   */
  version?: string;
}

/**
 * ContentType 是视频的 content_type，
 * 取自 com.dragon.read.rpc.model.VideoContentType。
 * 官方 App 的枚举混用小数值和从 1000 起的一段数值；这里两者都复现。
 *
 * 取值：
 * - ContentUnknown = 0
 * - ContentShortSeries = 1：真人短剧
 * - ContentScenePlay = 2：场景剧
 * - ContentExpertRecommend = 3：专家荐书
 * - ContentPPTRecommend = 4：PPT 荐书
 * - ContentMovie = 5：电影
 * - ContentTelePlay = 6：电视剧
 * - ContentShortPlay = 7：短剧
 * - ContentDropMaterial = 1000：投放素材
 * - ContentPUGC = 1001：PUGC
 * - ContentUnrealShortPlay = 1002：虚拍短剧
 * - ContentAlbum = 1003：合集
 * - ContentMotionComic = 1004：漫剧（动态漫）
 */
export type ContentType = number;

/**
 * Device 是匿名身份出示的手机。注册请求体、
 * 每个请求的设备参数和 User-Agent 都描述它，所以一个设备 id 始终对应一个机型。
 */
export interface Device {
  /**
   * Brand 是 device_brand，也是制造商（"Xiaomi"）；Model 是机型代号，
   * 作为 device_type、device_model 发送，也写进 User-Agent（"23127PN0CC"）。
   */
  brand: string;
  model: string;
  /**
   * OSVersion 是 Android 版本（"14"），OSAPI 是其 API 级别（34），
   * Build 是 User-Agent 携带的构建 id（"UKQ1.230804.001"）。
   */
  os_version: string;
  os_api: number;
  build: string;
  /**
   * Resolution 是 "height*width"，单位是像素；DPI 是屏幕密度，
   * 单位是每英寸点数。
   */
  resolution: string;
  dpi: number;
  /**
   * Pad 让设备以平板身份出现（dragon_device_type=pad）。
   */
  pad?: boolean;
  /**
   * Channel 是 App 构建上报的分发渠道（"xiaomi_8662_64" 来自小米应用商店）。
   */
  channel: string;
}

/**
 * Episode 是一个公开列出的分集。Number 对调用方从 1 开始。
 */
export interface Episode {
  id: string;
  series_id: string;
  number: number;
  title?: string;
  /**
   * 服务端禁止播放该分集时，Playable 为 false（video_list[].disable_play）。
   */
  playable: boolean;
  /**
   * Vertical 标记竖屏画幅的分集。
   */
  vertical?: boolean;
  /**
   * IsNew 标记本次更新中发布的分集。
   */
  is_new?: boolean;
  /**
   * CoverURL 是分集剧照；CoverExpiresAt 是它的签名过期时间。
   */
  cover_url?: string;
  cover_expires_at?: string;
  /**
   * ContentType 与 Series.ContentType 是同一枚举。
   */
  content_type?: ContentType;
  /**
   * CommentCount 和 DiggCount 是分集的计数。
   */
  comment_count?: number;
  digg_count?: number;
}

/**
 * FilterGroup 是一个筛选维度（状态、排序、子分类、篇幅）。
 */
export interface FilterGroup {
  /**
   * Key 是该组设置的查询参数，例如 sort_by。
   */
  key: string;
  options: FilterOption[];
}

/**
 * FilterOption 是分类页上的一个可选值。
 */
export interface FilterOption {
  /**
   * Name 是显示的标签；Value 是查询选中它时发送的值。
   */
  name: string;
  value: string;
}

/**
 * FilterRow 是短剧筛选页的一个选择行。Key 是该行的 select_items 键（genre、
 * category_dim_theme、category_dim_role、category_dim_epoch、sort、gender、
 * online_time、creation_status）；选项的 Value 是查询所选的值。
 * 有哪些行取决于 genre。
 */
export interface FilterRow {
  key: string;
  name: string;
  options: FilterOption[];
}

/**
 * HotBoard 是一个热榜。作品榜（短剧热搜榜、漫剧热搜榜、短剧新剧榜）填充 Items；
 * 热点话题榜填充 Topics。出现哪些榜单因调用而异。
 */
export interface HotBoard {
  name: string;
  items?: HotItem[];
  topics?: HotTopic[];
}

/**
 * HotItem 是热榜上的一部作品，按榜单顺序排列。
 */
export interface HotItem {
  /**
   * Position 是在榜单上的名次，从 1 开始。
   */
  position: number;
  series_id: string;
  title: string;
  cover_url?: string;
  cover_expires_at?: string;
  episode_count?: number;
  content_type?: ContentType;
  /**
   * SearchScore 是榜单自己的数值（"355万热搜值"，有时带 "·第1季"）；
   * Heat 是作品的总热度（"4102万热度"）。
   */
  search_score?: string;
  heat?: string;
  /**
   * Tag 是角标，例如新剧。
   */
  tag?: string;
}

/**
 * HotSearch 是查询前的搜索页：猜你想搜一行，以及其下的热榜。
 */
export interface HotSearch {
  guesses?: HotItem[];
  boards?: HotBoard[];
}

/**
 * HotTopic 是热点话题榜的一项。Query 是它打开的搜索。
 */
export interface HotTopic {
  id: string;
  title: string;
  query?: string;
  heat?: string;
}

/**
 * InfoKind 是 secondary_infos[].data_type，
 * 取自 com.dragon.read.rpc.model.SecondaryInfoDataType。
 *
 * 取值：
 * - InfoSeasonOrder = 0：InfoSeasonOrder 是 "第 N 季" 那一行（实测值为 0）。
 * - InfoRecommendReason = 1：InfoRecommendReason 是推荐理由。
 * - InfoRankScore = 2：InfoRankScore 是榜单名次。
 * - InfoCategory = 3：InfoCategory 是分类标签，例如 "末世"（实测值为 3）。
 * - InfoScore = 4：InfoScore 是用户评分。
 * - InfoAuthorName = 8：InfoAuthorName 是作者名。
 * - InfoTags = 9：InfoTags 是标签列表。
 * - InfoCreationStatus = 11：InfoCreationStatus 是连载中或已完结的说明。
 * - InfoTwoActors = 33：InfoTwoActors 是演员行 "演员·A B"（实测值为 33）。
 * - InfoPlotTag = 40：InfoPlotTag 是剧情标签。
 */
export type InfoKind = number;

/**
 * InfoRow 是 secondary_infos 的一项：详情页上带标签的一行。
 */
export interface InfoRow {
  kind: InfoKind;
  text: string;
  /**
   * Schema 是该行打开的应用内链接（如果有）。
   */
  schema?: string;
}

/**
 * NewRelease 是上新日历中的一部作品：已上线（Online），
 * 或已预告在 PublishAt 上线。
 */
export interface NewRelease {
  series_id: string;
  title: string;
  description?: string;
  cover_url?: string;
  cover_expires_at?: string;
  episode_count?: number;
  /**
   * 作品已上线时 Online 为 true；PublishAt 是预告作品的上线时间。
   */
  online: boolean;
  publish_at?: string;
  categories?: string[];
  /**
   * 已上线的作品 Heat 形如 "378万热度"，预告作品形如 "159.7万人预约"；
   * Note 是预告文案（"预计10月26日上线"）。
   */
  heat?: string;
  note?: string;
}

/**
 * NewReleasePage 是上新日历中的一天。
 */
export interface NewReleasePage {
  /**
   * Date 是本次返回的日期（YYYYMMDD）；Dates 是日历提供的日期，
   * 即今天前后各一周。
   */
  date?: string;
  dates?: string[];
  items: NewRelease[];
  has_more: boolean;
  cursor?: string;
}

/**
 * Novel 是短剧改编自的小说。
 */
export interface Novel {
  book_id: string;
  /**
   * Title 是小说本身的名称；ShortTitle 是它与短剧共用的名称。
   */
  title: string;
  short_title?: string;
  status: NovelStatus;
  tags?: string[];
  reads?: number;
  cover_url?: string;
  cover_expires_at?: string;
  /**
   * ShowAtEpisode 是播放器推荐该小说时所在的集数。
   */
  show_at_episode?: number;
}

/**
 * NovelStatus 是客户端的 NovelCreationStatus；未给出时为 -1。
 *
 * 取值：
 * - NovelFinished = 0：已完结
 * - NovelOngoing = 1：连载中
 * - NovelUpdatedToday = 3：今日更新
 */
export type NovelStatus = number;

/**
 * PayType 是 pay_info.pay_type，取自 com.dragon.read.rpc.model.VideoPayType。
 *
 * 取值：
 * - PayFree = 0：PayFree 表示无需账号即可观看。
 * - PayVip = 1：PayVip 表示需要订阅。
 * - PaySingle = 2：PaySingle 表示需要单次购买。
 */
export type PayType = number;

/**
 * Person 是演员或角色的主页。
 */
export interface Person {
  user_id: string;
  name: string;
  description?: string;
  avatar_url?: string;
  /**
   * Tags 是主页上的头衔（"女演员"、"虚拟人物"）；
   * Labels 是名字下方的小徽标（"水瓶座"）。
   */
  tags?: string[];
  labels?: string[];
  /**
   * Followers 是粉丝数；Likes 是获赞数。
   */
  followers?: number;
  likes?: number;
}

/**
 * PlatformType 是 video_platform，
 * 取自 com.dragon.read.rpc.model.VideoPlatformType。
 *
 * 取值：
 * - PlatformUnknown = 0：未知；有声书带此值
 * - PlatformDouyin = 1：抖音
 * - PlatformUGCVideo = 2：UGC 视频
 * - PlatformOuter = 3：站外与自制短剧
 * - PlatformXigua = 4：西瓜视频，电影来自这里
 * - PlatformChameleon = 5：Chameleon
 * - PlatformMixed = 1024：混合来源
 */
export type PlatformType = number;

/**
 * Playback 是一个已解析的匿名媒体模型。它描述服务端提供的内容，
 * 从不携带解密密钥或许可证材料。
 */
export interface Playback {
  episode_id: string;
  /**
   * VideoID 是分集背后的媒体 id。
   */
  video_id?: string;
  /**
   * MediaType 是服务端的媒体类型，例如 "video"。
   */
  media_type?: string;
  duration_seconds?: number;
  /**
   * ExpiresAt 是流 URL 失效的时间，单位是 unix 秒。
   */
  expires_at?: number;
  /**
   * Protected 标记加密的分集；参见 Stream.Protected。
   */
  protected: boolean;
  streams: Stream[];
  /**
   * AudioURL 是分段（DASH）作品（如长片电影）的独立音轨。
   * 单文件的表示形式留空。
   */
  audio_url?: string;
}

/**
 * RankBoard 是某个榜单内容下的一个榜（推荐榜、热播榜……）。
 */
export interface RankBoard {
  id: string;
  name: string;
  selected?: boolean;
}

/**
 * RankChange 是榜单行的变动（客户端中的 RankChangeType）。
 */
export interface RankChange {
  kind: RankChangeKind;
  /**
   * Places 是该行变动的位数；新上榜或不变的行为 0。
   */
  places?: number;
}

/**
 * RankChangeKind 的取值即客户端的 RankChangeType。
 *
 * 取值：
 * - RankNew = 0：新上榜
 * - RankUp = 1：上升 RankChange.Places 位
 * - RankDown = 2：下降 RankChange.Places 位
 * - RankUnchanged = 3：名次不变
 */
export type RankChangeKind = number;

/**
 * RankContent 是短剧榜单页的一个内容
 * （全部、真人剧、漫剧、AI剧、系列剧、演员），带有自己的各个榜。
 */
export interface RankContent {
  id: string;
  name: string;
  selected?: boolean;
  boards: RankBoard[];
}

/**
 * RankItem 是榜单的一行。
 */
export interface RankItem {
  /**
   * Rank 是服务端给该行的名次（参见 Position）。
   */
  rank?: number;
  /**
   * Votes 和 Kind 用于活动榜单行：分别是票数和服务端的条目类型，原样透传。
   */
  votes?: number;
  kind?: number;
  /**
   * Actor 在书籍榜单上是作者，在活动榜单或短剧演员榜上是演员名。
   */
  actor?: string;
  series_id?: string;
  title?: string;
  cover_url?: string;
  /**
   * Score 是显示的热度（"2550万热度"、"998万推荐"、"平台推荐"）。
   */
  score?: string;
  /**
   * EpisodeCount 由短剧榜单填充。
   */
  episode_count?: number;
  /**
   * Position 是该行在短剧榜单页上的名次，从 1 开始。Rank 是服务端自己的编号，
   * 在推广行前后会重复，在推荐榜上是个性化的。
   */
  position?: number;
  /**
   * Promoted 标记服务端插入短剧榜单的行：它显示 "平台推荐" 而非热度，
   * 名次照抄下一行（实测）。
   */
  promoted?: boolean;
  /**
   * ActorID、AvatarURL、Intro 和 Details 用于短剧榜单的演员行，
   * 该行的 Actor 是名字，Score 是人气。Intro 列出作品；
   * Details 存放 "67部作品" 这样的行。
   */
  actor_id?: string;
  /**
   * UserID 是演员的账号，即 ActorProfile 和 ActorWorks 接受的 id。
   */
  user_id?: string;
  avatar_url?: string;
  intro?: string;
  details?: string[];
  /**
   * Change 是该行相对上一期榜单的变动（如果有）。
   */
  change?: RankChange;
}

/**
 * RankList 是活动的一个榜单频道。
 */
export interface RankList {
  id: string;
  name?: string;
  /**
   * Type 和 Size 是服务端的榜单类型和长度，原样透传。
   */
  type?: number;
  size?: number;
}

/**
 * Ranking 是一个活动榜单页。
 */
export interface Ranking {
  list_name?: string;
  /**
   * Lists 是该活动的其他榜单频道。
   */
  lists?: RankList[];
  items: RankItem[];
  has_more: boolean;
  /**
   * NextOffset 用于续取列表（RankRows 把它当作 offset）；
   * SessionID 是服务端上报的分页会话。
   */
  next_offset?: number;
  /**
   * ActivityStage 和 CanVote 是服务端上报的活动状态。
   */
  activity_stage?: number;
  can_vote?: boolean;
  session_id?: string;
}

/**
 * Recommend 是一部短剧的相关视频单元：包含各行、单元标题，
 * 以及客户端为后续请求保留的分页会话。
 */
export interface Recommend {
  cell_title?: string;
  session_id?: string;
  items: SeriesSummary[];
}

/**
 * Season 是合集（多季合集）中的一项。
 */
export interface Season {
  series_id: string;
  title?: string;
  /**
   * Index 是季序号，从 1 开始。
   */
  index?: number;
  cover_url?: string;
  episode_count?: number;
  status?: SeriesStatus;
}

/**
 * Series 是一部短剧的详情视图。
 */
export interface Series extends SeriesSummary {
  /**
   * Status 是发布状态；PayType 是付费墙状态。
   */
  status?: SeriesStatus;
  pay_type?: PayType;
  /**
   * ContentType 和 Platform 描述作品是什么、来自哪里（短剧、电影、漫剧……）。
   */
  content_type?: ContentType;
  platform?: PlatformType;
  /**
   * Heat 是原始热度分；App 将其显示为 "红果热度值N万"。
   */
  heat?: number;
  /**
   * PlayCount 是短剧的总播放量（series_play_cnt）。
   */
  play_count?: number;
  /**
   * FollowCount 和 DiggCount 是标题旁显示的计数。
   */
  follow_count?: number;
  digg_count?: number;
  /**
   * CreatedAt 是发布时间（create_time，unix 秒）。
   */
  created_at?: string;
  /**
   * RightText 是集数旁的角标（"更新至12集"）。
   */
  right_text?: string;
  /**
   * Subtitles 是标题下方的简短状态行。
   */
  subtitles?: string[];
  /**
   * AgeGate 对不受限的作品为 0。
   */
  age_gate?: number;
  /**
   * IsMultiSeason 标记属于多季合集的作品。
   */
  is_multi_season?: boolean;
  /**
   * SeasonIndex 是该合集内的季序号，从 1 开始。
   */
  season_index?: number;
  /**
   * RelatedAlbumID 是各季共用的合集 id。
   */
  related_album_id?: string;
  /**
   * RelatedNovelBookID 是短剧改编的小说（如已公开）。
   */
  related_novel_book_id?: string;
  /**
   * Novel 是详情页链接的原著小说（video_relate_book），服务端给出时才有。
   */
  novel?: Novel;
  /**
   * Rows 是详情页的次要信息行（季、分类、演员）。
   */
  rows?: InfoRow[];
}

/**
 * SeriesDetail 是一部短剧及其分集，由批量读取返回。
 */
export interface SeriesDetail {
  series: Series;
  episodes: Episode[];
}

/**
 * SeriesStatus 是短剧的发布状态，由 data.video_data.series_status 携带。
 *
 * 名称取自官方 App 的枚举 com.dragon.read.rpc.model.SeriesStatus（
 * SeriesUpdating、SeriesEnd、SeriesUpdateTaday、SeriesUpdateStop）；
 * 值 0 和 1 也已在线上实测。
 *
 * 取值：
 * - SeriesUpdating = 0：SeriesUpdating 是未完结的短剧（"更新至 N 集"）。
 * - SeriesEnd = 1：SeriesEnd 是已完结的短剧（"全 N 集"）。
 * - SeriesUpdatedToday = 3：SeriesUpdatedToday 是枚举中的“今日更新”状态（推断）。
 * - SeriesUpdateStopped = 4：SeriesUpdateStopped 是枚举中的“停止更新”状态（推断）。
 */
export type SeriesStatus = number;

/**
 * SeriesSummary 是公开目录中的一行。ID 和标题是稳定字段。
 */
export interface SeriesSummary {
  id: string;
  title: string;
  description?: string;
  cover_url?: string;
  /**
   * CoverExpiresAt 是 CoverURL 的签名过期时间（x-expires）；
   * 过期后要从 API 重新获取封面。
   */
  cover_expires_at?: string;
  episode_count?: number;
  /**
   * Author、Category 和 Words 由分类页的行填充。
   */
  author?: string;
  category?: string;
  words?: number;
}

/**
 * ShortFilterPage 是符合筛选条件的短剧的一页。
 */
export interface ShortFilterPage {
  items: SeriesSummary[];
  has_more: boolean;
  /**
   * Cursor 用于续取列表；它携带已展示的短剧，服务端会跳过它们。
   */
  cursor?: string;
}

/**
 * ShortRanking 是短剧榜单的一页。
 */
export interface ShortRanking {
  /**
   * Contents 是该页的目录：每个内容及其各个榜。
   */
  contents: RankContent[];
  /**
   * Content 和 Board 是本页对应的 ID。
   */
  content: string;
  board: string;
  /**
   * Version 是服务端上报的榜单版本。
   */
  version?: string;
  items: RankItem[];
  has_more: boolean;
  cursor?: string;
}

/**
 * Stream 是媒体模型中的一个表示形式。Protected
 * 标记官方客户端在内部解密的表示形式；sdk.ContentKey 从模型的 KeyMaterial
 * 还原它的密钥。
 */
export interface Stream {
  /**
   * Definition 是分辨率标签（"720p"）；Quality 是服务端的画质标签
   * （"normal"）。
   */
  definition?: string;
  quality?: string;
  codec?: string;
  width?: number;
  height?: number;
  /**
   * Bitrate 的单位是 bit/s；Size 是文件大小，单位是字节。
   */
  bitrate?: number;
  size?: number;
  /**
   * BackupURL 是同一文件的第二个 CDN 地址。
   */
  url?: string;
  backup_url?: string;
  /**
   * HLS 标记 m3u8 播放列表；Dash 标记分段（DASH）的表示形式。
   */
  hls?: boolean;
  dash?: boolean;
  protected: boolean;
  /**
   * Encryption 是受保护流的加密方案，例如 "cenc-aes-ctr"。
   */
  encryption?: string;
}

/**
 * Suggestion 是一条搜索建议。Text 是搜索框显示的文本；
 * Keyword 是它提交的内容（直达行为内容 ID）。
 */
export interface Suggestion {
  text: string;
  keyword?: string;
  /**
   * Kind 是服务端的行类型，原样透传。
   */
  kind?: number;
}

/**
 * Work 是某人出演的一部短剧。
 */
export interface Work {
  series_id: string;
  title: string;
  description?: string;
  cover_url?: string;
  cover_expires_at?: string;
  episode_count?: number;
  /**
   * Tags 是卡片上的各行，例如 "爱情" 和 "70集"。
   */
  tags?: string[];
  /**
   * Heat 是短剧的原始热度分；Plays 是其播放量。
   */
  heat?: number;
  plays?: number;
}

/**
 * WorkPage 是某人作品的一页（作品 › 剧作）。
 */
export interface WorkPage {
  items: Work[];
  has_more: boolean;
  cursor?: string;
}

/**
 * DiscoverOptions 选择一个发现页。Channel 是书城频道名，空频道即推荐。
 * Cursor 是上一页给出的令牌。PageSize 是 cell/change 的 limit（0 表示 20，上限 50）。
 */
export interface DiscoverOptions {
  cursor?: string;
  channel?: string;
  pageSize?: number;
}

/**
 * SearchOptions 选择一个搜索页。Tab 取值为综合、剧集、视频、讨论、小说、
 * 用户或听书，空 tab 即综合。Cursor 是下一个偏移量。
 */
export interface SearchOptions {
  cursor?: string;
  pageSize?: number;
  tab?: string;
  /**
   * Filters 把综合 tab 收窄到 SearchFilter* 值；选多个时互相放宽。
   * 游标不携带这些筛选，所以要连同游标传入相同的筛选。
   */
  filters?: string[];
}

/**
 * EpisodeOptions 选择一个分集页。游标不在第 1 阶段的范围内。
 */
export interface EpisodeOptions {
  cursor?: string;
  pageSize?: number;
}

/**
 * LandingOptions 选择一个分类落地页。
 *
 * Tab 是 Categories 给出的 tab 值：8（全部，默认）、1（男生）或 0（女生）。
 * Filters 把 CategoryPage.Filters 中的筛选组键（book_status、sort_by、
 * sub_category、word_type）映射到该组的某个选项值。
 * Cursor 是上一页的 NextOffset。
 */
export interface LandingOptions {
  tab?: string;
  filters?: Record<string, string>;
  cursor?: string;
}

/**
 * NewReleaseQuery 选择上新日历中的一天：Date 格式为 YYYYMMDD（空即今天），
 * Cursor 是上一页的 Cursor，它带有所在的日期。
 */
export interface NewReleaseQuery {
  date?: string;
  cursor?: string;
}

/**
 * ShortFilterQuery 在筛选页（筛选）上挑选短剧。
 *
 * Selections 把 FilterRow.Key 映射到选中的选项 Values；缺省的行即全部。
 * Cursor 是上一页的 Cursor。
 */
export interface ShortFilterQuery {
  selections?: Record<string, string[]>;
  cursor?: string;
}

/**
 * ShortRankQuery 选择一个短剧榜单。Content 是 RankContent ID（all、human、
 * comic_series_rank、ai_playlet、series_album；空即 all），
 * Board 是它的某个 RankBoard ID（空即服务端对该 content 的默认值）。
 * Cursor 是上一页的 Cursor。
 */
export interface ShortRankQuery {
  content?: string;
  board?: string;
  cursor?: string;
}

