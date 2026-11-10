"""Draw a simple phone screenshot PNG for the demo's Compare tab (no dependencies).

Usage: python3 phone_png.py OUT.png VARIANT   (VARIANT: list | map | map-v2 | downloads)
"""
import math, struct, sys, zlib

W, H = 390, 700
out, variant = sys.argv[1], sys.argv[2]
px = bytearray(W * H * 3)

def put(x, y, c):
    if 0 <= x < W and 0 <= y < H:
        i = (y * W + x) * 3
        px[i:i + 3] = bytes(c)

def rect(x0, y0, x1, y1, c, r=0):
    for y in range(max(0, y0), min(H, y1)):
        for x in range(max(0, x0), min(W, x1)):
            if r:
                cx = x0 + r if x < x0 + r else x1 - r - 1 if x >= x1 - r else x
                cy = y0 + r if y < y0 + r else y1 - r - 1 if y >= y1 - r else y
                if (x - cx) ** 2 + (y - cy) ** 2 > r * r:
                    continue
            put(x, y, c)

def disc(cx, cy, r, c):
    rect(cx - r, cy - r, cx + r + 1, cy + r + 1, c, r)

for y in range(H):  # map background
    for x in range(W):
        t = (x + y) / (W + H)
        put(x, y, (int(214 - 40 * t), int(232 - 30 * t), int(217 - 35 * t)))
rect(0, 0, W, 64, (255, 255, 255))
rect(16, 24, 200, 40, (40, 52, 60), 4)
if variant == "list":
    for i in range(5):
        rect(14, 84 + i * 112, W - 14, 84 + i * 112 + 98, (255, 255, 255), 12)
        rect(28, 100 + i * 112, 170, 114 + i * 112, (40, 52, 60), 3)
        rect(28, 124 + i * 112, 120, 134 + i * 112, (150, 160, 168), 3)
else:
    for k in range(0, 360, 6):  # dashed trail loop
        a = math.radians(k)
        disc(int(190 + 110 * math.cos(a)), int(330 + 170 * math.sin(a) * (0.8 if k > 180 else 1)), 3, (192, 88, 47))
    disc(200, 380, 10, (255, 255, 255)); disc(200, 380, 7, (47, 111, 209))
    if variant == "map":
        rect(260, 76, 374, 100, (31, 157, 85), 12)       # offline badge
        rect(16, 620, W - 16, 676, (15, 157, 138), 14)   # full-width download button
    elif variant == "map-v2":
        rect(250, 628, 374, 670, (47, 111, 209), 12)     # small download button, no badge
    else:
        for i in range(4):
            rect(14, 430 + i * 62, W - 14, 480 + i * 62, (255, 255, 255), 10)
            rect(28, 446 + i * 62, 60 + 40 * i, 462 + i * 62, (15, 157, 138), 4)

raw = b"".join(b"\x00" + bytes(px[y * W * 3:(y + 1) * W * 3]) for y in range(H))
def chunk(t, d):
    return struct.pack(">I", len(d)) + t + d + struct.pack(">I", zlib.crc32(t + d) & 0xFFFFFFFF)
open(out, "wb").write(b"\x89PNG\r\n\x1a\n" + chunk(b"IHDR", struct.pack(">IIBBBBB", W, H, 8, 2, 0, 0, 0)) + chunk(b"IDAT", zlib.compress(raw, 6)) + chunk(b"IEND", b""))
