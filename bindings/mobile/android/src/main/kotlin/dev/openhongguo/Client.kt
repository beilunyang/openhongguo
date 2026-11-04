package dev.openhongguo

import android.content.Context
import dev.openhongguo.core.Core
import java.io.File
import kotlin.coroutines.resume
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow
import kotlinx.coroutines.suspendCancellableCoroutine
import org.json.JSONArray
import org.json.JSONObject

/** Page 是一页匿名结果。 */
data class Page<T>(
    val items: List<T> = emptyList(),
    /** 下一页的游标；没有下一页时为空。 */
    val cursor: String = "",
    val hasMore: Boolean = false,
) {
    internal companion object {
        fun <T> fromJson(o: JSONObject, item: (JSONObject) -> T) = Page(
            items = o.opt("items").asList { item(it.asObject()) },
            cursor = o.opt("cursor").asString(),
            hasMore = o.opt("has_more").asBoolean(),
        )
    }
}

/** PlaybackResult 是 resolvePlayback 的结果。 */
data class PlaybackResult(
    val playback: Playback,
    /** 加密剧集的内容密钥；未加密或推导失败时为 null。 */
    val contentKey: ContentKey? = null,
    /** 加密剧集的密钥推导失败时，说明原因。 */
    val contentKeyError: OpenhongguoException? = null,
)

/** ClientOptions 是 OpenhongguoClient 的设置，全部可选。 */
data class ClientOptions(
    /** 读取类接口的地址，如 "https://host"；默认与 App 一致。 */
    val apiOrigin: String? = null,
    /** 设备注册接口的地址；默认与 App 一致。 */
    val registerOrigin: String? = null,
    /** 短剧筛选和活动榜接口的地址；默认与 App 一致。 */
    val distributionOrigin: String? = null,
    /** 可重试的失败最多尝试几次，默认 3；0 或 1 表示不重试。 */
    val retryAttempts: Int = 3,
    /** 设备被拒或播放返回空结果时，一次调用最多换几台新的匿名设备，默认 2；0 表示关闭。 */
    val deviceResets: Int = 2,
    /** 新注册设备自称的机型；为空的字段取默认值（小米 14）。 */
    val device: Device? = null,
    /** 接收诊断日志，在 SDK 的线程里调用；日志里的 URL 已经脱敏。 */
    val logger: ((String) -> Unit)? = null,
) {
    internal fun toJson(): JSONObject = JSONObject().apply {
        put("apiOrigin", apiOrigin)
        put("registerOrigin", registerOrigin)
        put("distributionOrigin", distributionOrigin)
        put("retryAttempts", retryAttempts)
        put("deviceResets", deviceResets)
        put("device", device?.toJson())
    }
}

/**
 * Engine 发起一次按名字的调用，结果的 JSON 交给 done；返回的函数用来取消。
 * 正式实现是 Go 编译出的 core 库，测试里换成假的。
 */
internal fun interface Engine {
    fun start(method: String, args: String, done: (String) -> Unit): () -> Unit
}

private class GoEngine(private val client: dev.openhongguo.core.Client) : Engine {
    override fun start(method: String, args: String, done: (String) -> Unit): () -> Unit {
        val call = client.start(method, args) { done(it) }
        return { call.cancel() }
    }
}

/**
 * OpenhongguoClient 是红果短剧的匿名客户端。所有方法都是挂起函数，
 * 可以在任何线程调用，失败时抛出 OpenhongguoException；协程取消时，
 * 进行中的请求随之取消。同一个客户端可以并发使用。
 */
class OpenhongguoClient internal constructor(private val engine: Engine) {
    companion object {
        /**
         * 创建客户端，设备档案保存在 App 的 noBackupFilesDir 下，
         * 不会随系统备份迁移到别的手机。首次请求时注册一台匿名设备。
         */
        @JvmStatic
        @JvmOverloads
        fun create(context: Context, options: ClientOptions = ClientOptions()): OpenhongguoClient =
            create(File(context.noBackupFilesDir, "openhongguo/anonymous.json"), options)

        /** 创建客户端，设备档案保存在 profile 文件里，目录不存在时自动创建。 */
        @JvmStatic
        @JvmOverloads
        fun create(profile: File, options: ClientOptions = ClientOptions()): OpenhongguoClient {
            val logger = options.logger?.let { log -> dev.openhongguo.core.Logger { log(it) } }
            val client = try {
                Core.newClient(profile.path, options.toJson().toString(), logger)
            } catch (e: Exception) {
                throw OpenhongguoException.parse(e.message)
            }
            return OpenhongguoClient(GoEngine(client))
        }
    }

    /** 加载一个书城频道（默认“推荐”）。 */
    suspend fun discover(options: DiscoverOptions = DiscoverOptions()): Page<SeriesSummary> =
        page(call("discover", options.toJson()), SeriesSummary::fromJson)

    /** 搜索短剧。 */
    suspend fun search(query: String, options: SearchOptions = SearchOptions()): Page<SeriesSummary> =
        page(call("search", query, options.toJson()), SeriesSummary::fromJson)

    /** 搜索联想词。 */
    suspend fun suggest(prefix: String): List<Suggestion> =
        call("suggest", prefix).asList { Suggestion.fromJson(it.asObject()) }

    /** 搜索页的“猜你想搜”和热搜榜。 */
    suspend fun hotSearch(): HotSearch = HotSearch.fromJson(call("hotSearch").asObject())

    /** 搜索框里滚动的提示词。 */
    suspend fun searchCues(): List<String> = call("searchCues").asList { it.asString() }

    /** 上新日历的一天。 */
    suspend fun newReleases(query: NewReleaseQuery = NewReleaseQuery()): NewReleasePage =
        NewReleasePage.fromJson(call("newReleases", query.toJson()).asObject())

    /** 一部剧的详情。 */
    suspend fun getSeries(seriesId: String): Series = Series.fromJson(call("getSeries", seriesId).asObject())

    /** 一次读取多部剧及其各集，按请求顺序返回。 */
    suspend fun getSeriesBatch(seriesIds: List<String>): List<SeriesDetail> =
        call("getSeriesBatch", JSONArray(seriesIds)).asList { SeriesDetail.fromJson(it.asObject()) }

    /** 一部剧的分集列表。 */
    suspend fun listEpisodes(seriesId: String, options: EpisodeOptions = EpisodeOptions()): Page<Episode> =
        page(call("listEpisodes", seriesId, options.toJson()), Episode::fromJson)

    /** 多季剧的全部季。 */
    suspend fun seasons(seriesId: String): Album = Album.fromJson(call("seasons", seriesId).asObject())

    /** 一部剧的相关推荐。 */
    suspend fun seriesRecommend(seriesId: String): Recommend =
        Recommend.fromJson(call("seriesRecommend", seriesId).asObject())

    /** 一个合集及其各季。 */
    suspend fun getAlbum(albumId: String): Album = Album.fromJson(call("getAlbum", albumId).asObject())

    /** 短剧排行榜的一页。 */
    suspend fun shortRank(query: ShortRankQuery = ShortRankQuery()): ShortRanking =
        ShortRanking.fromJson(call("shortRank", query.toJson()).asObject())

    /** 短剧筛选面板；genre 如 "comic_series"。 */
    suspend fun shortFilterPanel(genre: String = ""): List<FilterRow> =
        call("shortFilterPanel", genre).asList { FilterRow.fromJson(it.asObject()) }

    /** 按筛选条件列出短剧。 */
    suspend fun shortFilter(query: ShortFilterQuery = ShortFilterQuery()): ShortFilterPage =
        ShortFilterPage.fromJson(call("shortFilter", query.toJson()).asObject())

    /** 活动榜单。 */
    suspend fun ranking(activityId: String, rankListId: String): Ranking =
        Ranking.fromJson(call("ranking", activityId, rankListId).asObject())

    /** 书城榜单的一页。 */
    suspend fun rankRows(cellId: String, offset: Int = 0): Ranking =
        Ranking.fromJson(call("rankRows", cellId, offset).asObject())

    /** 书城榜单的类型。 */
    suspend fun rankGroups(): List<RankList> = call("rankGroups").asList { RankList.fromJson(it.asObject()) }

    /** 演员或角色主页。 */
    suspend fun person(userId: String): Person = Person.fromJson(call("person", userId).asObject())

    /** 演员的作品，一页。 */
    suspend fun personWorks(userId: String, cursor: String = ""): WorkPage =
        WorkPage.fromJson(call("personWorks", userId, cursor).asObject())

    /** 书籍分类的标签页。 */
    suspend fun categories(categoryId: String): CategoryPage =
        CategoryPage.fromJson(call("categories", categoryId).asObject())

    /** 书籍分类下的列表。 */
    suspend fun categoryLanding(categoryId: String, options: LandingOptions = LandingOptions()): CategoryPage =
        CategoryPage.fromJson(call("categoryLanding", categoryId, options.toJson()).asObject())

    /** 一集的播放模型；加密剧集同时给出内容密钥。 */
    suspend fun resolvePlayback(episodeId: String): PlaybackResult {
        val o = call("resolvePlayback", episodeId).asObject()
        return PlaybackResult(
            playback = Playback.fromJson(o.opt("playback").asObject()),
            contentKey = (o.opt("contentKey") as? JSONObject)?.let(ContentKey::fromJson),
            contentKeyError = (o.opt("contentKeyError") as? JSONObject)?.let(OpenhongguoException::fromJson),
        )
    }

    /** 丢弃当前匿名设备，下次请求时重新注册。 */
    suspend fun resetDevice() {
        call("resetDevice")
    }

    /** 沿着游标逐部返回书城频道的剧，去掉跨页重复。 */
    fun discoverAll(options: DiscoverOptions = DiscoverOptions(), maxPages: Int = Int.MAX_VALUE): Flow<SeriesSummary> =
        walk(options.cursor, maxPages) { discover(options.copy(cursor = it)) }

    /** 沿着游标逐部返回搜索结果，去掉跨页重复。 */
    fun searchAll(query: String, options: SearchOptions = SearchOptions(), maxPages: Int = Int.MAX_VALUE): Flow<SeriesSummary> =
        walk(options.cursor, maxPages) { search(query, options.copy(cursor = it)) }

    private fun walk(start: String?, maxPages: Int, fetch: suspend (String) -> Page<SeriesSummary>): Flow<SeriesSummary> = flow {
        val seen = HashSet<String>()
        var cursor = start.orEmpty()
        for (pages in 0 until maxPages) {
            val page = fetch(cursor)
            for (item in page.items) {
                if (seen.add(item.id)) emit(item)
            }
            if (!page.hasMore || page.cursor.isEmpty()) return@flow
            cursor = page.cursor
        }
    }

    private fun <T> page(v: Any?, item: (JSONObject) -> T): Page<T> = Page.fromJson(v.asObject(), item)

    /** call 调用 Go 端的方法，返回结果的 JSON 值（JSONObject、JSONArray 或基本类型）。 */
    private suspend fun call(method: String, vararg args: Any?): Any? {
        val a = JSONArray()
        for (arg in args) a.put(arg ?: JSONObject.NULL)
        val out = suspendCancellableCoroutine { cont ->
            val cancel = engine.start(method, a.toString()) { cont.resume(it) }
            cont.invokeOnCancellation { cancel() }
        }
        val env = JSONObject(out)
        env.optJSONObject("error")?.let { throw OpenhongguoException.fromJson(it) }
        return env.opt("result")
    }
}
