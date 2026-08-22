#!/usr/bin/env python3
"""Swift brace/paren/bracket balance checker (string/comment-aware tokenizer).

swiftc is NOT installed in this environment, so this verifies that every
.swift file we create/modify has balanced (), {}, [] after stripping out
line comments, block comments, and string/character/interpolation literals.

Exit code 0 == all files balanced; 1 == imbalance detected.
"""
from __future__ import annotations

import sys
from pathlib import Path

PAIRS = {")": "(", "}": "{", "]": "["}


def scan(source: str) -> tuple[dict[str, int], list[str]]:
    """Tokenize Swift source, returning open-bracket counts + error list.

    Strips: // line comments, /* */ block comments, "..." and \"""...\"""
    multi-line + single-line string literals (including \\(...) interpolation,
    which is scanned as a self-contained balanced region), 'c' char literals
    (best-effort), and raw string delimiters #"..."#" (extended). Balanced
    pairs are tracked on the residual code.
    """
    counts = {"(": 0, "{": 0, "[": 0}
    errors: list[str] = []
    i, n = 0, len(source)
    line = 1

    def push(c: str) -> None:
        if c in counts:
            counts[c] += 1

    def pop(c: str, at_line: int) -> None:
        open_c = PAIRS.get(c)
        if open_c is None:
            return
        if counts[open_c] <= 0:
            errors.append(f"line {at_line}: unexpected '{c}' (no matching '{open_c}')")
        else:
            counts[open_c] -= 1

    # Recursively scan a balanced \\(...) interpolation region starting just
    # AFTER the '('; returns the index past the matching ')'. The contents are
    # part of the string literal, so they do NOT touch the global brace counts —
    # the scanner only tracks local `(` `)` depth (plus skipping nested strings
    # and comments) to locate the real close paren.
    def scan_interpolation(start: int) -> int:
        nonlocal line
        j = start
        depth = 1  # we entered after '('
        while j < n and depth > 0:
            c = source[j]
            nxt = source[j + 1] if j + 1 < n else ""
            if c == "\n":
                line += 1; j += 1; continue
            if c == "/" and nxt == "/":
                while j < n and source[j] != "\n":
                    j += 1
                continue
            if c == "/" and nxt == "*":
                j += 2
                while j < n:
                    if source[j] == "\n":
                        line += 1
                    if source[j] == "*" and j + 1 < n and source[j + 1] == "/":
                        j += 2; break
                    j += 1
                else:
                    errors.append(f"line {line}: unterminated block comment in interpolation")
                continue
            if c == '"' and source[j:j + 3] == '"""':
                j += 3
                while j < n:
                    if source[j] == "\n":
                        line += 1
                    if source[j:j + 3] == '"""':
                        j += 3; break
                    j += 1
                else:
                    errors.append(f"line {line}: unterminated triple-quoted string in interpolation")
                continue
            if c == '"':
                j += 1
                while j < n and source[j] != '"':
                    if source[j] == "\n":
                        line += 1
                    if source[j] == "\\" and j + 1 < n:
                        if source[j + 1] == "(":
                            j = scan_interpolation(j + 2)
                            continue
                        j += 2; continue
                    j += 1
                j += 1
                continue
            if c == "(":
                depth += 1
            elif c == ")":
                depth -= 1
            j += 1
        return j

    while i < n:
        c = source[i]
        nxt = source[i + 1] if i + 1 < n else ""

        if c == "\n":
            line += 1; i += 1; continue

        # Line comment.
        if c == "/" and nxt == "/":
            while i < n and source[i] != "\n":
                i += 1
            continue

        # Block comment.
        if c == "/" and nxt == "*":
            i += 2
            while i < n:
                if source[i] == "\n":
                    line += 1
                if source[i] == "*" and i + 1 < n and source[i + 1] == "/":
                    i += 2; break
                i += 1
            else:
                errors.append(f"line {line}: unterminated block comment")
            continue

        # Triple-quoted (multi-line) string. Interpolation is possible here too
        # but rare; treat content verbatim except the closing delimiter.
        if c == '"' and source[i:i + 3] == '"""':
            i += 3
            while i < n:
                if source[i] == "\n":
                    line += 1
                if source[i:i + 3] == '"""':
                    i += 3; break
                i += 1
            else:
                errors.append(f"line {line}: unterminated triple-quoted string")
            continue

        # Raw (extended) string #"..."#" / ##"..."##.
        if c == "#":
            j = i
            while j < n and source[j] == "#":
                j += 1
            if j < n and source[j] == '"':
                close_delim = '"' + "#" * (j - i)
                i = j + 1
                while i < n:
                    if source[i] == "\n":
                        line += 1
                    if source[i:i + len(close_delim)] == close_delim:
                        i += len(close_delim); break
                    i += 1
                else:
                    errors.append(f"line {line}: unterminated raw string")
                continue

        # Single-quoted / char literal (best-effort skip).
        if c == "'":
            i += 1
            while i < n and source[i] != "'":
                if source[i] == "\\" and i + 1 < n:
                    i += 2; continue
                i += 1
            i += 1
            continue

        # Double-quoted single-line string (handles \\(...) interpolation).
        if c == '"':
            i += 1
            while i < n and source[i] != '"':
                if source[i] == "\n":
                    line += 1
                if source[i] == "\\" and i + 1 < n:
                    if source[i + 1] == "(":
                        # Scan the balanced interpolation (does not touch global
                        # counts for the wrapping interpolation parens).
                        i = scan_interpolation(i + 2)
                        continue
                    i += 2; continue
                i += 1
            i += 1
            continue

        if c in "({[":
            push(c)
        elif c in ")}]":
            pop(c, line)

        i += 1

    return counts, errors


def check_file(path: Path) -> bool:
    src = path.read_text(encoding="utf-8")
    counts, errors = scan(src)
    ok = not errors and all(v == 0 for v in counts.values())
    status = "OK" if ok else "FAIL"
    print(f"[{status}] {path}")
    for e in errors:
        print(f"        {e}")
    for open_c, v in counts.items():
        if v != 0:
            close_c = {v: k for k, v in PAIRS.items()}.get(open_c, "?")
            print(f"        unbalanced '{open_c}{close_c}' delta={v}")
    return ok


def main(argv: list[str]) -> int:
    paths = [Path(a) for a in argv[1:]] if len(argv) > 1 else \
        sorted(Path("App").glob("*.swift"))
    if not paths:
        print("no .swift files found")
        return 1
    all_ok = True
    for p in paths:
        all_ok = check_file(p) and all_ok
    print("\n" + ("ALL BALANCED" if all_ok else "IMBALANCE DETECTED"))
    return 0 if all_ok else 1


if __name__ == "__main__":
    sys.exit(main(sys.argv))
