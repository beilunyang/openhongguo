// 由 go run ./bindings/mobile/gen 生成，请勿手工修改。

import Foundation

/// Album 是多季的合集，来自 /novel/player/album_detail/v1/。
public struct Album: Codable, Hashable, Sendable {
    public var id: String
    public var title: String
    public var description: String
    public var coverUrl: String
    public var coverExpiresAt: String
    public var seasonCount: Int
    public var seasons: [Season]

    public init(id: String = "", title: String = "", description: String = "", coverUrl: String = "", coverExpiresAt: String = "", seasonCount: Int = 0, seasons: [Season] = []) {
        self.id = id
        self.title = title
        self.description = description
        self.coverUrl = coverUrl
        self.coverExpiresAt = coverExpiresAt
        self.seasonCount = seasonCount
        self.seasons = seasons
    }

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case description
        case coverUrl = "cover_url"
        case coverExpiresAt = "cover_expires_at"
        case seasonCount = "season_count"
        case seasons
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try c.decodeIfPresent(String.self, forKey: .id) ?? ""
        self.title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        self.description = try c.decodeIfPresent(String.self, forKey: .description) ?? ""
        self.coverUrl = try c.decodeIfPresent(String.self, forKey: .coverUrl) ?? ""
        self.coverExpiresAt = try c.decodeIfPresent(String.self, forKey: .coverExpiresAt) ?? ""
        self.seasonCount = try c.decodeIfPresent(Int.self, forKey: .seasonCount) ?? 0
        self.seasons = try c.decodeIfPresent([Season].self, forKey: .seasons) ?? []
    }
}

/// CategoryPage 是一个分类落地页及其筛选项。
public struct CategoryPage: Codable, Hashable, Sendable {
    public var categoryId: String
    public var categoryName: String
    public var description: String
    public var filters: [FilterGroup]
    /// Tabs 是受众标签页（全部、男生、女生），每个 Value 是标签页类型编号；
    /// DefaultTab 是页面打开时所在的标签页类型。
    public var tabs: [FilterOption]
    public var defaultTab: Int
    public var items: [SeriesSummary]
    public var hasMore: Bool
    /// NextOffset 是下一页的偏移量，即 LandingOptions.Cursor 的值。
    public var nextOffset: Int

    public init(categoryId: String = "", categoryName: String = "", description: String = "", filters: [FilterGroup] = [], tabs: [FilterOption] = [], defaultTab: Int = 0, items: [SeriesSummary] = [], hasMore: Bool = false, nextOffset: Int = 0) {
        self.categoryId = categoryId
        self.categoryName = categoryName
        self.description = description
        self.filters = filters
        self.tabs = tabs
        self.defaultTab = defaultTab
        self.items = items
        self.hasMore = hasMore
        self.nextOffset = nextOffset
    }

    enum CodingKeys: String, CodingKey {
        case categoryId = "category_id"
        case categoryName = "category_name"
        case description
        case filters
        case tabs
        case defaultTab = "default_tab"
        case items
        case hasMore = "has_more"
        case nextOffset = "next_offset"
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.categoryId = try c.decodeIfPresent(String.self, forKey: .categoryId) ?? ""
        self.categoryName = try c.decodeIfPresent(String.self, forKey: .categoryName) ?? ""
        self.description = try c.decodeIfPresent(String.self, forKey: .description) ?? ""
        self.filters = try c.decodeIfPresent([FilterGroup].self, forKey: .filters) ?? []
        self.tabs = try c.decodeIfPresent([FilterOption].self, forKey: .tabs) ?? []
        self.defaultTab = try c.decodeIfPresent(Int.self, forKey: .defaultTab) ?? 0
        self.items = try c.decodeIfPresent([SeriesSummary].self, forKey: .items) ?? []
        self.hasMore = try c.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        self.nextOffset = try c.decodeIfPresent(Int.self, forKey: .nextOffset) ?? 0
    }
}

/// ContentKey 是受保护播放模型的 CENC（cenc-aes-ctr）内容密钥，
/// 由 sdk.ContentKey 还原。
public struct ContentKey: Codable, Hashable, Sendable {
    /// Hex 是 16 字节 AES 密钥的小写十六进制形式，
    /// 即 ffmpeg 的 -decryption_key 和播放器的 ClearKey 配置接受的形式。
    public var key: String
    /// Version 是播放器在密钥旁读取的方案标记。
    public var version: String

    public init(key: String = "", version: String = "") {
        self.key = key
        self.version = version
    }

    enum CodingKeys: String, CodingKey {
        case key
        case version
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.key = try c.decodeIfPresent(String.self, forKey: .key) ?? ""
        self.version = try c.decodeIfPresent(String.self, forKey: .version) ?? ""
    }
}

/// ContentType 是视频的 content_type，
/// 取自 com.dragon.read.rpc.model.VideoContentType。
/// 官方 App 的枚举混用小数值和从 1000 起的一段数值；这里两者都复现。
public struct ContentType: RawRepresentable, Codable, Hashable, Sendable {
    public var rawValue: Int

    public init(rawValue: Int) {
        self.rawValue = rawValue
    }

    public static let unknown = ContentType(rawValue: 0)
    /// 真人短剧
    public static let shortSeries = ContentType(rawValue: 1)
    /// 场景剧
    public static let scenePlay = ContentType(rawValue: 2)
    /// 专家荐书
    public static let expertRecommend = ContentType(rawValue: 3)
    /// PPT 荐书
    public static let pptRecommend = ContentType(rawValue: 4)
    /// 电影
    public static let movie = ContentType(rawValue: 5)
    /// 电视剧
    public static let telePlay = ContentType(rawValue: 6)
    /// 短剧
    public static let shortPlay = ContentType(rawValue: 7)
    /// 投放素材
    public static let dropMaterial = ContentType(rawValue: 1000)
    /// PUGC
    public static let pugc = ContentType(rawValue: 1001)
    /// 虚拍短剧
    public static let unrealShortPlay = ContentType(rawValue: 1002)
    /// 合集
    public static let album = ContentType(rawValue: 1003)
    /// 漫剧（动态漫）
    public static let motionComic = ContentType(rawValue: 1004)
}

/// Device 是匿名身份出示的手机。注册请求体、
/// 每个请求的设备参数和 User-Agent 都描述它，所以一个设备 id 始终对应一个机型。
public struct Device: Codable, Hashable, Sendable {
    /// Brand 是 device_brand，也是制造商（"Xiaomi"）；Model 是机型代号，
    /// 作为 device_type、device_model 发送，也写进 User-Agent（"23127PN0CC"）。
    public var brand: String
    public var model: String
    /// OSVersion 是 Android 版本（"14"），OSAPI 是其 API 级别（34），
    /// Build 是 User-Agent 携带的构建 id（"UKQ1.230804.001"）。
    public var osVersion: String
    public var osApi: Int
    public var build: String
    /// Resolution 是 "height*width"，单位是像素；DPI 是屏幕密度，
    /// 单位是每英寸点数。
    public var resolution: String
    public var dpi: Int
    /// Pad 让设备以平板身份出现（dragon_device_type=pad）。
    public var pad: Bool
    /// Channel 是 App 构建上报的分发渠道（"xiaomi_8662_64" 来自小米应用商店）。
    public var channel: String

    public init(brand: String = "", model: String = "", osVersion: String = "", osApi: Int = 0, build: String = "", resolution: String = "", dpi: Int = 0, pad: Bool = false, channel: String = "") {
        self.brand = brand
        self.model = model
        self.osVersion = osVersion
        self.osApi = osApi
        self.build = build
        self.resolution = resolution
        self.dpi = dpi
        self.pad = pad
        self.channel = channel
    }

    enum CodingKeys: String, CodingKey {
        case brand
        case model
        case osVersion = "os_version"
        case osApi = "os_api"
        case build
        case resolution
        case dpi
        case pad
        case channel
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.brand = try c.decodeIfPresent(String.self, forKey: .brand) ?? ""
        self.model = try c.decodeIfPresent(String.self, forKey: .model) ?? ""
        self.osVersion = try c.decodeIfPresent(String.self, forKey: .osVersion) ?? ""
        self.osApi = try c.decodeIfPresent(Int.self, forKey: .osApi) ?? 0
        self.build = try c.decodeIfPresent(String.self, forKey: .build) ?? ""
        self.resolution = try c.decodeIfPresent(String.self, forKey: .resolution) ?? ""
        self.dpi = try c.decodeIfPresent(Int.self, forKey: .dpi) ?? 0
        self.pad = try c.decodeIfPresent(Bool.self, forKey: .pad) ?? false
        self.channel = try c.decodeIfPresent(String.self, forKey: .channel) ?? ""
    }
}

/// Episode 是一个公开列出的分集。Number 对调用方从 1 开始。
public struct Episode: Codable, Hashable, Sendable {
    public var id: String
    public var seriesId: String
    public var number: Int
    public var title: String
    /// 服务端禁止播放该分集时，Playable 为 false（video_list[].disable_play）。
    public var playable: Bool
    /// Vertical 标记竖屏画幅的分集。
    public var vertical: Bool
    /// IsNew 标记本次更新中发布的分集。
    public var isNew: Bool
    /// CoverURL 是分集剧照；CoverExpiresAt 是它的签名过期时间。
    public var coverUrl: String
    public var coverExpiresAt: String
    /// ContentType 与 Series.ContentType 是同一枚举。
    public var contentType: ContentType
    /// CommentCount 和 DiggCount 是分集的计数。
    public var commentCount: Int
    public var diggCount: Int

    public init(id: String = "", seriesId: String = "", number: Int = 0, title: String = "", playable: Bool = false, vertical: Bool = false, isNew: Bool = false, coverUrl: String = "", coverExpiresAt: String = "", contentType: ContentType = ContentType(rawValue: 0), commentCount: Int = 0, diggCount: Int = 0) {
        self.id = id
        self.seriesId = seriesId
        self.number = number
        self.title = title
        self.playable = playable
        self.vertical = vertical
        self.isNew = isNew
        self.coverUrl = coverUrl
        self.coverExpiresAt = coverExpiresAt
        self.contentType = contentType
        self.commentCount = commentCount
        self.diggCount = diggCount
    }

    enum CodingKeys: String, CodingKey {
        case id
        case seriesId = "series_id"
        case number
        case title
        case playable
        case vertical
        case isNew = "is_new"
        case coverUrl = "cover_url"
        case coverExpiresAt = "cover_expires_at"
        case contentType = "content_type"
        case commentCount = "comment_count"
        case diggCount = "digg_count"
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try c.decodeIfPresent(String.self, forKey: .id) ?? ""
        self.seriesId = try c.decodeIfPresent(String.self, forKey: .seriesId) ?? ""
        self.number = try c.decodeIfPresent(Int.self, forKey: .number) ?? 0
        self.title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        self.playable = try c.decodeIfPresent(Bool.self, forKey: .playable) ?? false
        self.vertical = try c.decodeIfPresent(Bool.self, forKey: .vertical) ?? false
        self.isNew = try c.decodeIfPresent(Bool.self, forKey: .isNew) ?? false
        self.coverUrl = try c.decodeIfPresent(String.self, forKey: .coverUrl) ?? ""
        self.coverExpiresAt = try c.decodeIfPresent(String.self, forKey: .coverExpiresAt) ?? ""
        self.contentType = try c.decodeIfPresent(ContentType.self, forKey: .contentType) ?? ContentType(rawValue: 0)
        self.commentCount = try c.decodeIfPresent(Int.self, forKey: .commentCount) ?? 0
        self.diggCount = try c.decodeIfPresent(Int.self, forKey: .diggCount) ?? 0
    }
}

/// FilterGroup 是一个筛选维度（状态、排序、子分类、篇幅）。
public struct FilterGroup: Codable, Hashable, Sendable {
    /// Key 是该组设置的查询参数，例如 sort_by。
    public var key: String
    public var options: [FilterOption]

    public init(key: String = "", options: [FilterOption] = []) {
        self.key = key
        self.options = options
    }

    enum CodingKeys: String, CodingKey {
        case key
        case options
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.key = try c.decodeIfPresent(String.self, forKey: .key) ?? ""
        self.options = try c.decodeIfPresent([FilterOption].self, forKey: .options) ?? []
    }
}

/// FilterOption 是分类页上的一个可选值。
public struct FilterOption: Codable, Hashable, Sendable {
    /// Name 是显示的标签；Value 是查询选中它时发送的值。
    public var name: String
    public var value: String

    public init(name: String = "", value: String = "") {
        self.name = name
        self.value = value
    }

    enum CodingKeys: String, CodingKey {
        case name
        case value
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        self.value = try c.decodeIfPresent(String.self, forKey: .value) ?? ""
    }
}

/// FilterRow 是短剧筛选页的一个选择行。Key 是该行的 select_items 键（genre、
/// category_dim_theme、category_dim_role、category_dim_epoch、sort、gender、
/// online_time、creation_status）；选项的 Value 是查询所选的值。
/// 有哪些行取决于 genre。
public struct FilterRow: Codable, Hashable, Sendable {
    public var key: String
    public var name: String
    public var options: [FilterOption]

    public init(key: String = "", name: String = "", options: [FilterOption] = []) {
        self.key = key
        self.name = name
        self.options = options
    }

    enum CodingKeys: String, CodingKey {
        case key
        case name
        case options
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.key = try c.decodeIfPresent(String.self, forKey: .key) ?? ""
        self.name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        self.options = try c.decodeIfPresent([FilterOption].self, forKey: .options) ?? []
    }
}

/// HotBoard 是一个热榜。作品榜（短剧热搜榜、漫剧热搜榜、短剧新剧榜）填充 Items；
/// 热点话题榜填充 Topics。出现哪些榜单因调用而异。
public struct HotBoard: Codable, Hashable, Sendable {
    public var name: String
    public var items: [HotItem]
    public var topics: [HotTopic]

    public init(name: String = "", items: [HotItem] = [], topics: [HotTopic] = []) {
        self.name = name
        self.items = items
        self.topics = topics
    }

    enum CodingKeys: String, CodingKey {
        case name
        case items
        case topics
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        self.items = try c.decodeIfPresent([HotItem].self, forKey: .items) ?? []
        self.topics = try c.decodeIfPresent([HotTopic].self, forKey: .topics) ?? []
    }
}

/// HotItem 是热榜上的一部作品，按榜单顺序排列。
public struct HotItem: Codable, Hashable, Sendable {
    /// Position 是在榜单上的名次，从 1 开始。
    public var position: Int
    public var seriesId: String
    public var title: String
    public var coverUrl: String
    public var coverExpiresAt: String
    public var episodeCount: Int
    public var contentType: ContentType
    /// SearchScore 是榜单自己的数值（"355万热搜值"，有时带 "·第1季"）；
    /// Heat 是作品的总热度（"4102万热度"）。
    public var searchScore: String
    public var heat: String
    /// Tag 是角标，例如新剧。
    public var tag: String

    public init(position: Int = 0, seriesId: String = "", title: String = "", coverUrl: String = "", coverExpiresAt: String = "", episodeCount: Int = 0, contentType: ContentType = ContentType(rawValue: 0), searchScore: String = "", heat: String = "", tag: String = "") {
        self.position = position
        self.seriesId = seriesId
        self.title = title
        self.coverUrl = coverUrl
        self.coverExpiresAt = coverExpiresAt
        self.episodeCount = episodeCount
        self.contentType = contentType
        self.searchScore = searchScore
        self.heat = heat
        self.tag = tag
    }

    enum CodingKeys: String, CodingKey {
        case position
        case seriesId = "series_id"
        case title
        case coverUrl = "cover_url"
        case coverExpiresAt = "cover_expires_at"
        case episodeCount = "episode_count"
        case contentType = "content_type"
        case searchScore = "search_score"
        case heat
        case tag
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.position = try c.decodeIfPresent(Int.self, forKey: .position) ?? 0
        self.seriesId = try c.decodeIfPresent(String.self, forKey: .seriesId) ?? ""
        self.title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        self.coverUrl = try c.decodeIfPresent(String.self, forKey: .coverUrl) ?? ""
        self.coverExpiresAt = try c.decodeIfPresent(String.self, forKey: .coverExpiresAt) ?? ""
        self.episodeCount = try c.decodeIfPresent(Int.self, forKey: .episodeCount) ?? 0
        self.contentType = try c.decodeIfPresent(ContentType.self, forKey: .contentType) ?? ContentType(rawValue: 0)
        self.searchScore = try c.decodeIfPresent(String.self, forKey: .searchScore) ?? ""
        self.heat = try c.decodeIfPresent(String.self, forKey: .heat) ?? ""
        self.tag = try c.decodeIfPresent(String.self, forKey: .tag) ?? ""
    }
}

/// HotSearch 是查询前的搜索页：猜你想搜一行，以及其下的热榜。
public struct HotSearch: Codable, Hashable, Sendable {
    public var guesses: [HotItem]
    public var boards: [HotBoard]

    public init(guesses: [HotItem] = [], boards: [HotBoard] = []) {
        self.guesses = guesses
        self.boards = boards
    }

    enum CodingKeys: String, CodingKey {
        case guesses
        case boards
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.guesses = try c.decodeIfPresent([HotItem].self, forKey: .guesses) ?? []
        self.boards = try c.decodeIfPresent([HotBoard].self, forKey: .boards) ?? []
    }
}

/// HotTopic 是热点话题榜的一项。Query 是它打开的搜索。
public struct HotTopic: Codable, Hashable, Sendable {
    public var id: String
    public var title: String
    public var query: String
    public var heat: String

    public init(id: String = "", title: String = "", query: String = "", heat: String = "") {
        self.id = id
        self.title = title
        self.query = query
        self.heat = heat
    }

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case query
        case heat
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try c.decodeIfPresent(String.self, forKey: .id) ?? ""
        self.title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        self.query = try c.decodeIfPresent(String.self, forKey: .query) ?? ""
        self.heat = try c.decodeIfPresent(String.self, forKey: .heat) ?? ""
    }
}

/// InfoKind 是 secondary_infos[].data_type，
/// 取自 com.dragon.read.rpc.model.SecondaryInfoDataType。
public struct InfoKind: RawRepresentable, Codable, Hashable, Sendable {
    public var rawValue: Int

    public init(rawValue: Int) {
        self.rawValue = rawValue
    }

    /// InfoSeasonOrder 是 "第 N 季" 那一行（实测值为 0）。
    public static let seasonOrder = InfoKind(rawValue: 0)
    /// InfoRecommendReason 是推荐理由。
    public static let recommendReason = InfoKind(rawValue: 1)
    /// InfoRankScore 是榜单名次。
    public static let rankScore = InfoKind(rawValue: 2)
    /// InfoCategory 是分类标签，例如 "末世"（实测值为 3）。
    public static let category = InfoKind(rawValue: 3)
    /// InfoScore 是用户评分。
    public static let score = InfoKind(rawValue: 4)
    /// InfoAuthorName 是作者名。
    public static let authorName = InfoKind(rawValue: 8)
    /// InfoTags 是标签列表。
    public static let tags = InfoKind(rawValue: 9)
    /// InfoCreationStatus 是连载中或已完结的说明。
    public static let creationStatus = InfoKind(rawValue: 11)
    /// InfoTwoActors 是演员行 "演员·A B"（实测值为 33）。
    public static let twoActors = InfoKind(rawValue: 33)
    /// InfoPlotTag 是剧情标签。
    public static let plotTag = InfoKind(rawValue: 40)
}

/// InfoRow 是 secondary_infos 的一项：详情页上带标签的一行。
public struct InfoRow: Codable, Hashable, Sendable {
    public var kind: InfoKind
    public var text: String
    /// Schema 是该行打开的应用内链接（如果有）。
    public var schema: String

    public init(kind: InfoKind = InfoKind(rawValue: 0), text: String = "", schema: String = "") {
        self.kind = kind
        self.text = text
        self.schema = schema
    }

    enum CodingKeys: String, CodingKey {
        case kind
        case text
        case schema
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.kind = try c.decodeIfPresent(InfoKind.self, forKey: .kind) ?? InfoKind(rawValue: 0)
        self.text = try c.decodeIfPresent(String.self, forKey: .text) ?? ""
        self.schema = try c.decodeIfPresent(String.self, forKey: .schema) ?? ""
    }
}

/// NewRelease 是上新日历中的一部作品：已上线（Online），
/// 或已预告在 PublishAt 上线。
public struct NewRelease: Codable, Hashable, Sendable {
    public var seriesId: String
    public var title: String
    public var description: String
    public var coverUrl: String
    public var coverExpiresAt: String
    public var episodeCount: Int
    /// 作品已上线时 Online 为 true；PublishAt 是预告作品的上线时间。
    public var online: Bool
    public var publishAt: String
    public var categories: [String]
    /// 已上线的作品 Heat 形如 "378万热度"，预告作品形如 "159.7万人预约"；
    /// Note 是预告文案（"预计10月26日上线"）。
    public var heat: String
    public var note: String

    public init(seriesId: String = "", title: String = "", description: String = "", coverUrl: String = "", coverExpiresAt: String = "", episodeCount: Int = 0, online: Bool = false, publishAt: String = "", categories: [String] = [], heat: String = "", note: String = "") {
        self.seriesId = seriesId
        self.title = title
        self.description = description
        self.coverUrl = coverUrl
        self.coverExpiresAt = coverExpiresAt
        self.episodeCount = episodeCount
        self.online = online
        self.publishAt = publishAt
        self.categories = categories
        self.heat = heat
        self.note = note
    }

    enum CodingKeys: String, CodingKey {
        case seriesId = "series_id"
        case title
        case description
        case coverUrl = "cover_url"
        case coverExpiresAt = "cover_expires_at"
        case episodeCount = "episode_count"
        case online
        case publishAt = "publish_at"
        case categories
        case heat
        case note
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.seriesId = try c.decodeIfPresent(String.self, forKey: .seriesId) ?? ""
        self.title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        self.description = try c.decodeIfPresent(String.self, forKey: .description) ?? ""
        self.coverUrl = try c.decodeIfPresent(String.self, forKey: .coverUrl) ?? ""
        self.coverExpiresAt = try c.decodeIfPresent(String.self, forKey: .coverExpiresAt) ?? ""
        self.episodeCount = try c.decodeIfPresent(Int.self, forKey: .episodeCount) ?? 0
        self.online = try c.decodeIfPresent(Bool.self, forKey: .online) ?? false
        self.publishAt = try c.decodeIfPresent(String.self, forKey: .publishAt) ?? ""
        self.categories = try c.decodeIfPresent([String].self, forKey: .categories) ?? []
        self.heat = try c.decodeIfPresent(String.self, forKey: .heat) ?? ""
        self.note = try c.decodeIfPresent(String.self, forKey: .note) ?? ""
    }
}

/// NewReleasePage 是上新日历中的一天。
public struct NewReleasePage: Codable, Hashable, Sendable {
    /// Date 是本次返回的日期（YYYYMMDD）；Dates 是日历提供的日期，
    /// 即今天前后各一周。
    public var date: String
    public var dates: [String]
    public var items: [NewRelease]
    public var hasMore: Bool
    public var cursor: String

    public init(date: String = "", dates: [String] = [], items: [NewRelease] = [], hasMore: Bool = false, cursor: String = "") {
        self.date = date
        self.dates = dates
        self.items = items
        self.hasMore = hasMore
        self.cursor = cursor
    }

    enum CodingKeys: String, CodingKey {
        case date
        case dates
        case items
        case hasMore = "has_more"
        case cursor
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.date = try c.decodeIfPresent(String.self, forKey: .date) ?? ""
        self.dates = try c.decodeIfPresent([String].self, forKey: .dates) ?? []
        self.items = try c.decodeIfPresent([NewRelease].self, forKey: .items) ?? []
        self.hasMore = try c.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        self.cursor = try c.decodeIfPresent(String.self, forKey: .cursor) ?? ""
    }
}

/// Novel 是短剧改编自的小说。
public struct Novel: Codable, Hashable, Sendable {
    public var bookId: String
    /// Title 是小说本身的名称；ShortTitle 是它与短剧共用的名称。
    public var title: String
    public var shortTitle: String
    public var status: NovelStatus
    public var tags: [String]
    public var reads: Int
    public var coverUrl: String
    public var coverExpiresAt: String
    /// ShowAtEpisode 是播放器推荐该小说时所在的集数。
    public var showAtEpisode: Int

    public init(bookId: String = "", title: String = "", shortTitle: String = "", status: NovelStatus = NovelStatus(rawValue: 0), tags: [String] = [], reads: Int = 0, coverUrl: String = "", coverExpiresAt: String = "", showAtEpisode: Int = 0) {
        self.bookId = bookId
        self.title = title
        self.shortTitle = shortTitle
        self.status = status
        self.tags = tags
        self.reads = reads
        self.coverUrl = coverUrl
        self.coverExpiresAt = coverExpiresAt
        self.showAtEpisode = showAtEpisode
    }

    enum CodingKeys: String, CodingKey {
        case bookId = "book_id"
        case title
        case shortTitle = "short_title"
        case status
        case tags
        case reads
        case coverUrl = "cover_url"
        case coverExpiresAt = "cover_expires_at"
        case showAtEpisode = "show_at_episode"
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.bookId = try c.decodeIfPresent(String.self, forKey: .bookId) ?? ""
        self.title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        self.shortTitle = try c.decodeIfPresent(String.self, forKey: .shortTitle) ?? ""
        self.status = try c.decodeIfPresent(NovelStatus.self, forKey: .status) ?? NovelStatus(rawValue: 0)
        self.tags = try c.decodeIfPresent([String].self, forKey: .tags) ?? []
        self.reads = try c.decodeIfPresent(Int.self, forKey: .reads) ?? 0
        self.coverUrl = try c.decodeIfPresent(String.self, forKey: .coverUrl) ?? ""
        self.coverExpiresAt = try c.decodeIfPresent(String.self, forKey: .coverExpiresAt) ?? ""
        self.showAtEpisode = try c.decodeIfPresent(Int.self, forKey: .showAtEpisode) ?? 0
    }
}

/// NovelStatus 是客户端的 NovelCreationStatus；未给出时为 -1。
public struct NovelStatus: RawRepresentable, Codable, Hashable, Sendable {
    public var rawValue: Int

    public init(rawValue: Int) {
        self.rawValue = rawValue
    }

    /// 已完结
    public static let finished = NovelStatus(rawValue: 0)
    /// 连载中
    public static let ongoing = NovelStatus(rawValue: 1)
    /// 今日更新
    public static let updatedToday = NovelStatus(rawValue: 3)
}

/// PayType 是 pay_info.pay_type，取自 com.dragon.read.rpc.model.VideoPayType。
public struct PayType: RawRepresentable, Codable, Hashable, Sendable {
    public var rawValue: Int

    public init(rawValue: Int) {
        self.rawValue = rawValue
    }

    /// PayFree 表示无需账号即可观看。
    public static let free = PayType(rawValue: 0)
    /// PayVip 表示需要订阅。
    public static let vip = PayType(rawValue: 1)
    /// PaySingle 表示需要单次购买。
    public static let single = PayType(rawValue: 2)
}

/// Person 是演员或角色的主页。
public struct Person: Codable, Hashable, Sendable {
    public var userId: String
    public var name: String
    public var description: String
    public var avatarUrl: String
    /// Tags 是主页上的头衔（"女演员"、"虚拟人物"）；
    /// Labels 是名字下方的小徽标（"水瓶座"）。
    public var tags: [String]
    public var labels: [String]
    /// Followers 是粉丝数；Likes 是获赞数。
    public var followers: Int
    public var likes: Int

    public init(userId: String = "", name: String = "", description: String = "", avatarUrl: String = "", tags: [String] = [], labels: [String] = [], followers: Int = 0, likes: Int = 0) {
        self.userId = userId
        self.name = name
        self.description = description
        self.avatarUrl = avatarUrl
        self.tags = tags
        self.labels = labels
        self.followers = followers
        self.likes = likes
    }

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case name
        case description
        case avatarUrl = "avatar_url"
        case tags
        case labels
        case followers
        case likes
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.userId = try c.decodeIfPresent(String.self, forKey: .userId) ?? ""
        self.name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        self.description = try c.decodeIfPresent(String.self, forKey: .description) ?? ""
        self.avatarUrl = try c.decodeIfPresent(String.self, forKey: .avatarUrl) ?? ""
        self.tags = try c.decodeIfPresent([String].self, forKey: .tags) ?? []
        self.labels = try c.decodeIfPresent([String].self, forKey: .labels) ?? []
        self.followers = try c.decodeIfPresent(Int.self, forKey: .followers) ?? 0
        self.likes = try c.decodeIfPresent(Int.self, forKey: .likes) ?? 0
    }
}

/// PlatformType 是 video_platform，
/// 取自 com.dragon.read.rpc.model.VideoPlatformType。
public struct PlatformType: RawRepresentable, Codable, Hashable, Sendable {
    public var rawValue: Int

    public init(rawValue: Int) {
        self.rawValue = rawValue
    }

    /// 未知；有声书带此值
    public static let unknown = PlatformType(rawValue: 0)
    /// 抖音
    public static let douyin = PlatformType(rawValue: 1)
    /// UGC 视频
    public static let ugcVideo = PlatformType(rawValue: 2)
    /// 站外与自制短剧
    public static let outer = PlatformType(rawValue: 3)
    /// 西瓜视频，电影来自这里
    public static let xigua = PlatformType(rawValue: 4)
    /// Chameleon
    public static let chameleon = PlatformType(rawValue: 5)
    /// 混合来源
    public static let mixed = PlatformType(rawValue: 1024)
}

/// Playback 是一个已解析的匿名媒体模型。它描述服务端提供的内容，
/// 从不携带解密密钥或许可证材料。
public struct Playback: Codable, Hashable, Sendable {
    public var episodeId: String
    /// VideoID 是分集背后的媒体 id。
    public var videoId: String
    /// MediaType 是服务端的媒体类型，例如 "video"。
    public var mediaType: String
    public var durationSeconds: Int
    /// ExpiresAt 是流 URL 失效的时间，单位是 unix 秒。
    public var expiresAt: Int
    /// Protected 标记加密的分集；参见 Stream.Protected。
    public var protected: Bool
    public var streams: [Stream]
    /// AudioURL 是分段（DASH）作品（如长片电影）的独立音轨。
    /// 单文件的表示形式留空。
    public var audioUrl: String

    public init(episodeId: String = "", videoId: String = "", mediaType: String = "", durationSeconds: Int = 0, expiresAt: Int = 0, protected: Bool = false, streams: [Stream] = [], audioUrl: String = "") {
        self.episodeId = episodeId
        self.videoId = videoId
        self.mediaType = mediaType
        self.durationSeconds = durationSeconds
        self.expiresAt = expiresAt
        self.protected = protected
        self.streams = streams
        self.audioUrl = audioUrl
    }

    enum CodingKeys: String, CodingKey {
        case episodeId = "episode_id"
        case videoId = "video_id"
        case mediaType = "media_type"
        case durationSeconds = "duration_seconds"
        case expiresAt = "expires_at"
        case protected
        case streams
        case audioUrl = "audio_url"
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.episodeId = try c.decodeIfPresent(String.self, forKey: .episodeId) ?? ""
        self.videoId = try c.decodeIfPresent(String.self, forKey: .videoId) ?? ""
        self.mediaType = try c.decodeIfPresent(String.self, forKey: .mediaType) ?? ""
        self.durationSeconds = try c.decodeIfPresent(Int.self, forKey: .durationSeconds) ?? 0
        self.expiresAt = try c.decodeIfPresent(Int.self, forKey: .expiresAt) ?? 0
        self.protected = try c.decodeIfPresent(Bool.self, forKey: .protected) ?? false
        self.streams = try c.decodeIfPresent([Stream].self, forKey: .streams) ?? []
        self.audioUrl = try c.decodeIfPresent(String.self, forKey: .audioUrl) ?? ""
    }
}

/// RankBoard 是某个榜单内容下的一个榜（推荐榜、热播榜……）。
public struct RankBoard: Codable, Hashable, Sendable {
    public var id: String
    public var name: String
    public var selected: Bool

    public init(id: String = "", name: String = "", selected: Bool = false) {
        self.id = id
        self.name = name
        self.selected = selected
    }

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case selected
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try c.decodeIfPresent(String.self, forKey: .id) ?? ""
        self.name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        self.selected = try c.decodeIfPresent(Bool.self, forKey: .selected) ?? false
    }
}

/// RankChange 是榜单行的变动（客户端中的 RankChangeType）。
public struct RankChange: Codable, Hashable, Sendable {
    public var kind: RankChangeKind
    /// Places 是该行变动的位数；新上榜或不变的行为 0。
    public var places: Int

    public init(kind: RankChangeKind = RankChangeKind(rawValue: 0), places: Int = 0) {
        self.kind = kind
        self.places = places
    }

    enum CodingKeys: String, CodingKey {
        case kind
        case places
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.kind = try c.decodeIfPresent(RankChangeKind.self, forKey: .kind) ?? RankChangeKind(rawValue: 0)
        self.places = try c.decodeIfPresent(Int.self, forKey: .places) ?? 0
    }
}

/// RankChangeKind 的取值即客户端的 RankChangeType。
public struct RankChangeKind: RawRepresentable, Codable, Hashable, Sendable {
    public var rawValue: Int

    public init(rawValue: Int) {
        self.rawValue = rawValue
    }

    /// 新上榜
    public static let new = RankChangeKind(rawValue: 0)
    /// 上升 RankChange.Places 位
    public static let up = RankChangeKind(rawValue: 1)
    /// 下降 RankChange.Places 位
    public static let down = RankChangeKind(rawValue: 2)
    /// 名次不变
    public static let unchanged = RankChangeKind(rawValue: 3)
}

/// RankContent 是短剧榜单页的一个内容
/// （全部、真人剧、漫剧、AI剧、系列剧、演员），带有自己的各个榜。
public struct RankContent: Codable, Hashable, Sendable {
    public var id: String
    public var name: String
    public var selected: Bool
    public var boards: [RankBoard]

    public init(id: String = "", name: String = "", selected: Bool = false, boards: [RankBoard] = []) {
        self.id = id
        self.name = name
        self.selected = selected
        self.boards = boards
    }

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case selected
        case boards
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try c.decodeIfPresent(String.self, forKey: .id) ?? ""
        self.name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        self.selected = try c.decodeIfPresent(Bool.self, forKey: .selected) ?? false
        self.boards = try c.decodeIfPresent([RankBoard].self, forKey: .boards) ?? []
    }
}

/// RankItem 是榜单的一行。
public struct RankItem: Codable, Hashable, Sendable {
    /// Rank 是服务端给该行的名次（参见 Position）。
    public var rank: Int
    /// Votes 和 Kind 用于活动榜单行：分别是票数和服务端的条目类型，原样透传。
    public var votes: Int
    public var kind: Int
    /// Actor 在书籍榜单上是作者，在活动榜单或短剧演员榜上是演员名。
    public var actor: String
    public var seriesId: String
    public var title: String
    public var coverUrl: String
    /// Score 是显示的热度（"2550万热度"、"998万推荐"、"平台推荐"）。
    public var score: String
    /// EpisodeCount 由短剧榜单填充。
    public var episodeCount: Int
    /// Position 是该行在短剧榜单页上的名次，从 1 开始。Rank 是服务端自己的编号，
    /// 在推广行前后会重复，在推荐榜上是个性化的。
    public var position: Int
    /// Promoted 标记服务端插入短剧榜单的行：它显示 "平台推荐" 而非热度，
    /// 名次照抄下一行（实测）。
    public var promoted: Bool
    /// ActorID、AvatarURL、Intro 和 Details 用于短剧榜单的演员行，
    /// 该行的 Actor 是名字，Score 是人气。Intro 列出作品；
    /// Details 存放 "67部作品" 这样的行。
    public var actorId: String
    /// UserID 是演员的账号，即 ActorProfile 和 ActorWorks 接受的 id。
    public var userId: String
    public var avatarUrl: String
    public var intro: String
    public var details: [String]
    /// Change 是该行相对上一期榜单的变动（如果有）。
    public var change: RankChange?

    public init(rank: Int = 0, votes: Int = 0, kind: Int = 0, actor: String = "", seriesId: String = "", title: String = "", coverUrl: String = "", score: String = "", episodeCount: Int = 0, position: Int = 0, promoted: Bool = false, actorId: String = "", userId: String = "", avatarUrl: String = "", intro: String = "", details: [String] = [], change: RankChange? = nil) {
        self.rank = rank
        self.votes = votes
        self.kind = kind
        self.actor = actor
        self.seriesId = seriesId
        self.title = title
        self.coverUrl = coverUrl
        self.score = score
        self.episodeCount = episodeCount
        self.position = position
        self.promoted = promoted
        self.actorId = actorId
        self.userId = userId
        self.avatarUrl = avatarUrl
        self.intro = intro
        self.details = details
        self.change = change
    }

    enum CodingKeys: String, CodingKey {
        case rank
        case votes
        case kind
        case actor
        case seriesId = "series_id"
        case title
        case coverUrl = "cover_url"
        case score
        case episodeCount = "episode_count"
        case position
        case promoted
        case actorId = "actor_id"
        case userId = "user_id"
        case avatarUrl = "avatar_url"
        case intro
        case details
        case change
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.rank = try c.decodeIfPresent(Int.self, forKey: .rank) ?? 0
        self.votes = try c.decodeIfPresent(Int.self, forKey: .votes) ?? 0
        self.kind = try c.decodeIfPresent(Int.self, forKey: .kind) ?? 0
        self.actor = try c.decodeIfPresent(String.self, forKey: .actor) ?? ""
        self.seriesId = try c.decodeIfPresent(String.self, forKey: .seriesId) ?? ""
        self.title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        self.coverUrl = try c.decodeIfPresent(String.self, forKey: .coverUrl) ?? ""
        self.score = try c.decodeIfPresent(String.self, forKey: .score) ?? ""
        self.episodeCount = try c.decodeIfPresent(Int.self, forKey: .episodeCount) ?? 0
        self.position = try c.decodeIfPresent(Int.self, forKey: .position) ?? 0
        self.promoted = try c.decodeIfPresent(Bool.self, forKey: .promoted) ?? false
        self.actorId = try c.decodeIfPresent(String.self, forKey: .actorId) ?? ""
        self.userId = try c.decodeIfPresent(String.self, forKey: .userId) ?? ""
        self.avatarUrl = try c.decodeIfPresent(String.self, forKey: .avatarUrl) ?? ""
        self.intro = try c.decodeIfPresent(String.self, forKey: .intro) ?? ""
        self.details = try c.decodeIfPresent([String].self, forKey: .details) ?? []
        self.change = try c.decodeIfPresent(RankChange.self, forKey: .change)
    }
}

/// RankList 是活动的一个榜单频道。
public struct RankList: Codable, Hashable, Sendable {
    public var id: String
    public var name: String
    /// Type 和 Size 是服务端的榜单类型和长度，原样透传。
    public var type: Int
    public var size: Int

    public init(id: String = "", name: String = "", type: Int = 0, size: Int = 0) {
        self.id = id
        self.name = name
        self.type = type
        self.size = size
    }

    enum CodingKeys: String, CodingKey {
        case id
        case name
        case type
        case size
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try c.decodeIfPresent(String.self, forKey: .id) ?? ""
        self.name = try c.decodeIfPresent(String.self, forKey: .name) ?? ""
        self.type = try c.decodeIfPresent(Int.self, forKey: .type) ?? 0
        self.size = try c.decodeIfPresent(Int.self, forKey: .size) ?? 0
    }
}

/// Ranking 是一个活动榜单页。
public struct Ranking: Codable, Hashable, Sendable {
    public var listName: String
    /// Lists 是该活动的其他榜单频道。
    public var lists: [RankList]
    public var items: [RankItem]
    public var hasMore: Bool
    /// NextOffset 用于续取列表（RankRows 把它当作 offset）；
    /// SessionID 是服务端上报的分页会话。
    public var nextOffset: Int
    /// ActivityStage 和 CanVote 是服务端上报的活动状态。
    public var activityStage: Int
    public var canVote: Bool
    public var sessionId: String

    public init(listName: String = "", lists: [RankList] = [], items: [RankItem] = [], hasMore: Bool = false, nextOffset: Int = 0, activityStage: Int = 0, canVote: Bool = false, sessionId: String = "") {
        self.listName = listName
        self.lists = lists
        self.items = items
        self.hasMore = hasMore
        self.nextOffset = nextOffset
        self.activityStage = activityStage
        self.canVote = canVote
        self.sessionId = sessionId
    }

    enum CodingKeys: String, CodingKey {
        case listName = "list_name"
        case lists
        case items
        case hasMore = "has_more"
        case nextOffset = "next_offset"
        case activityStage = "activity_stage"
        case canVote = "can_vote"
        case sessionId = "session_id"
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.listName = try c.decodeIfPresent(String.self, forKey: .listName) ?? ""
        self.lists = try c.decodeIfPresent([RankList].self, forKey: .lists) ?? []
        self.items = try c.decodeIfPresent([RankItem].self, forKey: .items) ?? []
        self.hasMore = try c.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        self.nextOffset = try c.decodeIfPresent(Int.self, forKey: .nextOffset) ?? 0
        self.activityStage = try c.decodeIfPresent(Int.self, forKey: .activityStage) ?? 0
        self.canVote = try c.decodeIfPresent(Bool.self, forKey: .canVote) ?? false
        self.sessionId = try c.decodeIfPresent(String.self, forKey: .sessionId) ?? ""
    }
}

/// Recommend 是一部短剧的相关视频单元：包含各行、单元标题，
/// 以及客户端为后续请求保留的分页会话。
public struct Recommend: Codable, Hashable, Sendable {
    public var cellTitle: String
    public var sessionId: String
    public var items: [SeriesSummary]

    public init(cellTitle: String = "", sessionId: String = "", items: [SeriesSummary] = []) {
        self.cellTitle = cellTitle
        self.sessionId = sessionId
        self.items = items
    }

    enum CodingKeys: String, CodingKey {
        case cellTitle = "cell_title"
        case sessionId = "session_id"
        case items
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.cellTitle = try c.decodeIfPresent(String.self, forKey: .cellTitle) ?? ""
        self.sessionId = try c.decodeIfPresent(String.self, forKey: .sessionId) ?? ""
        self.items = try c.decodeIfPresent([SeriesSummary].self, forKey: .items) ?? []
    }
}

/// Season 是合集（多季合集）中的一项。
public struct Season: Codable, Hashable, Sendable {
    public var seriesId: String
    public var title: String
    /// Index 是季序号，从 1 开始。
    public var index: Int
    public var coverUrl: String
    public var episodeCount: Int
    public var status: SeriesStatus

    public init(seriesId: String = "", title: String = "", index: Int = 0, coverUrl: String = "", episodeCount: Int = 0, status: SeriesStatus = SeriesStatus(rawValue: 0)) {
        self.seriesId = seriesId
        self.title = title
        self.index = index
        self.coverUrl = coverUrl
        self.episodeCount = episodeCount
        self.status = status
    }

    enum CodingKeys: String, CodingKey {
        case seriesId = "series_id"
        case title
        case index
        case coverUrl = "cover_url"
        case episodeCount = "episode_count"
        case status
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.seriesId = try c.decodeIfPresent(String.self, forKey: .seriesId) ?? ""
        self.title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        self.index = try c.decodeIfPresent(Int.self, forKey: .index) ?? 0
        self.coverUrl = try c.decodeIfPresent(String.self, forKey: .coverUrl) ?? ""
        self.episodeCount = try c.decodeIfPresent(Int.self, forKey: .episodeCount) ?? 0
        self.status = try c.decodeIfPresent(SeriesStatus.self, forKey: .status) ?? SeriesStatus(rawValue: 0)
    }
}

/// Series 是一部短剧的详情视图。
public struct Series: Codable, Hashable, Sendable {
    public var id: String
    public var title: String
    public var description: String
    public var coverUrl: String
    /// CoverExpiresAt 是 CoverURL 的签名过期时间（x-expires）；
    /// 过期后要从 API 重新获取封面。
    public var coverExpiresAt: String
    public var episodeCount: Int
    /// Author、Category 和 Words 由分类页的行填充。
    public var author: String
    public var category: String
    public var words: Int
    /// Status 是发布状态；PayType 是付费墙状态。
    public var status: SeriesStatus
    public var payType: PayType
    /// ContentType 和 Platform 描述作品是什么、来自哪里（短剧、电影、漫剧……）。
    public var contentType: ContentType
    public var platform: PlatformType
    /// Heat 是原始热度分；App 将其显示为 "红果热度值N万"。
    public var heat: Int
    /// PlayCount 是短剧的总播放量（series_play_cnt）。
    public var playCount: Int
    /// FollowCount 和 DiggCount 是标题旁显示的计数。
    public var followCount: Int
    public var diggCount: Int
    /// CreatedAt 是发布时间（create_time，unix 秒）。
    public var createdAt: String
    /// RightText 是集数旁的角标（"更新至12集"）。
    public var rightText: String
    /// Subtitles 是标题下方的简短状态行。
    public var subtitles: [String]
    /// AgeGate 对不受限的作品为 0。
    public var ageGate: Int
    /// IsMultiSeason 标记属于多季合集的作品。
    public var isMultiSeason: Bool
    /// SeasonIndex 是该合集内的季序号，从 1 开始。
    public var seasonIndex: Int
    /// RelatedAlbumID 是各季共用的合集 id。
    public var relatedAlbumId: String
    /// RelatedNovelBookID 是短剧改编的小说（如已公开）。
    public var relatedNovelBookId: String
    /// Novel 是详情页链接的原著小说（video_relate_book），服务端给出时才有。
    public var novel: Novel?
    /// Rows 是详情页的次要信息行（季、分类、演员）。
    public var rows: [InfoRow]

    public init(id: String = "", title: String = "", description: String = "", coverUrl: String = "", coverExpiresAt: String = "", episodeCount: Int = 0, author: String = "", category: String = "", words: Int = 0, status: SeriesStatus = SeriesStatus(rawValue: 0), payType: PayType = PayType(rawValue: 0), contentType: ContentType = ContentType(rawValue: 0), platform: PlatformType = PlatformType(rawValue: 0), heat: Int = 0, playCount: Int = 0, followCount: Int = 0, diggCount: Int = 0, createdAt: String = "", rightText: String = "", subtitles: [String] = [], ageGate: Int = 0, isMultiSeason: Bool = false, seasonIndex: Int = 0, relatedAlbumId: String = "", relatedNovelBookId: String = "", novel: Novel? = nil, rows: [InfoRow] = []) {
        self.id = id
        self.title = title
        self.description = description
        self.coverUrl = coverUrl
        self.coverExpiresAt = coverExpiresAt
        self.episodeCount = episodeCount
        self.author = author
        self.category = category
        self.words = words
        self.status = status
        self.payType = payType
        self.contentType = contentType
        self.platform = platform
        self.heat = heat
        self.playCount = playCount
        self.followCount = followCount
        self.diggCount = diggCount
        self.createdAt = createdAt
        self.rightText = rightText
        self.subtitles = subtitles
        self.ageGate = ageGate
        self.isMultiSeason = isMultiSeason
        self.seasonIndex = seasonIndex
        self.relatedAlbumId = relatedAlbumId
        self.relatedNovelBookId = relatedNovelBookId
        self.novel = novel
        self.rows = rows
    }

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case description
        case coverUrl = "cover_url"
        case coverExpiresAt = "cover_expires_at"
        case episodeCount = "episode_count"
        case author
        case category
        case words
        case status
        case payType = "pay_type"
        case contentType = "content_type"
        case platform
        case heat
        case playCount = "play_count"
        case followCount = "follow_count"
        case diggCount = "digg_count"
        case createdAt = "created_at"
        case rightText = "right_text"
        case subtitles
        case ageGate = "age_gate"
        case isMultiSeason = "is_multi_season"
        case seasonIndex = "season_index"
        case relatedAlbumId = "related_album_id"
        case relatedNovelBookId = "related_novel_book_id"
        case novel
        case rows
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try c.decodeIfPresent(String.self, forKey: .id) ?? ""
        self.title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        self.description = try c.decodeIfPresent(String.self, forKey: .description) ?? ""
        self.coverUrl = try c.decodeIfPresent(String.self, forKey: .coverUrl) ?? ""
        self.coverExpiresAt = try c.decodeIfPresent(String.self, forKey: .coverExpiresAt) ?? ""
        self.episodeCount = try c.decodeIfPresent(Int.self, forKey: .episodeCount) ?? 0
        self.author = try c.decodeIfPresent(String.self, forKey: .author) ?? ""
        self.category = try c.decodeIfPresent(String.self, forKey: .category) ?? ""
        self.words = try c.decodeIfPresent(Int.self, forKey: .words) ?? 0
        self.status = try c.decodeIfPresent(SeriesStatus.self, forKey: .status) ?? SeriesStatus(rawValue: 0)
        self.payType = try c.decodeIfPresent(PayType.self, forKey: .payType) ?? PayType(rawValue: 0)
        self.contentType = try c.decodeIfPresent(ContentType.self, forKey: .contentType) ?? ContentType(rawValue: 0)
        self.platform = try c.decodeIfPresent(PlatformType.self, forKey: .platform) ?? PlatformType(rawValue: 0)
        self.heat = try c.decodeIfPresent(Int.self, forKey: .heat) ?? 0
        self.playCount = try c.decodeIfPresent(Int.self, forKey: .playCount) ?? 0
        self.followCount = try c.decodeIfPresent(Int.self, forKey: .followCount) ?? 0
        self.diggCount = try c.decodeIfPresent(Int.self, forKey: .diggCount) ?? 0
        self.createdAt = try c.decodeIfPresent(String.self, forKey: .createdAt) ?? ""
        self.rightText = try c.decodeIfPresent(String.self, forKey: .rightText) ?? ""
        self.subtitles = try c.decodeIfPresent([String].self, forKey: .subtitles) ?? []
        self.ageGate = try c.decodeIfPresent(Int.self, forKey: .ageGate) ?? 0
        self.isMultiSeason = try c.decodeIfPresent(Bool.self, forKey: .isMultiSeason) ?? false
        self.seasonIndex = try c.decodeIfPresent(Int.self, forKey: .seasonIndex) ?? 0
        self.relatedAlbumId = try c.decodeIfPresent(String.self, forKey: .relatedAlbumId) ?? ""
        self.relatedNovelBookId = try c.decodeIfPresent(String.self, forKey: .relatedNovelBookId) ?? ""
        self.novel = try c.decodeIfPresent(Novel.self, forKey: .novel)
        self.rows = try c.decodeIfPresent([InfoRow].self, forKey: .rows) ?? []
    }
}

/// SeriesDetail 是一部短剧及其分集，由批量读取返回。
public struct SeriesDetail: Codable, Hashable, Sendable {
    public var series: Series?
    public var episodes: [Episode]

    public init(series: Series? = nil, episodes: [Episode] = []) {
        self.series = series
        self.episodes = episodes
    }

    enum CodingKeys: String, CodingKey {
        case series
        case episodes
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.series = try c.decodeIfPresent(Series.self, forKey: .series)
        self.episodes = try c.decodeIfPresent([Episode].self, forKey: .episodes) ?? []
    }
}

/// SeriesStatus 是短剧的发布状态，由 data.video_data.series_status 携带。
///
/// 名称取自官方 App 的枚举 com.dragon.read.rpc.model.SeriesStatus（
/// SeriesUpdating、SeriesEnd、SeriesUpdateTaday、SeriesUpdateStop）；
/// 值 0 和 1 也已在线上实测。
public struct SeriesStatus: RawRepresentable, Codable, Hashable, Sendable {
    public var rawValue: Int

    public init(rawValue: Int) {
        self.rawValue = rawValue
    }

    /// SeriesUpdating 是未完结的短剧（"更新至 N 集"）。
    public static let updating = SeriesStatus(rawValue: 0)
    /// SeriesEnd 是已完结的短剧（"全 N 集"）。
    public static let end = SeriesStatus(rawValue: 1)
    /// SeriesUpdatedToday 是枚举中的“今日更新”状态（推断）。
    public static let updatedToday = SeriesStatus(rawValue: 3)
    /// SeriesUpdateStopped 是枚举中的“停止更新”状态（推断）。
    public static let updateStopped = SeriesStatus(rawValue: 4)
}

/// SeriesSummary 是公开目录中的一行。ID 和标题是稳定字段。
public struct SeriesSummary: Codable, Hashable, Sendable {
    public var id: String
    public var title: String
    public var description: String
    public var coverUrl: String
    /// CoverExpiresAt 是 CoverURL 的签名过期时间（x-expires）；
    /// 过期后要从 API 重新获取封面。
    public var coverExpiresAt: String
    public var episodeCount: Int
    /// Author、Category 和 Words 由分类页的行填充。
    public var author: String
    public var category: String
    public var words: Int

    public init(id: String = "", title: String = "", description: String = "", coverUrl: String = "", coverExpiresAt: String = "", episodeCount: Int = 0, author: String = "", category: String = "", words: Int = 0) {
        self.id = id
        self.title = title
        self.description = description
        self.coverUrl = coverUrl
        self.coverExpiresAt = coverExpiresAt
        self.episodeCount = episodeCount
        self.author = author
        self.category = category
        self.words = words
    }

    enum CodingKeys: String, CodingKey {
        case id
        case title
        case description
        case coverUrl = "cover_url"
        case coverExpiresAt = "cover_expires_at"
        case episodeCount = "episode_count"
        case author
        case category
        case words
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.id = try c.decodeIfPresent(String.self, forKey: .id) ?? ""
        self.title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        self.description = try c.decodeIfPresent(String.self, forKey: .description) ?? ""
        self.coverUrl = try c.decodeIfPresent(String.self, forKey: .coverUrl) ?? ""
        self.coverExpiresAt = try c.decodeIfPresent(String.self, forKey: .coverExpiresAt) ?? ""
        self.episodeCount = try c.decodeIfPresent(Int.self, forKey: .episodeCount) ?? 0
        self.author = try c.decodeIfPresent(String.self, forKey: .author) ?? ""
        self.category = try c.decodeIfPresent(String.self, forKey: .category) ?? ""
        self.words = try c.decodeIfPresent(Int.self, forKey: .words) ?? 0
    }
}

/// ShortFilterPage 是符合筛选条件的短剧的一页。
public struct ShortFilterPage: Codable, Hashable, Sendable {
    public var items: [SeriesSummary]
    public var hasMore: Bool
    /// Cursor 用于续取列表；它携带已展示的短剧，服务端会跳过它们。
    public var cursor: String

    public init(items: [SeriesSummary] = [], hasMore: Bool = false, cursor: String = "") {
        self.items = items
        self.hasMore = hasMore
        self.cursor = cursor
    }

    enum CodingKeys: String, CodingKey {
        case items
        case hasMore = "has_more"
        case cursor
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.items = try c.decodeIfPresent([SeriesSummary].self, forKey: .items) ?? []
        self.hasMore = try c.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        self.cursor = try c.decodeIfPresent(String.self, forKey: .cursor) ?? ""
    }
}

/// ShortRanking 是短剧榜单的一页。
public struct ShortRanking: Codable, Hashable, Sendable {
    /// Contents 是该页的目录：每个内容及其各个榜。
    public var contents: [RankContent]
    /// Content 和 Board 是本页对应的 ID。
    public var content: String
    public var board: String
    /// Version 是服务端上报的榜单版本。
    public var version: String
    public var items: [RankItem]
    public var hasMore: Bool
    public var cursor: String

    public init(contents: [RankContent] = [], content: String = "", board: String = "", version: String = "", items: [RankItem] = [], hasMore: Bool = false, cursor: String = "") {
        self.contents = contents
        self.content = content
        self.board = board
        self.version = version
        self.items = items
        self.hasMore = hasMore
        self.cursor = cursor
    }

    enum CodingKeys: String, CodingKey {
        case contents
        case content
        case board
        case version
        case items
        case hasMore = "has_more"
        case cursor
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.contents = try c.decodeIfPresent([RankContent].self, forKey: .contents) ?? []
        self.content = try c.decodeIfPresent(String.self, forKey: .content) ?? ""
        self.board = try c.decodeIfPresent(String.self, forKey: .board) ?? ""
        self.version = try c.decodeIfPresent(String.self, forKey: .version) ?? ""
        self.items = try c.decodeIfPresent([RankItem].self, forKey: .items) ?? []
        self.hasMore = try c.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        self.cursor = try c.decodeIfPresent(String.self, forKey: .cursor) ?? ""
    }
}

/// Stream 是媒体模型中的一个表示形式。Protected
/// 标记官方客户端在内部解密的表示形式；sdk.ContentKey 从模型的 KeyMaterial
/// 还原它的密钥。
public struct Stream: Codable, Hashable, Sendable {
    /// Definition 是分辨率标签（"720p"）；Quality 是服务端的画质标签
    /// （"normal"）。
    public var definition: String
    public var quality: String
    public var codec: String
    public var width: Int
    public var height: Int
    /// Bitrate 的单位是 bit/s；Size 是文件大小，单位是字节。
    public var bitrate: Int
    public var size: Int
    /// BackupURL 是同一文件的第二个 CDN 地址。
    public var url: String
    public var backupUrl: String
    /// HLS 标记 m3u8 播放列表；Dash 标记分段（DASH）的表示形式。
    public var hls: Bool
    public var dash: Bool
    public var protected: Bool
    /// Encryption 是受保护流的加密方案，例如 "cenc-aes-ctr"。
    public var encryption: String

    public init(definition: String = "", quality: String = "", codec: String = "", width: Int = 0, height: Int = 0, bitrate: Int = 0, size: Int = 0, url: String = "", backupUrl: String = "", hls: Bool = false, dash: Bool = false, protected: Bool = false, encryption: String = "") {
        self.definition = definition
        self.quality = quality
        self.codec = codec
        self.width = width
        self.height = height
        self.bitrate = bitrate
        self.size = size
        self.url = url
        self.backupUrl = backupUrl
        self.hls = hls
        self.dash = dash
        self.protected = protected
        self.encryption = encryption
    }

    enum CodingKeys: String, CodingKey {
        case definition
        case quality
        case codec
        case width
        case height
        case bitrate
        case size
        case url
        case backupUrl = "backup_url"
        case hls
        case dash
        case protected
        case encryption
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.definition = try c.decodeIfPresent(String.self, forKey: .definition) ?? ""
        self.quality = try c.decodeIfPresent(String.self, forKey: .quality) ?? ""
        self.codec = try c.decodeIfPresent(String.self, forKey: .codec) ?? ""
        self.width = try c.decodeIfPresent(Int.self, forKey: .width) ?? 0
        self.height = try c.decodeIfPresent(Int.self, forKey: .height) ?? 0
        self.bitrate = try c.decodeIfPresent(Int.self, forKey: .bitrate) ?? 0
        self.size = try c.decodeIfPresent(Int.self, forKey: .size) ?? 0
        self.url = try c.decodeIfPresent(String.self, forKey: .url) ?? ""
        self.backupUrl = try c.decodeIfPresent(String.self, forKey: .backupUrl) ?? ""
        self.hls = try c.decodeIfPresent(Bool.self, forKey: .hls) ?? false
        self.dash = try c.decodeIfPresent(Bool.self, forKey: .dash) ?? false
        self.protected = try c.decodeIfPresent(Bool.self, forKey: .protected) ?? false
        self.encryption = try c.decodeIfPresent(String.self, forKey: .encryption) ?? ""
    }
}

/// Suggestion 是一条搜索建议。Text 是搜索框显示的文本；
/// Keyword 是它提交的内容（直达行为内容 ID）。
public struct Suggestion: Codable, Hashable, Sendable {
    public var text: String
    public var keyword: String
    /// Kind 是服务端的行类型，原样透传。
    public var kind: Int

    public init(text: String = "", keyword: String = "", kind: Int = 0) {
        self.text = text
        self.keyword = keyword
        self.kind = kind
    }

    enum CodingKeys: String, CodingKey {
        case text
        case keyword
        case kind
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.text = try c.decodeIfPresent(String.self, forKey: .text) ?? ""
        self.keyword = try c.decodeIfPresent(String.self, forKey: .keyword) ?? ""
        self.kind = try c.decodeIfPresent(Int.self, forKey: .kind) ?? 0
    }
}

/// Work 是某人出演的一部短剧。
public struct Work: Codable, Hashable, Sendable {
    public var seriesId: String
    public var title: String
    public var description: String
    public var coverUrl: String
    public var coverExpiresAt: String
    public var episodeCount: Int
    /// Tags 是卡片上的各行，例如 "爱情" 和 "70集"。
    public var tags: [String]
    /// Heat 是短剧的原始热度分；Plays 是其播放量。
    public var heat: Int
    public var plays: Int

    public init(seriesId: String = "", title: String = "", description: String = "", coverUrl: String = "", coverExpiresAt: String = "", episodeCount: Int = 0, tags: [String] = [], heat: Int = 0, plays: Int = 0) {
        self.seriesId = seriesId
        self.title = title
        self.description = description
        self.coverUrl = coverUrl
        self.coverExpiresAt = coverExpiresAt
        self.episodeCount = episodeCount
        self.tags = tags
        self.heat = heat
        self.plays = plays
    }

    enum CodingKeys: String, CodingKey {
        case seriesId = "series_id"
        case title
        case description
        case coverUrl = "cover_url"
        case coverExpiresAt = "cover_expires_at"
        case episodeCount = "episode_count"
        case tags
        case heat
        case plays
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.seriesId = try c.decodeIfPresent(String.self, forKey: .seriesId) ?? ""
        self.title = try c.decodeIfPresent(String.self, forKey: .title) ?? ""
        self.description = try c.decodeIfPresent(String.self, forKey: .description) ?? ""
        self.coverUrl = try c.decodeIfPresent(String.self, forKey: .coverUrl) ?? ""
        self.coverExpiresAt = try c.decodeIfPresent(String.self, forKey: .coverExpiresAt) ?? ""
        self.episodeCount = try c.decodeIfPresent(Int.self, forKey: .episodeCount) ?? 0
        self.tags = try c.decodeIfPresent([String].self, forKey: .tags) ?? []
        self.heat = try c.decodeIfPresent(Int.self, forKey: .heat) ?? 0
        self.plays = try c.decodeIfPresent(Int.self, forKey: .plays) ?? 0
    }
}

/// WorkPage 是某人作品的一页（作品 › 剧作）。
public struct WorkPage: Codable, Hashable, Sendable {
    public var items: [Work]
    public var hasMore: Bool
    public var cursor: String

    public init(items: [Work] = [], hasMore: Bool = false, cursor: String = "") {
        self.items = items
        self.hasMore = hasMore
        self.cursor = cursor
    }

    enum CodingKeys: String, CodingKey {
        case items
        case hasMore = "has_more"
        case cursor
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        self.items = try c.decodeIfPresent([Work].self, forKey: .items) ?? []
        self.hasMore = try c.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
        self.cursor = try c.decodeIfPresent(String.self, forKey: .cursor) ?? ""
    }
}

/// DiscoverOptions 选择一个发现页。Channel 是书城频道名，空频道即推荐。
/// Cursor 是上一页给出的令牌。PageSize 是 cell/change 的 limit（0 表示 20，上限 50）。
public struct DiscoverOptions: Encodable, Hashable, Sendable {
    public var cursor: String?
    public var channel: String?
    public var pageSize: Int?

    public init(cursor: String? = nil, channel: String? = nil, pageSize: Int? = nil) {
        self.cursor = cursor
        self.channel = channel
        self.pageSize = pageSize
    }
}

/// SearchOptions 选择一个搜索页。Tab 取值为综合、剧集、视频、讨论、小说、
/// 用户或听书，空 tab 即综合。Cursor 是下一个偏移量。
public struct SearchOptions: Encodable, Hashable, Sendable {
    public var cursor: String?
    public var pageSize: Int?
    public var tab: String?
    /// Filters 把综合 tab 收窄到 SearchFilter* 值；选多个时互相放宽。
    /// 游标不携带这些筛选，所以要连同游标传入相同的筛选。
    public var filters: [String]?

    public init(cursor: String? = nil, pageSize: Int? = nil, tab: String? = nil, filters: [String]? = nil) {
        self.cursor = cursor
        self.pageSize = pageSize
        self.tab = tab
        self.filters = filters
    }
}

/// EpisodeOptions 选择一个分集页。游标不在第 1 阶段的范围内。
public struct EpisodeOptions: Encodable, Hashable, Sendable {
    public var cursor: String?
    public var pageSize: Int?

    public init(cursor: String? = nil, pageSize: Int? = nil) {
        self.cursor = cursor
        self.pageSize = pageSize
    }
}

/// LandingOptions 选择一个分类落地页。
///
/// Tab 是 Categories 给出的 tab 值：8（全部，默认）、1（男生）或 0（女生）。
/// Filters 把 CategoryPage.Filters 中的筛选组键（book_status、sort_by、
/// sub_category、word_type）映射到该组的某个选项值。
/// Cursor 是上一页的 NextOffset。
public struct LandingOptions: Encodable, Hashable, Sendable {
    public var tab: String?
    public var filters: [String: String]?
    public var cursor: String?

    public init(tab: String? = nil, filters: [String: String]? = nil, cursor: String? = nil) {
        self.tab = tab
        self.filters = filters
        self.cursor = cursor
    }
}

/// NewReleaseQuery 选择上新日历中的一天：Date 格式为 YYYYMMDD（空即今天），
/// Cursor 是上一页的 Cursor，它带有所在的日期。
public struct NewReleaseQuery: Encodable, Hashable, Sendable {
    public var date: String?
    public var cursor: String?

    public init(date: String? = nil, cursor: String? = nil) {
        self.date = date
        self.cursor = cursor
    }
}

/// ShortFilterQuery 在筛选页（筛选）上挑选短剧。
///
/// Selections 把 FilterRow.Key 映射到选中的选项 Values；缺省的行即全部。
/// Cursor 是上一页的 Cursor。
public struct ShortFilterQuery: Encodable, Hashable, Sendable {
    public var selections: [String: [String]]?
    public var cursor: String?

    public init(selections: [String: [String]]? = nil, cursor: String? = nil) {
        self.selections = selections
        self.cursor = cursor
    }
}

/// ShortRankQuery 选择一个短剧榜单。Content 是 RankContent ID（all、human、
/// comic_series_rank、ai_playlet、series_album；空即 all），
/// Board 是它的某个 RankBoard ID（空即服务端对该 content 的默认值）。
/// Cursor 是上一页的 Cursor。
public struct ShortRankQuery: Encodable, Hashable, Sendable {
    public var content: String?
    public var board: String?
    public var cursor: String?

    public init(content: String? = nil, board: String? = nil, cursor: String? = nil) {
        self.content = content
        self.board = board
        self.cursor = cursor
    }
}
