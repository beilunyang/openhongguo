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
    static final String URL = "https://api5-normal-sinfonlinea.fqnovel.com/reading/bookapi/search/tab/v?query=abc&aid=8662&device_id=1234567890123456&iid=1234567890123457&version_code=74332&ts=1763460000&_rticket=1763460000123";

    public static void main(String[] args) throws Exception {
        System.setIn(new java.io.ByteArrayInputStream(new byte[0]));
        String tool = args.length > 0 ? args[0] : "syscalls";
        if (tool.equals("determinism")) { determinism(); System.exit(0); }
        MetasecEmulator emu = boot();
        switch (tool) {
            case "syscalls": syscalls(emu); break;
            case "b64": b64(emu); break;
            case "natives": natives(emu); break;
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

    static final long FIXED_SECS = 1763460000L;

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

    /** Boots with the clock and rand() pinned from the start, then signs a few times. */
    static void determinism() throws Exception {
        MetasecEmulator emu = new MetasecEmulator(new File(SO));
        FixedClock clock = new FixedClock(1763460000_000_000_000L);
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
