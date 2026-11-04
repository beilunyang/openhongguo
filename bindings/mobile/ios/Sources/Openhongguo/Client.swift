import Foundation
import OpenhongguoCore

/// Page 是一页匿名结果。
public struct Page<Item: Codable & Sendable>: Codable, Sendable {
    public var items: [Item]
    /// 下一页的游标；没有下一页时为空。
    public var cursor: String
    public var hasMore: Bool

    public init(items: [Item] = [], cursor: String = "", hasMore: Bool = false) {
        self.items = items
        self.cursor = cursor
        self.hasMore = hasMore
    }

    enum CodingKeys: String, CodingKey {
        case items, cursor
        case hasMore = "has_more"
    }

    public init(from decoder: Decoder) throws {
        let c = try decoder.container(keyedBy: CodingKeys.self)
        items = try c.decodeIfPresent([Item].self, forKey: .items) ?? []
        cursor = try c.decodeIfPresent(String.self, forKey: .cursor) ?? ""
        hasMore = try c.decodeIfPresent(Bool.self, forKey: .hasMore) ?? false
    }
}

/// PlaybackResult 是 resolvePlayback 的结果。
public struct PlaybackResult: Sendable {
    public var playback: Playback
    /// 加密剧集的内容密钥；未加密或推导失败时为 nil。
    public var contentKey: ContentKey?
    /// 加密剧集的密钥推导失败时，说明原因。
    public var contentKeyError: OpenhongguoError?
}

/// 错误码，对应 Go SDK 的哨兵错误。
public enum ErrorCode: String, Sendable, Codable {
    case invalidOptions = "INVALID_OPTIONS"
    case profileNotFound = "PROFILE_NOT_FOUND"
    case signerUnavailable = "SIGNER_UNAVAILABLE"
    case authenticationRequired = "AUTHENTICATION_REQUIRED"
    case notProtected = "NOT_PROTECTED"
    case drmProtected = "DRM_PROTECTED"
    case anonymousLimited = "ANONYMOUS_LIMITED"
    case rateLimited = "RATE_LIMITED"
    case transport = "TRANSPORT"
    case signatureRejected = "SIGNATURE_REJECTED"
    case contentUnavailable = "CONTENT_UNAVAILABLE"
    case paramInvalid = "PARAM_INVALID"
    case challengeRequired = "CHALLENGE_REQUIRED"
    case deviceRejected = "DEVICE_REJECTED"
    case endOfList = "END_OF_LIST"
    case albumUnavailable = "ALBUM_UNAVAILABLE"
    /// 调用被取消。Task 取消时抛出的是 CancellationError，不会见到这个错误码。
    case cancelled = "CANCELLED"
    case unknown = "UNKNOWN"

    public init(from decoder: Decoder) throws {
        let raw = try decoder.singleValueContainer().decode(String.self)
        self = ErrorCode(rawValue: raw) ?? .unknown
    }
}

/// OpenhongguoError 是 SDK 返回的错误。
public struct OpenhongguoError: Error, LocalizedError, Sendable, Codable {
    /// 错误码，可以据此判断原因，而不必解析 message。
    public var code: ErrorCode
    public var message: String
    /// 原样重试是否可能成功（限流、网络错误、服务端 5xx）。
    public var retryable: Bool
    /// 服务端建议的等待时间，单位毫秒；0 表示没有建议。
    public var retryAfterMs: Int

    public init(code: ErrorCode, message: String, retryable: Bool = false, retryAfterMs: Int = 0) {
        self.code = code
        self.message = message
        self.retryable = retryable
        self.retryAfterMs = retryAfterMs
    }

    public var errorDescription: String? { message }

    /// 解析 Go 端错误信息里的 JSON；不是 JSON 时归为 unknown。
    static func parse(_ message: String) -> OpenhongguoError {
        (try? JSONDecoder().decode(OpenhongguoError.self, from: Data(message.utf8)))
            ?? OpenhongguoError(code: .unknown, message: message)
    }
}

/// ClientOptions 是 OpenhongguoClient 的设置，全部可选。
public struct ClientOptions: Sendable {
    /// 读取类接口的地址，如 "https://host"；默认与 App 一致。
    public var apiOrigin: String?
    /// 设备注册接口的地址；默认与 App 一致。
    public var registerOrigin: String?
    /// 短剧筛选和活动榜接口的地址；默认与 App 一致。
    public var distributionOrigin: String?
    /// 可重试的失败最多尝试几次，默认 3；0 或 1 表示不重试。
    public var retryAttempts: Int
    /// 设备被拒或播放返回空结果时，一次调用最多换几台新的匿名设备，默认 2；0 表示关闭。
    public var deviceResets: Int
    /// 新注册设备自称的机型；为空的字段取默认值（小米 14）。
    public var device: Device?
    /// 接收诊断日志，在 SDK 的线程里调用；日志里的 URL 已经脱敏。
    public var logger: (@Sendable (String) -> Void)?

    public init(
        apiOrigin: String? = nil, registerOrigin: String? = nil, distributionOrigin: String? = nil,
        retryAttempts: Int = 3, deviceResets: Int = 2, device: Device? = nil, logger: (@Sendable (String) -> Void)? = nil
    ) {
        self.apiOrigin = apiOrigin
        self.registerOrigin = registerOrigin
        self.distributionOrigin = distributionOrigin
        self.retryAttempts = retryAttempts
        self.deviceResets = deviceResets
        self.device = device
        self.logger = logger
    }

    struct Config: Encodable {
        var apiOrigin, registerOrigin, distributionOrigin: String?
        var retryAttempts: Int
        var deviceResets: Int
        var device: Device?
    }

    func json() throws -> String {
        let cfg = Config(
            apiOrigin: apiOrigin, registerOrigin: registerOrigin, distributionOrigin: distributionOrigin,
            retryAttempts: retryAttempts, deviceResets: deviceResets, device: device)
        return String(decoding: try JSONEncoder().encode(cfg), as: UTF8.self)
    }
}

/// Engine 发起一次按名字的调用，结果的 JSON 交给 done；返回的闭包用来取消。
/// 正式实现是 Go 编译出的 core 库，测试里换成假的。
protocol Engine: Sendable {
    func start(_ method: String, _ args: String, _ done: @escaping @Sendable (String) -> Void) -> @Sendable () -> Void
}

private final class Callback: NSObject, CoreCallbackProtocol, @unchecked Sendable {
    let handler: @Sendable (String) -> Void
    init(_ handler: @escaping @Sendable (String) -> Void) { self.handler = handler }
    func done(_ resultJSON: String?) { handler(resultJSON ?? "") }
}

private final class Logger: NSObject, CoreLoggerProtocol, @unchecked Sendable {
    let sink: @Sendable (String) -> Void
    init(_ sink: @escaping @Sendable (String) -> Void) { self.sink = sink }
    func log(_ line: String?) { sink(line ?? "") }
}

// CoreClient 和 CoreCall 由 Go 实现，可以在多个线程里使用。
private struct GoEngine: Engine, @unchecked Sendable {
    let client: CoreClient

    private struct CallRef: @unchecked Sendable {
        let call: CoreCall?
    }

    func start(_ method: String, _ args: String, _ done: @escaping @Sendable (String) -> Void) -> @Sendable () -> Void {
        let ref = CallRef(call: client.start(method, argsJSON: args, callback: Callback(done)))
        return { ref.call?.cancel() }
    }
}

/// 在取消先于调用开始时，记住取消，调用一开始就取消它。
private final class CancelBox: @unchecked Sendable {
    private let lock = NSLock()
    private var cancel: (@Sendable () -> Void)?
    private var cancelled = false

    func set(_ f: @escaping @Sendable () -> Void) {
        lock.lock()
        let now = cancelled
        if !now { cancel = f }
        lock.unlock()
        if now { f() }
    }

    func fire() {
        lock.lock()
        cancelled = true
        let f = cancel
        cancel = nil
        lock.unlock()
        f?()
    }
}

private struct Envelope<T: Decodable>: Decodable {
    var result: T?
    var error: OpenhongguoError?
}

private struct PlaybackEnvelope: Decodable {
    var playback: Playback
    var contentKey: ContentKey?
    var contentKeyError: OpenhongguoError?
}

private struct Nothing: Decodable {}

/// OpenhongguoClient 是红果短剧的匿名客户端。所有方法都是 async，
/// 失败时抛出 OpenhongguoError；Task 取消时，进行中的请求随之取消，
/// 方法抛出 CancellationError。同一个客户端可以并发使用。
public final class OpenhongguoClient: Sendable {
    private let engine: any Engine

    init(engine: any Engine) {
        self.engine = engine
    }

    /// 创建客户端。设备档案默认保存在 Application Support/openhongguo/anonymous.json，
    /// 这个目录不参与 iCloud 和 iTunes 备份，不会迁移到别的设备。
    /// 首次请求时注册一台匿名设备。
    public static func create(profile: URL? = nil, options: ClientOptions = ClientOptions()) throws -> OpenhongguoClient {
        let path = try profile ?? defaultProfileURL()
        var err: NSError?
        let logger = options.logger.map { Logger($0) }
        guard let client = CoreNewClient(path.path, try options.json(), logger, &err) else {
            throw OpenhongguoError.parse(err?.localizedDescription ?? "unknown error")
        }
        return OpenhongguoClient(engine: GoEngine(client: client))
    }

    private static func defaultProfileURL() throws -> URL {
        var dir = try FileManager.default.url(
            for: .applicationSupportDirectory, in: .userDomainMask, appropriateFor: nil, create: true
        ).appendingPathComponent("openhongguo", isDirectory: true)
        try FileManager.default.createDirectory(at: dir, withIntermediateDirectories: true)
        var values = URLResourceValues()
        values.isExcludedFromBackup = true
        try dir.setResourceValues(values)
        return dir.appendingPathComponent("anonymous.json")
    }

    /// 加载一个书城频道（默认“推荐”）。
    public func discover(_ options: DiscoverOptions = DiscoverOptions()) async throws -> Page<SeriesSummary> {
        try await call("discover", options)
    }

    /// 搜索短剧。
    public func search(_ query: String, options: SearchOptions = SearchOptions()) async throws -> Page<SeriesSummary> {
        try await call("search", query, options)
    }

    /// 搜索联想词。
    public func suggest(_ prefix: String) async throws -> [Suggestion] {
        try await list("suggest", prefix)
    }

    /// 搜索页的“猜你想搜”和热搜榜。
    public func hotSearch() async throws -> HotSearch {
        try await call("hotSearch")
    }

    /// 搜索框里滚动的提示词。
    public func searchCues() async throws -> [String] {
        try await list("searchCues")
    }

    /// 上新日历的一天。
    public func newReleases(_ query: NewReleaseQuery = NewReleaseQuery()) async throws -> NewReleasePage {
        try await call("newReleases", query)
    }

    /// 一部剧的详情。
    public func getSeries(_ seriesId: String) async throws -> Series {
        try await call("getSeries", seriesId)
    }

    /// 一次读取多部剧及其各集，按请求顺序返回。
    public func getSeriesBatch(_ seriesIds: [String]) async throws -> [SeriesDetail] {
        try await list("getSeriesBatch", seriesIds)
    }

    /// 一部剧的分集列表。
    public func listEpisodes(_ seriesId: String, options: EpisodeOptions = EpisodeOptions()) async throws -> Page<Episode> {
        try await call("listEpisodes", seriesId, options)
    }

    /// 多季剧的全部季。
    public func seasons(_ seriesId: String) async throws -> Album {
        try await call("seasons", seriesId)
    }

    /// 一部剧的相关推荐。
    public func seriesRecommend(_ seriesId: String) async throws -> Recommend {
        try await call("seriesRecommend", seriesId)
    }

    /// 一个合集及其各季。
    public func getAlbum(_ albumId: String) async throws -> Album {
        try await call("getAlbum", albumId)
    }

    /// 短剧排行榜的一页。
    public func shortRank(_ query: ShortRankQuery = ShortRankQuery()) async throws -> ShortRanking {
        try await call("shortRank", query)
    }

    /// 短剧筛选面板；genre 如 "comic_series"。
    public func shortFilterPanel(_ genre: String = "") async throws -> [FilterRow] {
        try await list("shortFilterPanel", genre)
    }

    /// 按筛选条件列出短剧。
    public func shortFilter(_ query: ShortFilterQuery = ShortFilterQuery()) async throws -> ShortFilterPage {
        try await call("shortFilter", query)
    }

    /// 活动榜单。
    public func ranking(activityId: String, rankListId: String) async throws -> Ranking {
        try await call("ranking", activityId, rankListId)
    }

    /// 书城榜单的一页。
    public func rankRows(_ cellId: String, offset: Int = 0) async throws -> Ranking {
        try await call("rankRows", cellId, offset)
    }

    /// 书城榜单的类型。
    public func rankGroups() async throws -> [RankList] {
        try await list("rankGroups")
    }

    /// 演员或角色主页。
    public func person(_ userId: String) async throws -> Person {
        try await call("person", userId)
    }

    /// 演员的作品，一页。
    public func personWorks(_ userId: String, cursor: String = "") async throws -> WorkPage {
        try await call("personWorks", userId, cursor)
    }

    /// 书籍分类的标签页。
    public func categories(_ categoryId: String) async throws -> CategoryPage {
        try await call("categories", categoryId)
    }

    /// 书籍分类下的列表。
    public func categoryLanding(_ categoryId: String, options: LandingOptions = LandingOptions()) async throws -> CategoryPage {
        try await call("categoryLanding", categoryId, options)
    }

    /// 一集的播放模型；加密剧集同时给出内容密钥。
    public func resolvePlayback(_ episodeId: String) async throws -> PlaybackResult {
        let r: PlaybackEnvelope = try await call("resolvePlayback", episodeId)
        return PlaybackResult(playback: r.playback, contentKey: r.contentKey, contentKeyError: r.contentKeyError)
    }

    /// 丢弃当前匿名设备，下次请求时重新注册。
    public func resetDevice() async throws {
        let _: Nothing? = try await optional("resetDevice")
    }

    /// 沿着游标逐部返回书城频道的剧，去掉跨页重复。
    public func discoverAll(_ options: DiscoverOptions = DiscoverOptions(), maxPages: Int = .max) -> AsyncThrowingStream<SeriesSummary, Error> {
        walk(start: options.cursor, maxPages: maxPages) { [self] cursor in
            var o = options
            o.cursor = cursor
            return try await discover(o)
        }
    }

    /// 沿着游标逐部返回搜索结果，去掉跨页重复。
    public func searchAll(_ query: String, options: SearchOptions = SearchOptions(), maxPages: Int = .max) -> AsyncThrowingStream<SeriesSummary, Error> {
        walk(start: options.cursor, maxPages: maxPages) { [self] cursor in
            var o = options
            o.cursor = cursor
            return try await search(query, options: o)
        }
    }

    private func walk(
        start: String?, maxPages: Int, fetch: @escaping @Sendable (String) async throws -> Page<SeriesSummary>
    ) -> AsyncThrowingStream<SeriesSummary, Error> {
        AsyncThrowingStream { continuation in
            let task = Task {
                do {
                    var seen = Set<String>()
                    var cursor = start ?? ""
                    var pages = 0
                    while pages < maxPages {
                        pages += 1
                        let page = try await fetch(cursor)
                        for item in page.items where seen.insert(item.id).inserted {
                            continuation.yield(item)
                        }
                        if !page.hasMore || page.cursor.isEmpty { break }
                        cursor = page.cursor
                    }
                    continuation.finish()
                } catch {
                    continuation.finish(throwing: error)
                }
            }
            continuation.onTermination = { _ in task.cancel() }
        }
    }

    // MARK: - 调用 Go 端

    private func call<T: Decodable>(_ method: String, _ args: any Encodable...) async throws -> T {
        guard let v: T = try await invoke(method, args) else {
            throw OpenhongguoError(code: .unknown, message: "\(method) returned no result")
        }
        return v
    }

    /// list 用于返回列表的方法：Go 的空切片编码成 null，这里换成空数组。
    private func list<T: Decodable>(_ method: String, _ args: any Encodable...) async throws -> [T] {
        let v: [T]? = try await invoke(method, args)
        return v ?? []
    }

    private func optional<T: Decodable>(_ method: String, _ args: any Encodable...) async throws -> T? {
        try await invoke(method, args)
    }

    private func invoke<T: Decodable>(_ method: String, _ args: [any Encodable]) async throws -> T? {
        let encoder = JSONEncoder()
        let parts = try args.map { String(decoding: try encoder.encode($0), as: UTF8.self) }
        let argsJSON = "[" + parts.joined(separator: ",") + "]"

        let box = CancelBox()
        let engine = self.engine
        let out: String = await withTaskCancellationHandler {
            await withCheckedContinuation { (cont: CheckedContinuation<String, Never>) in
                box.set(engine.start(method, argsJSON) { cont.resume(returning: $0) })
            }
        } onCancel: {
            box.fire()
        }
        try Task.checkCancellation()

        let env = try JSONDecoder().decode(Envelope<T>.self, from: Data(out.utf8))
        if let e = env.error { throw e }
        return env.result
    }
}
