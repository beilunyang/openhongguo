package com.openhongguo.signer;

import com.github.unidbg.Emulator;
import com.github.unidbg.linux.file.NetLinkSocket;
import com.sun.jna.Pointer;

import java.nio.ByteBuffer;
import java.nio.ByteOrder;

/**
 * unidbg answers only RTM_GETADDR dumps and throws on any other netlink
 * request, which kills the calling thread. Answer the rest with an empty
 * dump (a lone NLMSG_DONE), as a kernel with nothing to report would.
 */
class SafeNetLinkSocket extends NetLinkSocket {
    private int seq;
    private boolean pendingDone;

    SafeNetLinkSocket(Emulator<?> emulator) {
        super(emulator);
    }

    @Override
    public int write(byte[] data) {
        if (data.length >= 16) {
            ByteBuffer b = ByteBuffer.wrap(data).order(ByteOrder.LITTLE_ENDIAN);
            seq = b.getInt(8);
            System.out.println("[netlink] request type=0x" + Integer.toHexString(b.getShort(4) & 0xffff)
                    + " flags=0x" + Integer.toHexString(b.getShort(6) & 0xffff));
        }
        pendingDone = true;
        return super.write(data);
    }

    @Override
    protected int handleType(Pointer buffer, int count, short netlinkType) {
        try {
            int n = super.handleType(buffer, count, netlinkType);
            return n;
        } catch (UnsupportedOperationException e) {
            if (!pendingDone || count < 20) {
                return 0;
            }
            pendingDone = false;
            ByteBuffer b = ByteBuffer.allocate(20).order(ByteOrder.LITTLE_ENDIAN);
            b.putInt(20);
            b.putShort((short) 3); // NLMSG_DONE
            b.putShort((short) 2); // NLM_F_MULTI
            b.putInt(seq);
            b.putInt(0);
            b.putInt(0);
            buffer.write(0, b.array(), 0, 20);
            return 20;
        }
    }
}
