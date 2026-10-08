"""Build a COFF .syso containing Win32 resources (icon group, version info, manifest) for Go."""
import struct, sys

NAME = "시몬스 데스크톱 컴패니언"
VER = (1, 0, 0, 0)
LANG = 0x0412  # Korean

def align(b, n=4):
    return b + b"\0" * ((n - len(b) % n) % n)

# ---------- version info ----------
def vblock(key, value=b"", wtype=1, vlen=None, children=()):
    k = key.encode("utf-16-le") + b"\0\0"
    body = k + b"\0" * ((4 - (6 + len(k)) % 4) % 4)
    body += value
    if children:
        body += b"\0" * ((4 - (6 + len(body)) % 4) % 4)
        for c in children:
            body += align(c)
    if vlen is None:
        vlen = len(value)
    return struct.pack("<HHH", 6 + len(body), vlen, wtype) + body

def vstr(key, text):
    v = text.encode("utf-16-le") + b"\0\0"
    return vblock(key, v, 1, len(text) + 1)

ms = (VER[0] << 16) | VER[1]; ls = (VER[2] << 16) | VER[3]
fixed = struct.pack("<13I", 0xFEEF04BD, 0x00010000, ms, ls, ms, ls, 0x3F, 0, 0x00040004, 1, 0, 0, 0)
vs = ".".join(map(str, VER))
strings = [vstr("CompanyName", "Simons"), vstr("FileDescription", NAME), vstr("FileVersion", vs),
           vstr("InternalName", "SimonsCompanion"), vstr("OriginalFilename", NAME + ".exe"),
           vstr("ProductName", NAME), vstr("ProductVersion", vs)]
table = vblock("%04X04B0" % LANG, b"", 1, 0, strings)
sfi = vblock("StringFileInfo", b"", 1, 0, [table])
var = vblock("Translation", struct.pack("<HH", LANG, 1200), 0)
vfi = vblock("VarFileInfo", b"", 1, 0, [var])
version = vblock("VS_VERSION_INFO", fixed, 0, 52, [sfi, vfi])

manifest = ('<?xml version="1.0" encoding="UTF-8" standalone="yes"?>\n'
 '<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0" xmlns:asmv3="urn:schemas-microsoft-com:asm.v3">\n'
 ' <assemblyIdentity type="win32" name="Simons.DesktopCompanion" version="1.0.0.0"/>\n'
 ' <asmv3:application><asmv3:windowsSettings>\n'
 '  <dpiAware xmlns="http://schemas.microsoft.com/SMI/2005/WindowsSettings">true</dpiAware>\n'
 ' </asmv3:windowsSettings></asmv3:application>\n'
 ' <compatibility xmlns="urn:schemas-microsoft-com:compatibility.v1"><application>\n'
 '  <supportedOS Id="{8e0f7a12-bfb3-4fe8-b9a5-48fd50a15a9a}"/>\n'
 ' </application></compatibility>\n'
 ' <trustInfo xmlns="urn:schemas-microsoft-com:asm.v3"><security><requestedPrivileges>\n'
 '  <requestedExecutionLevel level="asInvoker" uiAccess="false"/>\n'
 ' </requestedPrivileges></security></trustInfo>\n'
 '</assembly>\n').encode("utf-8")

# ---------- icons ----------
icon_dir = sys.argv[1]
sizes = [256, 128, 64, 48, 32, 24, 16]
def dib_icon(path):
    from PIL import Image
    im = Image.open(path).convert("RGBA"); w, h = im.size
    px = im.tobytes()
    hdr = struct.pack("<IiiHHIIiiII", 40, w, h * 2, 1, 32, 0, 0, 0, 0, 0, 0)
    xor = b"".join(bytes(b for x in range(w) for b in (px[(y*w+x)*4+2], px[(y*w+x)*4+1], px[(y*w+x)*4], px[(y*w+x)*4+3])) for y in range(h - 1, -1, -1))
    stride = ((w + 31) // 32) * 4
    andm = bytearray(stride * h)
    for yy, y in enumerate(range(h - 1, -1, -1)):
        for x in range(w):
            if px[(y*w+x)*4+3] == 0: andm[yy*stride + x//8] |= 0x80 >> (x % 8)
    return hdr + xor + bytes(andm)
icons = [open(f"{icon_dir}/{s}.png", "rb").read() if s == 256 else dib_icon(f"{icon_dir}/{s}.png") for s in sizes]
grp = struct.pack("<HHH", 0, 1, len(icons))
for i, (s, d) in enumerate(zip(sizes, icons)):
    grp += struct.pack("<BBBBHHIH", s % 256, s % 256, 0, 0, 1, 32, len(d), i + 1)

# resources: type -> [(id, data)]
res = {3: [(i + 1, d) for i, d in enumerate(icons)], 14: [(1, grp)], 16: [(1, version)], 24: [(1, manifest)]}

# ---------- layout ----------
types = sorted(res)
dir_size = lambda n: 16 + 8 * n
off = dir_size(len(types))
type_dir_off = {}
for t in types:
    type_dir_off[t] = off; off += dir_size(len(res[t]))
name_dir_off = {}
for t in types:
    for rid, _ in res[t]:
        name_dir_off[(t, rid)] = off; off += dir_size(1)
entry_off = {}
for t in types:
    for rid, _ in res[t]:
        entry_off[(t, rid)] = off; off += 16
data_off = {}
for t in types:
    for rid, d in res[t]:
        off = (off + 7) & ~7
        data_off[(t, rid)] = off; off += len(d)
total = (off + 7) & ~7

buf = bytearray(total)
def put(o, b): buf[o:o + len(b)] = b
def dirhdr(n): return struct.pack("<IIHHHH", 0, 0, 0, 0, 0, n)
put(0, dirhdr(len(types)) + b"".join(struct.pack("<II", t, 0x80000000 | type_dir_off[t]) for t in types))
for t in types:
    ids = sorted(rid for rid, _ in res[t])
    put(type_dir_off[t], dirhdr(len(ids)) + b"".join(struct.pack("<II", rid, 0x80000000 | name_dir_off[(t, rid)]) for rid in ids))
relocs = []
for t in types:
    for rid, d in res[t]:
        put(name_dir_off[(t, rid)], dirhdr(1) + struct.pack("<II", LANG, entry_off[(t, rid)]))
        put(entry_off[(t, rid)], struct.pack("<IIII", data_off[(t, rid)], len(d), 0, 0))
        relocs.append(entry_off[(t, rid)])
        put(data_off[(t, rid)], d)

raw = bytes(buf)
sec_off = 20 + 40
rel_off = sec_off + len(raw)
sym_off = rel_off + 10 * len(relocs)
coff = struct.pack("<HHIIIHH", 0x8664, 1, 0, sym_off, 1, 0, 0)
coff += struct.pack("<8sIIIIIIHHI", b".rsrc", 0, 0, len(raw), sec_off, rel_off, 0, len(relocs), 0, 0x40000040)
coff += raw
for r in relocs:
    coff += struct.pack("<IIH", r, 0, 3)  # IMAGE_REL_AMD64_ADDR32NB
coff += struct.pack("<8sIhHBB", b".rsrc", 0, 1, 0, 3, 0)
coff += struct.pack("<I", 4)
open(sys.argv[2], "wb").write(coff)
print("syso bytes", len(coff), "resources", {t: len(v) for t, v in res.items()})
