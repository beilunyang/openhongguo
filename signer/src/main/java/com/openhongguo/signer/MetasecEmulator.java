package com.openhongguo.signer;

import com.github.unidbg.AndroidEmulator;
import com.github.unidbg.Emulator;
import com.github.unidbg.Module;
import com.github.unidbg.linux.android.AndroidResolver;
import com.github.unidbg.linux.android.dvm.*;
import com.github.unidbg.linux.android.dvm.array.ByteArray;
import com.github.unidbg.linux.android.dvm.wrapper.DvmBoolean;
import com.github.unidbg.memory.Memory;
import com.github.unidbg.pointer.UnidbgPointer;
import com.github.unidbg.arm.backend.Backend;
import com.github.unidbg.arm.backend.CodeHook;
import com.github.unidbg.arm.backend.UnHook;
import com.github.unidbg.arm.backend.WriteHook;
import com.github.unidbg.thread.ThreadDispatcher;
import unicorn.Arm64Const;

import java.io.*;
import java.nio.charset.StandardCharsets;
import java.util.*;
import java.util.concurrent.*;
import java.lang.reflect.Field;
import java.util.concurrent.atomic.AtomicInteger;

public class MetasecEmulator extends AbstractJni {

    private final AndroidEmulator emulator;
    private final VM vm;
    private final Module module;
    private final DvmClass k3Class;
    private long sdkHandle;

    private final List<long[]> capturedThreads = new ArrayList<>();

    private String deviceModel = "SM-T220";
    private String deviceBrand = "samsung";
    private String osVersion = "14";
    private int sdkInt = 34;
    private String packageName = "com.phoenix.read";

    // Debug: track futex calls during signing
    private volatile boolean signingActive = false;
    private final AtomicInteger futexCallCount = new AtomicInteger(0);
    private final Map<Long, AtomicInteger> futexAddrCounts = new ConcurrentHashMap<>();

    public MetasecEmulator(File soFile) throws IOException {
        emulator = new CustomEmulator(packageName);
        // Cooperative scheduler: pthread_create becomes a MarshmallowThread and
        // futex wait/wake switches tasks. Without this, clone only returns a tid.
        emulator.getSyscallHandler().setEnableThreadDispatcher(true);
        System.out.println("[openhongguo-signer] thread dispatcher enabled");

        Memory memory = emulator.getMemory();
        memory.setLibraryResolver(new AndroidResolver(23));

        vm = ((com.github.unidbg.linux.android.AndroidARM64Emulator) emulator).createDalvikVM();
        vm.setJni(this);
        vm.setVerbose(false);

        DvmClass objectClass = vm.resolveClass("java/lang/Object");
        DvmClass k3 = vm.resolveClass("ms/bd/c/k3", objectClass);
        DvmClass q2Class = vm.resolveClass("ms/bd/c/q2", k3);
        vm.resolveClass("com/bytedance/mobsec/metasec/ml/MS", q2Class);
        vm.resolveClass("ms/bd/c/f3", objectClass);
        vm.resolveClass("android/app/Application", vm.resolveClass("android/content/ContextWrapper",
                vm.resolveClass("android/content/Context", objectClass)));
        vm.resolveClass("java/util/HashMap", objectClass);

        this.k3Class = k3;

        DalvikModule dm = vm.loadLibrary(soFile, false);
        module = dm.getModule();

        System.out.println("[openhongguo-signer] .so loaded at base=0x" + Long.toHexString(module.base)
                + " size=0x" + Long.toHexString(module.size));

        // Configure CustomSyscallHandler with module info
        configureSyscallHandler();

        if (System.getenv("EMU_TRACE") != null) installTrace();
        dm.callJNI_OnLoad(emulator);
        System.out.println("[openhongguo-signer] JNI_OnLoad completed");
        UnidbgPointer fn = k3.findNativeFunction(emulator,
                "a(IIJLjava/lang/String;Ljava/lang/Object;)Ljava/lang/Object;");
        System.out.println("[openhongguo-signer] k3.a native=" + (fn == null ? "null" : loc(fn.peer)));
    }

    private int traceLogs;

    /** Log the constructor, the pool post, and each task the workers actually run. */
    private void installTrace() {
        hookSite(0x188a60L, "store+90");
        hookSite(0x178f28L, "sign-lock");
        // so+0x17467c stores 1 at [obj+0x28] when init actually finishes.
        // Signing returned X-Neptune -8 while that byte was still 0.
        hookSite(0x174870L, "ready-gate");
        hookSite(0x174c5cL, "ready-set");
        hookSite(0x17c930L, "pool-task");
        hookSite(0x2a5cd4L, "publish-ctx");
        hookSprintf();
        hookMinusEight();
        hookSingletonWrites();
    }

    /** Any store into the SDK context (so+0x460c40) or singleton (so+0x460c48). */
    private void hookSingletonWrites() {
        long begin = module.base + 0x460c40L;
        long end = module.base + 0x460c4fL;
        emulator.getBackend().hook_add_new(new WriteHook() {
            private int logs;

            @Override
            public void hook(Backend backend, long address, int size, long value, Object user) {
                if (logs++ > 20) {
                    return;
                }
                long pc = backend.reg_read(Arm64Const.UC_ARM64_REG_PC).longValue();
                long lr = backend.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                System.out.println("[slot] pc=" + loc(pc)
                        + " lr=" + loc(lr)
                        + " addr=" + loc(address)
                        + " size=" + size
                        + " value=" + loc(value));
            }

            @Override
            public void onAttach(UnHook unHook) {
            }

            @Override
            public void detach() {
            }
        }, begin, end, null);
    }

    /** Catch the store that plants the X-Neptune code -8. */
    private void hookMinusEight() {
        WriteHook hook = new WriteHook() {
            private int logs;

            @Override
            public void hook(Backend backend, long address, int size, long value, Object user) {
                if (!signingActive || size != 4 || (value & 0xffffffffL) != 0xfffffff8L || logs++ > 15) {
                    return;
                }
                long pc = backend.reg_read(Arm64Const.UC_ARM64_REG_PC).longValue();
                long x28 = backend.reg_read(Arm64Const.UC_ARM64_REG_X28).longValue();
                String pcWord = "";
                if (x28 > 0x10000) {
                    try {
                        pcWord = " [x28]=" + loc(readLong(backend, x28));
                    } catch (Exception ignored) {
                    }
                }
                System.out.println("[w-8] pc=" + loc(pc)
                        + " addr=" + loc(address)
                        + pcWord);
            }

            @Override
            public void onAttach(UnHook unHook) {
            }

            @Override
            public void detach() {
            }
        };
        emulator.getBackend().hook_add_new(hook, 0x40000000L, 0x50000000L, null);
        emulator.getBackend().hook_add_new(hook, 0xb0000000L, 0xc0000000L, null);
    }

    /** so+0x26f97c(std::string*, fmt, ...). The X-Neptune value is "%d|%s". */
    private void hookSprintf() {
        long addr = module.base + 0x26f97cL;
        emulator.getBackend().hook_add_new(new CodeHook() {
            private int logs;

            @Override
            public void hook(Backend backend, long address, int size, Object user) {
                if (!signingActive || logs++ > 40) {
                    return;
                }
                long x1 = backend.reg_read(Arm64Const.UC_ARM64_REG_X1).longValue();
                long w2 = backend.reg_read(Arm64Const.UC_ARM64_REG_X2).longValue() & 0xffffffffL;
                long x3 = backend.reg_read(Arm64Const.UC_ARM64_REG_X3).longValue();
                long lr = backend.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                String fmt = readCString(backend, x1, 48);
                if (!fmt.contains("%d|%s") && !fmt.contains("%d|")) {
                    return;
                }
                long sp = backend.reg_read(Arm64Const.UC_ARM64_REG_SP).longValue();
                int[] regs = {
                        Arm64Const.UC_ARM64_REG_X4, Arm64Const.UC_ARM64_REG_X5,
                        Arm64Const.UC_ARM64_REG_X6, Arm64Const.UC_ARM64_REG_X7,
                        Arm64Const.UC_ARM64_REG_X19, Arm64Const.UC_ARM64_REG_X20,
                        Arm64Const.UC_ARM64_REG_X21, Arm64Const.UC_ARM64_REG_X22,
                        Arm64Const.UC_ARM64_REG_X23, Arm64Const.UC_ARM64_REG_X24,
                        Arm64Const.UC_ARM64_REG_X25, Arm64Const.UC_ARM64_REG_X26,
                        Arm64Const.UC_ARM64_REG_X27, Arm64Const.UC_ARM64_REG_X28
                };
                String[] names = {"x4","x5","x6","x7","x19","x20","x21","x22","x23","x24","x25","x26","x27","x28"};
                StringBuilder sb = new StringBuilder();
                sb.append("[fmt] w2=").append((int) w2)
                        .append(" fmt=").append(fmt)
                        .append(" x3=").append(readCString(backend, x3, 80))
                        .append(" lr=").append(loc(lr));
                for (int i = 0; i < regs.length; i++) {
                    long v = backend.reg_read(regs[i]).longValue();
                    sb.append(' ').append(names[i]).append('=').append(loc(v));
                }
                System.out.println(sb);
                // VM register file is usually x4. Dump slots that look like small integers.
                long file = backend.reg_read(Arm64Const.UC_ARM64_REG_X4).longValue();
                if (file > 0x10000) {
                    StringBuilder slots = new StringBuilder("[fmt] regfile");
                    for (int i = 0; i < 40; i++) {
                        try {
                            long v = readLong(backend, file + i * 8L);
                            if (v == 0) continue;
                            slots.append(" [").append(i).append("]=");
                            if ((v >>> 32) == 0 || (v >>> 32) == 0xffffffffL) {
                                slots.append((int) v);
                            } else {
                                slots.append(loc(v));
                            }
                        } catch (Exception e) {
                            break;
                        }
                    }
                    System.out.println(slots);
                }
                long x28 = backend.reg_read(Arm64Const.UC_ARM64_REG_X28).longValue();
                if (x28 > 0x10000) {
                    StringBuilder words = new StringBuilder("[fmt] x28 words");
                    for (int i = 0; i < 8; i++) {
                        try {
                            words.append(" ").append(Long.toHexString(readLong(backend, x28 + i * 8L)));
                        } catch (Exception e) {
                            words.append(" ?");
                            break;
                        }
                    }
                    System.out.println(words);
                }
            }

            @Override
            public void onAttach(UnHook unHook) {
            }

            @Override
            public void detach() {
            }
        }, addr, addr, null);
    }

    private static String readCString(Backend backend, long addr, int max) {
        if (addr < 0x1000) {
            return "0x" + Long.toHexString(addr);
        }
        try {
            byte[] raw = backend.mem_read(addr, max);
            int n = 0;
            while (n < raw.length && raw[n] != 0) {
                n++;
            }
            String s = new String(raw, 0, n, java.nio.charset.StandardCharsets.ISO_8859_1);
            return s.length() == max ? s + "..." : s;
        } catch (Exception e) {
            return "unmapped";
        }
    }

    private void hookSite(long offset, String name) {
        long addr = module.base + offset;
        emulator.getBackend().hook_add_new(new CodeHook() {
            @Override
            public void hook(Backend backend, long address, int size, Object user) {
                if (traceLogs++ > 80) {
                    return;
                }
                long x0 = backend.reg_read(Arm64Const.UC_ARM64_REG_X0).longValue();
                long x19 = backend.reg_read(Arm64Const.UC_ARM64_REG_X19).longValue();
                long x20 = backend.reg_read(Arm64Const.UC_ARM64_REG_X20).longValue();
                long lr = backend.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                String extra = "";
                if ("store+90".equals(name) && x20 > 0x1000) {
                    try {
                        long vt = readLong(backend, x20);
                        long f28 = readLong(backend, x20 + 0x28);
                        long f48 = readLong(backend, x20 + 0x48);
                        long f58 = readLong(backend, x20 + 0x58);
                        long parent0 = x0 > 0x1000 ? readLong(backend, x0) : 0;
                        extra = " vt=" + loc(vt)
                                + " +28=0x" + Long.toHexString(f28)
                                + " +48=0x" + Long.toHexString(f48)
                                + " +58=" + loc(f58)
                                + " parent[0]=" + loc(parent0);
                        // Frida's signing handle is this object: [0] is the
                        // vtable and [0x48] is the mutex wrapper. The JNI
                        // callback registers an outer pointer whose [0x48] is
                        // still null, and signing then locks address 8.
                        if (vt == module.base + 0x3C4F08L && f48 != 0) {
                            sdkHandle = x20;
                            extra += " adopted";
                        }
                    } catch (Exception e) {
                        extra = " dump-err " + e.getClass().getSimpleName();
                    }
                }
                if ("sign-lock".equals(name) && x0 > 0x1000) {
                    try {
                        extra = " [0]=" + loc(readLong(backend, x0))
                                + " [0x48]=0x" + Long.toHexString(readLong(backend, x0 + 0x48));
                    } catch (Exception e) {
                        extra = " dump-err";
                    }
                }
                System.out.println("[trace] " + name
                        + " x0=0x" + Long.toHexString(x0)
                        + " x19=0x" + Long.toHexString(x19)
                        + " x20=0x" + Long.toHexString(x20)
                        + " lr=" + loc(lr) + extra);
            }

            @Override
            public void onAttach(UnHook unHook) {
            }

            @Override
            public void detach() {
            }
        }, addr, addr, null);
    }

    private static long readLong(Backend backend, long addr) {
        byte[] raw = backend.mem_read(addr, 8);
        long v = 0;
        for (int i = 0; i < 8; i++) {
            v |= (raw[i] & 0xffL) << (8 * i);
        }
        return v;
    }

    private void configureSyscallHandler() {
        // Get the custom syscall handler and configure it with our module info
        if (emulator instanceof CustomEmulator) {
            CustomSyscallHandler handler = ((CustomEmulator) emulator).getCustomSyscallHandler();
            if (handler != null) {
                handler.setModuleBase(module.base);
                handler.setVtableOffset(0x3C4F08);
                handler.setFunctionPointerOffset(0x17a9d4);
                System.out.println("[openhongguo-signer] CustomSyscallHandler configured with base=0x"
                        + Long.toHexString(module.base));
            }
        }
    }

    public void waitForInit() {
        System.out.println("[openhongguo-signer] checking handle state...");
        dumpHandle();
        // initSdk's main task returns before newly cloned workers are merged
        // into the runnable list. Pull them in and let them finish writing
        // the vtable / handle.
        // One worker is in pthread_cond_timedwait for ~5.5s. Give that
        // timeout time to fire and wake the other two.
        pumpWorkerThreads(Long.getLong("emu.pump", 8_000));
        printSlot("ctx-after-pump", 0x460c40L);
        printSlot("singleton-after-pump", 0x460c48L);
        dumpHandle();
        dumpVtable028();

        if (!isHandle0Set()) {
            // The slot is filled by a pool thread. Pointing handle[0] at the
            // vtable makes signing cond_wait on a function pointer.
            System.out.println("[openhongguo-signer] handle[0] not set; leaving it for the worker");
        }

        dumpHandle();
    }

    /**
     * Run cloned workers that unidbg left in the pending list when the main
     * JNI call returned without yielding.
     */
    @SuppressWarnings("unchecked")
    private void pumpWorkerThreads(long timeoutMs) {
        ThreadDispatcher dispatcher = emulator.getThreadDispatcher();
        try {
            Field pendingF = dispatcher.getClass().getDeclaredField("threadTaskList");
            Field activeF = dispatcher.getClass().getDeclaredField("taskList");
            pendingF.setAccessible(true);
            activeF.setAccessible(true);
            List<Object> pending = (List<Object>) pendingF.get(dispatcher);
            List<Object> active = (List<Object>) activeF.get(dispatcher);
            System.out.println("[openhongguo-signer] before pump active=" + active.size()
                    + " pending=" + pending.size());
            for (Object t : active) {
                System.out.println("[openhongguo-signer]   active " + t);
            }
            for (Object t : pending) {
                System.out.println("[openhongguo-signer]   pending " + t);
            }
            if (!pending.isEmpty()) {
                active.addAll(0, new ArrayList<>(pending));
                pending.clear();
            }
            active.removeIf(t -> {
                try {
                    return Boolean.TRUE.equals(t.getClass().getMethod("isFinish").invoke(t));
                } catch (Exception e) {
                    return false;
                }
            });
            if (active.isEmpty()) {
                System.out.println("[openhongguo-signer] no worker threads to pump");
                return;
            }
            dumpPausedWorkers(active);
            Thread stopper = new Thread(() -> {
                try {
                    Thread.sleep(timeoutMs);
                    System.out.println("[openhongguo-signer] pump timeout, emu_stop");
                    emulator.getBackend().emu_stop();
                } catch (InterruptedException ignored) {
                }
            });
            stopper.setDaemon(true);
            stopper.start();
            try {
                dispatcher.runThreads(timeoutMs, TimeUnit.MILLISECONDS);
            } finally {
                stopper.interrupt();
            }
            System.out.println("[openhongguo-signer] after pump active=" + active.size());
            for (Object t : active) {
                System.out.println("[openhongguo-signer]   active " + t);
            }
            dumpPausedWorkers(active);
        } catch (Exception e) {
            System.out.println("[openhongguo-signer] pumpWorkerThreads: " + e);
            e.printStackTrace(System.out);
        }
    }

    /** Where a parked worker is, and which futex word it is waiting on. */
    private void dumpPausedWorkers(List<Object> active) {
        Backend backend = emulator.getBackend();
        long saved = backend.context_alloc();
        backend.context_save(saved);
        try {
            for (Object t : active) {
                long ctx = 0;
                for (Class<?> c = t.getClass(); c != null; c = c.getSuperclass()) {
                    try {
                        Field f = c.getDeclaredField("context");
                        f.setAccessible(true);
                        ctx = f.getLong(t);
                        break;
                    } catch (NoSuchFieldException ignored) {
                    }
                }
                String at = "no-context";
                if (ctx != 0) {
                    backend.context_restore(ctx);
                    long pc = backend.reg_read(Arm64Const.UC_ARM64_REG_PC).longValue();
                    long lr = backend.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                    long fp = backend.reg_read(Arm64Const.UC_ARM64_REG_FP).longValue();
                    StringBuilder frames = new StringBuilder();
                    long cur = fp;
                    for (int frame = 0; frame < 8 && cur > 0x1000; frame++) {
                        byte[] raw;
                        try {
                            raw = backend.mem_read(cur, 16);
                        } catch (Exception e) {
                            break;
                        }
                        long savedFp = 0;
                        long savedLr = 0;
                        for (int i = 0; i < 8; i++) savedFp |= (raw[i] & 0xffL) << (8 * i);
                        for (int i = 0; i < 8; i++) savedLr |= (raw[8 + i] & 0xffL) << (8 * i);
                        frames.append(" #").append(frame).append("=").append(loc(savedLr));
                        if (savedFp == 0 || savedFp == cur) break;
                        cur = savedFp;
                    }
                    at = "pc=" + loc(pc) + " lr=" + loc(lr) + frames;
                }
                String wait = "";
                try {
                    Object waiter = t.getClass().getMethod("getWaiter").invoke(t);
                    if (waiter != null) {
                        StringBuilder sb = new StringBuilder(" waiter=").append(waiter.getClass().getSimpleName());
                        for (String name : new String[]{"uaddr", "val", "waitMillis"}) {
                            try {
                                Field f = null;
                                for (Class<?> c = waiter.getClass(); c != null && f == null; c = c.getSuperclass()) {
                                    try {
                                        f = c.getDeclaredField(name);
                                    } catch (NoSuchFieldException ignored2) {
                                    }
                                }
                                if (f == null) continue;
                                f.setAccessible(true);
                                Object v = f.get(waiter);
                                sb.append(" ").append(name).append("=");
                                if (v instanceof com.sun.jna.Pointer) {
                                    sb.append(v);
                                } else if (v instanceof Integer) {
                                    sb.append("0x").append(Integer.toHexString((Integer) v));
                                } else {
                                    sb.append(v);
                                }
                            } catch (ReflectiveOperationException ignored) {
                            }
                        }
                        wait = sb.toString();
                    }
                } catch (Exception e) {
                    wait = " waiter-err=" + e.getClass().getSimpleName();
                }
                System.out.println("[openhongguo-signer]   parked " + at + wait);
            }
        } catch (Exception e) {
            System.out.println("[openhongguo-signer] dumpPausedWorkers: " + e);
        } finally {
            backend.context_restore(saved);
            backend.context_free(saved);
        }
    }

    private String loc(long addr) {
        Module m = emulator.getMemory().findModuleByAddress(addr);
        if (m == null) return "0x" + Long.toHexString(addr);
        return m.name + "+0x" + Long.toHexString(addr - m.base);
    }

    private void dumpVtable028() {
        if (sdkHandle == 0) return;
        try {
            UnidbgPointer hp = UnidbgPointer.pointer(emulator, sdkHandle);
            if (hp == null) return;
            long vt = hp.getLong(0);
            if (vt == 0) {
                System.out.println("[openhongguo-signer] vtable[0x28] skipped, handle[0]=0");
                return;
            }
            UnidbgPointer slot = UnidbgPointer.pointer(emulator, vt + 0x28);
            if (slot == null) return;
            long v = slot.getLong(0);
            Module mod = emulator.getMemory().findModuleByAddress(v);
            String desc = mod != null ? " (" + mod.name + "+0x" + Long.toHexString(v - mod.base) + ")" : "";
            System.out.println("[openhongguo-signer] vtable[0x28] @0x" + Long.toHexString(vt + 0x28)
                    + " = 0x" + Long.toHexString(v) + desc);
        } catch (Exception e) {
            System.out.println("[openhongguo-signer] vtable dump error: " + e.getMessage());
        }
    }

    private void dumpHandle() {
        if (sdkHandle == 0) return;
        try {
            UnidbgPointer hp = UnidbgPointer.pointer(emulator, sdkHandle);
            if (hp != null) {
                byte[] mem = hp.getByteArray(0, 128);
                StringBuilder sb = new StringBuilder("[openhongguo-signer] handle dump: ");
                for (int i = 0; i < mem.length; i++) {
                    if (i > 0 && i % 8 == 0) sb.append(" ");
                    sb.append(String.format("%02x", mem[i] & 0xff));
                }
                System.out.println(sb);

                // Also dump as 64-bit values
                for (int off = 0; off < 0xc0; off += 8) {
                    long val = hp.getLong(off);
                    if (val != 0) {
                        Module mod = emulator.getMemory().findModuleByAddress(val);
                        String desc = "";
                        if (mod != null) desc = " (" + mod.name + "+0x" + Long.toHexString(val - mod.base) + ")";
                        System.out.println("[openhongguo-signer]   handle[0x" + Integer.toHexString(off) + "] = 0x"
                                + Long.toHexString(val) + desc);
                        if (off == 0x90 || off == 0x00) {
                            dumpPointed(val, "handle[0x" + Integer.toHexString(off) + "]");
                        }
                    }
                }
            }
        } catch (Exception e) {
            System.out.println("[openhongguo-signer] handle dump error: " + e.getMessage());
        }
    }

    /** Pull header strings out of the map or array k3.a returns. */
    private void harvest(Object val, Map<String, String> headers) {
        System.out.println("[openhongguo-signer] result " + (val == null ? "null" : val.getClass().getSimpleName()));
        harvestValue(unwrap(val), headers, 0);
        for (Map.Entry<String, String> e : headers.entrySet()) {
            String v = e.getValue();
            String shown = v.length() > 80 ? v.substring(0, 80) + "...(" + v.length() + ")" : v;
            System.out.println("[openhongguo-signer]   " + e.getKey() + ": " + shown);
        }
    }

    private void harvestValue(Object val, Map<String, String> headers, int depth) {
        if (val == null || depth > 6) return;
        if (val instanceof Map) {
            for (Map.Entry<?, ?> e : ((Map<?, ?>) val).entrySet()) {
                Object uv = unwrap(e.getValue());
                headers.put(String.valueOf(unwrap(e.getKey())), uv == null ? "" : String.valueOf(uv));
            }
            return;
        }
        if (val instanceof Object[]) {
            Object[] arr = (Object[]) val;
            System.out.println("[openhongguo-signer] array depth=" + depth + " len=" + arr.length);
            List<String> strings = new ArrayList<>();
            boolean allStrings = arr.length > 0;
            for (int i = 0; i < arr.length; i++) {
                Object u = unwrap(arr[i]);
                String kind = u == null ? "null" : u.getClass().getSimpleName();
                String shown;
                if (u instanceof byte[]) {
                    shown = "bytes[" + ((byte[]) u).length + "]";
                } else if (u instanceof String) {
                    String s = (String) u;
                    shown = s.length() > 120 ? s.substring(0, 120) + "...(" + s.length() + ")" : s;
                } else {
                    shown = String.valueOf(u);
                }
                System.out.println("[openhongguo-signer]   [" + i + "] " + kind + " " + shown);
                if (u instanceof String) {
                    strings.add((String) u);
                } else {
                    allStrings = false;
                    harvestValue(u, headers, depth + 1);
                }
            }
            if (allStrings && strings.size() % 2 == 0) {
                for (int i = 0; i < strings.size(); i += 2) {
                    headers.put(strings.get(i), strings.get(i + 1));
                }
            }
            return;
        }
        if (!(val instanceof String) && !(val instanceof byte[])) {
            System.out.println("[openhongguo-signer] value " + val.getClass().getName() + " = " + val);
        }
    }

    private static Object unwrap(Object o) {
        while (o instanceof DvmObject) {
            o = ((DvmObject<?>) o).getValue();
        }
        return o;
    }

    private String describeHandle(long addr) {
        if (addr == 0) return "";
        try {
            UnidbgPointer p = UnidbgPointer.pointer(emulator, addr);
            if (p == null) return "";
            return " [0]=" + loc(p.getLong(0)) + " [0x48]=0x" + Long.toHexString(p.getLong(0x48));
        } catch (Exception e) {
            return " unreadable";
        }
    }

    private void dumpPointed(long addr, String label) {
        try {
            UnidbgPointer p = UnidbgPointer.pointer(emulator, addr);
            if (p == null) return;
            System.out.println("[openhongguo-signer]   " + label + " ->");
            for (int off = 0; off < 0x40; off += 8) {
                long val = p.getLong(off);
                if (val == 0) continue;
                Module mod = emulator.getMemory().findModuleByAddress(val);
                String desc = mod != null ? " (" + mod.name + "+0x" + Long.toHexString(val - mod.base) + ")" : "";
                System.out.println("[openhongguo-signer]     [0x" + Integer.toHexString(off) + "] = 0x"
                        + Long.toHexString(val) + desc);
            }
        } catch (Exception e) {
            System.out.println("[openhongguo-signer]   " + label + " follow failed: " + e.getMessage());
        }
    }

    private boolean isHandle0Set() {
        if (sdkHandle == 0) return false;
        UnidbgPointer hp = UnidbgPointer.pointer(emulator, sdkHandle);
        if (hp == null) return false;
        long val = hp.getLong(0);
        return val != 0;
    }

    private void ensureHandle0() {
        if (sdkHandle == 0) return;
        UnidbgPointer handlePtr = UnidbgPointer.pointer(emulator, sdkHandle);
        if (handlePtr == null) return;

        long existing = handlePtr.getLong(0);
        if (existing != 0) {
            System.out.println("[openhongguo-signer] handle[0] already set: 0x" + Long.toHexString(existing));
            return;
        }

        long base = module.base;
        long subAddr = base + 0x3C4F08;
        UnidbgPointer sub = UnidbgPointer.pointer(emulator, subAddr);
        if (sub == null) return;

        // .data.rel.ro already holds R_AARCH64_RELATIVE slots, including
        // [0x28] -> so+0x17a9d4. Do not zero them. Only fill holes the
        // loader left empty.
        long[][] ptrs = {
            {0x00, 0x172e90}, {0x08, 0x173058},
            {0x20, 0x17a9c4}, {0x28, 0x17a9d4},
            {0x40, 0x17aaf0}, {0x48, 0x17ab4c},
            {0x60, 0x17457c}, {0x68, 0x17ad44},
            {0x80, 0x17af20}, {0x88, 0x17af24},
            {0xa0, 0x17bd50}, {0xa8, 0x17bdc0},
            {0xc0, 0x17bde4}, {0xc8, 0x17be7c},
            {0xe0, 0x17cb58}, {0xe8, 0x17c8f0}, {0xf0, 0x17c8f4}, {0xf8, 0x17c918},
            {0x100, 0x17c928}, {0x108, 0x17c92c}, {0x110, 0x17c930},
            {0x128, 0x17c988}, {0x130, 0x17c998}, {0x138, 0x17c9bc},
            {0x140, 0x17c9f8}, {0x148, 0x17ca14}, {0x150, 0x17ca1c}, {0x158, 0x17ca44},
            {0x170, 0x17cb58}, {0x178, 0x17cadc}, {0x180, 0x17cae0},
            {0x188, 0x17cb14}, {0x190, 0x17cb28}, {0x198, 0x17cb2c}, {0x1a0, 0x17cb30},
            {0x1b8, 0x17cb58}, {0x1c0, 0x17cb5c}, {0x1c8, 0x17cb60},
            {0x1d0, 0x17cb84}, {0x1d8, 0x17cb94}, {0x1e0, 0x17cb98}, {0x1e8, 0x17cb9c},
        };
        int filled = 0;
        for (long[] p : ptrs) {
            if (sub.getLong((int) p[0]) == 0) {
                sub.setLong((int) p[0], base + p[1]);
                filled++;
            }
        }

        handlePtr.setLong(0x00, subAddr);
        if (handlePtr.getLong(0x28) == 0) handlePtr.setLong(0x28, 0x0101010101L);
        if (handlePtr.getLong(0x48) == 0) handlePtr.setLong(0x48, subAddr + 0x20);
        if (handlePtr.getLong(0x58) == 0) handlePtr.setLong(0x58, base + 0x38c9f0);

        System.out.println("[openhongguo-signer] handle[0] = vtable at 0x" + Long.toHexString(subAddr)
                + " (filled " + filled + " empty slots, soBase=0x" + Long.toHexString(base) + ")");
        System.out.println("[openhongguo-signer]   vtable[0x28] = 0x" + Long.toHexString(sub.getLong(0x28)));
    }

    public void initSdk(String nativeValue) {
        System.out.println("[openhongguo-signer] initSdk with config length=" + nativeValue.length());
        StringObject nv = new StringObject(vm, nativeValue);
        if (System.getenv("EMU_NO_CTX") == null) {
            // i3.b: k3.a(0x1000003, 0, 0, null, applicationContext) runs once
            // before the init call.
            try {
                DvmObject<?> app = vm.resolveClass("android/app/Application").newObject(null);
                DvmObject<?> r = k3Class.callStaticJniMethodObject(emulator,
                        "a(IIJLjava/lang/String;Ljava/lang/Object;)Ljava/lang/Object;",
                        0x1000003, 0, 0L, null, app);
                System.out.println("[openhongguo-signer] op 0x1000003 -> " + (r == null ? null : r.getValue()));
            } catch (RuntimeException e) {
                System.out.println("[openhongguo-signer] op 0x1000003 failed: " + e);
            }
        }
        DvmObject<?> result;
        try {
            result = k3Class.callStaticJniMethodObject(emulator,
                    "a(IIJLjava/lang/String;Ljava/lang/Object;)Ljava/lang/Object;",
                    0x4000001, 0, 0L, nv, null);
        } catch (RuntimeException e) {
            System.out.println("[openhongguo-signer] initSdk aborted: " + e);
            result = null;
        }

        if (result != null) {
            Object val = result.getValue();
            if (val instanceof Number) {
                sdkHandle = ((Number) val).longValue();
            } else {
                System.out.println("[openhongguo-signer] init returned " + val.getClass().getSimpleName()
                        + " value=" + val
                        + " (handle from callback: 0x" + Long.toHexString(sdkHandle) + ")");
            }
        } else {
            System.out.println("[openhongguo-signer] init returned null (handle from callback: 0x"
                    + Long.toHexString(sdkHandle) + ")");
        }
        // i3.a: the f3 signer's handle is k3.a(0x4000002, 0, 0, null, null),
        // not the object the 0x2000001 callback registers.
        try {
            DvmObject<?> h = k3Class.callStaticJniMethodObject(emulator,
                    "a(IIJLjava/lang/String;Ljava/lang/Object;)Ljava/lang/Object;",
                    0x4000002, 0, 0L, null, null);
            Object hv = h == null ? null : h.getValue();
            System.out.println("[openhongguo-signer] op 0x4000002 -> " + hv
                    + (hv instanceof Number ? " (0x" + Long.toHexString(((Number) hv).longValue()) + ")" : ""));
            if (hv instanceof Number && ((Number) hv).longValue() != 0) {
                sdkHandle = ((Number) hv).longValue();
            }
        } catch (RuntimeException e) {
            System.out.println("[openhongguo-signer] op 0x4000002 failed: " + e);
        }
        System.out.println("[openhongguo-signer] SDK handle = 0x" + Long.toHexString(sdkHandle));
        printSlot("ctx", 0x460c40L);
        printSlot("singleton", 0x460c48L);
    }

    private void printSlot(String name, long off) {
        UnidbgPointer p = UnidbgPointer.pointer(emulator, module.base + off);
        long v = p == null ? 0 : p.getLong(0);
        System.out.println("[openhongguo-signer] " + name + " so+" + Long.toHexString(off) + " = " + loc(v));
    }

    public Map<String, String> getFeatureHash(String url, byte[] body) {
        System.out.println("[openhongguo-signer] getFeatureHash url=" + url
                + " body=" + (body != null ? body.length + "B" : "null")
                + " handle=0x" + Long.toHexString(sdkHandle));

        Backend backend = emulator.getBackend();
        signingActive = true;
        futexCallCount.set(0);
        futexAddrCounts.clear();

        // Notify the syscall handler that signing is active
        if (emulator instanceof CustomEmulator) {
            CustomSyscallHandler handler = ((CustomEmulator) emulator).getCustomSyscallHandler();
            if (handler != null) {
                handler.startSigning();
            }
        }

        ExecutorService exec = Executors.newSingleThreadExecutor();
        Future<Map<String, String>> future = exec.submit(() -> {
            StringObject urlObj = new StringObject(vm, url);
            ByteArray bodyObj = new ByteArray(vm, body != null ? body : new byte[0]);

            Thread watchdog = new Thread(() -> {
                try {
                    Thread.sleep(15_000);
                    System.out.println("[openhongguo-signer] getFeatureHash watchdog: stopping after 15s");
                    if (emulator instanceof CustomEmulator) {
                        CustomSyscallHandler h = ((CustomEmulator) emulator).getCustomSyscallHandler();
                        if (h != null) h.printStats();
                    }
                    backend.emu_stop();
                    Thread.sleep(200);
                    try {
                        long pc = backend.reg_read(Arm64Const.UC_ARM64_REG_PC).longValue();
                        long lr = backend.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                        com.github.unidbg.Module pcMod = emulator.getMemory().findModuleByAddress(pc);
                        String pcName = pcMod != null ? pcMod.name + "+0x" + Long.toHexString(pc - pcMod.base) : "0x" + Long.toHexString(pc);
                        com.github.unidbg.Module lrMod = emulator.getMemory().findModuleByAddress(lr);
                        String lrName = lrMod != null ? lrMod.name + "+0x" + Long.toHexString(lr - lrMod.base) : "0x" + Long.toHexString(lr);
                        System.out.println("[openhongguo-signer] STOPPED pc=" + pcName + " lr=" + lrName);
                    } catch (Exception ignored2) {}
                } catch (InterruptedException ignored) {}
            });
            watchdog.setDaemon(true);
            watchdog.start();

            DvmObject<?> result;
            try {
                if (System.getenv("EMU_FINISH_READY") != null) finishReady();
                syncIdentity(url);
                result = k3Class.callStaticJniMethodObject(emulator,
                        "a(IIJLjava/lang/String;Ljava/lang/Object;)Ljava/lang/Object;",
                        0x2000006, 0, sdkHandle, urlObj, bodyObj);
            } catch (Exception e) {
                System.out.println("[openhongguo-signer] getFeatureHash exception: " + e.getMessage());
                e.printStackTrace();
                return new HashMap<String, String>();
            } finally {
                watchdog.interrupt();
                signingActive = false;
                if (emulator instanceof CustomEmulator) {
                    CustomSyscallHandler h = ((CustomEmulator) emulator).getCustomSyscallHandler();
                    if (h != null) h.stopSigning();
                }
            }

            Map<String, String> headers = new HashMap<>();
            if (result == null) {
                System.out.println("[openhongguo-signer] getFeatureHash returned null");
                return headers;
            }
            harvest(result, headers);
            return headers;
        });

        try {
            return future.get(20, TimeUnit.SECONDS);
        } catch (TimeoutException e) {
            System.out.println("[openhongguo-signer] getFeatureHash timed out");
            backend.emu_stop();
            future.cancel(true);
            return new HashMap<>();
        } catch (Exception e) {
            System.out.println("[openhongguo-signer] getFeatureHash error: " + e);
            return new HashMap<>();
        } finally {
            exec.shutdownNow();
        }
    }

    private final Map<String, String> storageMap = new HashMap<>();
    private String appVersion = "7.4.3.32";
    private int cbLogs;
    private int initCbLogs;

    /**
     * The constructor leaves [obj+0x28] = 0 and four later stores set
     * bytes +0x29..+0x2c. so+0x17467c is the only writer of the low byte,
     * and it does the rest of init before storing 1. getFeatureHash checks
     * that byte and otherwise returns X-Neptune "-8|<failed ids>".
     */
    private void finishReady() {
        if (sdkHandle == 0) {
            return;
        }
        UnidbgPointer obj = UnidbgPointer.pointer(emulator, sdkHandle);
        if (obj == null) {
            return;
        }
        System.out.println("[openhongguo-signer] ready-flag before=0x" + Long.toHexString(obj.getLong(0x28)));
        // Do not publish sdkHandle into so+0x460c48. That slot is read by
        // so+0x1906ac; the inner signing object is the wrong type and faults
        // this call at so+0x298ab8. A null slot still lets the ready byte land.
        try {
            Number ret = module.callFunction(emulator, 0x17467cL, sdkHandle);
            System.out.println("[openhongguo-signer] finishReady ret=" + ret
                    + " flag=0x" + Long.toHexString(obj.getLong(0x28)));
        } catch (Exception e) {
            System.out.println("[openhongguo-signer] finishReady exception: " + e);
            e.printStackTrace(System.out);
        }
    }

    private static String systemProperty(String key) {
        if (key == null) return null;
        switch (key) {
            case "http.agent": return "Dalvik/2.1.0 (Linux; U; Android 12; PGT-AN10 Build/HONORPGT-AN10)";
            case "java.vm.version": return "2.1.0";
            case "java.vm.name": return "Dalvik";
            case "os.arch": return "aarch64";
            case "os.name": return "Linux";
            case "os.version": return "5.10.110-android12-9-00001-g0c6b0b4c4e3f";
            case "line.separator": return "\n";
            case "file.separator": return "/";
            case "path.separator": return ":";
            case "user.home": return "";
            case "java.io.tmpdir": return "/data/user/0/com.phoenix.read/cache";
            default: return null;
        }
    }

    private String lastDeviceId, lastInstallId;

    private static String queryParam(String url, String key) {
        int q = url.indexOf('?');
        if (q < 0) return null;
        for (String kv : url.substring(q + 1).split("&")) {
            int e = kv.indexOf('=');
            if (e > 0 && kv.substring(0, e).equals(key)) return kv.substring(e + 1);
        }
        return null;
    }

    /** f3.setDeviceID / setInstallID: ops 0x2000002 and 0x2000003 on the signing handle. */
    private void syncIdentity(String url) {
        String did = queryParam(url, "device_id");
        String iid = queryParam(url, "iid");
        DvmObject<?> app = vm.resolveClass("android/app/Application").newObject(null);
        String sig = "a(IIJLjava/lang/String;Ljava/lang/Object;)Ljava/lang/Object;";
        if (did != null && !did.equals(lastDeviceId)) {
            k3Class.callStaticJniMethodObject(emulator, sig, 0x2000002, 0, sdkHandle, new StringObject(vm, did), app);
            lastDeviceId = did;
            System.out.println("[openhongguo-signer] setDeviceID " + did);
        }
        if (iid != null && !iid.equals(lastInstallId)) {
            k3Class.callStaticJniMethodObject(emulator, sig, 0x2000003, 0, sdkHandle, new StringObject(vm, iid), app);
            lastInstallId = iid;
            System.out.println("[openhongguo-signer] setInstallID " + iid);
        }
    }

    private void logCb(String msg) {
        if (!signingActive || cbLogs++ > 40) {
            return;
        }
        System.out.println("[cb] " + msg);
    }

    private DvmObject<?> handleMSbCallback(int opcode, VaList vaList) {
        long handle = vaList.getLongArg(2);
        if (!signingActive && initCbLogs++ < 40) {
            System.out.println("[init-cb] op=0x" + Integer.toHexString(opcode)
                    + " handle=0x" + Long.toHexString(handle));
        }
        if (signingActive) {
            String str = null;
            try {
                DvmObject<?> s = vaList.getObjectArg(3);
                if (s != null && s.getValue() != null) {
                    str = String.valueOf(s.getValue());
                    if (str.length() > 80) {
                        str = str.substring(0, 80) + "...";
                    }
                }
            } catch (Exception ignored) {
            }
            logCb("op=0x" + Integer.toHexString(opcode) + " str=" + str);
        }
        switch (opcode) {
            case 0x1000000e:
                return vm.resolveClass("java/lang/Long").newObject(2L);
            case 0x10003:
                return new StringObject(vm, "/data/user/0/" + packageName + "/files/.msdata");
            case 0x10001:
                return new StringObject(vm, appVersion);
            case 0x10002:
                return new StringObject(vm, packageName);
            case 0x1000001:
                return new StringObject(vm, "/data/app/" + packageName + "/base.apk");
            case 0x1000022: {
                DvmObject<?> objArg = vaList.getObjectArg(4);
                String key = objArg != null ? String.valueOf(objArg.getValue()) : null;
                if (key != null) {
                    String stored = storageMap.get(key);
                    if (stored != null) return new StringObject(vm, stored);
                }
                return null;
            }
            case 0x30004:
                // so+0x299fa8 calls Boolean.booleanValue(). AbstractJni
                // casts the receiver to DvmBoolean; a plain DvmObject throws.
                return DvmBoolean.valueOf(vm, true);
            case 0x2000001:
                System.out.println("[openhongguo-signer] register handle=0x" + Long.toHexString(handle)
                        + describeHandle(handle));
                if (handle != 0 && sdkHandle == 0) {
                    sdkHandle = handle;
                }
                return vm.resolveClass("java/lang/Boolean").newObject(true);
            case 0x2000002:
                return vm.resolveClass("java/lang/Boolean").newObject(true);
            case 0x10008:
                // so+0x2bcb7c stores this as update_version_code. A null
                // return is replaced with the sentinel "!noperm!".
                return new StringObject(vm, "74332");
            case 0x20001:
                // so+0x2bcf14. so+0x2bc844 returns true only when this is
                // non-null, and the caller posts the follow-up pool task
                // only then. The value itself is released, not read.
                return new StringObject(vm, "1");
            case 0x1000041:
            case 0x30005:
            case 0x200000d:
            case 0x200000e:
            case 0x200000f:
            case 0x9000001:
                return null;
            default:
                System.out.println("[openhongguo-signer] UNHANDLED opcode=0x" + Integer.toHexString(opcode));
                return null;
        }
    }

    // ============ JNI callback handlers ============

    @Override
    public DvmObject<?> callStaticObjectMethodV(BaseVM vm, DvmClass dvmClass, String signature, VaList vaList) {
        switch (signature) {
            case "com/bytedance/mobsec/metasec/ml/MS->b(IIJLjava/lang/String;Ljava/lang/Object;)Ljava/lang/Object;":
            case "ms/bd/c/k3->b(IIJLjava/lang/String;Ljava/lang/Object;)Ljava/lang/Object;":
            case "ms/bd/c/q2->b(IIJLjava/lang/String;Ljava/lang/Object;)Ljava/lang/Object;": {
                int opcode = vaList.getIntArg(0);
                return handleMSbCallback(opcode, vaList);
            }
            case "java/lang/Boolean->valueOf(Z)Ljava/lang/Boolean;":
                return vm.resolveClass("java/lang/Boolean").newObject(vaList.getIntArg(0) != 0);
            case "java/lang/Integer->valueOf(I)Ljava/lang/Integer;":
                return vm.resolveClass("java/lang/Integer").newObject(vaList.getIntArg(0));
            case "java/lang/Long->valueOf(J)Ljava/lang/Long;":
                return vm.resolveClass("java/lang/Long").newObject(vaList.getLongArg(0));
            case "java/lang/System->getProperty(Ljava/lang/String;)Ljava/lang/String;": {
                DvmObject<?> k = vaList.getObjectArg(0);
                String key = k == null ? null : String.valueOf(k.getValue());
                String v = systemProperty(key);
                System.out.println("[jni] System.getProperty(" + key + ") -> " + v);
                return v == null ? null : new StringObject(vm, v);
            }
            default:
                logCb("static " + signature);
                try {
                    return super.callStaticObjectMethodV(vm, dvmClass, signature, vaList);
                } catch (UnsupportedOperationException e) {
                    System.out.println("[jni-miss] static " + signature + " -> null");
                    return null;
                }
        }
    }

    @Override
    public DvmObject<?> callStaticObjectMethod(BaseVM vm, DvmClass dvmClass, String signature, VarArg varArg) {
        switch (signature) {
            case "android/app/ActivityThread->currentApplication()Landroid/app/Application;":
                return vm.resolveClass("android/app/Application").newObject(null);
            case "android/os/Environment->getExternalStorageDirectory()Ljava/io/File;":
                return vm.resolveClass("java/io/File").newObject(new File("/sdcard"));
            case "java/lang/Boolean->valueOf(Z)Ljava/lang/Boolean;":
                return vm.resolveClass("java/lang/Boolean").newObject(varArg.getIntArg(0) != 0);
            case "java/lang/Integer->valueOf(I)Ljava/lang/Integer;":
                return vm.resolveClass("java/lang/Integer").newObject(varArg.getIntArg(0));
            case "java/lang/Long->valueOf(J)Ljava/lang/Long;":
                return vm.resolveClass("java/lang/Long").newObject(varArg.getLongArg(0));
            case "com/bytedance/mobsec/metasec/ml/MS->b(IIJLjava/lang/String;Ljava/lang/Object;)Ljava/lang/Object;":
            case "ms/bd/c/k3->b(IIJLjava/lang/String;Ljava/lang/Object;)Ljava/lang/Object;": {
                int opcode = varArg.getIntArg(0);
                switch (opcode) {
                    case 0x1000000e:
                        return vm.resolveClass("java/lang/Long").newObject(2L);
                    case 0x10003:
                        return new StringObject(vm, "/data/user/0/" + packageName + "/files/.msdata");
                    case 0x2000001:
                    case 0x2000002:
                        return vm.resolveClass("java/lang/Boolean").newObject(true);
                    default:
                        return null;
                }
            }
            default:
                return super.callStaticObjectMethod(vm, dvmClass, signature, varArg);
        }
    }

    @Override
    public DvmObject<?> callObjectMethodV(BaseVM vm, DvmObject<?> dvmObject, String signature, VaList vaList) {
        switch (signature) {
            case "android/app/Application->getPackageName()Ljava/lang/String;":
            case "android/content/Context->getPackageName()Ljava/lang/String;":
                return new StringObject(vm, packageName);
            case "android/app/Application->getApplicationContext()Landroid/content/Context;":
            case "android/content/Context->getApplicationContext()Landroid/content/Context;":
                return dvmObject;
            default:
                logCb("obj " + signature);
                try {
                    return super.callObjectMethodV(vm, dvmObject, signature, vaList);
                } catch (UnsupportedOperationException e) {
                    System.out.println("[jni-miss] " + signature + " -> null");
                    return null;
                }
        }
    }

    @Override
    public DvmObject<?> callObjectMethod(BaseVM vm, DvmObject<?> dvmObject, String signature, VarArg varArg) {
        switch (signature) {
            case "android/app/Application->getApplicationContext()Landroid/content/Context;":
            case "android/content/Context->getApplicationContext()Landroid/content/Context;":
                return dvmObject;
            case "android/content/Context->getPackageName()Ljava/lang/String;":
                return new StringObject(vm, packageName);
            case "android/content/Context->getFilesDir()Ljava/io/File;":
                return vm.resolveClass("java/io/File")
                        .newObject(new File("/data/data/" + packageName + "/files"));
            case "android/content/Context->getCacheDir()Ljava/io/File;":
                return vm.resolveClass("java/io/File")
                        .newObject(new File("/data/data/" + packageName + "/cache"));
            case "android/content/Context->getContentResolver()Landroid/content/ContentResolver;":
                return vm.resolveClass("android/content/ContentResolver").newObject(null);
            case "android/content/Context->getPackageManager()Landroid/content/pm/PackageManager;":
                return vm.resolveClass("android/content/pm/PackageManager").newObject(null);
            case "android/content/Context->getSharedPreferences(Ljava/lang/String;I)Landroid/content/SharedPreferences;":
                return vm.resolveClass("android/content/SharedPreferences").newObject(new HashMap<>());
            case "android/content/SharedPreferences->getString(Ljava/lang/String;Ljava/lang/String;)Ljava/lang/String;":
                return new StringObject(vm, "");
            case "android/content/SharedPreferences->edit()Landroid/content/SharedPreferences$Editor;":
                return vm.resolveClass("android/content/SharedPreferences$Editor").newObject(null);
            case "android/content/SharedPreferences$Editor->putString(Ljava/lang/String;Ljava/lang/String;)Landroid/content/SharedPreferences$Editor;":
                return dvmObject;
            case "java/io/File->getAbsolutePath()Ljava/lang/String;": {
                File f = (File) dvmObject.getValue();
                return new StringObject(vm, f != null ? f.getAbsolutePath() : "/data/data/" + packageName);
            }
            case "java/util/HashMap->keySet()Ljava/util/Set;":
            case "java/util/Map->keySet()Ljava/util/Set;": {
                Map<?, ?> map = (Map<?, ?>) dvmObject.getValue();
                return vm.resolveClass("java/util/Set").newObject(
                        map != null ? map.keySet() : new java.util.HashSet<>());
            }
            case "android/content/pm/PackageManager->getPackageInfo(Ljava/lang/String;I)Landroid/content/pm/PackageInfo;":
                return vm.resolveClass("android/content/pm/PackageInfo").newObject(null);
            case "java/lang/String->getBytes(Ljava/lang/String;)[B": {
                String s = (String) dvmObject.getValue();
                return new ByteArray(vm, s != null ? s.getBytes(StandardCharsets.UTF_8) : new byte[0]);
            }
            case "java/util/HashMap->put(Ljava/lang/Object;Ljava/lang/Object;)Ljava/lang/Object;": {
                @SuppressWarnings("unchecked")
                Map<Object, Object> map = (Map<Object, Object>) dvmObject.getValue();
                DvmObject<?> key = varArg.getObjectArg(0);
                DvmObject<?> val = varArg.getObjectArg(1);
                if (map != null && key != null) {
                    map.put(key.getValue(), val != null ? val.getValue() : null);
                }
                return null;
            }
            case "java/util/HashMap->get(Ljava/lang/Object;)Ljava/lang/Object;": {
                Map<?, ?> map = (Map<?, ?>) dvmObject.getValue();
                DvmObject<?> key = varArg.getObjectArg(0);
                if (map != null && key != null) {
                    Object val = map.get(key.getValue());
                    if (val instanceof String) return new StringObject(vm, (String) val);
                    if (val != null) return vm.resolveClass("java/lang/Object").newObject(val);
                }
                return null;
            }
            default:
                return super.callObjectMethod(vm, dvmObject, signature, varArg);
        }
    }

    @Override
    public long callLongMethodV(BaseVM vm, DvmObject<?> dvmObject, String signature, VaList vaList) {
        if ("java/lang/Long->longValue()J".equals(signature)) {
            return (Long) dvmObject.getValue();
        }
        try {
            return super.callLongMethodV(vm, dvmObject, signature, vaList);
        } catch (UnsupportedOperationException e) {
            System.out.println("[jni-miss] " + signature + " -> 0");
            return 0;
        }
    }

    @Override
    public long callLongMethod(BaseVM vm, DvmObject<?> dvmObject, String signature, VarArg varArg) {
        if ("java/lang/Long->longValue()J".equals(signature)) {
            return (Long) dvmObject.getValue();
        }
        return super.callLongMethod(vm, dvmObject, signature, varArg);
    }

    @Override
    public int callIntMethodV(BaseVM vm, DvmObject<?> dvmObject, String signature, VaList vaList) {
        if ("java/lang/Integer->intValue()I".equals(signature)) {
            return (Integer) dvmObject.getValue();
        }
        try {
            return super.callIntMethodV(vm, dvmObject, signature, vaList);
        } catch (UnsupportedOperationException e) {
            System.out.println("[jni-miss] " + signature + " -> 0");
            return 0;
        }
    }

    @Override
    public int callIntMethod(BaseVM vm, DvmObject<?> dvmObject, String signature, VarArg varArg) {
        switch (signature) {
            case "android/content/Context->checkSelfPermission(Ljava/lang/String;)I":
            case "android/content/pm/PackageManager->checkPermission(Ljava/lang/String;Ljava/lang/String;)I":
                return 0;
            case "java/lang/Integer->intValue()I":
                return (Integer) dvmObject.getValue();
            default:
                return super.callIntMethod(vm, dvmObject, signature, varArg);
        }
    }

    @Override
    public boolean callBooleanMethodV(BaseVM vm, DvmObject<?> dvmObject, String signature, VaList vaList) {
        if ("java/lang/Boolean->booleanValue()Z".equals(signature)) {
            return dvmObject.getValue() instanceof Boolean && (Boolean) dvmObject.getValue();
        }
        try {
            return super.callBooleanMethodV(vm, dvmObject, signature, vaList);
        } catch (UnsupportedOperationException e) {
            System.out.println("[jni-miss] " + signature + " -> false");
            return false;
        }
    }

    @Override
    public boolean callBooleanMethod(BaseVM vm, DvmObject<?> dvmObject, String signature, VarArg varArg) {
        switch (signature) {
            case "java/lang/Boolean->booleanValue()Z":
                return (Boolean) dvmObject.getValue();
            case "android/content/SharedPreferences$Editor->commit()Z":
            case "java/io/File->exists()Z":
            case "java/io/File->mkdirs()Z":
                return true;
            default:
                return super.callBooleanMethod(vm, dvmObject, signature, varArg);
        }
    }

    @Override
    public DvmObject<?> getStaticObjectField(BaseVM vm, DvmClass dvmClass, String signature) {
        switch (signature) {
            case "android/os/Build->MODEL:Ljava/lang/String;":
                return new StringObject(vm, deviceModel);
            case "android/os/Build->BRAND:Ljava/lang/String;":
            case "android/os/Build->MANUFACTURER:Ljava/lang/String;":
                return new StringObject(vm, deviceBrand);
            case "android/os/Build->PRODUCT:Ljava/lang/String;":
            case "android/os/Build->DEVICE:Ljava/lang/String;":
                return new StringObject(vm, "gta8wifi");
            case "android/os/Build->HARDWARE:Ljava/lang/String;":
            case "android/os/Build->BOARD:Ljava/lang/String;":
                return new StringObject(vm, "mt6765");
            case "android/os/Build->DISPLAY:Ljava/lang/String;":
                return new StringObject(vm, "TP1A.220624.014");
            case "android/os/Build->FINGERPRINT:Ljava/lang/String;":
                return new StringObject(vm, "samsung/gta8wifixx/gta8wifi:14/UP1A.231005.007/T220XXS4CXJ1:user/release-keys");
            case "android/os/Build->BOOTLOADER:Ljava/lang/String;":
                return new StringObject(vm, "unknown");
            case "android/os/Build->HOST:Ljava/lang/String;":
                return new StringObject(vm, "21DJR20306001124");
            case "android/os/Build->TYPE:Ljava/lang/String;":
                return new StringObject(vm, "user");
            case "android/os/Build->TAGS:Ljava/lang/String;":
                return new StringObject(vm, "release-keys");
            case "android/os/Build->CPU_ABI:Ljava/lang/String;":
                return new StringObject(vm, "arm64-v8a");
            case "android/os/Build->CPU_ABI2:Ljava/lang/String;":
                return new StringObject(vm, "");
            case "android/os/Build$VERSION->RELEASE:Ljava/lang/String;":
                return new StringObject(vm, osVersion);
            case "android/os/Build$VERSION->CODENAME:Ljava/lang/String;":
                return new StringObject(vm, "REL");
            case "android/os/Build$VERSION->INCREMENTAL:Ljava/lang/String;":
                return new StringObject(vm, "T220XXS4CXJ1");
            case "android/os/Build$VERSION->SDK:Ljava/lang/String;":
                return new StringObject(vm, String.valueOf(sdkInt));
            case "android/provider/Settings$System->SCREEN_BRIGHTNESS:Ljava/lang/String;":
                return new StringObject(vm, "screen_brightness");
            default:
                try {
                    return super.getStaticObjectField(vm, dvmClass, signature);
                } catch (UnsupportedOperationException e) {
                    System.out.println("[jni-miss] field " + signature + " -> null");
                    return null;
                }
        }
    }

    @Override
    public int getStaticIntField(BaseVM vm, DvmClass dvmClass, String signature) {
        switch (signature) {
            case "android/os/Build$VERSION->SDK_INT:I":
                return sdkInt;
            case "android/os/Build$VERSION->PREVIEW_SDK_INT:I":
                return 0;
            case "android/media/AudioManager->STREAM_VOICE_CALL:I": return 0;
            case "android/media/AudioManager->STREAM_SYSTEM:I": return 1;
            case "android/media/AudioManager->STREAM_RING:I": return 2;
            case "android/media/AudioManager->STREAM_MUSIC:I": return 3;
            case "android/media/AudioManager->STREAM_ALARM:I": return 4;
            case "android/media/AudioManager->STREAM_NOTIFICATION:I": return 5;
            default:
                try {
                    return super.getStaticIntField(vm, dvmClass, signature);
                } catch (UnsupportedOperationException e) {
                    System.out.println("[jni-miss] field " + signature + " -> 0");
                    return 0;
                }
        }
    }

    @Override
    public void callVoidMethod(BaseVM vm, DvmObject<?> dvmObject, String signature, VarArg varArg) {
        if (!"android/content/SharedPreferences$Editor->apply()V".equals(signature)) {
            super.callVoidMethod(vm, dvmObject, signature, varArg);
        }
    }

    @Override
    public DvmObject<?> newObject(BaseVM vm, DvmClass dvmClass, String signature, VarArg varArg) {
        switch (signature) {
            case "java/util/HashMap-><init>()V":
                return dvmClass.newObject(new HashMap<>());
            case "java/io/File-><init>(Ljava/lang/String;)V": {
                StringObject path = varArg.getObjectArg(0);
                return dvmClass.newObject(new File(path.getValue()));
            }
            case "java/lang/Long-><init>(J)V":
                return dvmClass.newObject(varArg.getLongArg(0));
            case "java/lang/Integer-><init>(I)V":
                return dvmClass.newObject(varArg.getIntArg(0));
            case "java/lang/StringBuilder-><init>()V":
                return dvmClass.newObject(new StringBuilder());
            case "java/lang/String-><init>([BLjava/lang/String;)V": {
                ByteArray bytes = varArg.getObjectArg(0);
                return new StringObject(vm, new String(bytes.getValue(), StandardCharsets.UTF_8));
            }
            default:
                return super.newObject(vm, dvmClass, signature, varArg);
        }
    }

    @Override
    public DvmObject<?> newObjectV(BaseVM vm, DvmClass dvmClass, String signature, VaList vaList) {
        switch (signature) {
            case "java/lang/Long-><init>(J)V":
                return dvmClass.newObject(vaList.getLongArg(0));
            case "java/lang/Integer-><init>(I)V":
                return dvmClass.newObject(vaList.getIntArg(0));
            default:
                try {
                    return super.newObjectV(vm, dvmClass, signature, vaList);
                } catch (UnsupportedOperationException e) {
                    System.out.println("[jni-miss] new " + signature + " -> stub");
                    return dvmClass.newObject(null);
                }
        }
    }

    @Override
    public DvmObject<?> getObjectField(BaseVM vm, DvmObject<?> dvmObject, String signature) {
        switch (signature) {
            case "android/content/pm/PackageInfo->signatures:[Landroid/content/pm/Signature;":
                return vm.resolveClass("[Landroid/content/pm/Signature;").newObject(new Object[0]);
            case "android/content/pm/PackageInfo->versionName:Ljava/lang/String;":
                return new StringObject(vm, appVersion);
            default:
                try {
                    return super.getObjectField(vm, dvmObject, signature);
                } catch (UnsupportedOperationException e) {
                    System.out.println("[jni-miss] field " + signature + " -> null");
                    return null;
                }
        }
    }

    @Override
    public int getIntField(BaseVM vm, DvmObject<?> dvmObject, String signature) {
        if ("android/content/pm/PackageInfo->versionCode:I".equals(signature)) {
            return 74332;
        }
        try {
            return super.getIntField(vm, dvmObject, signature);
        } catch (UnsupportedOperationException e) {
            System.out.println("[jni-miss] field " + signature + " -> 0");
            return 0;
        }
    }

    @Override
    public DvmObject<?> allocObject(BaseVM vm, DvmClass dvmClass, String signature) {
        if ("java/util/HashMap->allocObject".equals(signature)) {
            return dvmClass.newObject(new HashMap<>());
        }
        return super.allocObject(vm, dvmClass, signature);
    }

    @Override
    public long callStaticLongMethodV(BaseVM vm, DvmClass dvmClass, String signature, VaList vaList) {
        switch (signature) {
            case "java/lang/System->currentTimeMillis()J":
                return System.currentTimeMillis();
            case "java/lang/System->nanoTime()J":
                return System.nanoTime();
            case "android/os/SystemClock->elapsedRealtime()J":
            case "android/os/SystemClock->uptimeMillis()J":
                return 3_600_000L + System.nanoTime() / 1_000_000L;
        }
        try {
            return super.callStaticLongMethodV(vm, dvmClass, signature, vaList);
        } catch (UnsupportedOperationException e) {
            System.out.println("[jni-miss] static " + signature + " -> 0");
            return 0;
        }
    }

    @Override
    public int callStaticIntMethodV(BaseVM vm, DvmClass dvmClass, String signature, VaList vaList) {
        try {
            return super.callStaticIntMethodV(vm, dvmClass, signature, vaList);
        } catch (UnsupportedOperationException e) {
            System.out.println("[jni-miss] static " + signature + " -> 0");
            return 0;
        }
    }

    @Override
    public boolean callStaticBooleanMethodV(BaseVM vm, DvmClass dvmClass, String signature, VaList vaList) {
        try {
            return super.callStaticBooleanMethodV(vm, dvmClass, signature, vaList);
        } catch (UnsupportedOperationException e) {
            System.out.println("[jni-miss] static " + signature + " -> false");
            return false;
        }
    }

    @Override
    public void callStaticVoidMethodV(BaseVM vm, DvmClass dvmClass, String signature, VaList vaList) {
        try {
            super.callStaticVoidMethodV(vm, dvmClass, signature, vaList);
        } catch (UnsupportedOperationException e) {
            System.out.println("[jni-miss] static " + signature);
        }
    }

    @Override
    public void callVoidMethodV(BaseVM vm, DvmObject<?> dvmObject, String signature, VaList vaList) {
        try {
            super.callVoidMethodV(vm, dvmObject, signature, vaList);
        } catch (UnsupportedOperationException e) {
            System.out.println("[jni-miss] " + signature);
        }
    }

    public void close() {
        try {
            emulator.close();
        } catch (IOException e) {
            e.printStackTrace();
        }
    }

    public void setDeviceModel(String model) { this.deviceModel = model; }
    public void setDeviceBrand(String brand) { this.deviceBrand = brand; }
    public void setOsVersion(String version) { this.osVersion = version; }
    public void setSdkInt(int sdk) { this.sdkInt = sdk; }
    public long getSdkHandle() { return sdkHandle; }
}
