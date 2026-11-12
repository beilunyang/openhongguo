// 用假的原生模块检查客户端：参数编码、结果解码、错误映射、翻页和日志。
import assert from 'node:assert/strict';
import { test } from 'node:test';

import { createClientWith, OpenhongguoError, type Native } from '../build/client.js';

type Reply = (method: string, args: unknown[]) => unknown;

function fakeNative(reply: Reply, createResult = '{"id":7}') {
  const calls: { id: number; method: string; args: unknown[] }[] = [];
  const created: { profilePath: string; config: Record<string, unknown>; log: boolean }[] = [];
  const listeners: ((e: { client: number; line: string }) => void)[] = [];
  let released = 0;
  const native: Native = {
    defaultProfilePath: () => '/data/openhongguo/anonymous.json',
    createClient(profilePath, configJson, log) {
      created.push({ profilePath, config: JSON.parse(configJson), log });
      return createResult;
    },
    releaseClient: () => void released++,
    async call(id, method, argsJson) {
      const args = JSON.parse(argsJson);
      calls.push({ id, method, args });
      const out = reply(method, args);
      return JSON.stringify(out instanceof OpenhongguoError ? { error: { ...out, message: out.message } } : { result: out });
    },
    addListener(_event, listener) {
      listeners.push(listener);
      return { remove: () => listeners.splice(listeners.indexOf(listener), 1) };
    },
  };
  return { native, calls, created, listeners, released: () => released };
}

test('参数和设置原样传给原生模块，结果按 JSON 解码', async () => {
  const f = fakeNative(() => ({ items: [{ id: '1', title: '剧' }], cursor: 'c2', has_more: true }));
  const c = await createClientWith(f.native, { apiOrigin: 'http://x', device: { model: 'Pixel' } });
  assert.deepEqual(f.created[0], {
    profilePath: '/data/openhongguo/anonymous.json',
    config: { apiOrigin: 'http://x', retryAttempts: 3, deviceResets: 2, device: { model: 'Pixel' } },
    log: false,
  });

  const page = await c.search('短剧', { pageSize: 20 });
  assert.deepEqual(f.calls[0], { id: 7, method: 'search', args: ['短剧', { pageSize: 20 }] });
  assert.equal(page.items[0].title, '剧');
  assert.equal(page.has_more, true);
});

test('listDanmaku 传剧、集和选项', async () => {
  const f = fakeNative(() => ({ items: [{ id: '1', text: '好看', offset_ms: 1200 }], start_ms: 0, has_more: false }));
  const c = await createClientWith(f.native);
  const page = await c.listDanmaku('1001', '2001', { cursor: 'm1.x' });
  assert.deepEqual(f.calls[0].args, ['1001', '2001', { cursor: 'm1.x' }]);
  assert.equal(f.calls[0].method, 'listDanmaku');
  assert.equal(page.items[0].offset_ms, 1200);
});

test('错误带着错误码抛出', async () => {
  const f = fakeNative(() => new OpenhongguoError({ code: 'RATE_LIMITED', message: 'slow', retryable: true, retryAfterMs: 1500 }));
  const c = await createClientWith(f.native);
  await assert.rejects(c.hotSearch(), (e) => {
    assert.ok(e instanceof OpenhongguoError);
    assert.equal(e.code, 'RATE_LIMITED');
    assert.equal(e.retryable, true);
    assert.equal(e.retryAfterMs, 1500);
    return true;
  });
});

test('创建失败时抛出原生端给出的错误', async () => {
  const f = fakeNative(() => null, '{"error":{"code":"INVALID_OPTIONS","message":"bad device"}}');
  await assert.rejects(createClientWith(f.native, { device: { model: 'x; evil' } }), (e) => {
    return e instanceof OpenhongguoError && e.code === 'INVALID_OPTIONS' && e.message === 'bad device';
  });
});

test('列表方法把 null 换成空数组', async () => {
  const f = fakeNative(() => null);
  const c = await createClientWith(f.native);
  assert.deepEqual(await c.searchCues(), []);
  assert.deepEqual(await c.suggest('全'), []);
});

test('searchAll 沿游标翻页并去重', async () => {
  const f = fakeNative((_m, args) =>
    (args[1] as { cursor: string }).cursor === 'p2'
      ? { items: [{ id: '2' }, { id: '3' }], has_more: false }
      : { items: [{ id: '1' }, { id: '2' }], cursor: 'p2', has_more: true },
  );
  const c = await createClientWith(f.native);
  const ids: string[] = [];
  for await (const s of c.searchAll('短剧')) ids.push(s.id);
  assert.deepEqual(ids, ['1', '2', '3']);
  assert.equal(f.calls.length, 2);
});

test('日志只交给对应的客户端，release 后取消订阅', async () => {
  const f = fakeNative(() => null);
  const lines: string[] = [];
  const c = await createClientWith(f.native, { log: (l) => lines.push(l) });
  assert.equal(f.created[0].log, true);
  f.listeners.forEach((l) => l({ client: 7, line: 'mine' }));
  f.listeners.forEach((l) => l({ client: 8, line: 'other' }));
  assert.deepEqual(lines, ['mine']);
  c.release();
  assert.equal(f.listeners.length, 0);
  assert.equal(f.released(), 1);
});
