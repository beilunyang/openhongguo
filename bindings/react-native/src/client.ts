// 客户端的实现，和原生模块之间只通过 Native 接口交互，测试里可以换成假的。

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
} from './types';

/** Native 是原生模块的接口。 */
export interface Native {
  defaultProfilePath(): string;
  /** 返回 {"id": n} 或 {"error": {...}} 形式的 JSON。 */
  createClient(profilePath: string, configJson: string, log: boolean): string;
  releaseClient(id: number): void;
  /** 结果是 {"result": …} 或 {"error": {...}} 形式的 JSON，不会 reject。 */
  call(id: number, method: string, argsJson: string): Promise<string>;
  addListener(event: 'log', listener: (e: { client: number; line: string }) => void): { remove(): void };
}

/** SearchFilter 是综合 tab 的搜索筛选项（SearchOptions.filters 的取值）。 */
export const SearchFilter = Object.freeze({
  /** 真人剧 */
  ShortPlay: 'content_type_short_play',
  /** 漫剧 */
  Comic: 'content_type_dynamic_comic',
  /** AI剧 */
  AIComic: 'content_type_ai_dynamic_comic',
  /** 影视剧 */
  Film: 'content_type_long_video',
  /** 未看过 */
  Unwatched: 'search_range_unread',
} as const);

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
  | 'CANCELLED'
  | 'UNKNOWN';

interface ErrorObject {
  code?: ErrorCode;
  message?: string;
  retryable?: boolean;
  retryAfterMs?: number;
}

/** OpenhongguoError 是 SDK 返回的错误。 */
export class OpenhongguoError extends Error {
  /** 错误码，可以据此判断原因，而不必解析 message。 */
  readonly code: ErrorCode;
  /** 原样重试是否可能成功（限流、网络错误、服务端 5xx）。 */
  readonly retryable: boolean;
  /** 服务端建议的等待时间，单位毫秒；0 表示没有建议。 */
  readonly retryAfterMs: number;

  constructor({ code = 'UNKNOWN', message = '', retryable = false, retryAfterMs = 0 }: ErrorObject = {}) {
    super(message);
    this.name = 'OpenhongguoError';
    this.code = code;
    this.retryable = retryable;
    this.retryAfterMs = retryAfterMs;
  }
}

/** ClientOptions 是 createClient 的设置，全部可选。 */
export interface ClientOptions {
  /** 设备档案的路径；默认在 App 不参与备份的目录里。 */
  profilePath?: string;
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

/** createClientWith 用给定的原生模块创建客户端。 */
export async function createClientWith(native: Native, options: ClientOptions = {}): Promise<Client> {
  const { profilePath, log, apiOrigin, registerOrigin, distributionOrigin, retryAttempts = 3, deviceResets = 2, device } = options;
  const config = JSON.stringify({ apiOrigin, registerOrigin, distributionOrigin, retryAttempts, deviceResets, device });
  const res = JSON.parse(native.createClient(profilePath ?? native.defaultProfilePath(), config, !!log));
  if (res.error) throw new OpenhongguoError(res.error);
  const id: number = res.id;
  const sub = log ? native.addListener('log', (e) => e.client === id && log(e.line)) : undefined;
  return new Client(native, id, sub);
}

/**
 * Client 包装 Go SDK 的方法；每个方法返回 Promise，失败时抛出
 * OpenhongguoError。同一个客户端可以并发使用。
 */
export class Client {
  #native: Native;
  #id: number;
  #sub?: { remove(): void };

  /** @internal 用 createClient 创建。 */
  constructor(native: Native, id: number, sub?: { remove(): void }) {
    this.#native = native;
    this.#id = id;
    this.#sub = sub;
  }

  async #invoke<T>(name: string, ...args: unknown[]): Promise<T> {
    const out = await this.#native.call(this.#id, name, JSON.stringify(args));
    const env = JSON.parse(out);
    if (env.error) throw new OpenhongguoError(env.error);
    return env.result as T;
  }

  /** 加载一个书城频道（默认“推荐”）。 */
  discover(options: DiscoverOptions = {}): Promise<Page<SeriesSummary>> {
    return this.#invoke('discover', options);
  }
  /** 搜索短剧。 */
  search(query: string, options: SearchOptions = {}): Promise<Page<SeriesSummary>> {
    return this.#invoke('search', query, options);
  }
  /** 搜索联想词。 */
  async suggest(prefix: string): Promise<Suggestion[]> {
    return (await this.#invoke<Suggestion[] | null>('suggest', prefix)) ?? [];
  }
  /** 搜索页的“猜你想搜”和热搜榜。 */
  hotSearch(): Promise<HotSearch> {
    return this.#invoke('hotSearch');
  }
  /** 搜索框里滚动的提示词。 */
  async searchCues(): Promise<string[]> {
    return (await this.#invoke<string[] | null>('searchCues')) ?? [];
  }
  /** 上新日历的一天。 */
  newReleases(query: NewReleaseQuery = {}): Promise<NewReleasePage> {
    return this.#invoke('newReleases', query);
  }
  /** 一部剧的详情。 */
  getSeries(seriesId: string): Promise<Series> {
    return this.#invoke('getSeries', seriesId);
  }
  /** 一次读取多部剧及其各集，按请求顺序返回。 */
  async getSeriesBatch(seriesIds: string[]): Promise<SeriesDetail[]> {
    return (await this.#invoke<SeriesDetail[] | null>('getSeriesBatch', seriesIds)) ?? [];
  }
  /** 一部剧的分集列表。 */
  listEpisodes(seriesId: string, options: EpisodeOptions = {}): Promise<Page<Episode>> {
    return this.#invoke('listEpisodes', seriesId, options);
  }
  /** 多季剧的全部季。 */
  seasons(seriesId: string): Promise<Album> {
    return this.#invoke('seasons', seriesId);
  }
  /** 一部剧的相关推荐。 */
  seriesRecommend(seriesId: string): Promise<Recommend> {
    return this.#invoke('seriesRecommend', seriesId);
  }
  /** 一个合集及其各季。 */
  getAlbum(albumId: string): Promise<Album> {
    return this.#invoke('getAlbum', albumId);
  }
  /** 短剧排行榜的一页。 */
  shortRank(query: ShortRankQuery = {}): Promise<ShortRanking> {
    return this.#invoke('shortRank', query);
  }
  /** 短剧筛选面板；genre 如 "comic_series"。 */
  async shortFilterPanel(genre = ''): Promise<FilterRow[]> {
    return (await this.#invoke<FilterRow[] | null>('shortFilterPanel', genre)) ?? [];
  }
  /** 按筛选条件列出短剧。 */
  shortFilter(query: ShortFilterQuery = {}): Promise<ShortFilterPage> {
    return this.#invoke('shortFilter', query);
  }
  /** 活动榜单。 */
  ranking(activityId: string, rankListId: string): Promise<Ranking> {
    return this.#invoke('ranking', activityId, rankListId);
  }
  /** 书城榜单的一页。 */
  rankRows(cellId: string, offset = 0): Promise<Ranking> {
    return this.#invoke('rankRows', cellId, offset);
  }
  /** 书城榜单的类型。 */
  async rankGroups(): Promise<RankList[]> {
    return (await this.#invoke<RankList[] | null>('rankGroups')) ?? [];
  }
  /** 演员或角色主页。 */
  person(userId: string): Promise<Person> {
    return this.#invoke('person', userId);
  }
  /** 演员的作品，一页。 */
  personWorks(userId: string, cursor = ''): Promise<WorkPage> {
    return this.#invoke('personWorks', userId, cursor);
  }
  /** 书籍分类的标签页。 */
  categories(categoryId: string): Promise<CategoryPage> {
    return this.#invoke('categories', categoryId);
  }
  /** 书籍分类下的列表。 */
  categoryLanding(categoryId: string, options: LandingOptions = {}): Promise<CategoryPage> {
    return this.#invoke('categoryLanding', categoryId, options);
  }
  /** 一集的播放模型；加密剧集同时给出内容密钥。 */
  resolvePlayback(episodeId: string): Promise<PlaybackResult> {
    return this.#invoke('resolvePlayback', episodeId);
  }
  /** 丢弃当前匿名设备，下次请求时重新注册。 */
  async resetDevice(): Promise<void> {
    await this.#invoke('resetDevice');
  }

  /** 沿着游标逐部返回书城频道的剧，去掉跨页重复。 */
  discoverAll(options: DiscoverOptions = {}, { maxPages = Infinity } = {}): AsyncGenerator<SeriesSummary> {
    return this.#walk((cursor) => this.discover({ ...options, cursor }), options.cursor, maxPages);
  }

  /** 沿着游标逐部返回搜索结果，去掉跨页重复。 */
  searchAll(query: string, options: SearchOptions = {}, { maxPages = Infinity } = {}): AsyncGenerator<SeriesSummary> {
    return this.#walk((cursor) => this.search(query, { ...options, cursor }), options.cursor, maxPages);
  }

  async *#walk(
    fetch: (cursor: string) => Promise<Page<SeriesSummary>>,
    cursor: string | undefined,
    maxPages: number,
  ): AsyncGenerator<SeriesSummary> {
    const seen = new Set<string>();
    for (let pages = 0; pages < maxPages; pages++) {
      const page = await fetch(cursor ?? '');
      for (const item of page.items ?? []) {
        if (seen.has(item.id)) continue;
        seen.add(item.id);
        yield item;
      }
      if (!page.has_more || !page.cursor) return;
      cursor = page.cursor;
    }
  }

  /** 释放原生端的客户端；之后的调用都会失败。 */
  release(): void {
    this.#sub?.remove();
    this.#native.releaseClient(this.#id);
  }
}
