#!/usr/bin/env python3
"""Generate TigerWallet extension icons (RGBA PNG) with no external deps.

Every extension manifest referenced icons/icon16.png, icon48.png, icon128.png
(and icon256.png) that did not exist, so the extensions could not be loaded.
Run from the repo root:  python3 tools/gen_extension_icons.py
"""
import os
import struct
import zlib

# TigerWallet brand: deep orange tile, near-white "T" mark.
BG_TOP = (243, 118, 32)
BG_BOTTOM = (192, 57, 43)
FG = (255, 255, 255)


def blend(a, b, t):
    return tuple(round(a[i] + (b[i] - a[i]) * t) for i in range(3))


def rounded_alpha(x, y, size, radius):
    """Anti-aliased coverage of a rounded square at pixel (x, y)."""
    cx = min(max(x + 0.5, radius), size - radius)
    cy = min(max(y + 0.5, radius), size - radius)
    dx = x + 0.5 - cx
    dy = y + 0.5 - cy
    dist = (dx * dx + dy * dy) ** 0.5
    if dist <= radius - 1:
        return 1.0
    if dist >= radius:
        return 0.0
    return radius - dist


def render(size):
    radius = max(2.0, size * 0.22)
    # "T" glyph geometry, in fractions of the icon size.
    bar_top, bar_bottom = 0.28, 0.40
    stem_left, stem_right = 0.42, 0.58
    stem_bottom = 0.76

    rows = []
    for y in range(size):
        row = bytearray()
        for x in range(size):
            cov = rounded_alpha(x, y, size, radius)
            if cov <= 0:
                row += bytes((0, 0, 0, 0))
                continue
            bg = blend(BG_TOP, BG_BOTTOM, y / max(1, size - 1))
            fy = y + 0.5
            in_bar = bar_top * size <= fy <= bar_bottom * size
            in_stem = (
                stem_left * size <= x + 0.5 <= stem_right * size
                and bar_bottom * size <= fy <= stem_bottom * size
            )
            if in_bar or in_stem:
                r, g, b = FG
            else:
                r, g, b = bg
            row += bytes((r, g, b, round(255 * cov)))
        rows.append(bytes(row))
    return rows


def png(rows, size):
    raw = b"".join(b"\x00" + r for r in rows)

    def chunk(tag, data):
        body = tag + data
        return struct.pack(">I", len(data)) + body + struct.pack(">I", zlib.crc32(body))

    ihdr = struct.pack(">IIBBBBB", size, size, 8, 6, 0, 0, 0)
    return (
        b"\x89PNG\r\n\x1a\n"
        + chunk(b"IHDR", ihdr)
        + chunk(b"IDAT", zlib.compress(raw, 9))
        + chunk(b"IEND", b"")
    )


TARGETS = [
    ("admin/extensions/chrome", "icons"),
    ("admin/extensions/firefox", "icons"),
    ("admin/extensions/safari", "icons"),
    ("super_admin/extensions/chrome", "icons"),
    ("super_admin/extensions/firefox", "icons"),
    ("super_admin/extensions/safari", "icons"),
    ("white_label_admin/extensions/chrome", "icons"),
    ("white_label_admin/extensions/firefox", "icons"),
    ("white_label_admin/extensions/safari", "icons"),
    ("bots/extension", "assets"),
    ("project_party/extension", "assets"),
]
SIZES = (16, 48, 128, 256)

# Tauri reads fixed names from tauri.conf.json (bundle.icon + systemTray),
# so this tree is written by name rather than by size.
TAURI_DIR = "desktop_app/tauri/icons"
TAURI_NAMES = {
    "16x16.png": 16,
    "32x32.png": 32,
    "48x48.png": 48,
    "128x128.png": 128,
    "128x128@2x.png": 256,
    "256x256.png": 256,
    "512x512.png": 512,
    "icon.png": 512,
}

if __name__ == "__main__":
    root = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
    cache = {s: png(render(s), s) for s in sorted(set(SIZES) | set(TAURI_NAMES.values()))}
    for rel, sub in TARGETS:
        out = os.path.join(root, rel, sub)
        os.makedirs(out, exist_ok=True)
        for s in SIZES:
            with open(os.path.join(out, f"icon{s}.png"), "wb") as fh:
                fh.write(cache[s])
        print(f"wrote {len(SIZES)} icons -> {rel}/{sub}")

    out = os.path.join(root, TAURI_DIR)
    os.makedirs(out, exist_ok=True)
    for name, size in TAURI_NAMES.items():
        with open(os.path.join(out, name), "wb") as fh:
            fh.write(cache[size])
    print(f"wrote {len(TAURI_NAMES)} icons -> {TAURI_DIR}")
