part of '../openhongguo.dart';

/// PagedResult 是一页匿名结果。
///
/// Go 版叫 Page；这里换了名字，避免和 Flutter widgets 的 Page 重名。
class PagedResult<T> {
  final List<T> items;

  /// 下一页的游标；没有下一页时为空。
  final String cursor;
  final bool hasMore;

  const PagedResult({this.items = const [], this.cursor = '', this.hasMore = false});

  factory PagedResult._fromJson(Object? v, T Function(Map<String, dynamic>) item) {
    final j = _asMap(v);
    return PagedResult(
      items: _asList(j['items'], (e) => item(_asMap(e))),
      cursor: _asString(j['cursor']),
      hasMore: _asBool(j['has_more']),
    );
  }
}

/// SearchFilter 是综合 tab 的搜索筛选项（[SearchOptions.filters] 的取值）。
abstract final class SearchFilter {
  /// 真人剧
  static const shortPlay = 'content_type_short_play';

  /// 漫剧
  static const comic = 'content_type_dynamic_comic';

  /// AI剧
  static const aiComic = 'content_type_ai_dynamic_comic';

  /// 影视剧
  static const film = 'content_type_long_video';

  /// 未看过
  static const unwatched = 'search_range_unread';
}

/// PlaybackResult 是 resolvePlayback 的结果。
class PlaybackResult {
  final Playback playback;

  /// 加密剧集的内容密钥；未加密或推导失败时为 null。
  final ContentKey? contentKey;

  /// 加密剧集的密钥推导失败时，说明原因。
  final OpenhongguoException? contentKeyError;

  const PlaybackResult({required this.playback, this.contentKey, this.contentKeyError});
}

/// 错误码，对应 Go SDK 的哨兵错误。
enum ErrorCode {
  invalidOptions('INVALID_OPTIONS'),
  profileNotFound('PROFILE_NOT_FOUND'),
  signerUnavailable('SIGNER_UNAVAILABLE'),
  authenticationRequired('AUTHENTICATION_REQUIRED'),
  notProtected('NOT_PROTECTED'),
  drmProtected('DRM_PROTECTED'),
  anonymousLimited('ANONYMOUS_LIMITED'),
  rateLimited('RATE_LIMITED'),
  transport('TRANSPORT'),
  signatureRejected('SIGNATURE_REJECTED'),
  contentUnavailable('CONTENT_UNAVAILABLE'),
  paramInvalid('PARAM_INVALID'),
  challengeRequired('CHALLENGE_REQUIRED'),
  deviceRejected('DEVICE_REJECTED'),
  endOfList('END_OF_LIST'),
  albumUnavailable('ALBUM_UNAVAILABLE'),
  cancelled('CANCELLED'),
  unknown('UNKNOWN');

  /// Go 端使用的名字，如 RATE_LIMITED。
  final String wire;

  const ErrorCode(this.wire);

  static ErrorCode _of(Object? wire) => values.firstWhere((c) => c.wire == wire, orElse: () => unknown);
}

/// OpenhongguoException 是 SDK 返回的错误。
class OpenhongguoException implements Exception {
  /// 错误码，可以据此判断原因，而不必解析 message。
  final ErrorCode code;
  final String message;

  /// 原样重试是否可能成功（限流、网络错误、服务端 5xx）。
  final bool retryable;

  /// 服务端建议的等待时间，单位毫秒；0 表示没有建议。
  final int retryAfterMs;

  const OpenhongguoException(this.code, this.message, {this.retryable = false, this.retryAfterMs = 0});

  factory OpenhongguoException._fromJson(Object? v) {
    final j = _asMap(v);
    return OpenhongguoException(
      ErrorCode._of(j['code']),
      _asString(j['message']),
      retryable: _asBool(j['retryable']),
      retryAfterMs: _asInt(j['retryAfterMs']),
    );
  }

  @override
  String toString() => 'OpenhongguoException(${code.wire}): $message';
}

/// ClientOptions 是 OpenhongguoClient 的设置，全部可选。
class ClientOptions {
  /// 设备档案的路径；默认在 App 不参与备份的目录里。
  final String? profilePath;

  /// 读取类接口的地址，如 "https://host"；默认与 App 一致。
  final String? apiOrigin;

  /// 设备注册接口的地址；默认与 App 一致。
  final String? registerOrigin;

  /// 短剧筛选和活动榜接口的地址；默认与 App 一致。
  final String? distributionOrigin;

  /// 可重试的失败最多尝试几次，默认 3；0 或 1 表示不重试。
  final int retryAttempts;

  /// 设备被拒或播放返回空结果时，一次调用最多换几台新的匿名设备，默认 2；0 表示关闭。
  final int deviceResets;

  /// 新注册设备自称的机型；为空的字段取默认值（小米 14）。
  final Device? device;

  /// 接收诊断日志；日志里的 URL 已经脱敏。
  final void Function(String line)? log;

  const ClientOptions({
    this.profilePath,
    this.apiOrigin,
    this.registerOrigin,
    this.distributionOrigin,
    this.retryAttempts = 3,
    this.deviceResets = 2,
    this.device,
    this.log,
  });

  String _configJson() => jsonEncode({
        if (apiOrigin != null) 'apiOrigin': apiOrigin,
        if (registerOrigin != null) 'registerOrigin': registerOrigin,
        if (distributionOrigin != null) 'distributionOrigin': distributionOrigin,
        'retryAttempts': retryAttempts,
        'deviceResets': deviceResets,
        if (device != null) 'device': device!.toJson(),
      });
}

const _channel = MethodChannel('dev.openhongguo/flutter');

/// 各客户端的日志回调，按原生端的客户端编号区分。
final _loggers = <int, void Function(String)>{};
var _handlerSet = false;

void _setHandler() {
  if (_handlerSet) return;
  _handlerSet = true;
  _channel.setMethodCallHandler((call) async {
    if (call.method == 'log') {
      final args = _asMap(call.arguments);
      _loggers[_asInt(args['client'])]?.call(_asString(args['line']));
    }
  });
}

/// OpenhongguoClient 是红果短剧的匿名客户端。所有方法都返回 Future，
/// 失败时抛出 OpenhongguoException。同一个客户端可以并发使用。
class OpenhongguoClient {
  final int _id;

  OpenhongguoClient._(this._id);

  /// 创建客户端。首次请求时，SDK 注册一台匿名设备，并保存在
  /// [ClientOptions.profilePath]（默认在 App 不参与备份的目录里）。
  static Future<OpenhongguoClient> create([ClientOptions options = const ClientOptions()]) async {
    _setHandler();
    final out = await _channel.invokeMethod<String>('createClient', {
      'profilePath': options.profilePath,
      'config': options._configJson(),
      'log': options.log != null,
    });
    final res = _asMap(jsonDecode(out ?? '{}'));
    if (res['error'] != null) throw OpenhongguoException._fromJson(res['error']);
    final id = _asInt(res['id']);
    if (options.log != null) _loggers[id] = options.log!;
    return OpenhongguoClient._(id);
  }

  Future<Object?> _call(String method, [List<Object?> args = const []]) async {
    final out = await _channel.invokeMethod<String>('call', {
      'client': _id,
      'method': method,
      'args': jsonEncode(args),
    });
    final env = _asMap(jsonDecode(out ?? '{}'));
    if (env['error'] != null) throw OpenhongguoException._fromJson(env['error']);
    return env['result'];
  }

  /// 加载一个书城频道（默认“推荐”）。
  Future<PagedResult<SeriesSummary>> discover([DiscoverOptions options = const DiscoverOptions()]) async =>
      PagedResult._fromJson(await _call('discover', [options.toJson()]), SeriesSummary.fromJson);

  /// 搜索短剧。
  Future<PagedResult<SeriesSummary>> search(String query, [SearchOptions options = const SearchOptions()]) async =>
      PagedResult._fromJson(await _call('search', [query, options.toJson()]), SeriesSummary.fromJson);

  /// 搜索联想词。
  Future<List<Suggestion>> suggest(String prefix) async =>
      _asList(await _call('suggest', [prefix]), (e) => Suggestion.fromJson(_asMap(e)));

  /// 搜索页的“猜你想搜”和热搜榜。
  Future<HotSearch> hotSearch() async => HotSearch.fromJson(_asMap(await _call('hotSearch')));

  /// 搜索框里滚动的提示词。
  Future<List<String>> searchCues() async => _asList(await _call('searchCues'), _asString);

  /// 上新日历的一天。
  Future<NewReleasePage> newReleases([NewReleaseQuery query = const NewReleaseQuery()]) async =>
      NewReleasePage.fromJson(_asMap(await _call('newReleases', [query.toJson()])));

  /// 一部剧的详情。
  Future<Series> getSeries(String seriesId) async => Series.fromJson(_asMap(await _call('getSeries', [seriesId])));

  /// 一次读取多部剧及其各集，按请求顺序返回。
  Future<List<SeriesDetail>> getSeriesBatch(List<String> seriesIds) async =>
      _asList(await _call('getSeriesBatch', [seriesIds]), (e) => SeriesDetail.fromJson(_asMap(e)));

  /// 一部剧的分集列表。
  Future<PagedResult<Episode>> listEpisodes(String seriesId, [EpisodeOptions options = const EpisodeOptions()]) async =>
      PagedResult._fromJson(await _call('listEpisodes', [seriesId, options.toJson()]), Episode.fromJson);

  /// 多季剧的全部季。
  Future<Album> seasons(String seriesId) async => Album.fromJson(_asMap(await _call('seasons', [seriesId])));

  /// 一部剧的相关推荐。
  Future<Recommend> seriesRecommend(String seriesId) async =>
      Recommend.fromJson(_asMap(await _call('seriesRecommend', [seriesId])));

  /// 一个合集及其各季。
  Future<Album> getAlbum(String albumId) async => Album.fromJson(_asMap(await _call('getAlbum', [albumId])));

  /// 短剧排行榜的一页。
  Future<ShortRanking> shortRank([ShortRankQuery query = const ShortRankQuery()]) async =>
      ShortRanking.fromJson(_asMap(await _call('shortRank', [query.toJson()])));

  /// 短剧筛选面板；genre 如 "comic_series"。
  Future<List<FilterRow>> shortFilterPanel([String genre = '']) async =>
      _asList(await _call('shortFilterPanel', [genre]), (e) => FilterRow.fromJson(_asMap(e)));

  /// 按筛选条件列出短剧。
  Future<ShortFilterPage> shortFilter([ShortFilterQuery query = const ShortFilterQuery()]) async =>
      ShortFilterPage.fromJson(_asMap(await _call('shortFilter', [query.toJson()])));

  /// 活动榜单。
  Future<Ranking> ranking(String activityId, String rankListId) async =>
      Ranking.fromJson(_asMap(await _call('ranking', [activityId, rankListId])));

  /// 书城榜单的一页。
  Future<Ranking> rankRows(String cellId, [int offset = 0]) async =>
      Ranking.fromJson(_asMap(await _call('rankRows', [cellId, offset])));

  /// 书城榜单的类型。
  Future<List<RankList>> rankGroups() async =>
      _asList(await _call('rankGroups'), (e) => RankList.fromJson(_asMap(e)));

  /// 演员或角色主页。
  Future<Person> person(String userId) async => Person.fromJson(_asMap(await _call('person', [userId])));

  /// 演员的作品，一页。
  Future<WorkPage> personWorks(String userId, [String cursor = '']) async =>
      WorkPage.fromJson(_asMap(await _call('personWorks', [userId, cursor])));

  /// 书籍分类的标签页。
  Future<CategoryPage> categories(String categoryId) async =>
      CategoryPage.fromJson(_asMap(await _call('categories', [categoryId])));

  /// 书籍分类下的列表。
  Future<CategoryPage> categoryLanding(String categoryId, [LandingOptions options = const LandingOptions()]) async =>
      CategoryPage.fromJson(_asMap(await _call('categoryLanding', [categoryId, options.toJson()])));

  /// 一集的播放模型；加密剧集同时给出内容密钥。
  Future<PlaybackResult> resolvePlayback(String episodeId) async {
    final j = _asMap(await _call('resolvePlayback', [episodeId]));
    return PlaybackResult(
      playback: Playback.fromJson(_asMap(j['playback'])),
      contentKey: j['contentKey'] is Map ? ContentKey.fromJson(_asMap(j['contentKey'])) : null,
      contentKeyError: j['contentKeyError'] is Map ? OpenhongguoException._fromJson(j['contentKeyError']) : null,
    );
  }

  /// 丢弃当前匿名设备，下次请求时重新注册。
  Future<void> resetDevice() => _call('resetDevice');

  /// 沿着游标逐部返回书城频道的剧，去掉跨页重复。
  async.Stream<SeriesSummary> discoverAll([DiscoverOptions options = const DiscoverOptions(), int? maxPages]) =>
      _walk(options.cursor, maxPages, (cursor) => discover(options.copyWith(cursor: cursor)));

  /// 沿着游标逐部返回搜索结果，去掉跨页重复。
  async.Stream<SeriesSummary> searchAll(String query, [SearchOptions options = const SearchOptions(), int? maxPages]) =>
      _walk(options.cursor, maxPages, (cursor) => search(query, options.copyWith(cursor: cursor)));

  async.Stream<SeriesSummary> _walk(
    String? start,
    int? maxPages,
    Future<PagedResult<SeriesSummary>> Function(String cursor) fetch,
  ) async* {
    final seen = <String>{};
    var cursor = start ?? '';
    for (var pages = 0; maxPages == null || pages < maxPages; pages++) {
      final page = await fetch(cursor);
      for (final item in page.items) {
        if (seen.add(item.id)) yield item;
      }
      if (!page.hasMore || page.cursor.isEmpty) return;
      cursor = page.cursor;
    }
  }

  /// 释放原生端的客户端；之后的调用都会失败。
  Future<void> release() async {
    _loggers.remove(_id);
    await _channel.invokeMethod<void>('releaseClient', {'client': _id});
  }
}
