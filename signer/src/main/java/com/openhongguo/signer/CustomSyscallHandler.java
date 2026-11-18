package com.openhongguo.signer;

import com.github.unidbg.Emulator;
import com.github.unidbg.arm.ARMEmulator;
import com.github.unidbg.arm.backend.Backend;
import com.github.unidbg.linux.ARM64SyscallHandler;
import com.github.unidbg.memory.SvcMemory;
import unicorn.Arm64Const;

import java.util.HashMap;
import java.util.Map;

/**
 * Custom futex handler for metasec SDK emulation.
 *
 * The signing code in libmetasec_ml.so waits on a synchronization address
 * (vtable[0x28] = soBase + 0x3C4F30) for a background init thread to store
 * a function pointer. Without a real background thread, we simulate the
 * initialization by writing the function pointer when the signing code
 * enters FUTEX_WAIT with val=0 (waiting for init to complete).
 */
public class CustomSyscallHandler extends ARM64SyscallHandler {

    private static final int FUTEX_CMD_MASK = 0x7f;
    private static final int FUTEX_WAIT = 0;
    private static final int FUTEX_WAKE = 1;
    private static final int EAGAIN = 11;
    private static final int ETIMEDOUT = 110;

    // Module base address and key offsets
    private long moduleBase = 0;
    private long vtableOffset = 0;        // 0x3C4F08 — vtable base in .bss
    private long funcPtrOffset = 0;       // 0x17a9d4 — function pointer offset from base

    public CustomSyscallHandler(SvcMemory svcMemory) {
        super(svcMemory);
    }

    public void setModuleBase(long base) { this.moduleBase = base; }
    public void setVtableOffset(long offset) { this.vtableOffset = offset; }
    public void setFunctionPointerOffset(long offset) { this.funcPtrOffset = offset; }

    private long futexCount = 0;
    private long signingFutexCount = 0;
    private volatile boolean signing = false;
    private final Map<Long, long[]> addrStats = new HashMap<>();
    private boolean vtableInitDone = false;

    public void startSigning() {
        signing = true;
        signingFutexCount = 0;
        vtableInitDone = false;
    }

    public void stopSigning() { signing = false; }
    public long getFutexCount() { return futexCount; }
    public long getSigningFutexCount() { return signingFutexCount; }

    public void printStats() {
        System.out.println("[futex] === stats === total=" + futexCount + " signing=" + signingFutexCount);
        for (Map.Entry<Long, long[]> e : addrStats.entrySet()) {
            long addr = e.getKey();
            long[] stats = e.getValue();
            String desc = describeAddr(addr);
            System.out.println("[futex]   addr=0x" + Long.toHexString(addr)
                    + " wait=" + stats[0] + " wake=" + stats[1] + " " + desc);
        }
    }

    private String describeAddr(long addr) {
        if (moduleBase == 0) return "";
        long offset = addr - moduleBase;
        if (offset >= 0 && offset < 0x500000) {
            return "(so+0x" + Long.toHexString(offset) + ")";
        }
        return "";
    }

    private boolean isVtable028(long addr) {
        if (moduleBase == 0 || vtableOffset == 0) return false;
        return addr == moduleBase + vtableOffset + 0x28;
    }

    private int udefLogs = 0;
    private final long[] recentPc = new long[48];
    private int recentPcN = 0;

    public void notePc(long pc) {
        recentPc[recentPcN % recentPc.length] = pc;
        recentPcN++;
    }

    private int unknownSyscallLogs = 0;
    private int sig64Logs = 0;

    /** Global sigjmp_buf filled by so+0x2234f8 before raise(64). */
    private static final long JMP_BUF_OFF = 0x45ba10L;
    /** Flag word at so+0x3f1280. The handler publishes 0x4bd; 3 means the signal arrived too late. */
    private static final long SIG64_FLAG_OFF = 0x3f1280L;

    @Override
    protected boolean handleUnknownSyscall(Emulator<?> emulator, int NR) {
        if (unknownSyscallLogs++ < 24) {
            Backend backend = emulator.getBackend();
            long lr = backend.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
            long x0 = backend.reg_read(Arm64Const.UC_ARM64_REG_X0).longValue();
            long x1 = backend.reg_read(Arm64Const.UC_ARM64_REG_X1).longValue();
            long x2 = backend.reg_read(Arm64Const.UC_ARM64_REG_X2).longValue();
            long x3 = backend.reg_read(Arm64Const.UC_ARM64_REG_X3).longValue();
            System.out.println("[syscall] unknown NR=" + NR
                    + " x0=0x" + Long.toHexString(x0)
                    + " x1=0x" + Long.toHexString(x1)
                    + " x2=0x" + Long.toHexString(x2)
                    + " x3=0x" + Long.toHexString(x3)
                    + " lr=" + where(emulator, lr));
        }
        return false;
    }

    private static String where(Emulator<?> emulator, long addr) {
        com.github.unidbg.Module m = emulator.getMemory().findModuleByAddress(addr);
        if (m == null) {
            return "0x" + Long.toHexString(addr);
        }
        return m.name + "+0x" + Long.toHexString(addr - m.base);
    }

    @Override
    public void hook(Backend backend, int intno, int swi, Object user) {
        // A zeroed or unmapped instruction is UDF. The console debugger then
        // reads stdin and loops. Log the recent PCs and return the thread.
        if (intno == ARMEmulator.EXCP_UDEF) {
            Emulator<?> emulator = (Emulator<?>) user;
            if (udefLogs++ < 3) {
                long pc = backend.reg_read(Arm64Const.UC_ARM64_REG_PC).longValue();
                long lr = backend.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                long sp = backend.reg_read(Arm64Const.UC_ARM64_REG_SP).longValue();
                long fp = backend.reg_read(Arm64Const.UC_ARM64_REG_FP).longValue();
                byte[] insn = new byte[]{0, 0, 0, 0};
                try {
                    insn = backend.mem_read(pc, 4);
                } catch (Exception ignored) {
                }
                System.out.println("[udef] pc=0x" + Long.toHexString(pc)
                        + " lr=0x" + Long.toHexString(lr)
                        + " sp=0x" + Long.toHexString(sp)
                        + " fp=0x" + Long.toHexString(fp)
                        + " insn=" + String.format("%02x%02x%02x%02x", insn[0], insn[1], insn[2], insn[3]));
                System.out.println("[udef] x0=0x" + Long.toHexString(backend.reg_read(Arm64Const.UC_ARM64_REG_X0).longValue())
                        + " x1=0x" + Long.toHexString(backend.reg_read(Arm64Const.UC_ARM64_REG_X1).longValue())
                        + " x2=0x" + Long.toHexString(backend.reg_read(Arm64Const.UC_ARM64_REG_X2).longValue())
                        + " x8=0x" + Long.toHexString(backend.reg_read(Arm64Const.UC_ARM64_REG_X8).longValue())
                        + " x19=0x" + Long.toHexString(backend.reg_read(Arm64Const.UC_ARM64_REG_X19).longValue())
                        + " x20=0x" + Long.toHexString(backend.reg_read(Arm64Const.UC_ARM64_REG_X20).longValue()));
                System.out.print("[udef] stack");
                for (int i = -4; i < 8; i++) {
                    long a = sp + i * 8L;
                    String w;
                    try {
                        w = "0x" + Long.toHexString(readLong(backend, a));
                    } catch (Exception e) {
                        w = "?";
                    }
                    System.out.print(" [" + (i * 8) + "]=" + w);
                }
                System.out.println();
                int n = Math.min(recentPcN, recentPc.length);
                int start = recentPcN - n;
                if (n > 0) {
                    System.out.println("[udef] recent pcs:");
                    for (int i = 0; i < n; i++) {
                        long a = recentPc[(start + i) % recentPc.length];
                        System.out.println("[udef]   " + where(emulator, a));
                    }
                }
            }
            backend.reg_write(Arm64Const.UC_ARM64_REG_PC, emulator.getReturnAddress());
            return;
        }
        // NR 220 is clone. Log it before the parent handler, which either
        // enqueues a MarshmallowThread (bionic layout) or throws.
        // NR 131 sig 64 is the guest's setjmp/longjmp check. Do that longjmp
        // here. Queuing a SignalTask and then resuming the svc runs both
        // the longjmp target and the interrupted frame on the same stack.
        if (swi == 0) {
            try {
                int nr = backend.reg_read(Arm64Const.UC_ARM64_REG_X8).intValue();
                if (nr == 131) {
                    long sig = backend.reg_read(Arm64Const.UC_ARM64_REG_X2).longValue();
                    if (sig == 64 && longjmpSignal64(backend, (Emulator<?>) user)) {
                        return;
                    }
                }
                if (nr == 220) {
                    long flags = backend.reg_read(Arm64Const.UC_ARM64_REG_X0).longValue();
                    long stack = backend.reg_read(Arm64Const.UC_ARM64_REG_X1).longValue();
                    long fn = backend.reg_read(Arm64Const.UC_ARM64_REG_X5).longValue();
                    long arg = backend.reg_read(Arm64Const.UC_ARM64_REG_X6).longValue();
                    long lr = backend.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                    long slot0 = 0;
                    long slot1 = 0;
                    long startFn = 0;
                    long startArg = 0;
                    if (stack != 0) {
                        slot0 = readLong(backend, stack);
                        slot1 = readLong(backend, stack + 8);
                    }
                    if (arg != 0) {
                        // bionic pthread_internal_t: start_routine at +0x60, arg at +0x68.
                        startFn = readLong(backend, arg + 0x60);
                        startArg = readLong(backend, arg + 0x68);
                    }
                    System.out.println("[clone] flags=0x" + Long.toHexString(flags)
                            + " stack=0x" + Long.toHexString(stack)
                            + " fn=0x" + Long.toHexString(fn)
                            + " arg=0x" + Long.toHexString(arg)
                            + " stack[0]=0x" + Long.toHexString(slot0)
                            + " stack[1]=0x" + Long.toHexString(slot1)
                            + " lr=0x" + Long.toHexString(lr)
                            + " bionic=" + (stack != 0 && slot0 == fn && slot1 == arg)
                            + " start=" + where((Emulator<?>) user, startFn)
                            + " startArg=0x" + Long.toHexString(startArg));
                }
            } catch (Exception e) {
                System.out.println("[clone] log failed: " + e);
            }
        }
        int nrBefore = backend.reg_read(Arm64Const.UC_ARM64_REG_X8).intValue();
        long domain = backend.reg_read(Arm64Const.UC_ARM64_REG_X0).longValue();
        super.hook(backend, intno, swi, user);
        if (nrBefore == 198 && domain == 16) { // socket(AF_NETLINK, ...)
            int fd = backend.reg_read(Arm64Const.UC_ARM64_REG_X0).intValue();
            if (fd >= 0 && fdMap.get(fd) instanceof com.github.unidbg.linux.file.NetLinkSocket) {
                fdMap.put(fd, new SafeNetLinkSocket((Emulator<?>) user));
            }
        }
    }

    @Override
    protected int futex(Emulator<?> emulator) {
        futexCount++;
        Backend backend = emulator.getBackend();
        long uaddr = backend.reg_read(Arm64Const.UC_ARM64_REG_X0).longValue();
        int op = backend.reg_read(Arm64Const.UC_ARM64_REG_X1).intValue();
        int val = backend.reg_read(Arm64Const.UC_ARM64_REG_X2).intValue();
        int cmd = op & FUTEX_CMD_MASK;

        long[] stats = addrStats.computeIfAbsent(uaddr, k -> new long[2]);
        if (cmd == FUTEX_WAIT) stats[0]++;
        else if (cmd == FUTEX_WAKE) stats[1]++;

        // Cooperative dispatcher parks the caller via ThreadContextSwitchException.
        // Mutating the futex word here skips that park and the worker never runs.
        if (threadDispatcherEnabled) {
            if (cmd == FUTEX_WAIT) {
                long ts = backend.reg_read(Arm64Const.UC_ARM64_REG_X3).longValue();
                if (ts != 0 && readInt(backend, uaddr) == val) {
                    long sec = readLong(backend, ts);
                    long nsec = readLong(backend, ts + 8);
                    // FutexNanoSleepWaiter rejects a wait shorter than 1 ms
                    // with IllegalStateException, which kills the thread. The
                    // kernel just times out.
                    if (sec * 1000 + nsec / 1_000_000 <= 0) {
                        return -110; // ETIMEDOUT
                    }
                }
            }
            if (signing && signingFutexCount < 30 && cmd == FUTEX_WAIT) {
                signingFutexCount++;
                int curVal = readInt(backend, uaddr);
                long lr = backend.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                System.out.println("[futex] dispatch wait #" + signingFutexCount
                        + " addr=0x" + Long.toHexString(uaddr) + " " + describeAddr(uaddr)
                        + " val=0x" + Integer.toHexString(val)
                        + " cur=0x" + Integer.toHexString(curVal)
                        + " lr=0x" + Long.toHexString(lr));
            } else if (!signing && futexCount <= 15 && cmd == FUTEX_WAIT) {
                int curVal = readInt(backend, uaddr);
                System.out.println("[futex] init wait #" + futexCount
                        + " addr=0x" + Long.toHexString(uaddr)
                        + " val=0x" + Integer.toHexString(val)
                        + " cur=0x" + Integer.toHexString(curVal));
            }
            return super.futex(emulator);
        }

        if (signing && signingFutexCount < 100) {
            signingFutexCount++;
            if (signingFutexCount <= 30) {
                long lr = backend.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
                long pc = backend.reg_read(Arm64Const.UC_ARM64_REG_PC).longValue();
                long fp = backend.reg_read(Arm64Const.UC_ARM64_REG_FP).longValue();
                int curVal = readInt(backend, uaddr);
                String addrDesc = describeAddr(uaddr);
                com.github.unidbg.Module pcMod = emulator.getMemory().findModuleByAddress(pc);
                String pcName = pcMod != null ? pcMod.name + "+0x" + Long.toHexString(pc - pcMod.base) : "0x" + Long.toHexString(pc);
                com.github.unidbg.Module lrMod = emulator.getMemory().findModuleByAddress(lr);
                String lrName = lrMod != null ? lrMod.name + "+0x" + Long.toHexString(lr - lrMod.base) : "0x" + Long.toHexString(lr);
                System.out.println("[futex] signing #" + signingFutexCount
                        + " cmd=" + cmd + " addr=0x" + Long.toHexString(uaddr) + " " + addrDesc
                        + " val=0x" + Integer.toHexString(val) + " cur=0x" + Integer.toHexString(curVal)
                        + " pc=" + pcName + " lr=" + lrName);

                // Walk the frame pointer chain to find .so-level caller
                if (signingFutexCount <= 5) {
                    System.out.println("[futex]   backtrace:");
                    long curFp = fp;
                    for (int frame = 0; frame < 10 && curFp != 0; frame++) {
                        try {
                            long savedFp = readLong(backend, curFp);
                            long savedLr = readLong(backend, curFp + 8);
                            com.github.unidbg.Module mod = emulator.getMemory().findModuleByAddress(savedLr);
                            String modName = mod != null ? mod.name + "+0x" + Long.toHexString(savedLr - mod.base) : "0x" + Long.toHexString(savedLr);
                            System.out.println("[futex]     #" + frame + " lr=" + modName + " fp=0x" + Long.toHexString(curFp));
                            curFp = savedFp;
                        } catch (Exception e) {
                            break;
                        }
                    }
                }
            }
        } else if (!signing && futexCount <= 20) {
            long lr = backend.reg_read(Arm64Const.UC_ARM64_REG_LR).longValue();
            int curVal = readInt(backend, uaddr);
            System.out.println("[futex] init #" + futexCount
                    + " cmd=" + cmd + " addr=0x" + Long.toHexString(uaddr)
                    + " val=0x" + Integer.toHexString(val) + " cur=0x" + Integer.toHexString(curVal)
                    + " lr=0x" + Long.toHexString(lr));
        }

        if (cmd == FUTEX_WAIT) {
            int curVal = readInt(backend, uaddr);

            // ---- Special handling: vtable[0x28] init synchronization ----
            // The signing code waits here for a background thread to store a
            // function pointer. We simulate that by writing the pointer when
            // the code enters FUTEX_WAIT with val=0 (waiting for init).
            if (signing && isVtable028(uaddr) && !vtableInitDone) {
                long funcPtr = moduleBase + funcPtrOffset;
                // Write the function pointer as a 64-bit value
                writeLong(backend, uaddr, funcPtr);
                vtableInitDone = true;
                System.out.println("[futex] SIMULATED INIT: wrote funcPtr=0x"
                        + Long.toHexString(funcPtr) + " to vtable[0x28]");
                return -EAGAIN; // Value changed — caller re-reads and sees the pointer
            }

            // ---- General single-threaded futex emulation ----
            // Always write 0 (unlocked/idle) to the futex address so the
            // calling code's retry loop sees a changed value and can proceed.
            // Without another thread to modify the address, this is the only
            // way to break the wait.
            writeInt(backend, uaddr, 0);
            return -EAGAIN;
        }

        if (cmd == FUTEX_WAKE) {
            return 1;
        }

        // FUTEX_REQUEUE, etc.
        return 0;
    }

    /**
     * so+0x2235cc sets the flag to 0x4bd and longjmps to the sigsetjmp at
     * so+0x2234fc. Unicorn's svc hook already points PC at the next instruction,
     * so writing the restored PC here is the whole jump.
     *
     * @return true when this tgkill was consumed and the parent hook must not run
     */
    private boolean longjmpSignal64(Backend backend, Emulator<?> emulator) {
        if (moduleBase == 0) {
            return false;
        }
        long flagAddr = moduleBase + SIG64_FLAG_OFF;
        int flag = readInt(backend, flagAddr);
        if (flag == 3) {
            backend.reg_write(Arm64Const.UC_ARM64_REG_X0, 0);
            if (sig64Logs++ < 8) {
                long pc = backend.reg_read(Arm64Const.UC_ARM64_REG_PC).longValue();
                System.out.println("[sig64] signal arrived after raise returned, pc=" + where(emulator, pc));
            }
            return true;
        }
        long buf = moduleBase + JMP_BUF_OFF;
        long lr = readLong(backend, buf + 0x10);
        long sp = readLong(backend, buf + 0x18);
        if (lr < moduleBase || lr >= moduleBase + 0x468000L || sp == 0 || (sp & 0xf) != 0) {
            if (sig64Logs++ < 8) {
                System.out.println("[sig64] jmp_buf unusable lr=0x" + Long.toHexString(lr)
                        + " sp=0x" + Long.toHexString(sp) + " flag=" + flag);
            }
            return false;
        }
        writeInt(backend, flagAddr, 0x4bd);
        int[] gpr = {
                Arm64Const.UC_ARM64_REG_X28, Arm64Const.UC_ARM64_REG_X29,
                Arm64Const.UC_ARM64_REG_X26, Arm64Const.UC_ARM64_REG_X27,
                Arm64Const.UC_ARM64_REG_X24, Arm64Const.UC_ARM64_REG_X25,
                Arm64Const.UC_ARM64_REG_X22, Arm64Const.UC_ARM64_REG_X23,
                Arm64Const.UC_ARM64_REG_X20, Arm64Const.UC_ARM64_REG_X21,
                Arm64Const.UC_ARM64_REG_X19
        };
        int[] gprOff = {0x20, 0x28, 0x30, 0x38, 0x40, 0x48, 0x50, 0x58, 0x60, 0x68, 0x70};
        for (int i = 0; i < gpr.length; i++) {
            backend.reg_write(gpr[i], readLong(backend, buf + gprOff[i]));
        }
        // setjmp stores d14,d15 at +0x78, then d12,d13, d10,d11, d8,d9.
        int[] dreg = {
                Arm64Const.UC_ARM64_REG_D14, Arm64Const.UC_ARM64_REG_D15,
                Arm64Const.UC_ARM64_REG_D12, Arm64Const.UC_ARM64_REG_D13,
                Arm64Const.UC_ARM64_REG_D10, Arm64Const.UC_ARM64_REG_D11,
                Arm64Const.UC_ARM64_REG_D8, Arm64Const.UC_ARM64_REG_D9
        };
        try {
            for (int i = 0; i < dreg.length; i++) {
                backend.reg_write(dreg[i], readLong(backend, buf + 0x78 + i * 8L));
            }
        } catch (RuntimeException ignored) {
        }
        backend.reg_write(Arm64Const.UC_ARM64_REG_SP, sp);
        backend.reg_write(Arm64Const.UC_ARM64_REG_X0, 1L);
        backend.reg_write(Arm64Const.UC_ARM64_REG_LR, lr);
        backend.reg_write(Arm64Const.UC_ARM64_REG_PC, lr);
        if (sig64Logs++ < 8) {
            System.out.println("[sig64] longjmp -> " + where(emulator, lr)
                    + " sp=0x" + Long.toHexString(sp)
                    + " flag " + flag + " -> 0x4bd");
        }
        return true;
    }

    private long readLong(Backend backend, long addr) {
        try {
            byte[] b = backend.mem_read(addr, 8);
            long val = 0;
            for (int i = 7; i >= 0; i--) {
                val = (val << 8) | (b[i] & 0xFFL);
            }
            return val;
        } catch (Exception e) {
            return 0;
        }
    }

    private int readInt(Backend backend, long addr) {
        try {
            byte[] b = backend.mem_read(addr, 4);
            return (b[0] & 0xFF) | ((b[1] & 0xFF) << 8) | ((b[2] & 0xFF) << 16) | ((b[3] & 0xFF) << 24);
        } catch (Exception e) {
            return 0;
        }
    }

    private void writeInt(Backend backend, long addr, int value) {
        byte[] b = new byte[4];
        b[0] = (byte) (value & 0xFF);
        b[1] = (byte) ((value >> 8) & 0xFF);
        b[2] = (byte) ((value >> 16) & 0xFF);
        b[3] = (byte) ((value >> 24) & 0xFF);
        backend.mem_write(addr, b);
    }

    private void writeLong(Backend backend, long addr, long value) {
        byte[] b = new byte[8];
        for (int i = 0; i < 8; i++) {
            b[i] = (byte) ((value >> (i * 8)) & 0xFF);
        }
        backend.mem_write(addr, b);
    }
}
