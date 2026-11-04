import type {
  Album,
  CategoryPage,
  ContentKey,
  Device,
  DiscoverOptions,
  Episode,
  EpisodeOptions,
  FilterRow,
  HotSearch,
  LandingOptions,
  NewReleasePage,
  NewReleaseQuery,
  Page,
  Person,
  Playback,
  RankList,
  Ranking,
  Recommend,
  SearchOptions,
  Series,
  SeriesDetail,
  SeriesSummary,
  ShortFilterPage,
  ShortFilterQuery,
  ShortRanking,
  ShortRankQuery,
  Suggestion,
  WorkPage,
} from './types.js';

export * from './types.js';

/** SearchFilter 是综合 tab 的搜索筛选项（SearchOptions.filters 的取值）。 */
export declare const SearchFilter: {
  /** 真人剧 */
  readonly ShortPlay: 'content_type_short_play';
  /** 漫剧 */
  readonly Comic: 'content_type_dynamic_comic';
  /** AI剧 */
  readonly AIComic: 'content_type_ai_dynamic_comic';
  /** 影视剧 */
  readonly Film: 'content_type_long_video';
  /** 未看过 */
  readonly Unwatched: 'search_range_unread';
};

/** 错误码，对应 Go SDK 的哨兵错误。 */
export type ErrorCode =
  | 'INVALID_OPTIONS'
  | 'PROFILE_NOT_FOUND'
  | 'SIGNER_UNAVAILABLE'
  | 'AUTHENTICATION_REQUIRED'
  | 'NOT_PROTECTED'
  | 'DRM_PROTECTED'
  | 'ANONYMOUS_LIMITED'
  | 'RATE_LIMITED'
  | 'TRANSPORT'
  | 'SIGNATURE_REJECTED'
  | 'CONTENT_UNAVAILABLE'
  | 'PARAM_INVALID'
  | 'CHALLENGE_REQUIRED'
  | 'DEVICE_REJECTED'
  | 'END_OF_LIST'
  | 'ALBUM_UNAVAILABLE'
  | 'UNKNOWN';

/** OpenhongguoError 是 SDK 返回的错误。 */
export declare class OpenhongguoError extends Error {
  readonly name: 'OpenhongguoError';
  /** 错误码，可以据此判断原因，而不必解析 message。 */
  readonly code: ErrorCode;
  /** 原样重试是否可能成功（限流、网络错误、服务端 5xx）。 */
  readonly retryable: boolean;
  /** 服务端建议的等待时间，单位毫秒；0 表示没有建议。 */
  readonly retryAfterMs: number;
}

/** TransportRequest 是 SDK 要发送的一个请求。 */
export interface TransportRequest {
  method: string;
  url: string;
  headers: Record<string, string>;
  body: Uint8Array;
}

/** TransportResponse 是收到的响应。body 是原始字节，不要自动解压。 */
export interface TransportResponse {
  status: number;
  headers: Record<string, string>;
  /** Set-Cookie 解析出的名字和值。 */
  cookies?: Record<string, string>;
  body: Uint8Array;
}

/** Transport 发送一个请求；只有没收到响应时才 reject。 */
export type Transport = (request: TransportRequest) => Promise<TransportResponse>;

/** ProfileStore 读写匿名设备档案（JSON 字符串）。load 返回 null 表示还没有档案。 */
export interface ProfileStore {
  load(): Promise<string | null>;
  save(json: string): Promise<void>;
  reset(): Promise<void>;
}

/** ClientOptions 是 createClient 的设置，全部可选。 */
export interface ClientOptions {
  /**
   * 设备档案的存放位置：文件路径，或自定义的 ProfileStore。
   * 默认与 CLI 共用 <用户配置目录>/openhongguo/anonymous.json。
   */
  profile?: string | ProfileStore;
  /** 自定义请求发送方式；默认用 node:https。 */
  transport?: Transport;
  /** 默认 transport 的超时，单位毫秒，默认 30000。 */
  timeoutMs?: number;
  /** 接收诊断日志；日志里的 URL 已经脱敏。 */
  log?: (line: string) => void;
  /** 读取类接口的地址，如 "https://host"；默认与 App 一致。 */
  apiOrigin?: string;
  /** 设备注册接口的地址；默认与 App 一致。 */
  registerOrigin?: string;
  /** 短剧筛选和活动榜接口的地址；默认与 App 一致。 */
  distributionOrigin?: string;
  /** 可重试的失败最多尝试几次，默认 3；0 或 1 表示不重试。 */
  retryAttempts?: number;
  /** 设备被拒或播放返回空结果时，一次调用最多换几台新的匿名设备，默认 2；0 表示关闭。 */
  deviceResets?: number;
  /** 新注册设备自称的机型；未填的字段取默认值（小米 14）。 */
  device?: Partial<Device>;
}

/** PlaybackResult 是 resolvePlayback 的结果。 */
export interface PlaybackResult {
  playback: Playback;
  /** 加密剧集的内容密钥；未加密时没有这个字段。 */
  contentKey?: ContentKey;
  /** 加密剧集的密钥推导失败时，说明原因；此时没有 contentKey。 */
  contentKeyError?: { code: ErrorCode; message: string };
}

/** Client 包装 Go SDK 的方法；每个方法返回 Promise，失败时抛出 OpenhongguoError。 */
export declare class Client {
  private constructor();
  /** 加载一个书城频道（默认“推荐”）。 */
  discover(options?: DiscoverOptions): Promise<Page<SeriesSummary>>;
  /** 搜索短剧。 */
  search(query: string, options?: SearchOptions): Promise<Page<SeriesSummary>>;
  /** 搜索联想词。 */
  suggest(prefix: string): Promise<Suggestion[]>;
  /** 搜索页的“猜你想搜”和热搜榜。 */
  hotSearch(): Promise<HotSearch>;
  /** 搜索框里滚动的提示词。 */
  searchCues(): Promise<string[]>;
  /** 上新日历的一天。 */
  newReleases(query?: NewReleaseQuery): Promise<NewReleasePage>;
  /** 一部剧的详情。 */
  getSeries(seriesId: string): Promise<Series>;
  /** 一次读取多部剧及其各集，按请求顺序返回。 */
  getSeriesBatch(seriesIds: string[]): Promise<SeriesDetail[]>;
  /** 一部剧的分集列表。 */
  listEpisodes(seriesId: string, options?: EpisodeOptions): Promise<Page<Episode>>;
  /** 多季剧的全部季。 */
  seasons(seriesId: string): Promise<Album>;
  /** 一部剧的相关推荐。 */
  seriesRecommend(seriesId: string): Promise<Recommend>;
  /** 一个合集及其各季。 */
  getAlbum(albumId: string): Promise<Album>;
  /** 短剧排行榜的一页。 */
  shortRank(query?: ShortRankQuery): Promise<ShortRanking>;
  /** 短剧筛选面板；genre 如 "comic_series"。 */
  shortFilterPanel(genre?: string): Promise<FilterRow[]>;
  /** 按筛选条件列出短剧。 */
  shortFilter(query?: ShortFilterQuery): Promise<ShortFilterPage>;
  /** 活动榜单。 */
  ranking(activityId: string, rankListId: string): Promise<Ranking>;
  /** 书城榜单的一页。 */
  rankRows(cellId: string, offset?: number): Promise<Ranking>;
  /** 书城榜单的类型。 */
  rankGroups(): Promise<RankList[]>;
  /** 演员或角色主页。 */
  person(userId: string): Promise<Person>;
  /** 演员的作品，一页。 */
  personWorks(userId: string, cursor?: string): Promise<WorkPage>;
  /** 书籍分类的标签页。 */
  categories(categoryId: string): Promise<CategoryPage>;
  /** 书籍分类下的列表。 */
  categoryLanding(categoryId: string, options?: LandingOptions): Promise<CategoryPage>;
  /** 一集的播放模型；加密剧集同时给出内容密钥。 */
  resolvePlayback(episodeId: string): Promise<PlaybackResult>;
  /** 丢弃当前匿名设备，下次请求时重新注册。 */
  resetDevice(): Promise<void>;
  /** 沿着游标逐部返回书城频道的剧，去掉跨页重复。 */
  discoverAll(options?: DiscoverOptions, walk?: { maxPages?: number }): AsyncGenerator<SeriesSummary>;
  /** 沿着游标逐部返回搜索结果，去掉跨页重复。 */
  searchAll(query: string, options?: SearchOptions, walk?: { maxPages?: number }): AsyncGenerator<SeriesSummary>;
}

/** createClient 创建客户端。首次请求时注册一台匿名设备。 */
export declare function createClient(options?: ClientOptions): Promise<Client>;

/** defaultProfilePath 是默认的设备档案路径，与 CLI 共用。 */
export declare function defaultProfilePath(): string;

/** fileProfileStore 把设备档案存成一个 JSON 文件（权限 0600）。 */
export declare function fileProfileStore(path?: string): ProfileStore;

/** memoryProfileStore 只在内存里保存设备档案。 */
export declare function memoryProfileStore(): ProfileStore;

/** nodeTransport 是默认的请求发送方式（node:http/https）。 */
export declare function nodeTransport(options?: { timeoutMs?: number }): Transport;
