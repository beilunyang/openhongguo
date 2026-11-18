#!/usr/bin/env python3
# -I safe disassembler helper. Usage: dis.py <so> <start_hex> <count>
import sys
from capstone import Cs, CS_ARCH_ARM64, CS_MODE_LITTLE_ENDIAN

so = sys.argv[1]
start = int(sys.argv[2], 16)
count = int(sys.argv[3]) if len(sys.argv) > 3 else 60

with open(so, 'rb') as f:
    data = f.read()

# file offset == VMA for this .so (text PT_LOAD offset 0 VA 0)
code = data[start:start + count * 4]
md = Cs(CS_ARCH_ARM64, CS_MODE_LITTLE_ENDIAN)
md.detail = False
for insn in md.disasm(code, start):
    print(f"0x{insn.address:06x}: {insn.mnemonic:8} {insn.op_str}")
