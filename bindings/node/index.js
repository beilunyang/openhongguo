// openhongguo：红果短剧（Hongguo）的非官方 Node.js SDK。
//
// 底层是 Go SDK 编译成的 WebAssembly（dist/openhongguo.wasm）。本文件负责：
// 加载 wasm、用 node:https 发请求、读写匿名设备档案，并把 SDK 的方法
// 包装成 Promise 风格的 API。

import { mkdir, readFile, rename, rm, writeFile } from 'node:fs/promises';
import http from 'node:http';
import https from 'node:https';
import { createRequire } from 'node:module';
import { homedir } from 'node:os';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';
import { webcrypto } from 'node:crypto';

const here = dirname(fileURLToPath(import.meta.url));

// 响应体的上限，与 Go 的 HTTP 适配器一致。
const MAX_BODY = 8 << 20;

/** SearchFilter 是综合 tab 的搜索筛选项（SearchOptions.filters 的取值）。 */
export const SearchFilter = Object.freeze({
  ShortPlay: 'content_type_short_play', // 真人剧
  Comic: 'content_type_dynamic_comic', // 漫剧
  AIComic: 'content_type_ai_dynamic_comic', // AI剧
  Film: 'content_type_long_video', // 影视剧
  Unwatched: 'search_range_unread', // 未看过
});

/** OpenhongguoError 是 SDK 返回的错误，code 对应 Go SDK 的哨兵错误。 */
export class OpenhongguoError extends Error {
  constructor({ code = 'UNKNOWN', message = '', retryable = false, retryAfterMs = 0 } = {}) {
    super(message);
    this.name = 'OpenhongguoError';
    this.code = code;
    this.retryable = retryable;
    this.retryAfterMs = retryAfterMs;
  }
}

let runtime;

// loadRuntime 在进程内只加载一次 wasm。
function loadRuntime() {
  runtime ??= (async () => {
    globalThis.crypto ??= webcrypto;
    const require = createRequire(import.meta.url);
    require(join(here, 'dist', 'wasm_exec.js')); // 定义 globalThis.Go
    const go = new globalThis.Go();
    const bytes = await readFile(join(here, 'dist', 'openhongguo.wasm'));
    const { instance } = await WebAssembly.instantiate(bytes, go.importObject);
    go.run(instance); // Go 的 main 一直挂起，不会返回
    if (!globalThis.__openhongguo) {
      throw new OpenhongguoError({ code: 'UNKNOWN', message: 'openhongguo: wasm did not start' });
    }
    return globalThis.__openhongguo;
  })();
  return runtime;
}

/** defaultProfilePath 与 CLI 共用设备档案：<用户配置目录>/openhongguo/anonymous.json。 */
export function defaultProfilePath() {
  if (process.env.OHG_CONFIG_DIR) return join(process.env.OHG_CONFIG_DIR, 'anonymous.json');
  let base;
  if (process.platform === 'darwin') base = join(homedir(), 'Library', 'Application Support');
  else if (process.platform === 'win32') base = process.env.AppData ?? join(homedir(), 'AppData', 'Roaming');
  else base = process.env.XDG_CONFIG_HOME ?? join(homedir(), '.config');
  return join(base, 'openhongguo', 'anonymous.json');
}

/** fileProfileStore 把设备档案存成一个 JSON 文件（权限 0600）。 */
export function fileProfileStore(path = defaultProfilePath()) {
  return {
    async load() {
      try {
        return await readFile(path, 'utf8');
      } catch (e) {
        if (e.code === 'ENOENT') return null;
        throw e;
      }
    },
    async save(json) {
      await mkdir(dirname(path), { recursive: true, mode: 0o700 });
      const tmp = `${path}.${process.pid}.tmp`;
      await writeFile(tmp, json, { mode: 0o600 });
      await rename(tmp, path);
    },
    async reset() {
      await rm(path, { force: true });
    },
  };
}

/** memoryProfileStore 只在内存里保存设备档案，进程退出后丢失。 */
export function memoryProfileStore() {
  let saved = null;
  return {
    async load() {
      return saved;
    },
    async save(json) {
      saved = json;
    },
    async reset() {
      saved = null;
    },
  };
}

/**
 * nodeTransport 用 node:http/https 发送请求。
 * 响应体保持原样，不自动解压：SDK 会按 Content-Encoding 自己处理。
 */
export function nodeTransport({ timeoutMs = 30_000 } = {}) {
  return (req) =>
    new Promise((resolve, reject) => {
      const url = new URL(req.url);
      const mod = url.protocol === 'http:' ? http : https;
      const r = mod.request(url, { method: req.method || 'GET', headers: req.headers, timeout: timeoutMs }, (res) => {
        const chunks = [];
        let size = 0;
        res.on('data', (c) => {
          size += c.length;
          if (size > MAX_BODY) r.destroy(new Error('response body too large'));
          else chunks.push(c);
        });
        res.on('error', reject);
        res.on('end', () => {
          const headers = {};
          for (const [k, v] of Object.entries(res.headers)) {
            if (k === 'set-cookie' || k === 'cookie' || v === undefined) continue;
            headers[k] = Array.isArray(v) ? v[0] : String(v);
          }
          const cookies = {};
          for (const line of res.headers['set-cookie'] ?? []) {
            const pair = line.split(';', 1)[0];
            const i = pair.indexOf('=');
            if (i > 0) cookies[pair.slice(0, i).trim()] = pair.slice(i + 1).trim();
          }
          resolve({ status: res.statusCode, headers, cookies, body: new Uint8Array(Buffer.concat(chunks)) });
        });
      });
      r.on('timeout', () => r.destroy(new Error('request timed out')));
      r.on('error', reject);
      if (req.body && req.body.length > 0) r.write(Buffer.from(req.body));
      r.end();
    });
}

function toError(e) {
  if (e instanceof Error) return e;
  return new OpenhongguoError(e);
}

/**
 * createClient 创建一个客户端。首次请求时，SDK 注册一台匿名设备，
 * 并通过 profile 保存下来。
 */
export async function createClient(options = {}) {
  const rt = await loadRuntime();
  const {
    profile = fileProfileStore(),
    transport = nodeTransport({ timeoutMs: options.timeoutMs }),
    log,
    apiOrigin,
    registerOrigin,
    distributionOrigin,
    retryAttempts = 3,
    deviceResets = 2,
    device,
  } = options;
  const store = typeof profile === 'string' ? fileProfileStore(profile) : profile;
  const res = rt.newClient({
    transport,
    profile: store,
    log,
    options: JSON.stringify({ apiOrigin, registerOrigin, distributionOrigin, retryAttempts, deviceResets, device }),
  });
  if (res.error) throw new OpenhongguoError(res.error);
  return new Client(res.call);
}

/** Client 包装 Go SDK 的方法；每个方法返回 Promise。 */
export class Client {
  #call;

  constructor(call) {
    this.#call = call;
  }

  async #invoke(name, ...args) {
    let out;
    try {
      out = await this.#call(name, JSON.stringify(args));
    } catch (e) {
      throw toError(e);
    }
    return out === undefined || out === 'null' ? undefined : JSON.parse(out);
  }

  discover(options = {}) {
    return this.#invoke('discover', options);
  }
  search(query, options = {}) {
    return this.#invoke('search', query, options);
  }
  suggest(prefix) {
    return this.#invoke('suggest', prefix);
  }
  hotSearch() {
    return this.#invoke('hotSearch');
  }
  searchCues() {
    return this.#invoke('searchCues');
  }
  newReleases(query = {}) {
    return this.#invoke('newReleases', query);
  }
  getSeries(seriesId) {
    return this.#invoke('getSeries', seriesId);
  }
  getSeriesBatch(seriesIds) {
    return this.#invoke('getSeriesBatch', seriesIds);
  }
  listEpisodes(seriesId, options = {}) {
    return this.#invoke('listEpisodes', seriesId, options);
  }
  seasons(seriesId) {
    return this.#invoke('seasons', seriesId);
  }
  seriesRecommend(seriesId) {
    return this.#invoke('seriesRecommend', seriesId);
  }
  getAlbum(albumId) {
    return this.#invoke('getAlbum', albumId);
  }
  shortRank(query = {}) {
    return this.#invoke('shortRank', query);
  }
  shortFilterPanel(genre = '') {
    return this.#invoke('shortFilterPanel', genre);
  }
  shortFilter(query = {}) {
    return this.#invoke('shortFilter', query);
  }
  ranking(activityId, rankListId) {
    return this.#invoke('ranking', activityId, rankListId);
  }
  rankRows(cellId, offset = 0) {
    return this.#invoke('rankRows', cellId, offset);
  }
  rankGroups() {
    return this.#invoke('rankGroups');
  }
  person(userId) {
    return this.#invoke('person', userId);
  }
  personWorks(userId, cursor = '') {
    return this.#invoke('personWorks', userId, cursor);
  }
  categories(categoryId) {
    return this.#invoke('categories', categoryId);
  }
  categoryLanding(categoryId, options = {}) {
    return this.#invoke('categoryLanding', categoryId, options);
  }
  resolvePlayback(episodeId) {
    return this.#invoke('resolvePlayback', episodeId);
  }
  resetDevice() {
    return this.#invoke('resetDevice');
  }

  /** discoverAll 沿着游标逐部返回一个书城频道的剧，并去掉跨页重复的剧。 */
  discoverAll(options = {}, { maxPages = Infinity } = {}) {
    return this.#walk((cursor) => this.discover({ ...options, cursor }), options.cursor, maxPages);
  }

  /** searchAll 沿着游标逐部返回搜索结果，并去掉跨页重复的剧。 */
  searchAll(query, options = {}, { maxPages = Infinity } = {}) {
    return this.#walk((cursor) => this.search(query, { ...options, cursor }), options.cursor, maxPages);
  }

  async *#walk(fetch, cursor, maxPages) {
    const seen = new Set();
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
}
