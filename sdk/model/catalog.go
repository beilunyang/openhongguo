package model

import "time"

// SeriesStatus 是短剧的发布状态，由 data.video_data.series_status 携带。
//
// 名称取自官方 App 的枚举 com.dragon.read.rpc.model.SeriesStatus（
// SeriesUpdating、SeriesEnd、SeriesUpdateTaday、SeriesUpdateStop）；
// 值 0 和 1 也已在线上实测。
type SeriesStatus int

const (
	// SeriesUpdating 是未完结的短剧（"更新至 N 集"）。
	SeriesUpdating SeriesStatus = 0
	// SeriesEnd 是已完结的短剧（"全 N 集"）。
	SeriesEnd SeriesStatus = 1
	// SeriesUpdatedToday 是枚举中的“今日更新”状态（推断）。
	SeriesUpdatedToday SeriesStatus = 3
	// SeriesUpdateStopped 是枚举中的“停止更新”状态（推断）。
	SeriesUpdateStopped SeriesStatus = 4
)

// String 给出该状态用于输出的名称。
func (s SeriesStatus) String() string {
	switch s {
	case SeriesUpdating:
		return "updating"
	case SeriesEnd:
		return "finished"
	case SeriesUpdatedToday:
		return "updated_today"
	case SeriesUpdateStopped:
		return "stopped"
	default:
		return "unknown"
	}
}

// PayType 是 pay_info.pay_type，取自 com.dragon.read.rpc.model.VideoPayType。
type PayType int

const (
	// PayFree 表示无需账号即可观看。
	PayFree PayType = 0
	// PayVip 表示需要订阅。
	PayVip PayType = 1
	// PaySingle 表示需要单次购买。
	PaySingle PayType = 2
)

// String 给出该付费类型用于输出的名称。
func (p PayType) String() string {
	switch p {
	case PayFree:
		return "free"
	case PayVip:
		return "vip"
	case PaySingle:
		return "single"
	default:
		return "unknown"
	}
}

// ContentType 是视频的 content_type，
// 取自 com.dragon.read.rpc.model.VideoContentType。
// 官方 App 的枚举混用小数值和从 1000 起的一段数值；这里两者都复现。
type ContentType int

// 内容类型。下面每个值都在线上响应中见过，并与 App 代码里的枚举核对过。
const (
	ContentUnknown         ContentType = 0
	ContentShortSeries     ContentType = 1    // 真人短剧
	ContentScenePlay       ContentType = 2    // 场景剧
	ContentExpertRecommend ContentType = 3    // 专家荐书
	ContentPPTRecommend    ContentType = 4    // PPT 荐书
	ContentMovie           ContentType = 5    // 电影
	ContentTelePlay        ContentType = 6    // 电视剧
	ContentShortPlay       ContentType = 7    // 短剧
	ContentDropMaterial    ContentType = 1000 // 投放素材
	ContentPUGC            ContentType = 1001 // PUGC
	ContentUnrealShortPlay ContentType = 1002 // 虚拍短剧
	ContentAlbum           ContentType = 1003 // 合集
	ContentMotionComic     ContentType = 1004 // 漫剧（动态漫）
)

// String 给出该内容类型用于输出的名称。
func (c ContentType) String() string {
	switch c {
	case ContentShortSeries:
		return "short_series"
	case ContentScenePlay:
		return "scene_play"
	case ContentExpertRecommend:
		return "expert_recommend"
	case ContentPPTRecommend:
		return "ppt_recommend"
	case ContentMovie:
		return "movie"
	case ContentTelePlay:
		return "teleplay"
	case ContentShortPlay:
		return "short_play"
	case ContentDropMaterial:
		return "drop_material"
	case ContentPUGC:
		return "pugc"
	case ContentUnrealShortPlay:
		return "unreal_short_play"
	case ContentAlbum:
		return "album"
	case ContentMotionComic:
		return "motion_comic"
	default:
		return "unknown"
	}
}

// PlatformType 是 video_platform，
// 取自 com.dragon.read.rpc.model.VideoPlatformType。
type PlatformType int

// 作品的来源平台。下面每个值都在线上响应中见过，并与 App 代码里的枚举核对过。
const (
	PlatformUnknown   PlatformType = 0    // 未知；有声书带此值
	PlatformDouyin    PlatformType = 1    // 抖音
	PlatformUGCVideo  PlatformType = 2    // UGC 视频
	PlatformOuter     PlatformType = 3    // 站外与自制短剧
	PlatformXigua     PlatformType = 4    // 西瓜视频，电影来自这里
	PlatformChameleon PlatformType = 5    // Chameleon
	PlatformMixed     PlatformType = 1024 // 混合来源
)

// String 给出该平台用于输出的名称。
func (p PlatformType) String() string {
	switch p {
	case PlatformDouyin:
		return "douyin"
	case PlatformUGCVideo:
		return "ugc_video"
	case PlatformOuter:
		return "outer"
	case PlatformXigua:
		return "xigua"
	case PlatformChameleon:
		return "chameleon"
	case PlatformMixed:
		return "mixed"
	default:
		return "unknown"
	}
}

// InfoKind 是 secondary_infos[].data_type，
// 取自 com.dragon.read.rpc.model.SecondaryInfoDataType。
type InfoKind int

const (
	// InfoSeasonOrder 是 "第 N 季" 那一行（实测值为 0）。
	InfoSeasonOrder InfoKind = 0
	// InfoRecommendReason 是推荐理由。
	InfoRecommendReason InfoKind = 1
	// InfoRankScore 是榜单名次。
	InfoRankScore InfoKind = 2
	// InfoCategory 是分类标签，例如 "末世"（实测值为 3）。
	InfoCategory InfoKind = 3
	// InfoScore 是用户评分。
	InfoScore InfoKind = 4
	// InfoAuthorName 是作者名。
	InfoAuthorName InfoKind = 8
	// InfoTags 是标签列表。
	InfoTags InfoKind = 9
	// InfoCreationStatus 是连载中或已完结的说明。
	InfoCreationStatus InfoKind = 11
	// InfoTwoActors 是演员行 "演员·A B"（实测值为 33）。
	InfoTwoActors InfoKind = 33
	// InfoPlotTag 是剧情标签。
	InfoPlotTag InfoKind = 40
)

// String 给出该行类型用于输出的名称。
func (k InfoKind) String() string {
	switch k {
	case InfoSeasonOrder:
		return "season"
	case InfoRecommendReason:
		return "reason"
	case InfoRankScore:
		return "rank"
	case InfoCategory:
		return "category"
	case InfoScore:
		return "score"
	case InfoAuthorName:
		return "author"
	case InfoTags:
		return "tags"
	case InfoCreationStatus:
		return "creation"
	case InfoTwoActors:
		return "cast"
	case InfoPlotTag:
		return "plot"
	default:
		return "other"
	}
}

// InfoRow 是 secondary_infos 的一项：详情页上带标签的一行。
type InfoRow struct {
	Kind InfoKind `json:"kind"`
	Text string   `json:"text"`
	// Schema 是该行打开的应用内链接（如果有）。
	Schema string `json:"schema,omitempty"`
}

// FilterOption 是分类页上的一个可选值。
type FilterOption struct {
	// Name 是显示的标签；Value 是查询选中它时发送的值。
	Name  string `json:"name"`
	Value string `json:"value"`
}

// FilterGroup 是一个筛选维度（状态、排序、子分类、篇幅）。
type FilterGroup struct {
	// Key 是该组设置的查询参数，例如 sort_by。
	Key     string         `json:"key"`
	Options []FilterOption `json:"options"`
}

// CategoryPage 是一个分类落地页及其筛选项。
type CategoryPage struct {
	CategoryID   string        `json:"category_id,omitempty"`
	CategoryName string        `json:"category_name,omitempty"`
	Description  string        `json:"description,omitempty"`
	Filters      []FilterGroup `json:"filters,omitempty"`
	// Tabs 是受众标签页（全部、男生、女生），每个 Value 是标签页类型编号；
	// DefaultTab 是页面打开时所在的标签页类型。
	Tabs       []FilterOption  `json:"tabs,omitempty"`
	DefaultTab int             `json:"default_tab,omitempty"`
	Items      []SeriesSummary `json:"items"`
	HasMore    bool            `json:"has_more"`
	// NextOffset 是下一页的偏移量，即 LandingOptions.Cursor 的值。
	NextOffset int `json:"next_offset,omitempty"`
}

// SeriesSummary 是公开目录中的一行。ID 和标题是稳定字段。
type SeriesSummary struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	CoverURL    string `json:"cover_url,omitempty"`
	// CoverExpiresAt 是 CoverURL 的签名过期时间（x-expires）；
	// 过期后要从 API 重新获取封面。
	CoverExpiresAt time.Time `json:"cover_expires_at,omitempty"`
	EpisodeCount   int       `json:"episode_count,omitempty"`
	// Author、Category 和 Words 由分类页的行填充。
	Author   string `json:"author,omitempty"`
	Category string `json:"category,omitempty"`
	Words    int64  `json:"words,omitempty"`
}

// Series 是一部短剧的详情视图。
type Series struct {
	SeriesSummary
	// Status 是发布状态；PayType 是付费墙状态。
	Status  SeriesStatus `json:"status,omitempty"`
	PayType PayType      `json:"pay_type,omitempty"`
	// ContentType 和 Platform 描述作品是什么、来自哪里（短剧、电影、漫剧……）。
	ContentType ContentType  `json:"content_type,omitempty"`
	Platform    PlatformType `json:"platform,omitempty"`
	// Heat 是原始热度分；App 将其显示为 "红果热度值N万"。
	Heat int64 `json:"heat,omitempty"`
	// PlayCount 是短剧的总播放量（series_play_cnt）。
	PlayCount int64 `json:"play_count,omitempty"`
	// FollowCount 和 DiggCount 是标题旁显示的计数。
	FollowCount int64 `json:"follow_count,omitempty"`
	DiggCount   int64 `json:"digg_count,omitempty"`
	// CreatedAt 是发布时间（create_time，unix 秒）。
	CreatedAt time.Time `json:"created_at,omitempty"`
	// RightText 是集数旁的角标（"更新至12集"）。
	RightText string `json:"right_text,omitempty"`
	// Subtitles 是标题下方的简短状态行。
	Subtitles []string `json:"subtitles,omitempty"`
	// AgeGate 对不受限的作品为 0。
	AgeGate int `json:"age_gate,omitempty"`
	// IsMultiSeason 标记属于多季合集的作品。
	IsMultiSeason bool `json:"is_multi_season,omitempty"`
	// SeasonIndex 是该合集内的季序号，从 1 开始。
	SeasonIndex int `json:"season_index,omitempty"`
	// RelatedAlbumID 是各季共用的合集 id。
	RelatedAlbumID string `json:"related_album_id,omitempty"`
	// RelatedNovelBookID 是短剧改编的小说（如已公开）。
	RelatedNovelBookID string `json:"related_novel_book_id,omitempty"`
	// Novel 是详情页链接的原著小说（video_relate_book），服务端给出时才有。
	Novel *Novel `json:"novel,omitempty"`
	// Rows 是详情页的次要信息行（季、分类、演员）。
	Rows []InfoRow `json:"rows,omitempty"`
}

// Novel 是短剧改编自的小说。
type Novel struct {
	BookID string `json:"book_id"`
	// Title 是小说本身的名称；ShortTitle 是它与短剧共用的名称。
	Title          string      `json:"title"`
	ShortTitle     string      `json:"short_title,omitempty"`
	Status         NovelStatus `json:"status"`
	Tags           []string    `json:"tags,omitempty"`
	Reads          int64       `json:"reads,omitempty"`
	CoverURL       string      `json:"cover_url,omitempty"`
	CoverExpiresAt time.Time   `json:"cover_expires_at,omitempty"`
	// ShowAtEpisode 是播放器推荐该小说时所在的集数。
	ShowAtEpisode int `json:"show_at_episode,omitempty"`
}

// NovelStatus 是客户端的 NovelCreationStatus；未给出时为 -1。
type NovelStatus int

// 小说的创作状态。
const (
	NovelFinished     NovelStatus = 0 // 已完结
	NovelOngoing      NovelStatus = 1 // 连载中
	NovelUpdatedToday NovelStatus = 3 // 今日更新
)

// String 给出 CLI 打印的状态名称。
func (s NovelStatus) String() string {
	switch s {
	case NovelFinished:
		return "finished"
	case NovelOngoing:
		return "ongoing"
	case NovelUpdatedToday:
		return "updated today"
	}
	return "unknown"
}

// SeriesDetail 是一部短剧及其分集，由批量读取返回。
type SeriesDetail struct {
	Series   *Series   `json:"series"`
	Episodes []Episode `json:"episodes"`
}

// Episode 是一个公开列出的分集。Number 对调用方从 1 开始。
type Episode struct {
	ID       string `json:"id"`
	SeriesID string `json:"series_id"`
	Number   int    `json:"number"`
	Title    string `json:"title,omitempty"`
	// Duration 是分集时长，列表给出时才有。
	Duration time.Duration `json:"-"`
	// 服务端禁止播放该分集时，Playable 为 false（video_list[].disable_play）。
	Playable bool `json:"playable"`
	// Vertical 标记竖屏画幅的分集。
	Vertical bool `json:"vertical,omitempty"`
	// IsNew 标记本次更新中发布的分集。
	IsNew bool `json:"is_new,omitempty"`
	// CoverURL 是分集剧照；CoverExpiresAt 是它的签名过期时间。
	CoverURL       string    `json:"cover_url,omitempty"`
	CoverExpiresAt time.Time `json:"cover_expires_at,omitempty"`
	// ContentType 与 Series.ContentType 是同一枚举。
	ContentType ContentType `json:"content_type,omitempty"`
	// CommentCount 和 DiggCount 是分集的计数。
	CommentCount int64 `json:"comment_count,omitempty"`
	DiggCount    int64 `json:"digg_count,omitempty"`
}

// Season 是合集（多季合集）中的一项。
type Season struct {
	SeriesID string `json:"series_id"`
	Title    string `json:"title,omitempty"`
	// Index 是季序号，从 1 开始。
	Index        int          `json:"index,omitempty"`
	CoverURL     string       `json:"cover_url,omitempty"`
	EpisodeCount int          `json:"episode_count,omitempty"`
	Status       SeriesStatus `json:"status,omitempty"`
}

// Album 是多季的合集，来自 /novel/player/album_detail/v1/。
type Album struct {
	ID             string    `json:"id"`
	Title          string    `json:"title,omitempty"`
	Description    string    `json:"description,omitempty"`
	CoverURL       string    `json:"cover_url,omitempty"`
	CoverExpiresAt time.Time `json:"cover_expires_at,omitempty"`
	SeasonCount    int       `json:"season_count,omitempty"`
	Seasons        []Season  `json:"seasons,omitempty"`
}

// Suggestion 是一条搜索建议。Text 是搜索框显示的文本；
// Keyword 是它提交的内容（直达行为内容 ID）。
type Suggestion struct {
	Text    string `json:"text"`
	Keyword string `json:"keyword,omitempty"`
	// Kind 是服务端的行类型，原样透传。
	Kind int `json:"kind,omitempty"`
}

// Recommend 是一部短剧的相关视频单元：包含各行、单元标题，
// 以及客户端为后续请求保留的分页会话。
type Recommend struct {
	CellTitle string          `json:"cell_title,omitempty"`
	SessionID string          `json:"session_id,omitempty"`
	Items     []SeriesSummary `json:"items"`
}

// ShowTypeDetailRelatedVideo 是相关视频标签页的 plan
// 单元类型（ShowType.DetailRelatedVideo 的值；Kotlin 序号是 58）。
const ShowTypeDetailRelatedVideo = 160

// RankList 是活动的一个榜单频道。
type RankList struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
	// Type 和 Size 是服务端的榜单类型和长度，原样透传。
	Type int `json:"type,omitempty"`
	Size int `json:"size,omitempty"`
}

// NewReleasePage 是上新日历中的一天。
type NewReleasePage struct {
	// Date 是本次返回的日期（YYYYMMDD）；Dates 是日历提供的日期，
	// 即今天前后各一周。
	Date    string       `json:"date,omitempty"`
	Dates   []string     `json:"dates,omitempty"`
	Items   []NewRelease `json:"items"`
	HasMore bool         `json:"has_more"`
	Cursor  string       `json:"cursor,omitempty"`
}

// NewRelease 是上新日历中的一部作品：已上线（Online），
// 或已预告在 PublishAt 上线。
type NewRelease struct {
	SeriesID       string    `json:"series_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description,omitempty"`
	CoverURL       string    `json:"cover_url,omitempty"`
	CoverExpiresAt time.Time `json:"cover_expires_at,omitempty"`
	EpisodeCount   int       `json:"episode_count,omitempty"`
	// 作品已上线时 Online 为 true；PublishAt 是预告作品的上线时间。
	Online     bool      `json:"online"`
	PublishAt  time.Time `json:"publish_at,omitempty"`
	Categories []string  `json:"categories,omitempty"`
	// 已上线的作品 Heat 形如 "378万热度"，预告作品形如 "159.7万人预约"；
	// Note 是预告文案（"预计10月26日上线"）。
	Heat string `json:"heat,omitempty"`
	Note string `json:"note,omitempty"`
}

// Person 是演员或角色的主页。
type Person struct {
	UserID      string `json:"user_id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	AvatarURL   string `json:"avatar_url,omitempty"`
	// Tags 是主页上的头衔（"女演员"、"虚拟人物"）；
	// Labels 是名字下方的小徽标（"水瓶座"）。
	Tags   []string `json:"tags,omitempty"`
	Labels []string `json:"labels,omitempty"`
	// Followers 是粉丝数；Likes 是获赞数。
	Followers int64 `json:"followers,omitempty"`
	Likes     int64 `json:"likes,omitempty"`
}

// WorkPage 是某人作品的一页（作品 › 剧作）。
type WorkPage struct {
	Items   []Work `json:"items"`
	HasMore bool   `json:"has_more"`
	Cursor  string `json:"cursor,omitempty"`
}

// Work 是某人出演的一部短剧。
type Work struct {
	SeriesID       string    `json:"series_id"`
	Title          string    `json:"title"`
	Description    string    `json:"description,omitempty"`
	CoverURL       string    `json:"cover_url,omitempty"`
	CoverExpiresAt time.Time `json:"cover_expires_at,omitempty"`
	EpisodeCount   int       `json:"episode_count,omitempty"`
	// Tags 是卡片上的各行，例如 "爱情" 和 "70集"。
	Tags []string `json:"tags,omitempty"`
	// Heat 是短剧的原始热度分；Plays 是其播放量。
	Heat  int64 `json:"heat,omitempty"`
	Plays int64 `json:"plays,omitempty"`
}

// DanmakuPage 是一集弹幕的一页：服务端按播放时间划出的一段（2026-11-11
// 实测为 30 秒）。
type DanmakuPage struct {
	// Items 按出现时间排序。这一段没有弹幕时为空，但 HasMore 仍可能为真。
	Items []Danmaku `json:"items"`
	// StartMS 和 EndMS 是这一页覆盖的播放时间，单位毫秒。
	StartMS int64 `json:"start_ms"`
	EndMS   int64 `json:"end_ms,omitempty"`
	// Total 是服务端统计的本集弹幕总数，服务端给出时才有（有些设备拿不到）。
	// 服务端只下发其中精选的一部分，所以它通常远大于能取到的条数。
	Total   int64  `json:"total,omitempty"`
	HasMore bool   `json:"has_more"`
	Cursor  string `json:"cursor,omitempty"`
}

// Danmaku 是一条弹幕。它不带发送者的身份。
type Danmaku struct {
	ID   string `json:"id"`
	Text string `json:"text"`
	// OffsetMS 是弹幕在分集里出现的时间，单位毫秒。
	OffsetMS int64 `json:"offset_ms"`
	// CreatedAt 是发送时间。
	CreatedAt time.Time `json:"created_at,omitempty"`
	// Likes 是点赞数；PlusOnes 是其他观众跟发 "+1" 的次数。
	Likes    int64 `json:"likes,omitempty"`
	PlusOnes int64 `json:"plus_ones,omitempty"`
}

// HotSearch 是查询前的搜索页：猜你想搜一行，以及其下的热榜。
type HotSearch struct {
	Guesses []HotItem  `json:"guesses,omitempty"`
	Boards  []HotBoard `json:"boards,omitempty"`
}

// HotBoard 是一个热榜。作品榜（短剧热搜榜、漫剧热搜榜、短剧新剧榜）填充 Items；
// 热点话题榜填充 Topics。出现哪些榜单因调用而异。
type HotBoard struct {
	Name   string     `json:"name"`
	Items  []HotItem  `json:"items,omitempty"`
	Topics []HotTopic `json:"topics,omitempty"`
}

// HotItem 是热榜上的一部作品，按榜单顺序排列。
type HotItem struct {
	// Position 是在榜单上的名次，从 1 开始。
	Position       int         `json:"position"`
	SeriesID       string      `json:"series_id"`
	Title          string      `json:"title"`
	CoverURL       string      `json:"cover_url,omitempty"`
	CoverExpiresAt time.Time   `json:"cover_expires_at,omitempty"`
	EpisodeCount   int         `json:"episode_count,omitempty"`
	ContentType    ContentType `json:"content_type,omitempty"`
	// SearchScore 是榜单自己的数值（"355万热搜值"，有时带 "·第1季"）；
	// Heat 是作品的总热度（"4102万热度"）。
	SearchScore string `json:"search_score,omitempty"`
	Heat        string `json:"heat,omitempty"`
	// Tag 是角标，例如新剧。
	Tag string `json:"tag,omitempty"`
}

// HotTopic 是热点话题榜的一项。Query 是它打开的搜索。
type HotTopic struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Query string `json:"query,omitempty"`
	Heat  string `json:"heat,omitempty"`
}

// RankItem 是榜单的一行。
type RankItem struct {
	// Rank 是服务端给该行的名次（参见 Position）。
	Rank int `json:"rank,omitempty"`
	// Votes 和 Kind 用于活动榜单行：分别是票数和服务端的条目类型，原样透传。
	Votes int64 `json:"votes,omitempty"`
	Kind  int   `json:"kind,omitempty"`
	// Actor 在书籍榜单上是作者，在活动榜单或短剧演员榜上是演员名。
	Actor    string `json:"actor,omitempty"`
	SeriesID string `json:"series_id,omitempty"`
	Title    string `json:"title,omitempty"`
	CoverURL string `json:"cover_url,omitempty"`
	// Score 是显示的热度（"2550万热度"、"998万推荐"、"平台推荐"）。
	Score string `json:"score,omitempty"`
	// EpisodeCount 由短剧榜单填充。
	EpisodeCount int `json:"episode_count,omitempty"`
	// Position 是该行在短剧榜单页上的名次，从 1 开始。Rank 是服务端自己的编号，
	// 在推广行前后会重复，在推荐榜上是个性化的。
	Position int `json:"position,omitempty"`
	// Promoted 标记服务端插入短剧榜单的行：它显示 "平台推荐" 而非热度，
	// 名次照抄下一行（实测）。
	Promoted bool `json:"promoted,omitempty"`
	// ActorID、AvatarURL、Intro 和 Details 用于短剧榜单的演员行，
	// 该行的 Actor 是名字，Score 是人气。Intro 列出作品；
	// Details 存放 "67部作品" 这样的行。
	ActorID string `json:"actor_id,omitempty"`
	// UserID 是演员的账号，即 ActorProfile 和 ActorWorks 接受的 id。
	UserID    string   `json:"user_id,omitempty"`
	AvatarURL string   `json:"avatar_url,omitempty"`
	Intro     string   `json:"intro,omitempty"`
	Details   []string `json:"details,omitempty"`
	// Change 是该行相对上一期榜单的变动（如果有）。
	Change *RankChange `json:"change,omitempty"`
}

// RankChange 是榜单行的变动（客户端中的 RankChangeType）。
type RankChange struct {
	Kind RankChangeKind `json:"kind"`
	// Places 是该行变动的位数；新上榜或不变的行为 0。
	Places int `json:"places,omitempty"`
}

// RankChangeKind 的取值即客户端的 RankChangeType。
type RankChangeKind int

// 榜单变动。
const (
	RankNew       RankChangeKind = 0 // 新上榜
	RankUp        RankChangeKind = 1 // 上升 RankChange.Places 位
	RankDown      RankChangeKind = 2 // 下降 RankChange.Places 位
	RankUnchanged RankChangeKind = 3 // 名次不变
)

// String 是 CLI 为该变动打印的箭头。
func (k RankChangeKind) String() string {
	switch k {
	case RankNew:
		return "new"
	case RankUp:
		return "up"
	case RankDown:
		return "down"
	case RankUnchanged:
		return "same"
	}
	return "unknown"
}

// Ranking 是一个活动榜单页。
type Ranking struct {
	ListName string `json:"list_name,omitempty"`
	// Lists 是该活动的其他榜单频道。
	Lists   []RankList `json:"lists,omitempty"`
	Items   []RankItem `json:"items"`
	HasMore bool       `json:"has_more"`
	// NextOffset 用于续取列表（RankRows 把它当作 offset）；
	// SessionID 是服务端上报的分页会话。
	NextOffset int `json:"next_offset,omitempty"`
	// ActivityStage 和 CanVote 是服务端上报的活动状态。
	ActivityStage int    `json:"activity_stage,omitempty"`
	CanVote       bool   `json:"can_vote,omitempty"`
	SessionID     string `json:"session_id,omitempty"`
}

// FilterRow 是短剧筛选页的一个选择行。Key 是该行的 select_items 键（genre、
// category_dim_theme、category_dim_role、category_dim_epoch、sort、gender、
// online_time、creation_status）；选项的 Value 是查询所选的值。
// 有哪些行取决于 genre。
type FilterRow struct {
	Key     string         `json:"key"`
	Name    string         `json:"name"`
	Options []FilterOption `json:"options"`
}

// ShortFilterPage 是符合筛选条件的短剧的一页。
type ShortFilterPage struct {
	Items   []SeriesSummary `json:"items"`
	HasMore bool            `json:"has_more"`
	// Cursor 用于续取列表；它携带已展示的短剧，服务端会跳过它们。
	Cursor string `json:"cursor,omitempty"`
}

// RankBoard 是某个榜单内容下的一个榜（推荐榜、热播榜……）。
type RankBoard struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Selected bool   `json:"selected,omitempty"`
}

// RankContent 是短剧榜单页的一个内容
// （全部、真人剧、漫剧、AI剧、系列剧、演员），带有自己的各个榜。
type RankContent struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Selected bool        `json:"selected,omitempty"`
	Boards   []RankBoard `json:"boards"`
}

// ShortRanking 是短剧榜单的一页。
type ShortRanking struct {
	// Contents 是该页的目录：每个内容及其各个榜。
	Contents []RankContent `json:"contents"`
	// Content 和 Board 是本页对应的 ID。
	Content string `json:"content"`
	Board   string `json:"board"`
	// Version 是服务端上报的榜单版本。
	Version string     `json:"version,omitempty"`
	Items   []RankItem `json:"items"`
	HasMore bool       `json:"has_more"`
	Cursor  string     `json:"cursor,omitempty"`
}
