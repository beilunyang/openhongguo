// 由 go run ./bindings/mobile/gen 生成，请勿手工修改。

part of '../openhongguo.dart';

/// Album 是多季的合集，来自 /novel/player/album_detail/v1/。
class Album {
  final String id;
  final String title;
  final String description;
  final String coverUrl;
  final String coverExpiresAt;
  final int seasonCount;
  final List<Season> seasons;

  const Album({
    this.id = '',
    this.title = '',
    this.description = '',
    this.coverUrl = '',
    this.coverExpiresAt = '',
    this.seasonCount = 0,
    this.seasons = const [],
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory Album.fromJson(Map<String, dynamic> j) => Album(
        id: _asString(j['id']),
        title: _asString(j['title']),
        description: _asString(j['description']),
        coverUrl: _asString(j['cover_url']),
        coverExpiresAt: _asString(j['cover_expires_at']),
        seasonCount: _asInt(j['season_count']),
        seasons: _asList(j['seasons'], (e) => Season.fromJson(_asMap(e))),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'id': id,
        'title': title,
        'description': description,
        'cover_url': coverUrl,
        'cover_expires_at': coverExpiresAt,
        'season_count': seasonCount,
        'seasons': seasons.map((e) => e.toJson()).toList(),
      };
}

/// CategoryPage 是一个分类落地页及其筛选项。
class CategoryPage {
  final String categoryId;
  final String categoryName;
  final String description;
  final List<FilterGroup> filters;
  /// Tabs 是受众标签页（全部、男生、女生），每个 Value 是标签页类型编号；
  /// DefaultTab 是页面打开时所在的标签页类型。
  final List<FilterOption> tabs;
  final int defaultTab;
  final List<SeriesSummary> items;
  final bool hasMore;
  /// NextOffset 是下一页的偏移量，即 LandingOptions.Cursor 的值。
  final int nextOffset;

  const CategoryPage({
    this.categoryId = '',
    this.categoryName = '',
    this.description = '',
    this.filters = const [],
    this.tabs = const [],
    this.defaultTab = 0,
    this.items = const [],
    this.hasMore = false,
    this.nextOffset = 0,
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory CategoryPage.fromJson(Map<String, dynamic> j) => CategoryPage(
        categoryId: _asString(j['category_id']),
        categoryName: _asString(j['category_name']),
        description: _asString(j['description']),
        filters: _asList(j['filters'], (e) => FilterGroup.fromJson(_asMap(e))),
        tabs: _asList(j['tabs'], (e) => FilterOption.fromJson(_asMap(e))),
        defaultTab: _asInt(j['default_tab']),
        items: _asList(j['items'], (e) => SeriesSummary.fromJson(_asMap(e))),
        hasMore: _asBool(j['has_more']),
        nextOffset: _asInt(j['next_offset']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'category_id': categoryId,
        'category_name': categoryName,
        'description': description,
        'filters': filters.map((e) => e.toJson()).toList(),
        'tabs': tabs.map((e) => e.toJson()).toList(),
        'default_tab': defaultTab,
        'items': items.map((e) => e.toJson()).toList(),
        'has_more': hasMore,
        'next_offset': nextOffset,
      };
}

/// ContentKey 是受保护播放模型的 CENC（cenc-aes-ctr）内容密钥，
/// 由 sdk.ContentKey 还原。
class ContentKey {
  /// Hex 是 16 字节 AES 密钥的小写十六进制形式，
  /// 即 ffmpeg 的 -decryption_key 和播放器的 ClearKey 配置接受的形式。
  final String key;
  /// Version 是播放器在密钥旁读取的方案标记。
  final String version;

  const ContentKey({
    this.key = '',
    this.version = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory ContentKey.fromJson(Map<String, dynamic> j) => ContentKey(
        key: _asString(j['key']),
        version: _asString(j['version']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'key': key,
        'version': version,
      };
}

/// ContentType 是视频的 content_type，
/// 取自 com.dragon.read.rpc.model.VideoContentType。
/// 官方 App 的枚举混用小数值和从 1000 起的一段数值；这里两者都复现。
class ContentType {
  final int value;

  const ContentType(this.value);

  static const unknown = ContentType(0);
  /// 真人短剧
  static const shortSeries = ContentType(1);
  /// 场景剧
  static const scenePlay = ContentType(2);
  /// 专家荐书
  static const expertRecommend = ContentType(3);
  /// PPT 荐书
  static const pptRecommend = ContentType(4);
  /// 电影
  static const movie = ContentType(5);
  /// 电视剧
  static const telePlay = ContentType(6);
  /// 短剧
  static const shortPlay = ContentType(7);
  /// 投放素材
  static const dropMaterial = ContentType(1000);
  /// PUGC
  static const pugc = ContentType(1001);
  /// 虚拍短剧
  static const unrealShortPlay = ContentType(1002);
  /// 合集
  static const album = ContentType(1003);
  /// 漫剧（动态漫）
  static const motionComic = ContentType(1004);

  @override
  bool operator ==(Object other) => other is ContentType && other.value == value;

  @override
  int get hashCode => value.hashCode;

  @override
  String toString() => 'ContentType($value)';
}

/// Device 是匿名身份出示的手机。注册请求体、
/// 每个请求的设备参数和 User-Agent 都描述它，所以一个设备 id 始终对应一个机型。
class Device {
  /// Brand 是 device_brand，也是制造商（"Xiaomi"）；Model 是机型代号，
  /// 作为 device_type、device_model 发送，也写进 User-Agent（"23127PN0CC"）。
  final String brand;
  final String model;
  /// OSVersion 是 Android 版本（"14"），OSAPI 是其 API 级别（34），
  /// Build 是 User-Agent 携带的构建 id（"UKQ1.230804.001"）。
  final String osVersion;
  final int osApi;
  final String build;
  /// Resolution 是 "height*width"，单位是像素；DPI 是屏幕密度，
  /// 单位是每英寸点数。
  final String resolution;
  final int dpi;
  /// Pad 让设备以平板身份出现（dragon_device_type=pad）。
  final bool pad;
  /// Channel 是 App 构建上报的分发渠道（"xiaomi_8662_64" 来自小米应用商店）。
  final String channel;

  const Device({
    this.brand = '',
    this.model = '',
    this.osVersion = '',
    this.osApi = 0,
    this.build = '',
    this.resolution = '',
    this.dpi = 0,
    this.pad = false,
    this.channel = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory Device.fromJson(Map<String, dynamic> j) => Device(
        brand: _asString(j['brand']),
        model: _asString(j['model']),
        osVersion: _asString(j['os_version']),
        osApi: _asInt(j['os_api']),
        build: _asString(j['build']),
        resolution: _asString(j['resolution']),
        dpi: _asInt(j['dpi']),
        pad: _asBool(j['pad']),
        channel: _asString(j['channel']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'brand': brand,
        'model': model,
        'os_version': osVersion,
        'os_api': osApi,
        'build': build,
        'resolution': resolution,
        'dpi': dpi,
        'pad': pad,
        'channel': channel,
      };
}

/// Episode 是一个公开列出的分集。Number 对调用方从 1 开始。
class Episode {
  final String id;
  final String seriesId;
  final int number;
  final String title;
  /// 服务端禁止播放该分集时，Playable 为 false（video_list[].disable_play）。
  final bool playable;
  /// Vertical 标记竖屏画幅的分集。
  final bool vertical;
  /// IsNew 标记本次更新中发布的分集。
  final bool isNew;
  /// CoverURL 是分集剧照；CoverExpiresAt 是它的签名过期时间。
  final String coverUrl;
  final String coverExpiresAt;
  /// ContentType 与 Series.ContentType 是同一枚举。
  final ContentType contentType;
  /// CommentCount 和 DiggCount 是分集的计数。
  final int commentCount;
  final int diggCount;

  const Episode({
    this.id = '',
    this.seriesId = '',
    this.number = 0,
    this.title = '',
    this.playable = false,
    this.vertical = false,
    this.isNew = false,
    this.coverUrl = '',
    this.coverExpiresAt = '',
    this.contentType = const ContentType(0),
    this.commentCount = 0,
    this.diggCount = 0,
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory Episode.fromJson(Map<String, dynamic> j) => Episode(
        id: _asString(j['id']),
        seriesId: _asString(j['series_id']),
        number: _asInt(j['number']),
        title: _asString(j['title']),
        playable: _asBool(j['playable']),
        vertical: _asBool(j['vertical']),
        isNew: _asBool(j['is_new']),
        coverUrl: _asString(j['cover_url']),
        coverExpiresAt: _asString(j['cover_expires_at']),
        contentType: ContentType(_asInt(j['content_type'])),
        commentCount: _asInt(j['comment_count']),
        diggCount: _asInt(j['digg_count']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'id': id,
        'series_id': seriesId,
        'number': number,
        'title': title,
        'playable': playable,
        'vertical': vertical,
        'is_new': isNew,
        'cover_url': coverUrl,
        'cover_expires_at': coverExpiresAt,
        'content_type': contentType.value,
        'comment_count': commentCount,
        'digg_count': diggCount,
      };
}

/// FilterGroup 是一个筛选维度（状态、排序、子分类、篇幅）。
class FilterGroup {
  /// Key 是该组设置的查询参数，例如 sort_by。
  final String key;
  final List<FilterOption> options;

  const FilterGroup({
    this.key = '',
    this.options = const [],
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory FilterGroup.fromJson(Map<String, dynamic> j) => FilterGroup(
        key: _asString(j['key']),
        options: _asList(j['options'], (e) => FilterOption.fromJson(_asMap(e))),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'key': key,
        'options': options.map((e) => e.toJson()).toList(),
      };
}

/// FilterOption 是分类页上的一个可选值。
class FilterOption {
  /// Name 是显示的标签；Value 是查询选中它时发送的值。
  final String name;
  final String value;

  const FilterOption({
    this.name = '',
    this.value = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory FilterOption.fromJson(Map<String, dynamic> j) => FilterOption(
        name: _asString(j['name']),
        value: _asString(j['value']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'name': name,
        'value': value,
      };
}

/// FilterRow 是短剧筛选页的一个选择行。Key 是该行的 select_items 键（genre、
/// category_dim_theme、category_dim_role、category_dim_epoch、sort、gender、
/// online_time、creation_status）；选项的 Value 是查询所选的值。
/// 有哪些行取决于 genre。
class FilterRow {
  final String key;
  final String name;
  final List<FilterOption> options;

  const FilterRow({
    this.key = '',
    this.name = '',
    this.options = const [],
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory FilterRow.fromJson(Map<String, dynamic> j) => FilterRow(
        key: _asString(j['key']),
        name: _asString(j['name']),
        options: _asList(j['options'], (e) => FilterOption.fromJson(_asMap(e))),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'key': key,
        'name': name,
        'options': options.map((e) => e.toJson()).toList(),
      };
}

/// HotBoard 是一个热榜。作品榜（短剧热搜榜、漫剧热搜榜、短剧新剧榜）填充 Items；
/// 热点话题榜填充 Topics。出现哪些榜单因调用而异。
class HotBoard {
  final String name;
  final List<HotItem> items;
  final List<HotTopic> topics;

  const HotBoard({
    this.name = '',
    this.items = const [],
    this.topics = const [],
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory HotBoard.fromJson(Map<String, dynamic> j) => HotBoard(
        name: _asString(j['name']),
        items: _asList(j['items'], (e) => HotItem.fromJson(_asMap(e))),
        topics: _asList(j['topics'], (e) => HotTopic.fromJson(_asMap(e))),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'name': name,
        'items': items.map((e) => e.toJson()).toList(),
        'topics': topics.map((e) => e.toJson()).toList(),
      };
}

/// HotItem 是热榜上的一部作品，按榜单顺序排列。
class HotItem {
  /// Position 是在榜单上的名次，从 1 开始。
  final int position;
  final String seriesId;
  final String title;
  final String coverUrl;
  final String coverExpiresAt;
  final int episodeCount;
  final ContentType contentType;
  /// SearchScore 是榜单自己的数值（"355万热搜值"，有时带 "·第1季"）；
  /// Heat 是作品的总热度（"4102万热度"）。
  final String searchScore;
  final String heat;
  /// Tag 是角标，例如新剧。
  final String tag;

  const HotItem({
    this.position = 0,
    this.seriesId = '',
    this.title = '',
    this.coverUrl = '',
    this.coverExpiresAt = '',
    this.episodeCount = 0,
    this.contentType = const ContentType(0),
    this.searchScore = '',
    this.heat = '',
    this.tag = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory HotItem.fromJson(Map<String, dynamic> j) => HotItem(
        position: _asInt(j['position']),
        seriesId: _asString(j['series_id']),
        title: _asString(j['title']),
        coverUrl: _asString(j['cover_url']),
        coverExpiresAt: _asString(j['cover_expires_at']),
        episodeCount: _asInt(j['episode_count']),
        contentType: ContentType(_asInt(j['content_type'])),
        searchScore: _asString(j['search_score']),
        heat: _asString(j['heat']),
        tag: _asString(j['tag']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'position': position,
        'series_id': seriesId,
        'title': title,
        'cover_url': coverUrl,
        'cover_expires_at': coverExpiresAt,
        'episode_count': episodeCount,
        'content_type': contentType.value,
        'search_score': searchScore,
        'heat': heat,
        'tag': tag,
      };
}

/// HotSearch 是查询前的搜索页：猜你想搜一行，以及其下的热榜。
class HotSearch {
  final List<HotItem> guesses;
  final List<HotBoard> boards;

  const HotSearch({
    this.guesses = const [],
    this.boards = const [],
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory HotSearch.fromJson(Map<String, dynamic> j) => HotSearch(
        guesses: _asList(j['guesses'], (e) => HotItem.fromJson(_asMap(e))),
        boards: _asList(j['boards'], (e) => HotBoard.fromJson(_asMap(e))),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'guesses': guesses.map((e) => e.toJson()).toList(),
        'boards': boards.map((e) => e.toJson()).toList(),
      };
}

/// HotTopic 是热点话题榜的一项。Query 是它打开的搜索。
class HotTopic {
  final String id;
  final String title;
  final String query;
  final String heat;

  const HotTopic({
    this.id = '',
    this.title = '',
    this.query = '',
    this.heat = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory HotTopic.fromJson(Map<String, dynamic> j) => HotTopic(
        id: _asString(j['id']),
        title: _asString(j['title']),
        query: _asString(j['query']),
        heat: _asString(j['heat']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'id': id,
        'title': title,
        'query': query,
        'heat': heat,
      };
}

/// InfoKind 是 secondary_infos[].data_type，
/// 取自 com.dragon.read.rpc.model.SecondaryInfoDataType。
class InfoKind {
  final int value;

  const InfoKind(this.value);

  /// InfoSeasonOrder 是 "第 N 季" 那一行（实测值为 0）。
  static const seasonOrder = InfoKind(0);
  /// InfoRecommendReason 是推荐理由。
  static const recommendReason = InfoKind(1);
  /// InfoRankScore 是榜单名次。
  static const rankScore = InfoKind(2);
  /// InfoCategory 是分类标签，例如 "末世"（实测值为 3）。
  static const category = InfoKind(3);
  /// InfoScore 是用户评分。
  static const score = InfoKind(4);
  /// InfoAuthorName 是作者名。
  static const authorName = InfoKind(8);
  /// InfoTags 是标签列表。
  static const tags = InfoKind(9);
  /// InfoCreationStatus 是连载中或已完结的说明。
  static const creationStatus = InfoKind(11);
  /// InfoTwoActors 是演员行 "演员·A B"（实测值为 33）。
  static const twoActors = InfoKind(33);
  /// InfoPlotTag 是剧情标签。
  static const plotTag = InfoKind(40);

  @override
  bool operator ==(Object other) => other is InfoKind && other.value == value;

  @override
  int get hashCode => value.hashCode;

  @override
  String toString() => 'InfoKind($value)';
}

/// InfoRow 是 secondary_infos 的一项：详情页上带标签的一行。
class InfoRow {
  final InfoKind kind;
  final String text;
  /// Schema 是该行打开的应用内链接（如果有）。
  final String schema;

  const InfoRow({
    this.kind = const InfoKind(0),
    this.text = '',
    this.schema = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory InfoRow.fromJson(Map<String, dynamic> j) => InfoRow(
        kind: InfoKind(_asInt(j['kind'])),
        text: _asString(j['text']),
        schema: _asString(j['schema']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'kind': kind.value,
        'text': text,
        'schema': schema,
      };
}

/// NewRelease 是上新日历中的一部作品：已上线（Online），
/// 或已预告在 PublishAt 上线。
class NewRelease {
  final String seriesId;
  final String title;
  final String description;
  final String coverUrl;
  final String coverExpiresAt;
  final int episodeCount;
  /// 作品已上线时 Online 为 true；PublishAt 是预告作品的上线时间。
  final bool online;
  final String publishAt;
  final List<String> categories;
  /// 已上线的作品 Heat 形如 "378万热度"，预告作品形如 "159.7万人预约"；
  /// Note 是预告文案（"预计10月26日上线"）。
  final String heat;
  final String note;

  const NewRelease({
    this.seriesId = '',
    this.title = '',
    this.description = '',
    this.coverUrl = '',
    this.coverExpiresAt = '',
    this.episodeCount = 0,
    this.online = false,
    this.publishAt = '',
    this.categories = const [],
    this.heat = '',
    this.note = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory NewRelease.fromJson(Map<String, dynamic> j) => NewRelease(
        seriesId: _asString(j['series_id']),
        title: _asString(j['title']),
        description: _asString(j['description']),
        coverUrl: _asString(j['cover_url']),
        coverExpiresAt: _asString(j['cover_expires_at']),
        episodeCount: _asInt(j['episode_count']),
        online: _asBool(j['online']),
        publishAt: _asString(j['publish_at']),
        categories: _asList(j['categories'], (e) => _asString(e)),
        heat: _asString(j['heat']),
        note: _asString(j['note']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'series_id': seriesId,
        'title': title,
        'description': description,
        'cover_url': coverUrl,
        'cover_expires_at': coverExpiresAt,
        'episode_count': episodeCount,
        'online': online,
        'publish_at': publishAt,
        'categories': categories,
        'heat': heat,
        'note': note,
      };
}

/// NewReleasePage 是上新日历中的一天。
class NewReleasePage {
  /// Date 是本次返回的日期（YYYYMMDD）；Dates 是日历提供的日期，
  /// 即今天前后各一周。
  final String date;
  final List<String> dates;
  final List<NewRelease> items;
  final bool hasMore;
  final String cursor;

  const NewReleasePage({
    this.date = '',
    this.dates = const [],
    this.items = const [],
    this.hasMore = false,
    this.cursor = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory NewReleasePage.fromJson(Map<String, dynamic> j) => NewReleasePage(
        date: _asString(j['date']),
        dates: _asList(j['dates'], (e) => _asString(e)),
        items: _asList(j['items'], (e) => NewRelease.fromJson(_asMap(e))),
        hasMore: _asBool(j['has_more']),
        cursor: _asString(j['cursor']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'date': date,
        'dates': dates,
        'items': items.map((e) => e.toJson()).toList(),
        'has_more': hasMore,
        'cursor': cursor,
      };
}

/// Novel 是短剧改编自的小说。
class Novel {
  final String bookId;
  /// Title 是小说本身的名称；ShortTitle 是它与短剧共用的名称。
  final String title;
  final String shortTitle;
  final NovelStatus status;
  final List<String> tags;
  final int reads;
  final String coverUrl;
  final String coverExpiresAt;
  /// ShowAtEpisode 是播放器推荐该小说时所在的集数。
  final int showAtEpisode;

  const Novel({
    this.bookId = '',
    this.title = '',
    this.shortTitle = '',
    this.status = const NovelStatus(0),
    this.tags = const [],
    this.reads = 0,
    this.coverUrl = '',
    this.coverExpiresAt = '',
    this.showAtEpisode = 0,
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory Novel.fromJson(Map<String, dynamic> j) => Novel(
        bookId: _asString(j['book_id']),
        title: _asString(j['title']),
        shortTitle: _asString(j['short_title']),
        status: NovelStatus(_asInt(j['status'])),
        tags: _asList(j['tags'], (e) => _asString(e)),
        reads: _asInt(j['reads']),
        coverUrl: _asString(j['cover_url']),
        coverExpiresAt: _asString(j['cover_expires_at']),
        showAtEpisode: _asInt(j['show_at_episode']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'book_id': bookId,
        'title': title,
        'short_title': shortTitle,
        'status': status.value,
        'tags': tags,
        'reads': reads,
        'cover_url': coverUrl,
        'cover_expires_at': coverExpiresAt,
        'show_at_episode': showAtEpisode,
      };
}

/// NovelStatus 是客户端的 NovelCreationStatus；未给出时为 -1。
class NovelStatus {
  final int value;

  const NovelStatus(this.value);

  /// 已完结
  static const finished = NovelStatus(0);
  /// 连载中
  static const ongoing = NovelStatus(1);
  /// 今日更新
  static const updatedToday = NovelStatus(3);

  @override
  bool operator ==(Object other) => other is NovelStatus && other.value == value;

  @override
  int get hashCode => value.hashCode;

  @override
  String toString() => 'NovelStatus($value)';
}

/// PayType 是 pay_info.pay_type，取自 com.dragon.read.rpc.model.VideoPayType。
class PayType {
  final int value;

  const PayType(this.value);

  /// PayFree 表示无需账号即可观看。
  static const free = PayType(0);
  /// PayVip 表示需要订阅。
  static const vip = PayType(1);
  /// PaySingle 表示需要单次购买。
  static const single = PayType(2);

  @override
  bool operator ==(Object other) => other is PayType && other.value == value;

  @override
  int get hashCode => value.hashCode;

  @override
  String toString() => 'PayType($value)';
}

/// Person 是演员或角色的主页。
class Person {
  final String userId;
  final String name;
  final String description;
  final String avatarUrl;
  /// Tags 是主页上的头衔（"女演员"、"虚拟人物"）；
  /// Labels 是名字下方的小徽标（"水瓶座"）。
  final List<String> tags;
  final List<String> labels;
  /// Followers 是粉丝数；Likes 是获赞数。
  final int followers;
  final int likes;

  const Person({
    this.userId = '',
    this.name = '',
    this.description = '',
    this.avatarUrl = '',
    this.tags = const [],
    this.labels = const [],
    this.followers = 0,
    this.likes = 0,
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory Person.fromJson(Map<String, dynamic> j) => Person(
        userId: _asString(j['user_id']),
        name: _asString(j['name']),
        description: _asString(j['description']),
        avatarUrl: _asString(j['avatar_url']),
        tags: _asList(j['tags'], (e) => _asString(e)),
        labels: _asList(j['labels'], (e) => _asString(e)),
        followers: _asInt(j['followers']),
        likes: _asInt(j['likes']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'user_id': userId,
        'name': name,
        'description': description,
        'avatar_url': avatarUrl,
        'tags': tags,
        'labels': labels,
        'followers': followers,
        'likes': likes,
      };
}

/// PlatformType 是 video_platform，
/// 取自 com.dragon.read.rpc.model.VideoPlatformType。
class PlatformType {
  final int value;

  const PlatformType(this.value);

  /// 未知；有声书带此值
  static const unknown = PlatformType(0);
  /// 抖音
  static const douyin = PlatformType(1);
  /// UGC 视频
  static const ugcVideo = PlatformType(2);
  /// 站外与自制短剧
  static const outer = PlatformType(3);
  /// 西瓜视频，电影来自这里
  static const xigua = PlatformType(4);
  /// Chameleon
  static const chameleon = PlatformType(5);
  /// 混合来源
  static const mixed = PlatformType(1024);

  @override
  bool operator ==(Object other) => other is PlatformType && other.value == value;

  @override
  int get hashCode => value.hashCode;

  @override
  String toString() => 'PlatformType($value)';
}

/// Playback 是一个已解析的匿名媒体模型。它描述服务端提供的内容，
/// 从不携带解密密钥或许可证材料。
class Playback {
  final String episodeId;
  /// VideoID 是分集背后的媒体 id。
  final String videoId;
  /// MediaType 是服务端的媒体类型，例如 "video"。
  final String mediaType;
  final int durationSeconds;
  /// ExpiresAt 是流 URL 失效的时间，单位是 unix 秒。
  final int expiresAt;
  /// Protected 标记加密的分集；参见 Stream.Protected。
  final bool protected;
  final List<VideoStream> streams;
  /// AudioURL 是分段（DASH）作品（如长片电影）的独立音轨。
  /// 单文件的表示形式留空。
  final String audioUrl;

  const Playback({
    this.episodeId = '',
    this.videoId = '',
    this.mediaType = '',
    this.durationSeconds = 0,
    this.expiresAt = 0,
    this.protected = false,
    this.streams = const [],
    this.audioUrl = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory Playback.fromJson(Map<String, dynamic> j) => Playback(
        episodeId: _asString(j['episode_id']),
        videoId: _asString(j['video_id']),
        mediaType: _asString(j['media_type']),
        durationSeconds: _asInt(j['duration_seconds']),
        expiresAt: _asInt(j['expires_at']),
        protected: _asBool(j['protected']),
        streams: _asList(j['streams'], (e) => VideoStream.fromJson(_asMap(e))),
        audioUrl: _asString(j['audio_url']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'episode_id': episodeId,
        'video_id': videoId,
        'media_type': mediaType,
        'duration_seconds': durationSeconds,
        'expires_at': expiresAt,
        'protected': protected,
        'streams': streams.map((e) => e.toJson()).toList(),
        'audio_url': audioUrl,
      };
}

/// RankBoard 是某个榜单内容下的一个榜（推荐榜、热播榜……）。
class RankBoard {
  final String id;
  final String name;
  final bool selected;

  const RankBoard({
    this.id = '',
    this.name = '',
    this.selected = false,
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory RankBoard.fromJson(Map<String, dynamic> j) => RankBoard(
        id: _asString(j['id']),
        name: _asString(j['name']),
        selected: _asBool(j['selected']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'selected': selected,
      };
}

/// RankChange 是榜单行的变动（客户端中的 RankChangeType）。
class RankChange {
  final RankChangeKind kind;
  /// Places 是该行变动的位数；新上榜或不变的行为 0。
  final int places;

  const RankChange({
    this.kind = const RankChangeKind(0),
    this.places = 0,
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory RankChange.fromJson(Map<String, dynamic> j) => RankChange(
        kind: RankChangeKind(_asInt(j['kind'])),
        places: _asInt(j['places']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'kind': kind.value,
        'places': places,
      };
}

/// RankChangeKind 的取值即客户端的 RankChangeType。
class RankChangeKind {
  final int value;

  const RankChangeKind(this.value);

  /// 新上榜
  static const newValue = RankChangeKind(0);
  /// 上升 RankChange.Places 位
  static const up = RankChangeKind(1);
  /// 下降 RankChange.Places 位
  static const down = RankChangeKind(2);
  /// 名次不变
  static const unchanged = RankChangeKind(3);

  @override
  bool operator ==(Object other) => other is RankChangeKind && other.value == value;

  @override
  int get hashCode => value.hashCode;

  @override
  String toString() => 'RankChangeKind($value)';
}

/// RankContent 是短剧榜单页的一个内容
/// （全部、真人剧、漫剧、AI剧、系列剧、演员），带有自己的各个榜。
class RankContent {
  final String id;
  final String name;
  final bool selected;
  final List<RankBoard> boards;

  const RankContent({
    this.id = '',
    this.name = '',
    this.selected = false,
    this.boards = const [],
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory RankContent.fromJson(Map<String, dynamic> j) => RankContent(
        id: _asString(j['id']),
        name: _asString(j['name']),
        selected: _asBool(j['selected']),
        boards: _asList(j['boards'], (e) => RankBoard.fromJson(_asMap(e))),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'selected': selected,
        'boards': boards.map((e) => e.toJson()).toList(),
      };
}

/// RankItem 是榜单的一行。
class RankItem {
  /// Rank 是服务端给该行的名次（参见 Position）。
  final int rank;
  /// Votes 和 Kind 用于活动榜单行：分别是票数和服务端的条目类型，原样透传。
  final int votes;
  final int kind;
  /// Actor 在书籍榜单上是作者，在活动榜单或短剧演员榜上是演员名。
  final String actor;
  final String seriesId;
  final String title;
  final String coverUrl;
  /// Score 是显示的热度（"2550万热度"、"998万推荐"、"平台推荐"）。
  final String score;
  /// EpisodeCount 由短剧榜单填充。
  final int episodeCount;
  /// Position 是该行在短剧榜单页上的名次，从 1 开始。Rank 是服务端自己的编号，
  /// 在推广行前后会重复，在推荐榜上是个性化的。
  final int position;
  /// Promoted 标记服务端插入短剧榜单的行：它显示 "平台推荐" 而非热度，
  /// 名次照抄下一行（实测）。
  final bool promoted;
  /// ActorID、AvatarURL、Intro 和 Details 用于短剧榜单的演员行，
  /// 该行的 Actor 是名字，Score 是人气。Intro 列出作品；
  /// Details 存放 "67部作品" 这样的行。
  final String actorId;
  /// UserID 是演员的账号，即 ActorProfile 和 ActorWorks 接受的 id。
  final String userId;
  final String avatarUrl;
  final String intro;
  final List<String> details;
  /// Change 是该行相对上一期榜单的变动（如果有）。
  final RankChange? change;

  const RankItem({
    this.rank = 0,
    this.votes = 0,
    this.kind = 0,
    this.actor = '',
    this.seriesId = '',
    this.title = '',
    this.coverUrl = '',
    this.score = '',
    this.episodeCount = 0,
    this.position = 0,
    this.promoted = false,
    this.actorId = '',
    this.userId = '',
    this.avatarUrl = '',
    this.intro = '',
    this.details = const [],
    this.change,
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory RankItem.fromJson(Map<String, dynamic> j) => RankItem(
        rank: _asInt(j['rank']),
        votes: _asInt(j['votes']),
        kind: _asInt(j['kind']),
        actor: _asString(j['actor']),
        seriesId: _asString(j['series_id']),
        title: _asString(j['title']),
        coverUrl: _asString(j['cover_url']),
        score: _asString(j['score']),
        episodeCount: _asInt(j['episode_count']),
        position: _asInt(j['position']),
        promoted: _asBool(j['promoted']),
        actorId: _asString(j['actor_id']),
        userId: _asString(j['user_id']),
        avatarUrl: _asString(j['avatar_url']),
        intro: _asString(j['intro']),
        details: _asList(j['details'], (e) => _asString(e)),
        change: (j['change'] is Map ? RankChange.fromJson(_asMap(j['change'])) : null),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'rank': rank,
        'votes': votes,
        'kind': kind,
        'actor': actor,
        'series_id': seriesId,
        'title': title,
        'cover_url': coverUrl,
        'score': score,
        'episode_count': episodeCount,
        'position': position,
        'promoted': promoted,
        'actor_id': actorId,
        'user_id': userId,
        'avatar_url': avatarUrl,
        'intro': intro,
        'details': details,
        if (change != null) 'change': change!.toJson(),
      };
}

/// RankList 是活动的一个榜单频道。
class RankList {
  final String id;
  final String name;
  /// Type 和 Size 是服务端的榜单类型和长度，原样透传。
  final int type;
  final int size;

  const RankList({
    this.id = '',
    this.name = '',
    this.type = 0,
    this.size = 0,
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory RankList.fromJson(Map<String, dynamic> j) => RankList(
        id: _asString(j['id']),
        name: _asString(j['name']),
        type: _asInt(j['type']),
        size: _asInt(j['size']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'id': id,
        'name': name,
        'type': type,
        'size': size,
      };
}

/// Ranking 是一个活动榜单页。
class Ranking {
  final String listName;
  /// Lists 是该活动的其他榜单频道。
  final List<RankList> lists;
  final List<RankItem> items;
  final bool hasMore;
  /// NextOffset 用于续取列表（RankRows 把它当作 offset）；
  /// SessionID 是服务端上报的分页会话。
  final int nextOffset;
  /// ActivityStage 和 CanVote 是服务端上报的活动状态。
  final int activityStage;
  final bool canVote;
  final String sessionId;

  const Ranking({
    this.listName = '',
    this.lists = const [],
    this.items = const [],
    this.hasMore = false,
    this.nextOffset = 0,
    this.activityStage = 0,
    this.canVote = false,
    this.sessionId = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory Ranking.fromJson(Map<String, dynamic> j) => Ranking(
        listName: _asString(j['list_name']),
        lists: _asList(j['lists'], (e) => RankList.fromJson(_asMap(e))),
        items: _asList(j['items'], (e) => RankItem.fromJson(_asMap(e))),
        hasMore: _asBool(j['has_more']),
        nextOffset: _asInt(j['next_offset']),
        activityStage: _asInt(j['activity_stage']),
        canVote: _asBool(j['can_vote']),
        sessionId: _asString(j['session_id']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'list_name': listName,
        'lists': lists.map((e) => e.toJson()).toList(),
        'items': items.map((e) => e.toJson()).toList(),
        'has_more': hasMore,
        'next_offset': nextOffset,
        'activity_stage': activityStage,
        'can_vote': canVote,
        'session_id': sessionId,
      };
}

/// Recommend 是一部短剧的相关视频单元：包含各行、单元标题，
/// 以及客户端为后续请求保留的分页会话。
class Recommend {
  final String cellTitle;
  final String sessionId;
  final List<SeriesSummary> items;

  const Recommend({
    this.cellTitle = '',
    this.sessionId = '',
    this.items = const [],
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory Recommend.fromJson(Map<String, dynamic> j) => Recommend(
        cellTitle: _asString(j['cell_title']),
        sessionId: _asString(j['session_id']),
        items: _asList(j['items'], (e) => SeriesSummary.fromJson(_asMap(e))),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'cell_title': cellTitle,
        'session_id': sessionId,
        'items': items.map((e) => e.toJson()).toList(),
      };
}

/// Season 是合集（多季合集）中的一项。
class Season {
  final String seriesId;
  final String title;
  /// Index 是季序号，从 1 开始。
  final int index;
  final String coverUrl;
  final int episodeCount;
  final SeriesStatus status;

  const Season({
    this.seriesId = '',
    this.title = '',
    this.index = 0,
    this.coverUrl = '',
    this.episodeCount = 0,
    this.status = const SeriesStatus(0),
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory Season.fromJson(Map<String, dynamic> j) => Season(
        seriesId: _asString(j['series_id']),
        title: _asString(j['title']),
        index: _asInt(j['index']),
        coverUrl: _asString(j['cover_url']),
        episodeCount: _asInt(j['episode_count']),
        status: SeriesStatus(_asInt(j['status'])),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'series_id': seriesId,
        'title': title,
        'index': index,
        'cover_url': coverUrl,
        'episode_count': episodeCount,
        'status': status.value,
      };
}

/// Series 是一部短剧的详情视图。
class Series {
  final String id;
  final String title;
  final String description;
  final String coverUrl;
  /// CoverExpiresAt 是 CoverURL 的签名过期时间（x-expires）；
  /// 过期后要从 API 重新获取封面。
  final String coverExpiresAt;
  final int episodeCount;
  /// Author、Category 和 Words 由分类页的行填充。
  final String author;
  final String category;
  final int words;
  /// Status 是发布状态；PayType 是付费墙状态。
  final SeriesStatus status;
  final PayType payType;
  /// ContentType 和 Platform 描述作品是什么、来自哪里（短剧、电影、漫剧……）。
  final ContentType contentType;
  final PlatformType platform;
  /// Heat 是原始热度分；App 将其显示为 "红果热度值N万"。
  final int heat;
  /// PlayCount 是短剧的总播放量（series_play_cnt）。
  final int playCount;
  /// FollowCount 和 DiggCount 是标题旁显示的计数。
  final int followCount;
  final int diggCount;
  /// CreatedAt 是发布时间（create_time，unix 秒）。
  final String createdAt;
  /// RightText 是集数旁的角标（"更新至12集"）。
  final String rightText;
  /// Subtitles 是标题下方的简短状态行。
  final List<String> subtitles;
  /// AgeGate 对不受限的作品为 0。
  final int ageGate;
  /// IsMultiSeason 标记属于多季合集的作品。
  final bool isMultiSeason;
  /// SeasonIndex 是该合集内的季序号，从 1 开始。
  final int seasonIndex;
  /// RelatedAlbumID 是各季共用的合集 id。
  final String relatedAlbumId;
  /// RelatedNovelBookID 是短剧改编的小说（如已公开）。
  final String relatedNovelBookId;
  /// Novel 是详情页链接的原著小说（video_relate_book），服务端给出时才有。
  final Novel? novel;
  /// Rows 是详情页的次要信息行（季、分类、演员）。
  final List<InfoRow> rows;

  const Series({
    this.id = '',
    this.title = '',
    this.description = '',
    this.coverUrl = '',
    this.coverExpiresAt = '',
    this.episodeCount = 0,
    this.author = '',
    this.category = '',
    this.words = 0,
    this.status = const SeriesStatus(0),
    this.payType = const PayType(0),
    this.contentType = const ContentType(0),
    this.platform = const PlatformType(0),
    this.heat = 0,
    this.playCount = 0,
    this.followCount = 0,
    this.diggCount = 0,
    this.createdAt = '',
    this.rightText = '',
    this.subtitles = const [],
    this.ageGate = 0,
    this.isMultiSeason = false,
    this.seasonIndex = 0,
    this.relatedAlbumId = '',
    this.relatedNovelBookId = '',
    this.novel,
    this.rows = const [],
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory Series.fromJson(Map<String, dynamic> j) => Series(
        id: _asString(j['id']),
        title: _asString(j['title']),
        description: _asString(j['description']),
        coverUrl: _asString(j['cover_url']),
        coverExpiresAt: _asString(j['cover_expires_at']),
        episodeCount: _asInt(j['episode_count']),
        author: _asString(j['author']),
        category: _asString(j['category']),
        words: _asInt(j['words']),
        status: SeriesStatus(_asInt(j['status'])),
        payType: PayType(_asInt(j['pay_type'])),
        contentType: ContentType(_asInt(j['content_type'])),
        platform: PlatformType(_asInt(j['platform'])),
        heat: _asInt(j['heat']),
        playCount: _asInt(j['play_count']),
        followCount: _asInt(j['follow_count']),
        diggCount: _asInt(j['digg_count']),
        createdAt: _asString(j['created_at']),
        rightText: _asString(j['right_text']),
        subtitles: _asList(j['subtitles'], (e) => _asString(e)),
        ageGate: _asInt(j['age_gate']),
        isMultiSeason: _asBool(j['is_multi_season']),
        seasonIndex: _asInt(j['season_index']),
        relatedAlbumId: _asString(j['related_album_id']),
        relatedNovelBookId: _asString(j['related_novel_book_id']),
        novel: (j['novel'] is Map ? Novel.fromJson(_asMap(j['novel'])) : null),
        rows: _asList(j['rows'], (e) => InfoRow.fromJson(_asMap(e))),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'id': id,
        'title': title,
        'description': description,
        'cover_url': coverUrl,
        'cover_expires_at': coverExpiresAt,
        'episode_count': episodeCount,
        'author': author,
        'category': category,
        'words': words,
        'status': status.value,
        'pay_type': payType.value,
        'content_type': contentType.value,
        'platform': platform.value,
        'heat': heat,
        'play_count': playCount,
        'follow_count': followCount,
        'digg_count': diggCount,
        'created_at': createdAt,
        'right_text': rightText,
        'subtitles': subtitles,
        'age_gate': ageGate,
        'is_multi_season': isMultiSeason,
        'season_index': seasonIndex,
        'related_album_id': relatedAlbumId,
        'related_novel_book_id': relatedNovelBookId,
        if (novel != null) 'novel': novel!.toJson(),
        'rows': rows.map((e) => e.toJson()).toList(),
      };
}

/// SeriesDetail 是一部短剧及其分集，由批量读取返回。
class SeriesDetail {
  final Series? series;
  final List<Episode> episodes;

  const SeriesDetail({
    this.series,
    this.episodes = const [],
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory SeriesDetail.fromJson(Map<String, dynamic> j) => SeriesDetail(
        series: (j['series'] is Map ? Series.fromJson(_asMap(j['series'])) : null),
        episodes: _asList(j['episodes'], (e) => Episode.fromJson(_asMap(e))),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        if (series != null) 'series': series!.toJson(),
        'episodes': episodes.map((e) => e.toJson()).toList(),
      };
}

/// SeriesStatus 是短剧的发布状态，由 data.video_data.series_status 携带。
///
/// 名称取自官方 App 的枚举 com.dragon.read.rpc.model.SeriesStatus（
/// SeriesUpdating、SeriesEnd、SeriesUpdateTaday、SeriesUpdateStop）；
/// 值 0 和 1 也已在线上实测。
class SeriesStatus {
  final int value;

  const SeriesStatus(this.value);

  /// SeriesUpdating 是未完结的短剧（"更新至 N 集"）。
  static const updating = SeriesStatus(0);
  /// SeriesEnd 是已完结的短剧（"全 N 集"）。
  static const end = SeriesStatus(1);
  /// SeriesUpdatedToday 是枚举中的“今日更新”状态（推断）。
  static const updatedToday = SeriesStatus(3);
  /// SeriesUpdateStopped 是枚举中的“停止更新”状态（推断）。
  static const updateStopped = SeriesStatus(4);

  @override
  bool operator ==(Object other) => other is SeriesStatus && other.value == value;

  @override
  int get hashCode => value.hashCode;

  @override
  String toString() => 'SeriesStatus($value)';
}

/// SeriesSummary 是公开目录中的一行。ID 和标题是稳定字段。
class SeriesSummary {
  final String id;
  final String title;
  final String description;
  final String coverUrl;
  /// CoverExpiresAt 是 CoverURL 的签名过期时间（x-expires）；
  /// 过期后要从 API 重新获取封面。
  final String coverExpiresAt;
  final int episodeCount;
  /// Author、Category 和 Words 由分类页的行填充。
  final String author;
  final String category;
  final int words;

  const SeriesSummary({
    this.id = '',
    this.title = '',
    this.description = '',
    this.coverUrl = '',
    this.coverExpiresAt = '',
    this.episodeCount = 0,
    this.author = '',
    this.category = '',
    this.words = 0,
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory SeriesSummary.fromJson(Map<String, dynamic> j) => SeriesSummary(
        id: _asString(j['id']),
        title: _asString(j['title']),
        description: _asString(j['description']),
        coverUrl: _asString(j['cover_url']),
        coverExpiresAt: _asString(j['cover_expires_at']),
        episodeCount: _asInt(j['episode_count']),
        author: _asString(j['author']),
        category: _asString(j['category']),
        words: _asInt(j['words']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'id': id,
        'title': title,
        'description': description,
        'cover_url': coverUrl,
        'cover_expires_at': coverExpiresAt,
        'episode_count': episodeCount,
        'author': author,
        'category': category,
        'words': words,
      };
}

/// ShortFilterPage 是符合筛选条件的短剧的一页。
class ShortFilterPage {
  final List<SeriesSummary> items;
  final bool hasMore;
  /// Cursor 用于续取列表；它携带已展示的短剧，服务端会跳过它们。
  final String cursor;

  const ShortFilterPage({
    this.items = const [],
    this.hasMore = false,
    this.cursor = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory ShortFilterPage.fromJson(Map<String, dynamic> j) => ShortFilterPage(
        items: _asList(j['items'], (e) => SeriesSummary.fromJson(_asMap(e))),
        hasMore: _asBool(j['has_more']),
        cursor: _asString(j['cursor']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'items': items.map((e) => e.toJson()).toList(),
        'has_more': hasMore,
        'cursor': cursor,
      };
}

/// ShortRanking 是短剧榜单的一页。
class ShortRanking {
  /// Contents 是该页的目录：每个内容及其各个榜。
  final List<RankContent> contents;
  /// Content 和 Board 是本页对应的 ID。
  final String content;
  final String board;
  /// Version 是服务端上报的榜单版本。
  final String version;
  final List<RankItem> items;
  final bool hasMore;
  final String cursor;

  const ShortRanking({
    this.contents = const [],
    this.content = '',
    this.board = '',
    this.version = '',
    this.items = const [],
    this.hasMore = false,
    this.cursor = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory ShortRanking.fromJson(Map<String, dynamic> j) => ShortRanking(
        contents: _asList(j['contents'], (e) => RankContent.fromJson(_asMap(e))),
        content: _asString(j['content']),
        board: _asString(j['board']),
        version: _asString(j['version']),
        items: _asList(j['items'], (e) => RankItem.fromJson(_asMap(e))),
        hasMore: _asBool(j['has_more']),
        cursor: _asString(j['cursor']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'contents': contents.map((e) => e.toJson()).toList(),
        'content': content,
        'board': board,
        'version': version,
        'items': items.map((e) => e.toJson()).toList(),
        'has_more': hasMore,
        'cursor': cursor,
      };
}

/// Stream 是媒体模型中的一个表示形式。Protected
/// 标记官方客户端在内部解密的表示形式；sdk.ContentKey 从模型的 KeyMaterial
/// 还原它的密钥。
class VideoStream {
  /// Definition 是分辨率标签（"720p"）；Quality 是服务端的画质标签
  /// （"normal"）。
  final String definition;
  final String quality;
  final String codec;
  final int width;
  final int height;
  /// Bitrate 的单位是 bit/s；Size 是文件大小，单位是字节。
  final int bitrate;
  final int size;
  /// BackupURL 是同一文件的第二个 CDN 地址。
  final String url;
  final String backupUrl;
  /// HLS 标记 m3u8 播放列表；Dash 标记分段（DASH）的表示形式。
  final bool hls;
  final bool dash;
  final bool protected;
  /// Encryption 是受保护流的加密方案，例如 "cenc-aes-ctr"。
  final String encryption;

  const VideoStream({
    this.definition = '',
    this.quality = '',
    this.codec = '',
    this.width = 0,
    this.height = 0,
    this.bitrate = 0,
    this.size = 0,
    this.url = '',
    this.backupUrl = '',
    this.hls = false,
    this.dash = false,
    this.protected = false,
    this.encryption = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory VideoStream.fromJson(Map<String, dynamic> j) => VideoStream(
        definition: _asString(j['definition']),
        quality: _asString(j['quality']),
        codec: _asString(j['codec']),
        width: _asInt(j['width']),
        height: _asInt(j['height']),
        bitrate: _asInt(j['bitrate']),
        size: _asInt(j['size']),
        url: _asString(j['url']),
        backupUrl: _asString(j['backup_url']),
        hls: _asBool(j['hls']),
        dash: _asBool(j['dash']),
        protected: _asBool(j['protected']),
        encryption: _asString(j['encryption']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'definition': definition,
        'quality': quality,
        'codec': codec,
        'width': width,
        'height': height,
        'bitrate': bitrate,
        'size': size,
        'url': url,
        'backup_url': backupUrl,
        'hls': hls,
        'dash': dash,
        'protected': protected,
        'encryption': encryption,
      };
}

/// Suggestion 是一条搜索建议。Text 是搜索框显示的文本；
/// Keyword 是它提交的内容（直达行为内容 ID）。
class Suggestion {
  final String text;
  final String keyword;
  /// Kind 是服务端的行类型，原样透传。
  final int kind;

  const Suggestion({
    this.text = '',
    this.keyword = '',
    this.kind = 0,
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory Suggestion.fromJson(Map<String, dynamic> j) => Suggestion(
        text: _asString(j['text']),
        keyword: _asString(j['keyword']),
        kind: _asInt(j['kind']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'text': text,
        'keyword': keyword,
        'kind': kind,
      };
}

/// Work 是某人出演的一部短剧。
class Work {
  final String seriesId;
  final String title;
  final String description;
  final String coverUrl;
  final String coverExpiresAt;
  final int episodeCount;
  /// Tags 是卡片上的各行，例如 "爱情" 和 "70集"。
  final List<String> tags;
  /// Heat 是短剧的原始热度分；Plays 是其播放量。
  final int heat;
  final int plays;

  const Work({
    this.seriesId = '',
    this.title = '',
    this.description = '',
    this.coverUrl = '',
    this.coverExpiresAt = '',
    this.episodeCount = 0,
    this.tags = const [],
    this.heat = 0,
    this.plays = 0,
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory Work.fromJson(Map<String, dynamic> j) => Work(
        seriesId: _asString(j['series_id']),
        title: _asString(j['title']),
        description: _asString(j['description']),
        coverUrl: _asString(j['cover_url']),
        coverExpiresAt: _asString(j['cover_expires_at']),
        episodeCount: _asInt(j['episode_count']),
        tags: _asList(j['tags'], (e) => _asString(e)),
        heat: _asInt(j['heat']),
        plays: _asInt(j['plays']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'series_id': seriesId,
        'title': title,
        'description': description,
        'cover_url': coverUrl,
        'cover_expires_at': coverExpiresAt,
        'episode_count': episodeCount,
        'tags': tags,
        'heat': heat,
        'plays': plays,
      };
}

/// WorkPage 是某人作品的一页（作品 › 剧作）。
class WorkPage {
  final List<Work> items;
  final bool hasMore;
  final String cursor;

  const WorkPage({
    this.items = const [],
    this.hasMore = false,
    this.cursor = '',
  });

  /// 从 Go SDK 输出的 JSON 解码；缺省的字段取零值。
  factory WorkPage.fromJson(Map<String, dynamic> j) => WorkPage(
        items: _asList(j['items'], (e) => Work.fromJson(_asMap(e))),
        hasMore: _asBool(j['has_more']),
        cursor: _asString(j['cursor']),
      );

  /// 编码成与 Go SDK 相同的 JSON。
  Map<String, dynamic> toJson() => {
        'items': items.map((e) => e.toJson()).toList(),
        'has_more': hasMore,
        'cursor': cursor,
      };
}

/// DiscoverOptions 选择一个发现页。Channel 是书城频道名，空频道即推荐。
/// Cursor 是上一页给出的令牌。PageSize 是 cell/change 的 limit（0 表示 20，上限 50）。
class DiscoverOptions {
  final String? cursor;
  final String? channel;
  final int? pageSize;

  const DiscoverOptions({this.cursor, this.channel, this.pageSize});

  DiscoverOptions copyWith({String? cursor, String? channel, int? pageSize}) =>
      DiscoverOptions(cursor: cursor ?? this.cursor, channel: channel ?? this.channel, pageSize: pageSize ?? this.pageSize);

  Map<String, dynamic> toJson() => {
        if (cursor != null) 'cursor': cursor!,
        if (channel != null) 'channel': channel!,
        if (pageSize != null) 'pageSize': pageSize!,
      };
}

/// SearchOptions 选择一个搜索页。Tab 取值为综合、剧集、视频、讨论、小说、
/// 用户或听书，空 tab 即综合。Cursor 是下一个偏移量。
class SearchOptions {
  final String? cursor;
  final int? pageSize;
  final String? tab;
  /// Filters 把综合 tab 收窄到 SearchFilter* 值；选多个时互相放宽。
  /// 游标不携带这些筛选，所以要连同游标传入相同的筛选。
  final List<String>? filters;

  const SearchOptions({this.cursor, this.pageSize, this.tab, this.filters});

  SearchOptions copyWith({String? cursor, int? pageSize, String? tab, List<String>? filters}) =>
      SearchOptions(cursor: cursor ?? this.cursor, pageSize: pageSize ?? this.pageSize, tab: tab ?? this.tab, filters: filters ?? this.filters);

  Map<String, dynamic> toJson() => {
        if (cursor != null) 'cursor': cursor!,
        if (pageSize != null) 'pageSize': pageSize!,
        if (tab != null) 'tab': tab!,
        if (filters != null) 'filters': filters!,
      };
}

/// EpisodeOptions 选择一个分集页。游标不在第 1 阶段的范围内。
class EpisodeOptions {
  final String? cursor;
  final int? pageSize;

  const EpisodeOptions({this.cursor, this.pageSize});

  EpisodeOptions copyWith({String? cursor, int? pageSize}) =>
      EpisodeOptions(cursor: cursor ?? this.cursor, pageSize: pageSize ?? this.pageSize);

  Map<String, dynamic> toJson() => {
        if (cursor != null) 'cursor': cursor!,
        if (pageSize != null) 'pageSize': pageSize!,
      };
}

/// LandingOptions 选择一个分类落地页。
///
/// Tab 是 Categories 给出的 tab 值：8（全部，默认）、1（男生）或 0（女生）。
/// Filters 把 CategoryPage.Filters 中的筛选组键（book_status、sort_by、
/// sub_category、word_type）映射到该组的某个选项值。
/// Cursor 是上一页的 NextOffset。
class LandingOptions {
  final String? tab;
  final Map<String, String>? filters;
  final String? cursor;

  const LandingOptions({this.tab, this.filters, this.cursor});

  LandingOptions copyWith({String? tab, Map<String, String>? filters, String? cursor}) =>
      LandingOptions(tab: tab ?? this.tab, filters: filters ?? this.filters, cursor: cursor ?? this.cursor);

  Map<String, dynamic> toJson() => {
        if (tab != null) 'tab': tab!,
        if (filters != null) 'filters': filters!,
        if (cursor != null) 'cursor': cursor!,
      };
}

/// NewReleaseQuery 选择上新日历中的一天：Date 格式为 YYYYMMDD（空即今天），
/// Cursor 是上一页的 Cursor，它带有所在的日期。
class NewReleaseQuery {
  final String? date;
  final String? cursor;

  const NewReleaseQuery({this.date, this.cursor});

  NewReleaseQuery copyWith({String? date, String? cursor}) =>
      NewReleaseQuery(date: date ?? this.date, cursor: cursor ?? this.cursor);

  Map<String, dynamic> toJson() => {
        if (date != null) 'date': date!,
        if (cursor != null) 'cursor': cursor!,
      };
}

/// ShortFilterQuery 在筛选页（筛选）上挑选短剧。
///
/// Selections 把 FilterRow.Key 映射到选中的选项 Values；缺省的行即全部。
/// Cursor 是上一页的 Cursor。
class ShortFilterQuery {
  final Map<String, List<String>>? selections;
  final String? cursor;

  const ShortFilterQuery({this.selections, this.cursor});

  ShortFilterQuery copyWith({Map<String, List<String>>? selections, String? cursor}) =>
      ShortFilterQuery(selections: selections ?? this.selections, cursor: cursor ?? this.cursor);

  Map<String, dynamic> toJson() => {
        if (selections != null) 'selections': selections!,
        if (cursor != null) 'cursor': cursor!,
      };
}

/// ShortRankQuery 选择一个短剧榜单。Content 是 RankContent ID（all、human、
/// comic_series_rank、ai_playlet、series_album；空即 all），
/// Board 是它的某个 RankBoard ID（空即服务端对该 content 的默认值）。
/// Cursor 是上一页的 Cursor。
class ShortRankQuery {
  final String? content;
  final String? board;
  final String? cursor;

  const ShortRankQuery({this.content, this.board, this.cursor});

  ShortRankQuery copyWith({String? content, String? board, String? cursor}) =>
      ShortRankQuery(content: content ?? this.content, board: board ?? this.board, cursor: cursor ?? this.cursor);

  Map<String, dynamic> toJson() => {
        if (content != null) 'content': content!,
        if (board != null) 'board': board!,
        if (cursor != null) 'cursor': cursor!,
      };
}
