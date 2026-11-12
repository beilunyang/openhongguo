import Foundation
import XCTest

@testable import Openhongguo

/// 用 bindings/mobile/fixtureserver 回放 testdata/protocol 的样本，
/// 走通 Go 的 core 库、设备注册、档案保存和错误映射，不访问网络。
final class FixtureTests: XCTestCase {
    private static var server: Process?
    private static var origin = ""

    override class func setUp() {
        let root = URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
            .deletingLastPathComponent().deletingLastPathComponent().deletingLastPathComponent()
        // 先编译再直接运行：go run 被终止时不会结束它启动的子进程。
        let bin = FileManager.default.temporaryDirectory.appendingPathComponent("ohg-fixtureserver-\(UUID().uuidString)")
        let build = Process()
        build.executableURL = URL(fileURLWithPath: "/usr/bin/env")
        build.arguments = ["go", "build", "-o", bin.path, "./bindings/mobile/fixtureserver"]
        build.currentDirectoryURL = root
        try! build.run()
        build.waitUntilExit()

        let p = Process()
        p.executableURL = bin
        p.arguments = ["-dir", root.appendingPathComponent("testdata/protocol").path]
        let out = Pipe()
        p.standardOutput = out
        p.standardError = FileHandle.nullDevice
        try! p.run()
        server = p
        // 第一行是监听地址。
        var line = Data()
        while let b = try? out.fileHandleForReading.read(upToCount: 1), !b.isEmpty, b != Data("\n".utf8) {
            line.append(b)
        }
        origin = String(decoding: line, as: UTF8.self)
        try? FileManager.default.removeItem(at: bin)
    }

    override class func tearDown() {
        server?.terminate()
    }

    private func client(api: String? = nil, device: Device? = nil) throws -> (OpenhongguoClient, URL) {
        let profile = FileManager.default.temporaryDirectory
            .appendingPathComponent(UUID().uuidString).appendingPathComponent("anonymous.json")
        let o = Self.origin
        let c = try OpenhongguoClient.create(
            profile: profile,
            options: ClientOptions(apiOrigin: api ?? o, registerOrigin: api ?? o, distributionOrigin: o, retryAttempts: 1, device: device))
        return (c, profile)
    }

    func testSearchSeriesAndEpisodes() async throws {
        XCTAssertTrue(Self.origin.hasPrefix("http://"), "fixture server: \(Self.origin)")
        let (c, profile) = try client()

        let page = try await c.search("短剧", options: SearchOptions(pageSize: 20))
        XCTAssertEqual(page.items.first?.id, "7100000000000101")
        XCTAssertEqual(page.items.first?.title, "Fixture Lantern Court")
        XCTAssertTrue(FileManager.default.fileExists(atPath: profile.path), "注册得到的设备存进了档案")

        let episodes = try await c.listEpisodes("7100000000000101")
        XCTAssertEqual(episodes.items.first?.number, 1)
        XCTAssertEqual(episodes.items.first?.id, "7100000000000201")
    }

    func testDanmaku() async throws {
        let (c, _) = try client()
        let page = try await c.listDanmaku("7100000000000101", episodeId: "7100000000000201")
        XCTAssertEqual(page.items.map(\.offsetMs), [329, 12500])
        XCTAssertEqual(page.items.first?.text, "Fixture danmaku one")
        XCTAssertTrue(page.hasMore)
        XCTAssertFalse(page.cursor.isEmpty)
    }

    func testProtectedPlaybackWithoutKey() async throws {
        let (c, _) = try client()
        let r = try await c.resolvePlayback("7100000000000201")
        XCTAssertTrue(r.playback.protected)
        XCTAssertEqual(r.playback.streams.count, 1)
        XCTAssertNil(r.contentKey) // 样本里的密钥材料是假的
        XCTAssertEqual(r.contentKeyError?.code, .drmProtected)
    }

    func testRateLimited() async throws {
        let (c, _) = try client(api: Self.origin + "/limited")
        do {
            _ = try await c.search("短剧")
            XCTFail("no error")
        } catch let e as OpenhongguoError {
            XCTAssertEqual(e.code, .rateLimited)
            XCTAssertTrue(e.retryable)
        }
    }

    func testInvalidDevice() throws {
        XCTAssertThrowsError(try client(device: Device(model: "x; evil"))) { e in
            XCTAssertEqual((e as? OpenhongguoError)?.code, .invalidOptions)
        }
    }
}

/// 用假的 Engine 检查翻页、去重和取消。
final class EngineTests: XCTestCase {
    private struct FakeEngine: Engine {
        let reply: @Sendable (String, String) -> String?
        let onCancel: @Sendable () -> Void

        func start(_ method: String, _ args: String, _ done: @escaping @Sendable (String) -> Void) -> @Sendable () -> Void {
            if let out = reply(method, args) { done(out) }
            return { onCancel(); done(#"{"error":{"code":"CANCELLED","message":"cancelled"}}"#) }
        }
    }

    func testSearchAllFollowsCursorAndDropsDuplicates() async throws {
        let engine = FakeEngine(reply: { _, args in
            if args.contains(#""cursor":"p2""#) {
                return #"{"result":{"items":[{"id":"2"},{"id":"3"}],"has_more":false}}"#
            }
            return #"{"result":{"items":[{"id":"1"},{"id":"2"}],"cursor":"p2","has_more":true}}"#
        }, onCancel: {})
        var ids: [String] = []
        for try await s in OpenhongguoClient(engine: engine).searchAll("短剧") {
            ids.append(s.id)
        }
        XCTAssertEqual(ids, ["1", "2", "3"])
    }

    func testNullListBecomesEmpty() async throws {
        let engine = FakeEngine(reply: { _, _ in #"{"result":null}"# }, onCancel: {})
        let cues = try await OpenhongguoClient(engine: engine).searchCues()
        XCTAssertEqual(cues, [])
    }

    func testCancellingTheTaskCancelsTheCall() async throws {
        let cancelled = expectation(description: "cancelled")
        let engine = FakeEngine(reply: { _, _ in nil }, onCancel: { cancelled.fulfill() })
        let task = Task { try await OpenhongguoClient(engine: engine).rankGroups() }
        try await Task.sleep(nanoseconds: 50_000_000)
        task.cancel()
        await fulfillment(of: [cancelled], timeout: 5)
        do {
            _ = try await task.value
            XCTFail("no error")
        } catch is CancellationError {}
    }

    func testEnumsDecode() throws {
        let ep = try JSONDecoder().decode(Episode.self, from: Data(#"{"id":"1","content_type":7}"#.utf8))
        XCTAssertEqual(ep.contentType, .shortPlay)
        XCTAssertEqual(ep.title, "")
    }
}
