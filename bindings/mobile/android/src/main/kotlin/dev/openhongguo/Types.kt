// 由 go run ./bindings/mobile/gen 生成，请勿手工修改。

package dev.openhongguo

import org.json.JSONArray
import org.json.JSONObject

/** Album 是多季的合集，来自 /novel/player/album_detail/v1/。 */
data class Album(
    val id: String = "",
    val title: String = "",
    val description: String = "",
    val coverUrl: String = "",
    val coverExpiresAt: String = "",
    val seasonCount: Int = 0,
    val seasons: List<Season> = emptyList(),
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("id", id)
        put("title", title)
        put("description", description)
        put("cover_url", coverUrl)
        put("cover_expires_at", coverExpiresAt)
        put("season_count", seasonCount)
        put("seasons", JSONArray(seasons.map { it.toJson() }))
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Album = Album(
            id = o.opt("id").asString(),
            title = o.opt("title").asString(),
            description = o.opt("description").asString(),
            coverUrl = o.opt("cover_url").asString(),
            coverExpiresAt = o.opt("cover_expires_at").asString(),
            seasonCount = o.opt("season_count").asInt(),
            seasons = o.opt("seasons").asList { Season.fromJson(it.asObject()) },
        )
    }
}

/** CategoryPage 是一个分类落地页及其筛选项。 */
data class CategoryPage(
    val categoryId: String = "",
    val categoryName: String = "",
    val description: String = "",
    val filters: List<FilterGroup> = emptyList(),
    /**
     * Tabs 是受众标签页（全部、男生、女生），每个 Value 是标签页类型编号；
     * DefaultTab 是页面打开时所在的标签页类型。
     */
    val tabs: List<FilterOption> = emptyList(),
    val defaultTab: Int = 0,
    val items: List<SeriesSummary> = emptyList(),
    val hasMore: Boolean = false,
    /** NextOffset 是下一页的偏移量，即 LandingOptions.Cursor 的值。 */
    val nextOffset: Int = 0,
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("category_id", categoryId)
        put("category_name", categoryName)
        put("description", description)
        put("filters", JSONArray(filters.map { it.toJson() }))
        put("tabs", JSONArray(tabs.map { it.toJson() }))
        put("default_tab", defaultTab)
        put("items", JSONArray(items.map { it.toJson() }))
        put("has_more", hasMore)
        put("next_offset", nextOffset)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): CategoryPage = CategoryPage(
            categoryId = o.opt("category_id").asString(),
            categoryName = o.opt("category_name").asString(),
            description = o.opt("description").asString(),
            filters = o.opt("filters").asList { FilterGroup.fromJson(it.asObject()) },
            tabs = o.opt("tabs").asList { FilterOption.fromJson(it.asObject()) },
            defaultTab = o.opt("default_tab").asInt(),
            items = o.opt("items").asList { SeriesSummary.fromJson(it.asObject()) },
            hasMore = o.opt("has_more").asBoolean(),
            nextOffset = o.opt("next_offset").asInt(),
        )
    }
}

/**
 * ContentKey 是受保护播放模型的 CENC（cenc-aes-ctr）内容密钥，
 * 由 sdk.ContentKey 还原。
 */
data class ContentKey(
    /**
     * Hex 是 16 字节 AES 密钥的小写十六进制形式，
     * 即 ffmpeg 的 -decryption_key 和播放器的 ClearKey 配置接受的形式。
     */
    val key: String = "",
    /** Version 是播放器在密钥旁读取的方案标记。 */
    val version: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("key", key)
        put("version", version)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): ContentKey = ContentKey(
            key = o.opt("key").asString(),
            version = o.opt("version").asString(),
        )
    }
}

/**
 * ContentType 是视频的 content_type，
 * 取自 com.dragon.read.rpc.model.VideoContentType。
 * 官方 App 的枚举混用小数值和从 1000 起的一段数值；这里两者都复现。
 */
@JvmInline
value class ContentType(val value: Int) {
    companion object {
        val UNKNOWN = ContentType(0)
        /** 真人短剧 */
        val SHORT_SERIES = ContentType(1)
        /** 场景剧 */
        val SCENE_PLAY = ContentType(2)
        /** 专家荐书 */
        val EXPERT_RECOMMEND = ContentType(3)
        /** PPT 荐书 */
        val PPT_RECOMMEND = ContentType(4)
        /** 电影 */
        val MOVIE = ContentType(5)
        /** 电视剧 */
        val TELE_PLAY = ContentType(6)
        /** 短剧 */
        val SHORT_PLAY = ContentType(7)
        /** 投放素材 */
        val DROP_MATERIAL = ContentType(1000)
        /** PUGC */
        val PUGC = ContentType(1001)
        /** 虚拍短剧 */
        val UNREAL_SHORT_PLAY = ContentType(1002)
        /** 合集 */
        val ALBUM = ContentType(1003)
        /** 漫剧（动态漫） */
        val MOTION_COMIC = ContentType(1004)
    }
}

/**
 * Device 是匿名身份出示的手机。注册请求体、
 * 每个请求的设备参数和 User-Agent 都描述它，所以一个设备 id 始终对应一个机型。
 */
data class Device(
    /**
     * Brand 是 device_brand，也是制造商（"Xiaomi"）；Model 是机型代号，
     * 作为 device_type、device_model 发送，也写进 User-Agent（"23127PN0CC"）。
     */
    val brand: String = "",
    val model: String = "",
    /**
     * OSVersion 是 Android 版本（"14"），OSAPI 是其 API 级别（34），
     * Build 是 User-Agent 携带的构建 id（"UKQ1.230804.001"）。
     */
    val osVersion: String = "",
    val osApi: Int = 0,
    val build: String = "",
    /**
     * Resolution 是 "height*width"，单位是像素；DPI 是屏幕密度，
     * 单位是每英寸点数。
     */
    val resolution: String = "",
    val dpi: Int = 0,
    /** Pad 让设备以平板身份出现（dragon_device_type=pad）。 */
    val pad: Boolean = false,
    /** Channel 是 App 构建上报的分发渠道（"xiaomi_8662_64" 来自小米应用商店）。 */
    val channel: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("brand", brand)
        put("model", model)
        put("os_version", osVersion)
        put("os_api", osApi)
        put("build", build)
        put("resolution", resolution)
        put("dpi", dpi)
        put("pad", pad)
        put("channel", channel)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Device = Device(
            brand = o.opt("brand").asString(),
            model = o.opt("model").asString(),
            osVersion = o.opt("os_version").asString(),
            osApi = o.opt("os_api").asInt(),
            build = o.opt("build").asString(),
            resolution = o.opt("resolution").asString(),
            dpi = o.opt("dpi").asInt(),
            pad = o.opt("pad").asBoolean(),
            channel = o.opt("channel").asString(),
        )
    }
}

/** Episode 是一个公开列出的分集。Number 对调用方从 1 开始。 */
data class Episode(
    val id: String = "",
    val seriesId: String = "",
    val number: Int = 0,
    val title: String = "",
    /** 服务端禁止播放该分集时，Playable 为 false（video_list[].disable_play）。 */
    val playable: Boolean = false,
    /** Vertical 标记竖屏画幅的分集。 */
    val vertical: Boolean = false,
    /** IsNew 标记本次更新中发布的分集。 */
    val isNew: Boolean = false,
    /** CoverURL 是分集剧照；CoverExpiresAt 是它的签名过期时间。 */
    val coverUrl: String = "",
    val coverExpiresAt: String = "",
    /** ContentType 与 Series.ContentType 是同一枚举。 */
    val contentType: ContentType = ContentType(0),
    /** CommentCount 和 DiggCount 是分集的计数。 */
    val commentCount: Long = 0L,
    val diggCount: Long = 0L,
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("id", id)
        put("series_id", seriesId)
        put("number", number)
        put("title", title)
        put("playable", playable)
        put("vertical", vertical)
        put("is_new", isNew)
        put("cover_url", coverUrl)
        put("cover_expires_at", coverExpiresAt)
        put("content_type", contentType.value)
        put("comment_count", commentCount)
        put("digg_count", diggCount)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Episode = Episode(
            id = o.opt("id").asString(),
            seriesId = o.opt("series_id").asString(),
            number = o.opt("number").asInt(),
            title = o.opt("title").asString(),
            playable = o.opt("playable").asBoolean(),
            vertical = o.opt("vertical").asBoolean(),
            isNew = o.opt("is_new").asBoolean(),
            coverUrl = o.opt("cover_url").asString(),
            coverExpiresAt = o.opt("cover_expires_at").asString(),
            contentType = ContentType(o.opt("content_type").asInt()),
            commentCount = o.opt("comment_count").asLong(),
            diggCount = o.opt("digg_count").asLong(),
        )
    }
}

/** FilterGroup 是一个筛选维度（状态、排序、子分类、篇幅）。 */
data class FilterGroup(
    /** Key 是该组设置的查询参数，例如 sort_by。 */
    val key: String = "",
    val options: List<FilterOption> = emptyList(),
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("key", key)
        put("options", JSONArray(options.map { it.toJson() }))
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): FilterGroup = FilterGroup(
            key = o.opt("key").asString(),
            options = o.opt("options").asList { FilterOption.fromJson(it.asObject()) },
        )
    }
}

/** FilterOption 是分类页上的一个可选值。 */
data class FilterOption(
    /** Name 是显示的标签；Value 是查询选中它时发送的值。 */
    val name: String = "",
    val value: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("name", name)
        put("value", value)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): FilterOption = FilterOption(
            name = o.opt("name").asString(),
            value = o.opt("value").asString(),
        )
    }
}

/**
 * FilterRow 是短剧筛选页的一个选择行。Key 是该行的 select_items 键（genre、
 * category_dim_theme、category_dim_role、category_dim_epoch、sort、gender、
 * online_time、creation_status）；选项的 Value 是查询所选的值。
 * 有哪些行取决于 genre。
 */
data class FilterRow(
    val key: String = "",
    val name: String = "",
    val options: List<FilterOption> = emptyList(),
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("key", key)
        put("name", name)
        put("options", JSONArray(options.map { it.toJson() }))
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): FilterRow = FilterRow(
            key = o.opt("key").asString(),
            name = o.opt("name").asString(),
            options = o.opt("options").asList { FilterOption.fromJson(it.asObject()) },
        )
    }
}

/**
 * HotBoard 是一个热榜。作品榜（短剧热搜榜、漫剧热搜榜、短剧新剧榜）填充 Items；
 * 热点话题榜填充 Topics。出现哪些榜单因调用而异。
 */
data class HotBoard(
    val name: String = "",
    val items: List<HotItem> = emptyList(),
    val topics: List<HotTopic> = emptyList(),
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("name", name)
        put("items", JSONArray(items.map { it.toJson() }))
        put("topics", JSONArray(topics.map { it.toJson() }))
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): HotBoard = HotBoard(
            name = o.opt("name").asString(),
            items = o.opt("items").asList { HotItem.fromJson(it.asObject()) },
            topics = o.opt("topics").asList { HotTopic.fromJson(it.asObject()) },
        )
    }
}

/** HotItem 是热榜上的一部作品，按榜单顺序排列。 */
data class HotItem(
    /** Position 是在榜单上的名次，从 1 开始。 */
    val position: Int = 0,
    val seriesId: String = "",
    val title: String = "",
    val coverUrl: String = "",
    val coverExpiresAt: String = "",
    val episodeCount: Int = 0,
    val contentType: ContentType = ContentType(0),
    /**
     * SearchScore 是榜单自己的数值（"355万热搜值"，有时带 "·第1季"）；
     * Heat 是作品的总热度（"4102万热度"）。
     */
    val searchScore: String = "",
    val heat: String = "",
    /** Tag 是角标，例如新剧。 */
    val tag: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("position", position)
        put("series_id", seriesId)
        put("title", title)
        put("cover_url", coverUrl)
        put("cover_expires_at", coverExpiresAt)
        put("episode_count", episodeCount)
        put("content_type", contentType.value)
        put("search_score", searchScore)
        put("heat", heat)
        put("tag", tag)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): HotItem = HotItem(
            position = o.opt("position").asInt(),
            seriesId = o.opt("series_id").asString(),
            title = o.opt("title").asString(),
            coverUrl = o.opt("cover_url").asString(),
            coverExpiresAt = o.opt("cover_expires_at").asString(),
            episodeCount = o.opt("episode_count").asInt(),
            contentType = ContentType(o.opt("content_type").asInt()),
            searchScore = o.opt("search_score").asString(),
            heat = o.opt("heat").asString(),
            tag = o.opt("tag").asString(),
        )
    }
}

/** HotSearch 是查询前的搜索页：猜你想搜一行，以及其下的热榜。 */
data class HotSearch(
    val guesses: List<HotItem> = emptyList(),
    val boards: List<HotBoard> = emptyList(),
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("guesses", JSONArray(guesses.map { it.toJson() }))
        put("boards", JSONArray(boards.map { it.toJson() }))
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): HotSearch = HotSearch(
            guesses = o.opt("guesses").asList { HotItem.fromJson(it.asObject()) },
            boards = o.opt("boards").asList { HotBoard.fromJson(it.asObject()) },
        )
    }
}

/** HotTopic 是热点话题榜的一项。Query 是它打开的搜索。 */
data class HotTopic(
    val id: String = "",
    val title: String = "",
    val query: String = "",
    val heat: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("id", id)
        put("title", title)
        put("query", query)
        put("heat", heat)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): HotTopic = HotTopic(
            id = o.opt("id").asString(),
            title = o.opt("title").asString(),
            query = o.opt("query").asString(),
            heat = o.opt("heat").asString(),
        )
    }
}

/**
 * InfoKind 是 secondary_infos[].data_type，
 * 取自 com.dragon.read.rpc.model.SecondaryInfoDataType。
 */
@JvmInline
value class InfoKind(val value: Int) {
    companion object {
        /** InfoSeasonOrder 是 "第 N 季" 那一行（实测值为 0）。 */
        val SEASON_ORDER = InfoKind(0)
        /** InfoRecommendReason 是推荐理由。 */
        val RECOMMEND_REASON = InfoKind(1)
        /** InfoRankScore 是榜单名次。 */
        val RANK_SCORE = InfoKind(2)
        /** InfoCategory 是分类标签，例如 "末世"（实测值为 3）。 */
        val CATEGORY = InfoKind(3)
        /** InfoScore 是用户评分。 */
        val SCORE = InfoKind(4)
        /** InfoAuthorName 是作者名。 */
        val AUTHOR_NAME = InfoKind(8)
        /** InfoTags 是标签列表。 */
        val TAGS = InfoKind(9)
        /** InfoCreationStatus 是连载中或已完结的说明。 */
        val CREATION_STATUS = InfoKind(11)
        /** InfoTwoActors 是演员行 "演员·A B"（实测值为 33）。 */
        val TWO_ACTORS = InfoKind(33)
        /** InfoPlotTag 是剧情标签。 */
        val PLOT_TAG = InfoKind(40)
    }
}

/** InfoRow 是 secondary_infos 的一项：详情页上带标签的一行。 */
data class InfoRow(
    val kind: InfoKind = InfoKind(0),
    val text: String = "",
    /** Schema 是该行打开的应用内链接（如果有）。 */
    val schema: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("kind", kind.value)
        put("text", text)
        put("schema", schema)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): InfoRow = InfoRow(
            kind = InfoKind(o.opt("kind").asInt()),
            text = o.opt("text").asString(),
            schema = o.opt("schema").asString(),
        )
    }
}

/**
 * NewRelease 是上新日历中的一部作品：已上线（Online），
 * 或已预告在 PublishAt 上线。
 */
data class NewRelease(
    val seriesId: String = "",
    val title: String = "",
    val description: String = "",
    val coverUrl: String = "",
    val coverExpiresAt: String = "",
    val episodeCount: Int = 0,
    /** 作品已上线时 Online 为 true；PublishAt 是预告作品的上线时间。 */
    val online: Boolean = false,
    val publishAt: String = "",
    val categories: List<String> = emptyList(),
    /**
     * 已上线的作品 Heat 形如 "378万热度"，预告作品形如 "159.7万人预约"；
     * Note 是预告文案（"预计10月26日上线"）。
     */
    val heat: String = "",
    val note: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("series_id", seriesId)
        put("title", title)
        put("description", description)
        put("cover_url", coverUrl)
        put("cover_expires_at", coverExpiresAt)
        put("episode_count", episodeCount)
        put("online", online)
        put("publish_at", publishAt)
        put("categories", JSONArray(categories))
        put("heat", heat)
        put("note", note)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): NewRelease = NewRelease(
            seriesId = o.opt("series_id").asString(),
            title = o.opt("title").asString(),
            description = o.opt("description").asString(),
            coverUrl = o.opt("cover_url").asString(),
            coverExpiresAt = o.opt("cover_expires_at").asString(),
            episodeCount = o.opt("episode_count").asInt(),
            online = o.opt("online").asBoolean(),
            publishAt = o.opt("publish_at").asString(),
            categories = o.opt("categories").asList { it.asString() },
            heat = o.opt("heat").asString(),
            note = o.opt("note").asString(),
        )
    }
}

/** NewReleasePage 是上新日历中的一天。 */
data class NewReleasePage(
    /**
     * Date 是本次返回的日期（YYYYMMDD）；Dates 是日历提供的日期，
     * 即今天前后各一周。
     */
    val date: String = "",
    val dates: List<String> = emptyList(),
    val items: List<NewRelease> = emptyList(),
    val hasMore: Boolean = false,
    val cursor: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("date", date)
        put("dates", JSONArray(dates))
        put("items", JSONArray(items.map { it.toJson() }))
        put("has_more", hasMore)
        put("cursor", cursor)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): NewReleasePage = NewReleasePage(
            date = o.opt("date").asString(),
            dates = o.opt("dates").asList { it.asString() },
            items = o.opt("items").asList { NewRelease.fromJson(it.asObject()) },
            hasMore = o.opt("has_more").asBoolean(),
            cursor = o.opt("cursor").asString(),
        )
    }
}

/** Novel 是短剧改编自的小说。 */
data class Novel(
    val bookId: String = "",
    /** Title 是小说本身的名称；ShortTitle 是它与短剧共用的名称。 */
    val title: String = "",
    val shortTitle: String = "",
    val status: NovelStatus = NovelStatus(0),
    val tags: List<String> = emptyList(),
    val reads: Long = 0L,
    val coverUrl: String = "",
    val coverExpiresAt: String = "",
    /** ShowAtEpisode 是播放器推荐该小说时所在的集数。 */
    val showAtEpisode: Int = 0,
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("book_id", bookId)
        put("title", title)
        put("short_title", shortTitle)
        put("status", status.value)
        put("tags", JSONArray(tags))
        put("reads", reads)
        put("cover_url", coverUrl)
        put("cover_expires_at", coverExpiresAt)
        put("show_at_episode", showAtEpisode)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Novel = Novel(
            bookId = o.opt("book_id").asString(),
            title = o.opt("title").asString(),
            shortTitle = o.opt("short_title").asString(),
            status = NovelStatus(o.opt("status").asInt()),
            tags = o.opt("tags").asList { it.asString() },
            reads = o.opt("reads").asLong(),
            coverUrl = o.opt("cover_url").asString(),
            coverExpiresAt = o.opt("cover_expires_at").asString(),
            showAtEpisode = o.opt("show_at_episode").asInt(),
        )
    }
}

/** NovelStatus 是客户端的 NovelCreationStatus；未给出时为 -1。 */
@JvmInline
value class NovelStatus(val value: Int) {
    companion object {
        /** 已完结 */
        val FINISHED = NovelStatus(0)
        /** 连载中 */
        val ONGOING = NovelStatus(1)
        /** 今日更新 */
        val UPDATED_TODAY = NovelStatus(3)
    }
}

/** PayType 是 pay_info.pay_type，取自 com.dragon.read.rpc.model.VideoPayType。 */
@JvmInline
value class PayType(val value: Int) {
    companion object {
        /** PayFree 表示无需账号即可观看。 */
        val FREE = PayType(0)
        /** PayVip 表示需要订阅。 */
        val VIP = PayType(1)
        /** PaySingle 表示需要单次购买。 */
        val SINGLE = PayType(2)
    }
}

/** Person 是演员或角色的主页。 */
data class Person(
    val userId: String = "",
    val name: String = "",
    val description: String = "",
    val avatarUrl: String = "",
    /**
     * Tags 是主页上的头衔（"女演员"、"虚拟人物"）；
     * Labels 是名字下方的小徽标（"水瓶座"）。
     */
    val tags: List<String> = emptyList(),
    val labels: List<String> = emptyList(),
    /** Followers 是粉丝数；Likes 是获赞数。 */
    val followers: Long = 0L,
    val likes: Long = 0L,
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("user_id", userId)
        put("name", name)
        put("description", description)
        put("avatar_url", avatarUrl)
        put("tags", JSONArray(tags))
        put("labels", JSONArray(labels))
        put("followers", followers)
        put("likes", likes)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Person = Person(
            userId = o.opt("user_id").asString(),
            name = o.opt("name").asString(),
            description = o.opt("description").asString(),
            avatarUrl = o.opt("avatar_url").asString(),
            tags = o.opt("tags").asList { it.asString() },
            labels = o.opt("labels").asList { it.asString() },
            followers = o.opt("followers").asLong(),
            likes = o.opt("likes").asLong(),
        )
    }
}

/**
 * PlatformType 是 video_platform，
 * 取自 com.dragon.read.rpc.model.VideoPlatformType。
 */
@JvmInline
value class PlatformType(val value: Int) {
    companion object {
        /** 未知；有声书带此值 */
        val UNKNOWN = PlatformType(0)
        /** 抖音 */
        val DOUYIN = PlatformType(1)
        /** UGC 视频 */
        val UGC_VIDEO = PlatformType(2)
        /** 站外与自制短剧 */
        val OUTER = PlatformType(3)
        /** 西瓜视频，电影来自这里 */
        val XIGUA = PlatformType(4)
        /** Chameleon */
        val CHAMELEON = PlatformType(5)
        /** 混合来源 */
        val MIXED = PlatformType(1024)
    }
}

/**
 * Playback 是一个已解析的匿名媒体模型。它描述服务端提供的内容，
 * 从不携带解密密钥或许可证材料。
 */
data class Playback(
    val episodeId: String = "",
    /** VideoID 是分集背后的媒体 id。 */
    val videoId: String = "",
    /** MediaType 是服务端的媒体类型，例如 "video"。 */
    val mediaType: String = "",
    val durationSeconds: Int = 0,
    /** ExpiresAt 是流 URL 失效的时间，单位是 unix 秒。 */
    val expiresAt: Long = 0L,
    /** Protected 标记加密的分集；参见 Stream.Protected。 */
    val protected: Boolean = false,
    val streams: List<Stream> = emptyList(),
    /**
     * AudioURL 是分段（DASH）作品（如长片电影）的独立音轨。
     * 单文件的表示形式留空。
     */
    val audioUrl: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("episode_id", episodeId)
        put("video_id", videoId)
        put("media_type", mediaType)
        put("duration_seconds", durationSeconds)
        put("expires_at", expiresAt)
        put("protected", protected)
        put("streams", JSONArray(streams.map { it.toJson() }))
        put("audio_url", audioUrl)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Playback = Playback(
            episodeId = o.opt("episode_id").asString(),
            videoId = o.opt("video_id").asString(),
            mediaType = o.opt("media_type").asString(),
            durationSeconds = o.opt("duration_seconds").asInt(),
            expiresAt = o.opt("expires_at").asLong(),
            protected = o.opt("protected").asBoolean(),
            streams = o.opt("streams").asList { Stream.fromJson(it.asObject()) },
            audioUrl = o.opt("audio_url").asString(),
        )
    }
}

/** RankBoard 是某个榜单内容下的一个榜（推荐榜、热播榜……）。 */
data class RankBoard(
    val id: String = "",
    val name: String = "",
    val selected: Boolean = false,
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("id", id)
        put("name", name)
        put("selected", selected)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): RankBoard = RankBoard(
            id = o.opt("id").asString(),
            name = o.opt("name").asString(),
            selected = o.opt("selected").asBoolean(),
        )
    }
}

/** RankChange 是榜单行的变动（客户端中的 RankChangeType）。 */
data class RankChange(
    val kind: RankChangeKind = RankChangeKind(0),
    /** Places 是该行变动的位数；新上榜或不变的行为 0。 */
    val places: Int = 0,
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("kind", kind.value)
        put("places", places)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): RankChange = RankChange(
            kind = RankChangeKind(o.opt("kind").asInt()),
            places = o.opt("places").asInt(),
        )
    }
}

/** RankChangeKind 的取值即客户端的 RankChangeType。 */
@JvmInline
value class RankChangeKind(val value: Int) {
    companion object {
        /** 新上榜 */
        val NEW = RankChangeKind(0)
        /** 上升 RankChange.Places 位 */
        val UP = RankChangeKind(1)
        /** 下降 RankChange.Places 位 */
        val DOWN = RankChangeKind(2)
        /** 名次不变 */
        val UNCHANGED = RankChangeKind(3)
    }
}

/**
 * RankContent 是短剧榜单页的一个内容
 * （全部、真人剧、漫剧、AI剧、系列剧、演员），带有自己的各个榜。
 */
data class RankContent(
    val id: String = "",
    val name: String = "",
    val selected: Boolean = false,
    val boards: List<RankBoard> = emptyList(),
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("id", id)
        put("name", name)
        put("selected", selected)
        put("boards", JSONArray(boards.map { it.toJson() }))
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): RankContent = RankContent(
            id = o.opt("id").asString(),
            name = o.opt("name").asString(),
            selected = o.opt("selected").asBoolean(),
            boards = o.opt("boards").asList { RankBoard.fromJson(it.asObject()) },
        )
    }
}

/** RankItem 是榜单的一行。 */
data class RankItem(
    /** Rank 是服务端给该行的名次（参见 Position）。 */
    val rank: Int = 0,
    /** Votes 和 Kind 用于活动榜单行：分别是票数和服务端的条目类型，原样透传。 */
    val votes: Long = 0L,
    val kind: Int = 0,
    /** Actor 在书籍榜单上是作者，在活动榜单或短剧演员榜上是演员名。 */
    val actor: String = "",
    val seriesId: String = "",
    val title: String = "",
    val coverUrl: String = "",
    /** Score 是显示的热度（"2550万热度"、"998万推荐"、"平台推荐"）。 */
    val score: String = "",
    /** EpisodeCount 由短剧榜单填充。 */
    val episodeCount: Int = 0,
    /**
     * Position 是该行在短剧榜单页上的名次，从 1 开始。Rank 是服务端自己的编号，
     * 在推广行前后会重复，在推荐榜上是个性化的。
     */
    val position: Int = 0,
    /**
     * Promoted 标记服务端插入短剧榜单的行：它显示 "平台推荐" 而非热度，
     * 名次照抄下一行（实测）。
     */
    val promoted: Boolean = false,
    /**
     * ActorID、AvatarURL、Intro 和 Details 用于短剧榜单的演员行，
     * 该行的 Actor 是名字，Score 是人气。Intro 列出作品；
     * Details 存放 "67部作品" 这样的行。
     */
    val actorId: String = "",
    /** UserID 是演员的账号，即 ActorProfile 和 ActorWorks 接受的 id。 */
    val userId: String = "",
    val avatarUrl: String = "",
    val intro: String = "",
    val details: List<String> = emptyList(),
    /** Change 是该行相对上一期榜单的变动（如果有）。 */
    val change: RankChange? = null,
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("rank", rank)
        put("votes", votes)
        put("kind", kind)
        put("actor", actor)
        put("series_id", seriesId)
        put("title", title)
        put("cover_url", coverUrl)
        put("score", score)
        put("episode_count", episodeCount)
        put("position", position)
        put("promoted", promoted)
        put("actor_id", actorId)
        put("user_id", userId)
        put("avatar_url", avatarUrl)
        put("intro", intro)
        put("details", JSONArray(details))
        put("change", change?.toJson())
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): RankItem = RankItem(
            rank = o.opt("rank").asInt(),
            votes = o.opt("votes").asLong(),
            kind = o.opt("kind").asInt(),
            actor = o.opt("actor").asString(),
            seriesId = o.opt("series_id").asString(),
            title = o.opt("title").asString(),
            coverUrl = o.opt("cover_url").asString(),
            score = o.opt("score").asString(),
            episodeCount = o.opt("episode_count").asInt(),
            position = o.opt("position").asInt(),
            promoted = o.opt("promoted").asBoolean(),
            actorId = o.opt("actor_id").asString(),
            userId = o.opt("user_id").asString(),
            avatarUrl = o.opt("avatar_url").asString(),
            intro = o.opt("intro").asString(),
            details = o.opt("details").asList { it.asString() },
            change = (o.opt("change") as? JSONObject)?.let(RankChange::fromJson),
        )
    }
}

/** RankList 是活动的一个榜单频道。 */
data class RankList(
    val id: String = "",
    val name: String = "",
    /** Type 和 Size 是服务端的榜单类型和长度，原样透传。 */
    val type: Int = 0,
    val size: Int = 0,
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("id", id)
        put("name", name)
        put("type", type)
        put("size", size)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): RankList = RankList(
            id = o.opt("id").asString(),
            name = o.opt("name").asString(),
            type = o.opt("type").asInt(),
            size = o.opt("size").asInt(),
        )
    }
}

/** Ranking 是一个活动榜单页。 */
data class Ranking(
    val listName: String = "",
    /** Lists 是该活动的其他榜单频道。 */
    val lists: List<RankList> = emptyList(),
    val items: List<RankItem> = emptyList(),
    val hasMore: Boolean = false,
    /**
     * NextOffset 用于续取列表（RankRows 把它当作 offset）；
     * SessionID 是服务端上报的分页会话。
     */
    val nextOffset: Int = 0,
    /** ActivityStage 和 CanVote 是服务端上报的活动状态。 */
    val activityStage: Int = 0,
    val canVote: Boolean = false,
    val sessionId: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("list_name", listName)
        put("lists", JSONArray(lists.map { it.toJson() }))
        put("items", JSONArray(items.map { it.toJson() }))
        put("has_more", hasMore)
        put("next_offset", nextOffset)
        put("activity_stage", activityStage)
        put("can_vote", canVote)
        put("session_id", sessionId)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Ranking = Ranking(
            listName = o.opt("list_name").asString(),
            lists = o.opt("lists").asList { RankList.fromJson(it.asObject()) },
            items = o.opt("items").asList { RankItem.fromJson(it.asObject()) },
            hasMore = o.opt("has_more").asBoolean(),
            nextOffset = o.opt("next_offset").asInt(),
            activityStage = o.opt("activity_stage").asInt(),
            canVote = o.opt("can_vote").asBoolean(),
            sessionId = o.opt("session_id").asString(),
        )
    }
}

/**
 * Recommend 是一部短剧的相关视频单元：包含各行、单元标题，
 * 以及客户端为后续请求保留的分页会话。
 */
data class Recommend(
    val cellTitle: String = "",
    val sessionId: String = "",
    val items: List<SeriesSummary> = emptyList(),
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("cell_title", cellTitle)
        put("session_id", sessionId)
        put("items", JSONArray(items.map { it.toJson() }))
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Recommend = Recommend(
            cellTitle = o.opt("cell_title").asString(),
            sessionId = o.opt("session_id").asString(),
            items = o.opt("items").asList { SeriesSummary.fromJson(it.asObject()) },
        )
    }
}

/** Season 是合集（多季合集）中的一项。 */
data class Season(
    val seriesId: String = "",
    val title: String = "",
    /** Index 是季序号，从 1 开始。 */
    val index: Int = 0,
    val coverUrl: String = "",
    val episodeCount: Int = 0,
    val status: SeriesStatus = SeriesStatus(0),
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("series_id", seriesId)
        put("title", title)
        put("index", index)
        put("cover_url", coverUrl)
        put("episode_count", episodeCount)
        put("status", status.value)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Season = Season(
            seriesId = o.opt("series_id").asString(),
            title = o.opt("title").asString(),
            index = o.opt("index").asInt(),
            coverUrl = o.opt("cover_url").asString(),
            episodeCount = o.opt("episode_count").asInt(),
            status = SeriesStatus(o.opt("status").asInt()),
        )
    }
}

/** Series 是一部短剧的详情视图。 */
data class Series(
    val id: String = "",
    val title: String = "",
    val description: String = "",
    val coverUrl: String = "",
    /**
     * CoverExpiresAt 是 CoverURL 的签名过期时间（x-expires）；
     * 过期后要从 API 重新获取封面。
     */
    val coverExpiresAt: String = "",
    val episodeCount: Int = 0,
    /** Author、Category 和 Words 由分类页的行填充。 */
    val author: String = "",
    val category: String = "",
    val words: Long = 0L,
    /** Status 是发布状态；PayType 是付费墙状态。 */
    val status: SeriesStatus = SeriesStatus(0),
    val payType: PayType = PayType(0),
    /** ContentType 和 Platform 描述作品是什么、来自哪里（短剧、电影、漫剧……）。 */
    val contentType: ContentType = ContentType(0),
    val platform: PlatformType = PlatformType(0),
    /** Heat 是原始热度分；App 将其显示为 "红果热度值N万"。 */
    val heat: Long = 0L,
    /** PlayCount 是短剧的总播放量（series_play_cnt）。 */
    val playCount: Long = 0L,
    /** FollowCount 和 DiggCount 是标题旁显示的计数。 */
    val followCount: Long = 0L,
    val diggCount: Long = 0L,
    /** CreatedAt 是发布时间（create_time，unix 秒）。 */
    val createdAt: String = "",
    /** RightText 是集数旁的角标（"更新至12集"）。 */
    val rightText: String = "",
    /** Subtitles 是标题下方的简短状态行。 */
    val subtitles: List<String> = emptyList(),
    /** AgeGate 对不受限的作品为 0。 */
    val ageGate: Int = 0,
    /** IsMultiSeason 标记属于多季合集的作品。 */
    val isMultiSeason: Boolean = false,
    /** SeasonIndex 是该合集内的季序号，从 1 开始。 */
    val seasonIndex: Int = 0,
    /** RelatedAlbumID 是各季共用的合集 id。 */
    val relatedAlbumId: String = "",
    /** RelatedNovelBookID 是短剧改编的小说（如已公开）。 */
    val relatedNovelBookId: String = "",
    /** Novel 是详情页链接的原著小说（video_relate_book），服务端给出时才有。 */
    val novel: Novel? = null,
    /** Rows 是详情页的次要信息行（季、分类、演员）。 */
    val rows: List<InfoRow> = emptyList(),
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("id", id)
        put("title", title)
        put("description", description)
        put("cover_url", coverUrl)
        put("cover_expires_at", coverExpiresAt)
        put("episode_count", episodeCount)
        put("author", author)
        put("category", category)
        put("words", words)
        put("status", status.value)
        put("pay_type", payType.value)
        put("content_type", contentType.value)
        put("platform", platform.value)
        put("heat", heat)
        put("play_count", playCount)
        put("follow_count", followCount)
        put("digg_count", diggCount)
        put("created_at", createdAt)
        put("right_text", rightText)
        put("subtitles", JSONArray(subtitles))
        put("age_gate", ageGate)
        put("is_multi_season", isMultiSeason)
        put("season_index", seasonIndex)
        put("related_album_id", relatedAlbumId)
        put("related_novel_book_id", relatedNovelBookId)
        put("novel", novel?.toJson())
        put("rows", JSONArray(rows.map { it.toJson() }))
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Series = Series(
            id = o.opt("id").asString(),
            title = o.opt("title").asString(),
            description = o.opt("description").asString(),
            coverUrl = o.opt("cover_url").asString(),
            coverExpiresAt = o.opt("cover_expires_at").asString(),
            episodeCount = o.opt("episode_count").asInt(),
            author = o.opt("author").asString(),
            category = o.opt("category").asString(),
            words = o.opt("words").asLong(),
            status = SeriesStatus(o.opt("status").asInt()),
            payType = PayType(o.opt("pay_type").asInt()),
            contentType = ContentType(o.opt("content_type").asInt()),
            platform = PlatformType(o.opt("platform").asInt()),
            heat = o.opt("heat").asLong(),
            playCount = o.opt("play_count").asLong(),
            followCount = o.opt("follow_count").asLong(),
            diggCount = o.opt("digg_count").asLong(),
            createdAt = o.opt("created_at").asString(),
            rightText = o.opt("right_text").asString(),
            subtitles = o.opt("subtitles").asList { it.asString() },
            ageGate = o.opt("age_gate").asInt(),
            isMultiSeason = o.opt("is_multi_season").asBoolean(),
            seasonIndex = o.opt("season_index").asInt(),
            relatedAlbumId = o.opt("related_album_id").asString(),
            relatedNovelBookId = o.opt("related_novel_book_id").asString(),
            novel = (o.opt("novel") as? JSONObject)?.let(Novel::fromJson),
            rows = o.opt("rows").asList { InfoRow.fromJson(it.asObject()) },
        )
    }
}

/** SeriesDetail 是一部短剧及其分集，由批量读取返回。 */
data class SeriesDetail(
    val series: Series? = null,
    val episodes: List<Episode> = emptyList(),
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("series", series?.toJson())
        put("episodes", JSONArray(episodes.map { it.toJson() }))
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): SeriesDetail = SeriesDetail(
            series = (o.opt("series") as? JSONObject)?.let(Series::fromJson),
            episodes = o.opt("episodes").asList { Episode.fromJson(it.asObject()) },
        )
    }
}

/**
 * SeriesStatus 是短剧的发布状态，由 data.video_data.series_status 携带。
 *
 * 名称取自官方 App 的枚举 com.dragon.read.rpc.model.SeriesStatus（
 * SeriesUpdating、SeriesEnd、SeriesUpdateTaday、SeriesUpdateStop）；
 * 值 0 和 1 也已在线上实测。
 */
@JvmInline
value class SeriesStatus(val value: Int) {
    companion object {
        /** SeriesUpdating 是未完结的短剧（"更新至 N 集"）。 */
        val UPDATING = SeriesStatus(0)
        /** SeriesEnd 是已完结的短剧（"全 N 集"）。 */
        val END = SeriesStatus(1)
        /** SeriesUpdatedToday 是枚举中的“今日更新”状态（推断）。 */
        val UPDATED_TODAY = SeriesStatus(3)
        /** SeriesUpdateStopped 是枚举中的“停止更新”状态（推断）。 */
        val UPDATE_STOPPED = SeriesStatus(4)
    }
}

/** SeriesSummary 是公开目录中的一行。ID 和标题是稳定字段。 */
data class SeriesSummary(
    val id: String = "",
    val title: String = "",
    val description: String = "",
    val coverUrl: String = "",
    /**
     * CoverExpiresAt 是 CoverURL 的签名过期时间（x-expires）；
     * 过期后要从 API 重新获取封面。
     */
    val coverExpiresAt: String = "",
    val episodeCount: Int = 0,
    /** Author、Category 和 Words 由分类页的行填充。 */
    val author: String = "",
    val category: String = "",
    val words: Long = 0L,
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("id", id)
        put("title", title)
        put("description", description)
        put("cover_url", coverUrl)
        put("cover_expires_at", coverExpiresAt)
        put("episode_count", episodeCount)
        put("author", author)
        put("category", category)
        put("words", words)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): SeriesSummary = SeriesSummary(
            id = o.opt("id").asString(),
            title = o.opt("title").asString(),
            description = o.opt("description").asString(),
            coverUrl = o.opt("cover_url").asString(),
            coverExpiresAt = o.opt("cover_expires_at").asString(),
            episodeCount = o.opt("episode_count").asInt(),
            author = o.opt("author").asString(),
            category = o.opt("category").asString(),
            words = o.opt("words").asLong(),
        )
    }
}

/** ShortFilterPage 是符合筛选条件的短剧的一页。 */
data class ShortFilterPage(
    val items: List<SeriesSummary> = emptyList(),
    val hasMore: Boolean = false,
    /** Cursor 用于续取列表；它携带已展示的短剧，服务端会跳过它们。 */
    val cursor: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("items", JSONArray(items.map { it.toJson() }))
        put("has_more", hasMore)
        put("cursor", cursor)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): ShortFilterPage = ShortFilterPage(
            items = o.opt("items").asList { SeriesSummary.fromJson(it.asObject()) },
            hasMore = o.opt("has_more").asBoolean(),
            cursor = o.opt("cursor").asString(),
        )
    }
}

/** ShortRanking 是短剧榜单的一页。 */
data class ShortRanking(
    /** Contents 是该页的目录：每个内容及其各个榜。 */
    val contents: List<RankContent> = emptyList(),
    /** Content 和 Board 是本页对应的 ID。 */
    val content: String = "",
    val board: String = "",
    /** Version 是服务端上报的榜单版本。 */
    val version: String = "",
    val items: List<RankItem> = emptyList(),
    val hasMore: Boolean = false,
    val cursor: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("contents", JSONArray(contents.map { it.toJson() }))
        put("content", content)
        put("board", board)
        put("version", version)
        put("items", JSONArray(items.map { it.toJson() }))
        put("has_more", hasMore)
        put("cursor", cursor)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): ShortRanking = ShortRanking(
            contents = o.opt("contents").asList { RankContent.fromJson(it.asObject()) },
            content = o.opt("content").asString(),
            board = o.opt("board").asString(),
            version = o.opt("version").asString(),
            items = o.opt("items").asList { RankItem.fromJson(it.asObject()) },
            hasMore = o.opt("has_more").asBoolean(),
            cursor = o.opt("cursor").asString(),
        )
    }
}

/**
 * Stream 是媒体模型中的一个表示形式。Protected
 * 标记官方客户端在内部解密的表示形式；sdk.ContentKey 从模型的 KeyMaterial
 * 还原它的密钥。
 */
data class Stream(
    /**
     * Definition 是分辨率标签（"720p"）；Quality 是服务端的画质标签
     * （"normal"）。
     */
    val definition: String = "",
    val quality: String = "",
    val codec: String = "",
    val width: Int = 0,
    val height: Int = 0,
    /** Bitrate 的单位是 bit/s；Size 是文件大小，单位是字节。 */
    val bitrate: Long = 0L,
    val size: Long = 0L,
    /** BackupURL 是同一文件的第二个 CDN 地址。 */
    val url: String = "",
    val backupUrl: String = "",
    /** HLS 标记 m3u8 播放列表；Dash 标记分段（DASH）的表示形式。 */
    val hls: Boolean = false,
    val dash: Boolean = false,
    val protected: Boolean = false,
    /** Encryption 是受保护流的加密方案，例如 "cenc-aes-ctr"。 */
    val encryption: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("definition", definition)
        put("quality", quality)
        put("codec", codec)
        put("width", width)
        put("height", height)
        put("bitrate", bitrate)
        put("size", size)
        put("url", url)
        put("backup_url", backupUrl)
        put("hls", hls)
        put("dash", dash)
        put("protected", protected)
        put("encryption", encryption)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Stream = Stream(
            definition = o.opt("definition").asString(),
            quality = o.opt("quality").asString(),
            codec = o.opt("codec").asString(),
            width = o.opt("width").asInt(),
            height = o.opt("height").asInt(),
            bitrate = o.opt("bitrate").asLong(),
            size = o.opt("size").asLong(),
            url = o.opt("url").asString(),
            backupUrl = o.opt("backup_url").asString(),
            hls = o.opt("hls").asBoolean(),
            dash = o.opt("dash").asBoolean(),
            protected = o.opt("protected").asBoolean(),
            encryption = o.opt("encryption").asString(),
        )
    }
}

/**
 * Suggestion 是一条搜索建议。Text 是搜索框显示的文本；
 * Keyword 是它提交的内容（直达行为内容 ID）。
 */
data class Suggestion(
    val text: String = "",
    val keyword: String = "",
    /** Kind 是服务端的行类型，原样透传。 */
    val kind: Int = 0,
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("text", text)
        put("keyword", keyword)
        put("kind", kind)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Suggestion = Suggestion(
            text = o.opt("text").asString(),
            keyword = o.opt("keyword").asString(),
            kind = o.opt("kind").asInt(),
        )
    }
}

/** Work 是某人出演的一部短剧。 */
data class Work(
    val seriesId: String = "",
    val title: String = "",
    val description: String = "",
    val coverUrl: String = "",
    val coverExpiresAt: String = "",
    val episodeCount: Int = 0,
    /** Tags 是卡片上的各行，例如 "爱情" 和 "70集"。 */
    val tags: List<String> = emptyList(),
    /** Heat 是短剧的原始热度分；Plays 是其播放量。 */
    val heat: Long = 0L,
    val plays: Long = 0L,
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("series_id", seriesId)
        put("title", title)
        put("description", description)
        put("cover_url", coverUrl)
        put("cover_expires_at", coverExpiresAt)
        put("episode_count", episodeCount)
        put("tags", JSONArray(tags))
        put("heat", heat)
        put("plays", plays)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): Work = Work(
            seriesId = o.opt("series_id").asString(),
            title = o.opt("title").asString(),
            description = o.opt("description").asString(),
            coverUrl = o.opt("cover_url").asString(),
            coverExpiresAt = o.opt("cover_expires_at").asString(),
            episodeCount = o.opt("episode_count").asInt(),
            tags = o.opt("tags").asList { it.asString() },
            heat = o.opt("heat").asLong(),
            plays = o.opt("plays").asLong(),
        )
    }
}

/** WorkPage 是某人作品的一页（作品 › 剧作）。 */
data class WorkPage(
    val items: List<Work> = emptyList(),
    val hasMore: Boolean = false,
    val cursor: String = "",
) {
    /** 编码成与 Go SDK 相同的 JSON。 */
    fun toJson(): JSONObject = JSONObject().apply {
        put("items", JSONArray(items.map { it.toJson() }))
        put("has_more", hasMore)
        put("cursor", cursor)
    }

    companion object {
        /** 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。 */
        fun fromJson(o: JSONObject): WorkPage = WorkPage(
            items = o.opt("items").asList { Work.fromJson(it.asObject()) },
            hasMore = o.opt("has_more").asBoolean(),
            cursor = o.opt("cursor").asString(),
        )
    }
}

/**
 * DiscoverOptions 选择一个发现页。Channel 是书城频道名，空频道即推荐。
 * Cursor 是上一页给出的令牌。PageSize 是 cell/change 的 limit（0 表示 20，上限 50）。
 */
data class DiscoverOptions(
    val cursor: String? = null,
    val channel: String? = null,
    val pageSize: Int? = null,
) {
    internal fun toJson(): JSONObject = JSONObject().apply {
        cursor?.let { put("cursor", it) }
        channel?.let { put("channel", it) }
        pageSize?.let { put("pageSize", it) }
    }
}

/**
 * SearchOptions 选择一个搜索页。Tab 取值为综合、剧集、视频、讨论、小说、
 * 用户或听书，空 tab 即综合。Cursor 是下一个偏移量。
 */
data class SearchOptions(
    val cursor: String? = null,
    val pageSize: Int? = null,
    val tab: String? = null,
    /**
     * Filters 把综合 tab 收窄到 SearchFilter* 值；选多个时互相放宽。
     * 游标不携带这些筛选，所以要连同游标传入相同的筛选。
     */
    val filters: List<String>? = null,
) {
    internal fun toJson(): JSONObject = JSONObject().apply {
        cursor?.let { put("cursor", it) }
        pageSize?.let { put("pageSize", it) }
        tab?.let { put("tab", it) }
        filters?.let { put("filters", JSONArray(it)) }
    }
}

/** EpisodeOptions 选择一个分集页。游标不在第 1 阶段的范围内。 */
data class EpisodeOptions(
    val cursor: String? = null,
    val pageSize: Int? = null,
) {
    internal fun toJson(): JSONObject = JSONObject().apply {
        cursor?.let { put("cursor", it) }
        pageSize?.let { put("pageSize", it) }
    }
}

/**
 * LandingOptions 选择一个分类落地页。
 *
 * Tab 是 Categories 给出的 tab 值：8（全部，默认）、1（男生）或 0（女生）。
 * Filters 把 CategoryPage.Filters 中的筛选组键（book_status、sort_by、
 * sub_category、word_type）映射到该组的某个选项值。
 * Cursor 是上一页的 NextOffset。
 */
data class LandingOptions(
    val tab: String? = null,
    val filters: Map<String, String>? = null,
    val cursor: String? = null,
) {
    internal fun toJson(): JSONObject = JSONObject().apply {
        tab?.let { put("tab", it) }
        filters?.let { put("filters", JSONObject(it)) }
        cursor?.let { put("cursor", it) }
    }
}

/**
 * NewReleaseQuery 选择上新日历中的一天：Date 格式为 YYYYMMDD（空即今天），
 * Cursor 是上一页的 Cursor，它带有所在的日期。
 */
data class NewReleaseQuery(
    val date: String? = null,
    val cursor: String? = null,
) {
    internal fun toJson(): JSONObject = JSONObject().apply {
        date?.let { put("date", it) }
        cursor?.let { put("cursor", it) }
    }
}

/**
 * ShortFilterQuery 在筛选页（筛选）上挑选短剧。
 *
 * Selections 把 FilterRow.Key 映射到选中的选项 Values；缺省的行即全部。
 * Cursor 是上一页的 Cursor。
 */
data class ShortFilterQuery(
    val selections: Map<String, List<String>>? = null,
    val cursor: String? = null,
) {
    internal fun toJson(): JSONObject = JSONObject().apply {
        selections?.let { put("selections", JSONObject(it.mapValues { JSONArray(it.value) })) }
        cursor?.let { put("cursor", it) }
    }
}

/**
 * ShortRankQuery 选择一个短剧榜单。Content 是 RankContent ID（all、human、
 * comic_series_rank、ai_playlet、series_album；空即 all），
 * Board 是它的某个 RankBoard ID（空即服务端对该 content 的默认值）。
 * Cursor 是上一页的 Cursor。
 */
data class ShortRankQuery(
    val content: String? = null,
    val board: String? = null,
    val cursor: String? = null,
) {
    internal fun toJson(): JSONObject = JSONObject().apply {
        content?.let { put("content", it) }
        board?.let { put("board", it) }
        cursor?.let { put("cursor", it) }
    }
}
