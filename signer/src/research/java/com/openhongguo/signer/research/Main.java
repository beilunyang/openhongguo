package com.openhongguo.signer.research;

import com.github.unidbg.Emulator;
import com.github.unidbg.arm.backend.Backend;
import com.openhongguo.signer.CustomEmulator;
import com.openhongguo.signer.CustomSyscallHandler;
import com.openhongguo.signer.MetasecEmulator;
import unicorn.Arm64Const;

import java.io.File;
import java.nio.charset.StandardCharsets;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.Map;
import java.util.TreeMap;

/** Entry point for the research tools. */
public class Main {
    static final String SO = "src/main/resources/lib/arm64-v8a/libmetasec_ml.so";
    static final String CONFIG = "src/main/resources/native-config.txt";
    static final String URL = System.getProperty("url") != null ? System.getProperty("url") : "https://api5-normal-sinfonlinea.fqnovel.com/reading/bookapi/search/tab/v?query=abc&aid=8662&device_id=1234567890123456&iid=1234567890123457&version_code=74332&ts=1763460000&_rticket=1763460000123";

    public static void main(String[] args) throws Exception {
        System.setIn(new java.io.ByteArrayInputStream(new byte[0]));
        String tool = args.length > 0 ? args[0] : "syscalls";
        if (tool.equals("determinism")) { determinism(); System.exit(0); }
        if (tool.equals("trace")) { trace(bootPinned(), args.length > 1 ? Long.decode(args[1]) : 0, args.length > 2 ? Integer.decode(args[2]) : 0); System.exit(0); }
        if (tool.equals("readmap")) { readMap(); System.exit(0); }
        if (tool.equals("reads")) { reads(Long.decode(args[1]), Integer.decode(args[2])); System.exit(0); }
        if (tool.equals("watch")) { watch(Long.decode(args[1])); System.exit(0); }
        if (tool.equals("allwriters")) { allWriters(Long.decode(args[1]), Integer.decode(args[2])); System.exit(0); }
        // Every tool runs with the known entropy sources pinned, so heap
        // addresses and values match across runs.
        MetasecEmulator emu = bootPinned();
        switch (tool) {
            case "syscalls": syscalls(emu); break;
            case "b64": b64(emu); break;
            case "natives": natives(emu); break;
            case "sboxes": sboxes(emu); break;
            case "f13": f13(emu, args); break;
            case "vectors": vectors(emu, args); break;
            case "f13prof": f13prof(emu, args[1]); break;
            case "f13vm": f13vm(emu, args[1]); break;
            case "f13handlers": f13handlers(emu, args[1]); break;
            case "probe": probe(emu, args[1], args[2]); break;
            case "mem": {
                emu.getFeatureHash(URL, null);
                long a = emu.moduleBase() + Long.decode(args[1]);
                int l = Integer.decode(args[2]);
                System.out.println("[mem] " + hex(emu.emulator().getBackend().mem_read(a, l), l));
                break;
            }
            case "at": at(emu, Long.decode(args[1]), args.length > 2 ? Integer.parseInt(args[2]) : 20); break;
            case "writers": writers(emu, Long.decode(args[1]), Integer.decode(args[2])); break;
            default: throw new IllegalArgumentException("unknown tool " + tool);
        }
        System.exit(0);
    }

    static MetasecEmulator boot() throws Exception {
        MetasecEmulator emu = new MetasecEmulator(new File(SO));
        emu.initSdk(new String(Files.readAllBytes(Paths.get(CONFIG)), StandardCharsets.UTF_8).trim());
        emu.waitForInit();
        return emu;
    }

    static CustomSyscallHandler handler(MetasecEmulator emu) {
        return ((CustomEmulator) emu.emulator()).getCustomSyscallHandler();
    }

    /** Logs the syscalls one signing makes, with the calling site. */
    static void syscalls(MetasecEmulator emu) {
        emu.getFeatureHash(URL, null); // warm up
        Map<String, Integer> counts = new TreeMap<>();
        long base = emu.moduleBase();
        handler(emu).setSyscallObserver((Backend b, Emulator<?> e, int nr) -> {
            long lr = b.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
            long x0 = b.reg_read(Arm64Const.UC_ARM64_REG_X0).longValue();
            String extra = "";
            if (nr == 56) { // openat
                long p = b.reg_read(Arm64Const.UC_ARM64_REG_X1).longValue();
                extra = " " + cstr(b, p);
            }
            String site = (lr >= base && lr < base + 0x500000) ? "so+0x" + Long.toHexString(lr - base) : e.getMemory().findModuleByAddress(lr) != null ? e.getMemory().findModuleByAddress(lr).name : "?";
            counts.merge("nr=" + nr + " " + site + extra, 1, Integer::sum);
            return false;
        });
        Map<String, String> h1 = emu.getFeatureHash(URL, null);
        handler(emu).setSyscallObserver(null);
        counts.forEach((k, v) -> System.out.println("[sys] " + v + "x " + k));
        Map<String, String> h2 = emu.getFeatureHash(URL, null);
        System.out.println("medusa1=" + h1.get("X-Medusa"));
        System.out.println("medusa2=" + h2.get("X-Medusa"));
    }

    /** A clock that starts at a fixed instant and ticks 1 µs per read. */
    static final class FixedClock implements CustomSyscallHandler.SyscallObserver {
        final long startNanos;
        long reads;
        FixedClock(long startNanos) { this.startNanos = startNanos; }
        void reset() { reads = 0; }
        @Override public boolean before(Backend b, Emulator<?> e, int nr) {
            if (nr == 172 || nr == 173) { // getpid, getppid
                b.reg_write(Arm64Const.UC_ARM64_REG_X0, nr == 173 ? 1234L : (long) PID);
                return true;
            }
            if (nr == 178) { // gettid: unidbg answers the host pid (+ task id); shift it onto PID
                com.github.unidbg.thread.Task task = e.get(com.github.unidbg.thread.Task.TASK_KEY);
                long tid = task == null ? e.getPid() : task.getId();
                b.reg_write(Arm64Const.UC_ARM64_REG_X0, tid - e.getPid() + PID);
                return true;
            }
            if (nr != 113 && nr != 169) return false;
            long t = startNanos + 1000L * reads++;
            if (nr == 113) {
                long ts = b.reg_read(Arm64Const.UC_ARM64_REG_X1).longValue();
                b.mem_write(ts, java.nio.ByteBuffer.allocate(16).order(java.nio.ByteOrder.LITTLE_ENDIAN)
                        .putLong(t / 1_000_000_000L).putLong(t % 1_000_000_000L).array());
            } else {
                long tv = b.reg_read(Arm64Const.UC_ARM64_REG_X0).longValue();
                if (tv != 0) b.mem_write(tv, java.nio.ByteBuffer.allocate(16).order(java.nio.ByteOrder.LITTLE_ENDIAN)
                        .putLong(t / 1_000_000_000L).putLong((t % 1_000_000_000L) / 1000).array());
            }
            b.reg_write(Arm64Const.UC_ARM64_REG_X0, 0L);
            return true;
        }
    }

    static final long FIXED_SECS = Long.getLong("secs", 1763460000L);
    static final int PID = 4321;

    /** Replaces the mrs CNTVCT_EL0 reads with a counter that restarts on reset(). */
    static final class FixedCounter {
        static final long[] SITES = {0x20e4b0, 0x20e4c8, 0x20e528, 0x20e540, 0x20e5a0, 0x20e5b8, 0x20e608, 0x20e620};
        long value;
        int reads;
        void reset() { value = 0x1000000L; reads = 0; }
        void install(MetasecEmulator emu) {
            reset();
            long base = emu.moduleBase();
            for (long off : SITES) {
                long addr = base + off;
                emu.emulator().getBackend().hook_add_new(new com.github.unidbg.arm.backend.CodeHook() {
                    @Override public void hook(Backend b, long address, int size, Object user) {
                        int insn = java.nio.ByteBuffer.wrap(b.mem_read(address, 4)).order(java.nio.ByteOrder.LITTLE_ENDIAN).getInt();
                        int rt = insn & 0x1f;
                        value += 1920; // 100 µs at 19.2 MHz between reads
                        reads++;
                        b.reg_write(Arm64Const.UC_ARM64_REG_X0 + rt, value);
                        b.reg_write(Arm64Const.UC_ARM64_REG_PC, address + 4);
                    }
                    @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
                    @Override public void detach() {}
                }, addr, addr, null);
            }
        }
    }

    /** Replaces libc rand() with a sequence that restarts on reset(); also pins time(). */
    static final class FixedRand {
        int calls;
        final Map<String, Integer> sites = new TreeMap<>();
        boolean log;
        void reset() { calls = 0; sites.clear(); }
        int next() {
            // A plain LCG, so values look random but repeat across runs.
            int x = 0x12345678 + calls * 0x9E3779B9;
            x ^= x >>> 15; x *= 0x2c1b3c6d; x ^= x >>> 12;
            calls++;
            return x & 0x7fffffff;
        }
        void install(MetasecEmulator emu) {
            com.github.unidbg.Module libc = emu.emulator().getMemory().findModule("libc.so");
            for (String name : new String[]{"time", "gettimeofday"}) {
                com.github.unidbg.Symbol sym = libc.findSymbolByName(name, false);
                if (sym == null) continue;
                long addr = sym.getAddress();
                boolean isTime = name.equals("time");
                emu.emulator().getBackend().hook_add_new(new com.github.unidbg.arm.backend.CodeHook() {
                    @Override public void hook(Backend b, long address, int size, Object user) {
                        long lr = b.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                        long base = emu.moduleBase();
                        String site = (lr >= base && lr < base + 0x500000) ? "so+0x" + Long.toHexString(lr - base) : "0x" + Long.toHexString(lr);
                        sites.merge(name + "@" + site, 1, Integer::sum);
                        long x0 = b.reg_read(Arm64Const.UC_ARM64_REG_X0).longValue();
                        long secs = FIXED_SECS;
                        if (isTime) {
                            if (x0 != 0) b.mem_write(x0, java.nio.ByteBuffer.allocate(8).order(java.nio.ByteOrder.LITTLE_ENDIAN).putLong(secs).array());
                            b.reg_write(Arm64Const.UC_ARM64_REG_X0, secs);
                        } else {
                            if (x0 != 0) b.mem_write(x0, java.nio.ByteBuffer.allocate(16).order(java.nio.ByteOrder.LITTLE_ENDIAN).putLong(secs).putLong(0).array());
                            b.reg_write(Arm64Const.UC_ARM64_REG_X0, 0L);
                        }
                        b.reg_write(Arm64Const.UC_ARM64_REG_PC, lr);
                    }
                    @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
                    @Override public void detach() {}
                }, addr, addr, null);
            }
            for (String name : new String[]{"rand", "srand"}) {
                long addr = libc.findSymbolByName(name, false).getAddress();
                boolean isRand = name.equals("rand");
                emu.emulator().getBackend().hook_add_new(new com.github.unidbg.arm.backend.CodeHook() {
                    @Override public void hook(Backend b, long address, int size, Object user) {
                        long lr = b.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                        long base = emu.moduleBase();
                        String site = (lr >= base && lr < base + 0x500000) ? "so+0x" + Long.toHexString(lr - base) : "0x" + Long.toHexString(lr);
                        if (isRand) {
                            int v = next();
                            sites.merge(name + "@" + site, 1, Integer::sum);
                            b.reg_write(Arm64Const.UC_ARM64_REG_X0, (long) v);
                        } else {
                            sites.merge("srand(" + b.reg_read(Arm64Const.UC_ARM64_REG_X0).longValue() + ")@" + site, 1, Integer::sum);
                        }
                        b.reg_write(Arm64Const.UC_ARM64_REG_PC, lr);
                    }
                    @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
                    @Override public void detach() {}
                }, addr, addr, null);
            }
        }
    }

    /** Boots with every known entropy source pinned (clock, rand, counter, JNI clock). */
    static MetasecEmulator bootPinned() throws Exception {
        return bootPinned(null);
    }

    /**
     * Code hook over [EARLY_FROM, EARLY_TO) of the .so, installed before it
     * loads (-DearlyHook=from:to, so offsets). Unicorn caches translated
     * blocks, and hooks added later never see blocks that already ran.
     */
    static volatile com.github.unidbg.arm.backend.CodeHook EARLY;
    static volatile com.github.unidbg.arm.backend.WriteHook EARLYW;

    /** The .so loads here in every run; hooks that must precede loading use it. */
    static final long SO_BASE = 0x40000000L;

    static MetasecEmulator bootPinned(java.util.function.Consumer<MetasecEmulator> beforeInit) throws Exception {
        return bootPinned(null, beforeInit);
    }

    static MetasecEmulator bootPinned(java.util.function.Consumer<com.github.unidbg.AndroidEmulator> beforeLoad,
                                      java.util.function.Consumer<MetasecEmulator> beforeInit) throws Exception {
        // The clock goes in before the .so loads: its initializers record the launch time.
        MetasecEmulator emu = new MetasecEmulator(new File(SO), e -> {
            ((CustomEmulator) e).getCustomSyscallHandler().setSyscallObserver(new FixedClock(FIXED_SECS * 1_000_000_000L));
            String eh = System.getProperty("earlyHook");
            if (eh != null) {
                for (String range : eh.split(",")) {
                    long from = Long.decode(range.split(":")[0]), to = Long.decode(range.split(":")[1]);
                    e.getBackend().hook_add_new(new com.github.unidbg.arm.backend.CodeHook() {
                        @Override public void hook(Backend b, long address, int size, Object user) {
                            com.github.unidbg.arm.backend.CodeHook h = EARLY;
                            if (h != null) h.hook(b, address, size, user);
                        }
                        @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
                        @Override public void detach() {}
                    }, SO_BASE + from, SO_BASE + to, null);
                }
            }
            if (Boolean.getBoolean("vmwrites")) {
                com.github.unidbg.arm.backend.WriteHook wh = new com.github.unidbg.arm.backend.WriteHook() {
                    @Override public void hook(Backend b, long address, int size, long value, Object user) {
                        com.github.unidbg.arm.backend.WriteHook h = EARLYW;
                        if (h != null) h.hook(b, address, size, value, user);
                    }
                    @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
                    @Override public void detach() {}
                };
                e.getBackend().hook_add_new(wh, 0xbff00000L, 0xc0000000L, null);
                e.getBackend().hook_add_new(wh, 0x40400000L, 0x40800000L, null);
            }
            if (beforeLoad != null) beforeLoad.accept(e);
        });
        if (emu.moduleBase() != SO_BASE) throw new IllegalStateException("so base moved: 0x" + Long.toHexString(emu.moduleBase()));
        if (beforeInit != null) beforeInit.accept(emu);
        new FixedRand().install(emu);
        new FixedCounter().install(emu);
        final long[] reads = {0};
        emu.setJavaClock(new MetasecEmulator.JavaClock() {
            public long currentTimeMillis() { return FIXED_SECS * 1000 + ++reads[0]; }
            public long nanoTime() { return 5_000_000_000L + ++reads[0] * 1000; }
        });
        emu.initSdk(new String(Files.readAllBytes(Paths.get(CONFIG)), StandardCharsets.UTF_8).trim());
        emu.waitForInit();
        return emu;
    }

    /**
     * Prints, in order, each native call the VM makes during the second signing:
     * target, x0..x3, and a short digest of the 32 bytes each pointer argument points at.
     * Diff two runs to find the first call whose inputs diverge.
     */
    static void trace(MetasecEmulator emu, long watchAddr, int watchLen) throws Exception {
        long base = emu.moduleBase();
        final boolean[] on = {false};
        final int[] seq = {0};
        final String[] lastWatch = {null};
        for (long off : thunkSites()) {
            long addr = base + off;
            int insn = java.nio.ByteBuffer.wrap(emu.emulator().getBackend().mem_read(addr, 4)).order(java.nio.ByteOrder.LITTLE_ENDIAN).getInt();
            int rn = (insn >> 5) & 0x1f;
            emu.emulator().getBackend().hook_add_new(new com.github.unidbg.arm.backend.CodeHook() {
                @Override public void hook(Backend b, long address, int size, Object user) {
                    if (!on[0]) return;
                    long target = b.reg_read(Arm64Const.UC_ARM64_REG_X0 + rn).longValue();
                    if (watchLen > 0) {
                        String w;
                        try { w = hex(b.mem_read(watchAddr, watchLen), watchLen); } catch (Exception e) { w = "unmapped"; }
                        if (!w.equals(lastWatch[0])) {
                            System.out.println(String.format("[t] ---- watch 0x%x changed before %04d: %s", watchAddr, seq[0], w));
                            lastWatch[0] = w;
                        }
                    }
                    if (DUMP != null) {
                        for (String d : DUMP) {
                            String[] f = d.split(":");
                            if (seq[0] != Integer.parseInt(f[0])) continue;
                            long a = Long.decode(f[1]);
                            int l = Integer.decode(f[2]);
                            System.out.println("[dump] " + d + " " + hex(b.mem_read(a, l), l));
                        }
                    }
                    StringBuilder sb = new StringBuilder(String.format("[t] %04d %s", seq[0]++, sym(emu, target)));
                    for (int r = 0; r <= 3; r++) {
                        long v = b.reg_read(Arm64Const.UC_ARM64_REG_X0 + r).longValue();
                        sb.append(" x").append(r).append("=0x").append(Long.toHexString(v));
                        if (v > 0x10000 && v < 0xc0000000L) {
                            try { sb.append("{").append(hex(b.mem_read(v, PREVIEW), PREVIEW)).append("}"); } catch (Exception e) {}
                        }
                    }
                    System.out.println(sb);
                }
                @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
                @Override public void detach() {}
            }, addr, addr, null);
        }
        on[0] = Boolean.getBoolean("traceFirst");
        Map<String, String> h1 = emu.getFeatureHash(URL, null);
        System.out.println("[t] sign1 medusa=" + h1.get("X-Medusa") + " khronos=" + h1.get("X-Khronos"));
        on[0] = true;
        Map<String, String> h = emu.getFeatureHash(URL, null);
        on[0] = false;
        System.out.println("[t] medusa md5=" + md5(h.get("X-Medusa")));
    }

    /** Boots with the clock and rand() pinned from the start, then signs a few times. */
    static void determinism() throws Exception {
        MetasecEmulator emu = new MetasecEmulator(new File(SO));
        FixedClock clock = new FixedClock(FIXED_SECS * 1_000_000_000L);
        FixedRand rand = new FixedRand();
        rand.install(emu);
        FixedCounter counter = new FixedCounter();
        counter.install(emu);
        final long[] jniReads = {0};
        emu.setJavaClock(new MetasecEmulator.JavaClock() {
            public long currentTimeMillis() { jniReads[0]++; return FIXED_SECS * 1000 + jniReads[0]; }
            public long nanoTime() { jniReads[0]++; return 5_000_000_000L + jniReads[0] * 1000; }
        });
        handler(emu).setSyscallObserver(clock);
        emu.initSdk(new String(Files.readAllBytes(Paths.get(CONFIG)), StandardCharsets.UTF_8).trim());
        emu.waitForInit();
        System.out.println("init rand sites=" + rand.sites + " clock reads=" + clock.reads);
        for (int i = 0; i < 3; i++) {
            rand.sites.clear();
            int before = counter.reads;
            Map<String, String> h = emu.getFeatureHash(URL, null);
            String m = h.get("X-Medusa");
            System.out.println("run" + i + " khronos=" + h.get("X-Khronos") + " gorgon=" + h.get("X-Gorgon") + " helios=" + h.get("X-Helios"));
            System.out.println("run" + i + " medusa=" + (m == null ? null : m.substring(0, Math.min(120, m.length()))) + " len=" + (m == null ? 0 : m.length()) + " md5=" + md5(m));
            System.out.println("run" + i + " rand sites=" + rand.sites + " counter reads=" + (counter.reads - before) + " jni clock reads=" + jniReads[0]);
        }
    }

    static String md5(String s) {
        if (s == null) return "-";
        try {
            byte[] d = java.security.MessageDigest.getInstance("MD5").digest(s.getBytes(StandardCharsets.UTF_8));
            StringBuilder sb = new StringBuilder();
            for (int i = 0; i < 6; i++) sb.append(String.format("%02x", d[i]));
            return sb.toString();
        } catch (Exception e) { return "?"; }
    }

    /** Logs every mbedtls_base64_encode call while signing: caller and raw input. */
    static void b64(MetasecEmulator emu) {
        long base = emu.moduleBase();
        long fn = base + 0x26c720;
        emu.emulator().getBackend().hook_add_new(new com.github.unidbg.arm.backend.CodeHook() {
            @Override public void hook(Backend b, long address, int size, Object user) {
                long src = b.reg_read(Arm64Const.UC_ARM64_REG_X3).longValue();
                long len = b.reg_read(Arm64Const.UC_ARM64_REG_X4).longValue();
                long lr = b.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                byte[] data = len > 0 && len < 4096 ? b.mem_read(src, len) : new byte[0];
                System.out.println("[b64] lr=" + rel(base, lr) + " src=0x" + Long.toHexString(src) + " len=" + len + " head=" + hex(data, 48));
            }
            @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
            @Override public void detach() {}
        }, fn, fn, null);
        Map<String, String> h = emu.getFeatureHash(URL, null);
        System.out.println("[b64] --- second signing ---");
        h = emu.getFeatureHash(URL, null);
        System.out.println("medusa len=" + h.get("X-Medusa").length() + " head=" + hex(java.util.Base64.getDecoder().decode(h.get("X-Medusa")), 48));
    }

    /** The branch sites in the VM's native-call thunks (so offsets). */
    static long[] thunkSites() throws java.io.IOException {
        String txt = new String(Main.class.getResourceAsStream("/thunks.txt").readAllBytes(), StandardCharsets.UTF_8).trim();
        return java.util.Arrays.stream(txt.split(",")).filter(x -> !x.isEmpty()).mapToLong(x -> Long.parseLong(x, 16)).toArray();
    }

    static String sym(MetasecEmulator emu, long a) {
        long base = emu.moduleBase();
        if (a >= base && a < base + 0x500000) return "so+0x" + Long.toHexString(a - base);
        com.github.unidbg.Module m = emu.emulator().getMemory().findModuleByAddress(a);
        if (m == null) return "0x" + Long.toHexString(a);
        com.github.unidbg.Symbol s = m.findClosestSymbolByAddress(a, false);
        return m.name + ":" + (s != null ? s.getName() + "+0x" + Long.toHexString(a - s.getAddress()) : "0x" + Long.toHexString(a - m.base));
    }

    /** After one signing, scans mapped memory for 256-byte permutations (S-box candidates). */
    static void sboxes(MetasecEmulator emu) {
        emu.getFeatureHash(URL, null);
        long base = emu.moduleBase();
        for (com.github.unidbg.memory.MemoryMap mm : emu.emulator().getMemory().getMemoryMap()) {
            if (mm.size > 64 << 20) continue;
            byte[] m;
            try { m = emu.emulator().getBackend().mem_read(mm.base, mm.size); } catch (Exception e) { continue; }
            for (int off = 0; off + 256 <= m.length; off += 4) {
                boolean[] seen = new boolean[256];
                boolean ok = true;
                boolean identity = true;
                for (int i = 0; i < 256; i++) {
                    int v = m[off + i] & 0xff;
                    if (seen[v]) { ok = false; break; }
                    seen[v] = true;
                    if (v != i) identity = false;
                }
                if (ok && !identity) {
                    long a = mm.base + off;
                    System.out.println("[sbox] " + rel(base, a) + " head=" + hex(java.util.Arrays.copyOfRange(m, off, off + 16), 16));
                    off += 252;
                }
            }
        }
    }

    /**
     * Calls the F13 native (so+0x2aac34) directly. Each argument is one input
     * as hex: 32-byte query SM3, 16-byte body MD5, 4-byte LE ts (52 bytes),
     * optionally followed by ":khronos". Prints the 20-byte result.
     */
    static void f13(MetasecEmulator emu, String[] args) {
        emu.getFeatureHash(URL, null);
        com.github.unidbg.memory.Memory mem = emu.emulator().getMemory();
        Backend b = emu.emulator().getBackend();
        long base = emu.moduleBase();
        for (int i = 1; i < args.length; i++) {
            String[] f = args[i].split(":");
            byte[] data = hexToBytes(f[0]);
            long khronos = f.length > 1 && !f[1].isEmpty() ? Long.parseLong(f[1]) : (java.nio.ByteBuffer.wrap(data, 48, 4).order(java.nio.ByteOrder.LITTLE_ENDIAN).getInt() & 0xffffffffL);
            long variant = f.length > 2 ? Long.parseLong(f[2]) : Long.getLong("variant", 5L);
            com.github.unidbg.pointer.UnidbgPointer buf = mem.malloc(64, true).getPointer();
            buf.write(0, data, 0, data.length);
            com.github.unidbg.pointer.UnidbgPointer str = mem.malloc(48, true).getPointer();
            str.setLong(0, base + 0x3c8288L);
            str.setInt(8, data.length + 1);
            str.setInt(12, data.length);
            str.setLong(16, buf.peer);
            com.github.unidbg.pointer.UnidbgPointer out = mem.malloc(64, true).getPointer();
            out.write(0, new byte[64], 0, 64);
            System.out.println(String.format("[f13buf] in=0x%x str=0x%x out=0x%x", buf.peer, str.peer, out.peer));
            b.reg_write(Arm64Const.UC_ARM64_REG_X8, out.peer);
            if (F13_HOOK_ON != null) F13_HOOK_ON[0] = true;
            emu.emulator().eFunc(base + 0x2aac34L, str.peer, variant, khronos);
            if (F13_HOOK_ON != null) F13_HOOK_ON[0] = false;
            long ptr = out.getLong(16);
            int len = out.getInt(12);
            String res = ptr != 0 && len > 0 && len < 256 ? hex(b.mem_read(ptr, len), len) : "out=" + hex(out.getByteArray(0, 32), 32);
            System.out.println("[f13] " + args[i] + " -> " + res);
        }
    }

    /** Runs F13 once with a code hook over .text and reports the executed footprint. */
    static void f13prof(MetasecEmulator emu, String in) {
        long base = emu.moduleBase();
        final boolean[] on = {false};
        final long[] total = {0};
        Map<Long, long[]> pages = new TreeMap<>();
        emu.emulator().getBackend().hook_add_new(new com.github.unidbg.arm.backend.CodeHook() {
            @Override public void hook(Backend b, long address, int size, Object user) {
                if (!on[0]) return;
                total[0]++;
                pages.computeIfAbsent((address - base) & ~0xfffL, k -> new long[1])[0]++;
            }
            @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
            @Override public void detach() {}
        }, base + 0x172900L, base + 0x3c0654L, null);
        F13_HOOK_ON = on;
        f13(emu, new String[]{"f13", in});
        System.out.println("[prof] total native instructions " + total[0]);
        pages.forEach((k, v) -> System.out.println("[prof] so+0x" + Long.toHexString(k) + " " + v[0]));
    }

    static boolean[] F13_HOOK_ON;

    interface Code { void at(Backend b, long address); }

    static com.github.unidbg.arm.backend.CodeHook code(Code c) {
        return new com.github.unidbg.arm.backend.CodeHook() {
            @Override public void hook(Backend b, long address, int size, Object user) { c.at(b, address); }
            @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
            @Override public void detach() {}
        };
    }

    /**
     * Probes VM handlers in isolation. Takes a CPU snapshot at the dispatch
     * code of handler 0x3b (so+0x183724: ldr w14,[x13,#4]!; str x13,[x28];
     * ... br) during F13, then for each template word in the file (one hex
     * word per line) runs: the word as is, each single-bit flip, and a few
     * random register files. Prints written VM register slots and the VM pc delta.
     */
    static void probe(MetasecEmulator emu, String in, String wordsFile) throws Exception {
        long base = emu.moduleBase();
        Backend b = emu.emulator().getBackend();
        final long[] snap = new long[34];
        final byte[][] stack = new byte[1][];
        final long[] stackLo = {0};
        final boolean[] want = {false};
        final long dispatch = base + 0x183724L;
        EARLY = code((bb, address) -> {
            if (want[0] && address == dispatch) {
                for (int i = 0; i <= 28; i++) snap[i] = bb.reg_read(Arm64Const.UC_ARM64_REG_X0 + i).longValue();
                snap[29] = bb.reg_read(Arm64Const.UC_ARM64_REG_FP).longValue();
                snap[30] = bb.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                snap[31] = bb.reg_read(Arm64Const.UC_ARM64_REG_SP).longValue();
                snap[32] = bb.reg_read(Arm64Const.UC_ARM64_REG_NZCV).longValue();
                stackLo[0] = (snap[31] - 0x2000) & ~0xfffL;
                long hi = Math.max(snap[29], Math.max(snap[4], snap[28])) + 0x2000;
                stack[0] = bb.mem_read(stackLo[0], (int) (hi - stackLo[0]));
                want[0] = false;
            }
        });
        want[0] = true;
        F13_HOOK_ON = new boolean[]{false};
        f13(emu, new String[]{"f13", in});
        EARLY = null;
        if (want[0]) throw new IllegalStateException("no snapshot");
        long regfile = snap[4];
        long slot = snap[28];
        long savedSlot = readLong(b, slot);
        byte[] savedRegs = b.mem_read(regfile, 32 * 8);
        com.github.unidbg.pointer.UnidbgPointer wb = emu.emulator().getMemory().malloc(64, true).getPointer();
        boolean memprobe = Boolean.getBoolean("memprobe");
        boolean flips = !"false".equals(System.getProperty("flips"));
        final int SCR = 1 << 20;
        com.github.unidbg.pointer.UnidbgPointer scr = memprobe ? emu.emulator().getMemory().malloc(SCR, true).getPointer() : null;
        byte[] scrInit = new byte[SCR];
        new java.util.Random(99).nextBytes(scrInit);
        long wordbuf = wb.peer;
        java.util.Random rnd = new java.util.Random(1);
        final long[] stopAt = {0};
        com.github.unidbg.arm.backend.CodeHook stopper = new com.github.unidbg.arm.backend.CodeHook() {
            @Override public void hook(Backend bb, long address, int size, Object user) {
                long v = readLong(bb, slot);
                if (v != wordbuf + 4) { stopAt[0] = v; bb.emu_stop(); }
            }
            @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
            @Override public void detach() {}
        };
        final boolean[] armed = {false};
        EARLY = code((bb, address) -> { if (armed[0]) stopper.hook(bb, address, 4, null); });
        for (String line : java.nio.file.Files.readAllLines(java.nio.file.Paths.get(wordsFile))) {
            line = line.trim();
            if (line.isEmpty()) continue;
            int tmpl = (int) Long.parseLong(line, 16);
            java.util.List<int[]> runs = new java.util.ArrayList<>();
            for (int k = 0; k < 4; k++) runs.add(new int[]{tmpl, k});
            if (flips) for (int bit = 0; bit < 32; bit++) runs.add(new int[]{tmpl ^ (1 << bit), 0});
            for (int[] run : runs) {
                long[] regs = new long[32];
                java.util.Random rr = new java.util.Random(run[1] * 7919L + 13);
                for (int i = 0; i < 32; i++) regs[i] = rr.nextLong() & (rr.nextBoolean() ? 0xffffffffL : -1L);
                if (memprobe) {
                    for (int i = 0; i < 32; i++) regs[i] = scr.peer + 0x4000L + 0x7000L * i + (rr.nextInt(0x200) << 3);
                    b.mem_write(scr.peer, scrInit);
                }
                regs[0] = 0;
                java.nio.ByteBuffer bbuf = java.nio.ByteBuffer.allocate(256).order(java.nio.ByteOrder.LITTLE_ENDIAN);
                for (long v : regs) bbuf.putLong(v);
                b.mem_write(stackLo[0], stack[0]);
                b.mem_write(regfile, bbuf.array());
                byte[] wbytes = java.nio.ByteBuffer.allocate(8).order(java.nio.ByteOrder.LITTLE_ENDIAN).putInt(0).putInt(run[0]).array();
                b.mem_write(wordbuf, wbytes);
                for (int i = 0; i <= 28; i++) b.reg_write(Arm64Const.UC_ARM64_REG_X0 + i, snap[i]);
                b.reg_write(Arm64Const.UC_ARM64_REG_X13, wordbuf);
                b.reg_write(Arm64Const.UC_ARM64_REG_FP, snap[29]);
                b.reg_write(Arm64Const.UC_ARM64_REG_LR, snap[30]);
                b.reg_write(Arm64Const.UC_ARM64_REG_SP, snap[31]);
                b.reg_write(Arm64Const.UC_ARM64_REG_NZCV, snap[32]);
                stopAt[0] = 0;
                String err = "";
                // First store to [x28] (= wordbuf+4) happens in the dispatch; arm after it.
                armed[0] = false;
                try {
                    b.emu_start(dispatch, dispatch + 8, 0, 0);   // ldr + str
                    armed[0] = true;
                    b.emu_start(dispatch + 8, 0, 0, 5000);
                } catch (Exception ex) {
                    err = " err=" + ex.getMessage() + " pc=" + rel(base, b.reg_read(Arm64Const.UC_ARM64_REG_PC).longValue());
                }
                armed[0] = false;
                byte[] after = b.mem_read(regfile, 256);
                java.nio.ByteBuffer ab = java.nio.ByteBuffer.wrap(after).order(java.nio.ByteOrder.LITTLE_ENDIAN);
                StringBuilder sb = new StringBuilder(String.format("[p] tmpl=%08x word=%08x seed=%d dpc=%d", tmpl, run[0], run[1], stopAt[0] == 0 ? 0 : stopAt[0] - (wordbuf + 4)));
                sb.append(" in=");
                for (int i = 0; i < 32; i++) sb.append(i == 0 ? "" : ",").append(Long.toHexString(regs[i]));
                sb.append(" out=");
                for (int i = 0; i < 32; i++) { long v = ab.getLong(); if (v != regs[i]) sb.append(i).append(':').append(Long.toHexString(v)).append(';'); }
                if (memprobe && Boolean.getBoolean("probedebug")) {
                    System.out.println("[dbg] r3 mem after=" + hex(b.mem_read(regs[3], 8), 8) + " before=" + hex(java.util.Arrays.copyOfRange(scrInit, (int) (regs[3] - scr.peer), (int) (regs[3] - scr.peer) + 8), 8) + " stop=" + Long.toHexString(stopAt[0]));
                }
                if (memprobe) {
                    byte[] now = b.mem_read(scr.peer, SCR);
                    sb.append(" mem=");
                    for (int i = 0; i < SCR; i++) {
                        if (now[i] != scrInit[i]) {
                            int j = i;
                            while (j < SCR && j < i + 16 && now[j] != scrInit[j]) j++;
                            sb.append(Long.toHexString(scr.peer + i)).append(':').append(hex(java.util.Arrays.copyOfRange(now, i, j), j - i)).append(';');
                            i = j;
                        }
                    }
                    sb.append(" scr=").append(Long.toHexString(scr.peer));
                }
                sb.append(err);
                System.out.println(sb);
            }
        }
        b.mem_write(regfile, savedRegs);
    }

    /** Maps each VM (op, sub) executed by F13 to its handler: logs br targets in the interpreter. */
    static void f13handlers(MetasecEmulator emu, String in) {
        long base = emu.moduleBase();
        final boolean[] on = {false};
        Map<String, Integer> seen = new TreeMap<>();
        Backend backend = emu.emulator().getBackend();
        EARLY = (new com.github.unidbg.arm.backend.CodeHook() {
            @Override public void hook(Backend b, long address, int size, Object user) {
                if (!on[0]) return;
                int insn = java.nio.ByteBuffer.wrap(b.mem_read(address, 4)).order(java.nio.ByteOrder.LITTLE_ENDIAN).getInt();
                if ((insn & 0xfffffc1f) != 0xd61f0000) return; // br xN
                int rn = (insn >> 5) & 31;
                long target = b.reg_read(Arm64Const.UC_ARM64_REG_X0 + rn).longValue();
                long vpc;
                try { vpc = readLong(b, b.reg_read(Arm64Const.UC_ARM64_REG_X28).longValue()); } catch (Exception e) { return; }
                int word = vpc > base && vpc < base + 0x500000 ? java.nio.ByteBuffer.wrap(b.mem_read(vpc, 4)).order(java.nio.ByteOrder.LITTLE_ENDIAN).getInt() : 0;
                String key = String.format("op=%02x sub=%02x br@%s -> %s", word & 0x3f, (word >> 6) & 0x3f, rel(base, address), rel(base, target));
                seen.merge(key, 1, Integer::sum);
            }
            @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
            @Override public void detach() {}
        });
        F13_HOOK_ON = on;
        f13(emu, new String[]{"f13", in});
        seen.forEach((k, v) -> System.out.println("[h] " + v + "x " + k));
    }

    /**
     * Traces F13 at VM level: whenever the VM pc ([x28]) changes, prints the
     * instruction word, the native pc of the handler and the 32 VM registers (x4).
     */
    static final long WATCH = Long.decode(System.getProperty("watchaddr", "0"));
    static final Map<Long, String[]> STORES = loadStores();

    /** Reads general register xN (x29 and x30 have their own unicorn ids). */
    static long xr(Backend b, int n) {
        int id = n == 29 ? Arm64Const.UC_ARM64_REG_FP : n == 30 ? Arm64Const.UC_ARM64_REG_LR : Arm64Const.UC_ARM64_REG_X0 + n;
        return b.reg_read(id).longValue();
    }

    static Map<Long, String[]> loadStores() {
        Map<Long, String[]> m = new java.util.HashMap<>();
        String path = System.getProperty("stores");
        if (path == null) return m;
        try {
            for (String line : java.nio.file.Files.readAllLines(java.nio.file.Paths.get(path))) {
                String[] f = line.trim().split(" ");
                if (f.length >= 8) m.put(Long.parseLong(f[0], 16), f);
            }
        } catch (java.io.IOException e) { throw new RuntimeException(e); }
        return m;
    }
    static final long[] watchLast = {Long.MIN_VALUE};

    static void f13vm(MetasecEmulator emu, String in) {
        long base = emu.moduleBase();
        final boolean[] on = {false};
        final long[] lastPc = {0};
        final int[] n = {0};
        EARLY = (new com.github.unidbg.arm.backend.CodeHook() {
            @Override public void hook(Backend b, long address, int size, Object user) {
                if (!on[0]) return;
                String[] st = STORES.get(address - base);
                if (st != null) {
                    // addr kind size rt rt2 rn mode val [post]
                    int sz = Integer.parseInt(st[2]);
                    long rn = xr(b, Integer.parseInt(st[5]));
                    long ea;
                    String mode = st[6];
                    long v = Long.parseLong(st[7]);
                    if (mode.startsWith("reg")) {
                        long rm = xr(b, (int) v);
                        if (mode.contains("sxtw")) rm = (int) rm;
                        else if (mode.contains("uxtw")) rm &= 0xffffffffL;
                        int shift = mode.matches(".*\\d$") ? Integer.parseInt(mode.replaceAll("\\D", "")) : 0;
                        ea = rn + (rm << shift);
                    } else {
                        ea = (st.length > 8 && st[8].equals("post")) ? rn : rn + v;
                    }
                    long val1 = st[3].equals("zr") || sz == 16 ? 0 : xr(b, Integer.parseInt(st[3]));
                    if (st[1].equals("pair")) {
                        long val2 = st[4].equals("zr") || sz == 16 ? 0 : xr(b, Integer.parseInt(st[4]));
                        System.out.println(String.format("[vm-store] addr=0x%x size=%d value=0x%x%s", ea, sz, val1, sz == 16 ? " q" : ""));
                        System.out.println(String.format("[vm-store] addr=0x%x size=%d value=0x%x%s", ea + sz, sz, val2, sz == 16 ? " q" : ""));
                    } else {
                        System.out.println(String.format("[vm-store] addr=0x%x size=%d value=0x%x%s", ea, sz, val1, sz == 16 ? " q" : ""));
                    }
                }
                if (address >= base + 0x188000L) {
                    if (address == base + 0x5cc640L) {
                        System.out.println(String.format("[vm-memset] dst=0x%x val=0x%x len=%d", b.reg_read(Arm64Const.UC_ARM64_REG_X0).longValue(), b.reg_read(Arm64Const.UC_ARM64_REG_X1).longValue(), b.reg_read(Arm64Const.UC_ARM64_REG_X2).longValue()));
                        return;
                    }
                    if (address == base + 0x3a0840L || address == base + 0x5cc148L) {
                        System.out.println(String.format("[vm-memcpy] dst=0x%x src=0x%x len=%d", b.reg_read(Arm64Const.UC_ARM64_REG_X0).longValue(), b.reg_read(Arm64Const.UC_ARM64_REG_X1).longValue(), b.reg_read(Arm64Const.UC_ARM64_REG_X2).longValue()));
                    }
                    return;
                }
                long x28 = b.reg_read(Arm64Const.UC_ARM64_REG_X28).longValue();
                long vpc;
                try { vpc = readLong(b, x28); } catch (Exception e) { return; }
                if (vpc == lastPc[0] || vpc < base || vpc > base + 0x500000) return;
                lastPc[0] = vpc;
                if (WATCH != 0) {
                    long wv = readLong(b, WATCH);
                    if (wv != watchLast[0]) { System.out.println(String.format("[watchmem] before n=%d vpc=so+0x%x value=0x%x", n[0], vpc - base, wv)); watchLast[0] = wv; }
                }
                int word = java.nio.ByteBuffer.wrap(b.mem_read(vpc, 4)).order(java.nio.ByteOrder.LITTLE_ENDIAN).getInt();
                long x4 = Long.getLong("regfile", b.reg_read(Arm64Const.UC_ARM64_REG_X4).longValue());
                StringBuilder sb = new StringBuilder(String.format("[vm] %04d vpc=so+0x%x word=%08x op=%02x pc=%s x4=0x%x regs", n[0]++, vpc - base, word, word & 0x3f, rel(base, address), x4));
                try {
                    byte[] r = b.mem_read(x4, 32 * 8);
                    java.nio.ByteBuffer bb = java.nio.ByteBuffer.wrap(r).order(java.nio.ByteOrder.LITTLE_ENDIAN);
                    for (int i = 0; i < 32; i++) sb.append(' ').append(Long.toHexString(bb.getLong()));
                } catch (Exception e) { sb.append(" ?"); }
                System.out.println(sb);
            }
            @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
            @Override public void detach() {}
        });
        long rf = Long.getLong("regfile", 0L);
        EARLYW = new com.github.unidbg.arm.backend.WriteHook() {
            @Override public void hook(Backend b, long address, int size, long value, Object user) {
                if (!on[0]) return;
                if (rf != 0 && address >= rf && address < rf + 256) return;
                long pc = b.reg_read(Arm64Const.UC_ARM64_REG_PC).longValue();
                if (pc < base + 0x17d000L || pc >= base + 0x188000L) return; // VM handlers only
                System.out.println(String.format("[vm-write] addr=0x%x size=%d value=0x%x", address, size, value));
            }
            @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
            @Override public void detach() {}
        };
        F13_HOOK_ON = on;
        f13(emu, new String[]{"f13", in});
    }

    /**
     * Signs each request (url or url|bodyhex) and prints the full header set.
     * The khronos comes from the pinned clock (-Dsecs), so vary it per run.
     */
    static void vectors(MetasecEmulator emu, String[] args) {
        for (int i = 1; i < args.length; i++) {
            String[] parts = args[i].split("\\|", 2);
            byte[] body = parts.length > 1 ? hexToBytes(parts[1]) : null;
            Map<String, String> h = emu.getFeatureHash(parts[0], body);
            System.out.println("[vec] url=" + parts[0] + " body=" + (body == null ? "-" : parts[1])
                    + " khronos=" + h.get("X-Khronos") + " gorgon=" + h.get("X-Gorgon") + " helios=" + h.get("X-Helios")
                    + " argus=" + h.get("X-Argus") + " ladon=" + h.get("X-Ladon") + " medusa=" + h.get("X-Medusa"));
        }
    }

    static byte[] hexToBytes(String h) {
        byte[] d = new byte[h.length() / 2];
        for (int i = 0; i < d.length; i++) d[i] = (byte) Integer.parseInt(h.substring(2 * i, 2 * i + 2), 16);
        return d;
    }

    /** Lists the native functions the VM calls during the second signing. */
    static void natives(MetasecEmulator emu) throws Exception {
        long base = emu.moduleBase();
        final boolean[] on = {false};
        Map<String, Integer> counts = new java.util.LinkedHashMap<>();
        for (long off : thunkSites()) {
            long addr = base + off;
            int insn = java.nio.ByteBuffer.wrap(emu.emulator().getBackend().mem_read(addr, 4)).order(java.nio.ByteOrder.LITTLE_ENDIAN).getInt();
            int rn = (insn >> 5) & 0x1f;
            emu.emulator().getBackend().hook_add_new(new com.github.unidbg.arm.backend.CodeHook() {
                @Override public void hook(Backend b, long address, int size, Object user) {
                    if (!on[0]) return;
                    long target = b.reg_read(Arm64Const.UC_ARM64_REG_X0 + rn).longValue();
                    counts.merge(sym(emu, target) + " via so+0x" + Long.toHexString(address - base), 1, Integer::sum);
                }
                @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
                @Override public void detach() {}
            }, addr, addr, null);
        }
        emu.getFeatureHash(URL, null);
        on[0] = true;
        emu.getFeatureHash(URL, null);
        on[0] = false;
        counts.forEach((k, v) -> System.out.println("[n] " + v + "x " + k));
    }

    /** At so+off during the second signing: registers, frame-pointer backtrace, and x0..x2 memory. */
    static void at(MetasecEmulator emu, long off, int max) {
        long base = emu.moduleBase();
        final boolean[] on = {false};
        final int[] n = {0};
        long addr = base + off;
        emu.emulator().getBackend().hook_add_new(new com.github.unidbg.arm.backend.CodeHook() {
            @Override public void hook(Backend b, long address, int size, Object user) {
                if (!on[0] || n[0]++ >= max) return;
                StringBuilder sb = new StringBuilder("[at] #" + n[0]);
                for (int r = 0; r <= 8; r++) sb.append(" x").append(r).append("=0x").append(Long.toHexString(b.reg_read(Arm64Const.UC_ARM64_REG_X0 + r).longValue()));
                sb.append("\n[at]   bt: ").append(rel(base, b.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue()));
                long fp = b.reg_read(Arm64Const.UC_ARM64_REG_FP).longValue();
                for (int i = 0; i < 10 && fp != 0; i++) {
                    try {
                        long next = readLong(b, fp);
                        long ret = readLong(b, fp + 8);
                        sb.append(" <- ").append(rel(base, ret));
                        if (next <= fp) break;
                        fp = next;
                    } catch (Exception e) { break; }
                }
                for (int r = 0; r <= 1; r++) {
                    long p = b.reg_read(Arm64Const.UC_ARM64_REG_X0 + r).longValue();
                    try { sb.append("\n[at]   *x").append(r).append("=").append(hex(b.mem_read(p, 32), 32)); } catch (Exception e) {}
                }
                System.out.println(sb);
            }
            @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
            @Override public void detach() {}
        }, addr, addr, null);
        emu.getFeatureHash(URL, null);
        on[0] = true;
        emu.getFeatureHash(URL, null);
        on[0] = false;
    }

    static long readLong(Backend b, long a) {
        return java.nio.ByteBuffer.wrap(b.mem_read(a, 8)).order(java.nio.ByteOrder.LITTLE_ENDIAN).getLong();
    }

    static volatile String phase = "boot";

    /**
     * Watches 8 bytes at addr from boot on and reports, at each syscall, if
     * they changed since the previous syscall: catches writes made from Java
     * (syscall handlers, JNI), which memory hooks do not see.
     */
    static void watch(long addr) throws Exception {
        final long[] last = {Long.MIN_VALUE};
        final int[] prevNr = {-1};
        MetasecEmulator emu = bootPinned(e -> {
            CustomSyscallHandler h = handler(e);
            FixedClock clock = new FixedClock(FIXED_SECS * 1_000_000_000L);
            h.setSyscallObserver((b, em, nr) -> {
                long v;
                try { v = readLong(b, addr); } catch (Exception ex) { v = Long.MIN_VALUE + 1; }
                if (v != last[0]) {
                    System.out.println("[watch] " + phase + " changed to 0x" + Long.toHexString(v) + " after nr=" + prevNr[0] + " (now nr=" + nr + " lr=" + rel(e.moduleBase(), b.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue()) + ")");
                    last[0] = v;
                }
                prevNr[0] = nr;
                return clock.before(b, em, nr);
            });
        });
        com.github.unidbg.Module m = emu.emulator().getMemory().findModuleByAddress(addr);
        System.out.println("[watch] module at addr: " + (m == null ? "none" : m.name + " base=0x" + Long.toHexString(m.base) + " size=0x" + Long.toHexString(m.size)));
        for (com.github.unidbg.memory.MemoryMap mm : emu.emulator().getMemory().getMemoryMap()) {
            if (addr >= mm.base && addr < mm.base + mm.size) System.out.println("[watch] map base=0x" + Long.toHexString(mm.base) + " size=0x" + Long.toHexString(mm.size));
        }
        phase = "sign1";
        emu.getFeatureHash(URL, null);
        phase = "sign2";
        emu.getFeatureHash(URL, null);
    }

    /** Counts reads per 256-byte bucket of the .so's data sections during the first signing. */
    static void readMap() throws Exception {
        final boolean[] on = {false};
        Map<Long, int[]> buckets = new TreeMap<>();
        MetasecEmulator emu = bootPinned(e -> {
            long base = e.moduleBase();
            com.github.unidbg.arm.backend.ReadHook hook = new com.github.unidbg.arm.backend.ReadHook() {
                @Override public void hook(Backend b, long address, int size, Object user) {
                    if (!on[0]) return;
                    buckets.computeIfAbsent((address - base) & ~0xffL, k -> new int[1])[0]++;
                }
                @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
                @Override public void detach() {}
            };
            e.emulator().getBackend().hook_add_new(hook, base + 0x663b0, base + 0x12a340, null);
            e.emulator().getBackend().hook_add_new(hook, base + 0x3c4ef0, base + 0x467560, null);
        });
        on[0] = true;
        emu.getFeatureHash(URL, null);
        on[0] = false;
        buckets.entrySet().stream().sorted((x, y) -> y.getValue()[0] - x.getValue()[0]).limit(40)
                .forEach(en -> System.out.println("[rm] so+0x" + Long.toHexString(en.getKey()) + " " + en.getValue()[0]));
    }

    /** Logs every read of so+[off, off+len) during the first signing: offset, size, value, pc. */
    static void reads(long off, int len) throws Exception {
        final boolean[] on = {false};
        final int[] n = {0};
        final int[] seq = {0};
        int from = Integer.getInteger("from", 0), to = Integer.getInteger("to", Integer.MAX_VALUE);
        final com.github.unidbg.arm.backend.ReadHook[] rhHolder = new com.github.unidbg.arm.backend.ReadHook[1];
        MetasecEmulator emu = bootPinned(ae -> {
            long a = SO_BASE + off;
            rhHolder[0] = new com.github.unidbg.arm.backend.ReadHook() {
                @Override public void hook(Backend b, long address, int size, Object user) {
                    if (!on[0] || seq[0] < from || seq[0] > to) return;
                    long pc = b.reg_read(Arm64Const.UC_ARM64_REG_PC).longValue();
                    byte[] v = b.mem_read(address, size);
                    System.out.println(String.format("[r] %05d call=%d +0x%x size=%d val=%s pc=%s", n[0]++, seq[0], address - SO_BASE, size, hex(v, size), rel(SO_BASE, pc)));
                }
                @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
                @Override public void detach() {}
            };
            if (Boolean.getBoolean("viaLdrb")) {
                // The VM's indexed byte load (ldrb w15, [x16, x17]); memory hooks miss some pages.
                long pcAddr = SO_BASE + 0x185e98L;
                ae.getBackend().hook_add_new(new com.github.unidbg.arm.backend.CodeHook() {
                    @Override public void hook(Backend b, long address, int size, Object user) {
                        if (!on[0] || seq[0] < from || seq[0] > to) return;
                        long ea = b.reg_read(Arm64Const.UC_ARM64_REG_X16).longValue() + b.reg_read(Arm64Const.UC_ARM64_REG_X17).longValue();
                        if (ea < a || ea >= a + len) return;
                        System.out.println(String.format("[r] %05d call=%d +0x%x size=1 val=%s pc=ldrb", n[0]++, seq[0], ea - SO_BASE, hex(b.mem_read(ea, 1), 1)));
                    }
                    @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
                    @Override public void detach() {}
                }, pcAddr, pcAddr, null);
            } else {
                ae.getBackend().hook_add_new(rhHolder[0], a, a + len, null);
            }
        }, e -> {
            try {
                for (long t : thunkSites()) {
                    long ta = e.moduleBase() + t;
                    e.emulator().getBackend().hook_add_new(new com.github.unidbg.arm.backend.CodeHook() {
                        @Override public void hook(Backend b, long address, int size, Object user) {
                            if (!on[0]) return;
                            seq[0]++;
                            if (seq[0] == 534) System.out.println("[bits] " + hex(b.mem_read(0x4079ef80L, 32), 32));
                            if (seq[0] == 553) System.out.println("[ct] " + hex(b.mem_read(0x4079ef80L, 32), 32));
                        }
                        @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
                        @Override public void detach() {}
                    }, ta, ta, null);
                }
            } catch (java.io.IOException ex) { throw new RuntimeException(ex); }
        });
        on[0] = true;
        emu.getFeatureHash(URL, null);
        on[0] = false;
    }

    /** Logs every write to [addr, addr+len) from boot on, tagged with the phase and value. */
    static void allWriters(long addr, int len) throws Exception {
        final long[] baseHolder = {0};
        MetasecEmulator emu = bootPinned(e -> e.emulator().getBackend().hook_add_new(new com.github.unidbg.arm.backend.WriteHook() {
            int n;
            @Override public void hook(Backend b, long address, int size, long value, Object user) {
                if (n++ > 60) return;
                long base = e.moduleBase();
                long pc = b.reg_read(Arm64Const.UC_ARM64_REG_PC).longValue();
                long lr = b.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                System.out.println("[aw] " + phase + " +" + (address - addr) + " size=" + size + " value=0x" + Long.toHexString(value) + " pc=" + rel(base, pc) + " lr=" + rel(base, lr));
            }
            @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
            @Override public void detach() {}
        }, addr, addr + len, null));
        phase = "sign1";
        emu.getFeatureHash(URL, null);
        phase = "sign2";
        emu.getFeatureHash(URL, null);
    }

    /** Logs which code writes [addr, addr+len) during the second signing. */
    static void writers(MetasecEmulator emu, long addr, int len) {
        long base = emu.moduleBase();
        final boolean[] on = {false};
        Map<String, int[]> pcs = new java.util.LinkedHashMap<>();
        emu.emulator().getBackend().hook_add_new(new com.github.unidbg.arm.backend.WriteHook() {
            @Override public void hook(Backend b, long address, int size, long value, Object user) {
                if (!on[0]) return;
                long pc = b.reg_read(Arm64Const.UC_ARM64_REG_PC).longValue();
                long lr = b.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                String key = rel(base, pc) + " (lr " + rel(base, lr) + ")";
                int off = (int) (address - addr);
                int[] st = pcs.computeIfAbsent(key, k -> new int[]{Integer.MAX_VALUE, Integer.MIN_VALUE, 0});
                st[0] = Math.min(st[0], off); st[1] = Math.max(st[1], off + size); st[2]++;
            }
            @Override public void onAttach(com.github.unidbg.arm.backend.UnHook u) {}
            @Override public void detach() {}
        }, addr, addr + len, null);
        emu.getFeatureHash(URL, null);
        on[0] = true;
        emu.getFeatureHash(URL, null);
        on[0] = false;
        pcs.forEach((k, v) -> System.out.println("[w] " + k + " off=[" + v[0] + "," + v[1] + ") n=" + v[2]));
    }

    static final int PREVIEW = Integer.getInteger("preview", 16);
    /** -Ddump=seq:addr:len dumps memory when the trace reaches call seq. */
    static final String[] DUMP = System.getProperty("dump") == null ? null : System.getProperty("dump").split(",");

    static String hex(byte[] d, int max) {
        StringBuilder sb = new StringBuilder();
        for (int i = 0; i < Math.min(d.length, max); i++) sb.append(String.format("%02x", d[i]));
        return sb.toString();
    }

    static String rel(long base, long a) {
        return (a >= base && a < base + 0x500000) ? "so+0x" + Long.toHexString(a - base) : "0x" + Long.toHexString(a);
    }

    static String cstr(Backend b, long p) {
        try {
            byte[] m = b.mem_read(p, 256);
            int n = 0;
            while (n < m.length && m[n] != 0) n++;
            return new String(m, 0, n, StandardCharsets.UTF_8);
        } catch (Exception ex) {
            return "?";
        }
    }
}
