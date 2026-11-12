// 用本地 HTTP 服务回放 testdata/protocol 的样本，走通 wasm、请求转发、
// 设备注册、档案保存和错误映射，不访问网络。先运行 npm run build。
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createServer } from 'node:http';
import { dirname, join } from 'node:path';
import { after, before, test } from 'node:test';
import { fileURLToPath } from 'node:url';

import { createClient, memoryProfileStore, OpenhongguoError } from '../index.js';

const fixtures = join(dirname(fileURLToPath(import.meta.url)), '..', '..', '..', 'testdata', 'protocol');
const routes = {
  '/service/2/device_register/': 'register-response.json',
  '/reading/bookapi/bookmall/tab/v': 'discover-response.json',
  '/reading/bookapi/search/tab/v': 'search-response.json',
  '/novel/player/video_detail/v1/': 'series-response.json',
  '/novel/player/video_model/v1/': 'drm-playback.json',
  '/novel/commentapi/comment/list/7100000000000201/v1/': 'danmaku-response.json',
};

let server;
let origin;
const hits = [];

before(async () => {
  server = createServer((req, res) => {
    const path = new URL(req.url, 'http://x').pathname;
    hits.push(path);
    if (path.startsWith('/limited/')) {
      res.writeHead(429).end();
      return;
    }
    const file = routes[path];
    if (!file) {
      res.writeHead(404).end();
      return;
    }
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(readFileSync(join(fixtures, file)));
  });
  await new Promise((resolve) => server.listen(0, '127.0.0.1', resolve));
  origin = `http://127.0.0.1:${server.address().port}`;
});

after(() => server.close());

const client = (profile, extra = {}) =>
  createClient({ profile, apiOrigin: origin, registerOrigin: origin, distributionOrigin: origin, retryAttempts: 1, ...extra });

test('搜索、剧集详情和分集', async () => {
  const profile = memoryProfileStore();
  const c = await client(profile);

  const page = await c.search('短剧', { pageSize: 20 });
  assert.equal(page.items[0].id, '7100000000000101');
  assert.equal(page.items[0].title, 'Fixture Lantern Court');
  assert.ok(hits.includes('/service/2/device_register/'), '首次请求前注册了设备');

  const saved = JSON.parse(await profile.load());
  assert.ok(saved.device_id_str, '注册得到的设备 ID 存进了档案');

  const episodes = await c.listEpisodes('7100000000000101');
  assert.equal(episodes.items[0].number, 1);
  assert.equal(episodes.items[0].id, '7100000000000201');
});

test('一集的弹幕按出现时间排好，带着下一页的游标', async () => {
  const c = await client(memoryProfileStore());
  const page = await c.listDanmaku('7100000000000101', '7100000000000201');
  assert.deepEqual(page.items.map((d) => [d.offset_ms, d.text]), [
    [329, 'Fixture danmaku one'],
    [12500, 'Fixture danmaku two'],
  ]);
  assert.equal(page.has_more, true);
  assert.ok(page.cursor);
});

test('加密剧集返回播放模型，密钥推导失败时说明原因', async () => {
  const c = await client(memoryProfileStore());
  const { playback, contentKey, contentKeyError } = await c.resolvePlayback('7100000000000201');
  assert.equal(playback.protected, true);
  assert.equal(playback.streams.length, 1);
  assert.equal(contentKey, undefined); // 样本里的密钥材料是假的
  assert.equal(contentKeyError.code, 'DRM_PROTECTED');
});

test('限流映射成可重试的 RATE_LIMITED', async () => {
  const c = await client(memoryProfileStore(), { apiOrigin: `${origin}/limited`, registerOrigin: `${origin}/limited` });
  await assert.rejects(c.search('短剧'), (e) => {
    assert.ok(e instanceof OpenhongguoError);
    assert.equal(e.code, 'RATE_LIMITED');
    assert.equal(e.retryable, true);
    return true;
  });
});

test('请求发送失败映射成 TRANSPORT', async () => {
  const c = await client(memoryProfileStore(), {
    transport: async () => {
      throw new Error('offline');
    },
  });
  await assert.rejects(c.search('短剧'), (e) => e instanceof OpenhongguoError && e.code === 'TRANSPORT');
});

test('无效的机型设置在创建时报 INVALID_OPTIONS', async () => {
  await assert.rejects(client(memoryProfileStore(), { device: { model: 'x; evil' } }), (e) => e.code === 'INVALID_OPTIONS');
});
