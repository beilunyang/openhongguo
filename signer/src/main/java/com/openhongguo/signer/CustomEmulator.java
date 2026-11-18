package com.openhongguo.signer;

import com.github.unidbg.file.linux.AndroidFileIO;
import com.github.unidbg.linux.android.AndroidARM64Emulator;
import com.github.unidbg.memory.SvcMemory;
import com.github.unidbg.unix.UnixSyscallHandler;

import java.util.Collections;

public class CustomEmulator extends AndroidARM64Emulator {

    private CustomSyscallHandler customSyscallHandler;

    public CustomEmulator(String processName) {
        super(processName, null, Collections.emptyList());
    }

    @Override
    protected UnixSyscallHandler<AndroidFileIO> createSyscallHandler(SvcMemory svcMemory) {
        customSyscallHandler = new CustomSyscallHandler(svcMemory);
        return customSyscallHandler;
    }

    public CustomSyscallHandler getCustomSyscallHandler() {
        return customSyscallHandler;
    }
}
