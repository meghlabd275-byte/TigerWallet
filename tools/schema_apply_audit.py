#!/usr/bin/env python3
"""Apply every Go module's embedded DDL to a real PostgreSQL database.

Preparing a statement only catches syntax and column errors at the statement
level. It does not catch:
  * a multi-statement script where a later statement references a table the
    script has not created yet (the whole Exec aborts),
  * a script whose first statement is syntactically invalid,
  * an index created on a table that never exists.
This tool extracts the DDL string literals from each module, applies them in a
scratch schema, and reports what fails.

Usage: python3 tools/schema_apply_audit.py [--db URL] [module ...]
"""
from __future__ import annotations

import argparse
import os
import re
import sys
import time

import pg8000

DDL_START = re.compile(
    r"\s*(?:"
    r"CREATE\s+(?:OR\s+REPLACE\s+)?(?:TABLE|INDEX|UNIQUE\s+INDEX|SCHEMA|EXTENSION|VIEW|"
    r"MATERIALIZED\s+VIEW|FUNCTION|TRIGGER|TYPE|SEQUENCE|DATABASE|POLICY|ROLE)"
    r"|ALTER\s+(?:TABLE|INDEX|SEQUENCE|TYPE|VIEW|POLICY|ROLE|DATABASE)"
    r"|DROP\s+(?:TABLE|INDEX|SCHEMA|VIEW|TYPE|SEQUENCE|POLICY|TRIGGER|FUNCTION)"
    r")\b",
    re.I,
)


def go_string_literals(src: str):
    """Yield (value, kind) for backtick and double-quoted Go string literals."""
    i, n = 0, len(src)
    while i < n:
        ch = src[i]
        if ch == "`":
            j = src.find("`", i + 1)
            if j == -1:
                return
            yield src[i + 1 : j], "raw"
            i = j + 1
        elif ch == '"':
            j = i + 1
            out = []
            while j < n and src[j] != '"':
                if src[j] == "\\" and j + 1 < n:
                    out.append(src[j + 1])
                    j += 2
                    continue
                out.append(src[j])
                j += 1
            yield "".join(out), "quoted"
            i = j + 1
        else:
            i += 1


def strip_comments(sql: str) -> str:
    sql = re.sub(r"--[^\n]*", "", sql)
    return sql


def split_statements(sql: str):
    """Split on newline-terminated semicolons (the shape used in these repos)."""
    return [s for s in sql.split(";\n") if s.strip()]


def ddl_literals(src: str):
    """Return DDL literals, filtering out fragments and format templates."""
    out = []
    for value, kind in go_string_literals(src):
        v = value.strip()
        if not DDL_START.match(v):
            continue
        if "fmt.Sprintf" in v or "%s" in v or "%v" in v:
            continue
        out.append(v)
    return out


# Files that must be applied after everything else in their module, because
# their DDL depends on tables defined elsewhere in the same module. The value
# lists file basenames in application order.
APPLY_LAST = {
    "finance_ledger.go": ["store.go"],
}


def order_scripts(scripts):
    """Sort (path, literal) pairs so module dependencies are applied first."""
    def rank(item):
        path = item[0]
        base = os.path.basename(path)
        for last, after in APPLY_LAST.items():
            if base == last:
                return 2
            if base in after:
                return 0
        return 1

    return sorted(scripts, key=rank)


def find_modules(root: str):
    for dirpath, dirnames, filenames in os.walk(root):
        dirnames[:] = [
            d
            for d in dirnames
            if d not in {"node_modules", "vendor", ".git", "target", "dist", "build"}
        ]
        if "go.mod" in filenames:
            yield dirpath


def apply_script(cur, statements, debug=None, allow_retries=True):
    """Apply statements, in order, reporting the first failure.

    allow_retries mirrors how a module runs its own statements. Modules that
    execute one statement per Exec abort on the first failure, so a script that
    only succeeds after reordering is a real defect: retries are off by default
    so the audit reports it. Pass allow_retries=True only to model a runner that
    already retries in passes.
    """
    pending = [s for s in statements if s.strip()]
    passes = 0
    last_err = None
    while pending:
        passes += 1
        if passes > 12:
            return False, passes, f"no convergence after {passes} passes: {last_err}"
        remaining, lasterr = [], None
        for s in pending:
            try:
                cur.execute(s)
            except Exception as e:  # noqa: BLE001 - reporting raw driver error
                if not allow_retries:
                    if debug is not None:
                        debug.append(s)
                    return False, passes, str(e)
                remaining.append(s)
                lasterr = e
        if len(remaining) == len(pending):
            if debug is not None:
                debug.extend(remaining)
            return False, passes, str(lasterr)
        pending = remaining
        last_err = lasterr
    return True, passes, None


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--db", default=os.environ.get("TEST_DATABASE_URL") or os.environ.get("DATABASE_URL"))
    ap.add_argument("--root", default=".")
    ap.add_argument("--include-tests", action="store_true", help="also scan *_test.go DDL")
    ap.add_argument(
        "--retry",
        action="store_true",
        help="model a runner that retries statements in passes; off by default so "
        "scripts whose order only works after reordering are reported",
    )
    ap.add_argument("modules", nargs="*")
    args = ap.parse_args()

    if not args.db:
        print("no database url (set TEST_DATABASE_URL or pass --db)")
        return 2

    conn = pg8000.connect(**parse_dsn(args.db))
    conn.autocommit = True
    cur = conn.cursor()

    modules = args.modules or sorted(find_modules(args.root))
    bad = []
    counter = [0]
    for mod in modules:
        scripts = []
        for dirpath, dirnames, filenames in os.walk(mod):
            dirnames[:] = [d for d in dirnames if d not in {"node_modules", "vendor", ".git", "target", "dist", "build"}]
            for fn in filenames:
                if not fn.endswith(".go"):
                    continue
                if fn.endswith("_test.go") and not args.include_tests:
                    continue
                path = os.path.join(dirpath, fn)
                try:
                    src = open(path, encoding="utf-8", errors="replace").read()
                except OSError:
                    continue
                for lit in ddl_literals(src):
                    if lit.count(";") >= 1:
                        scripts.append((path, lit))

        if not scripts:
            continue

        counter[0] += 1
        schema = f"aud_{counter[0]:04d}_{os.getpid()}"
        cur.execute(f'DROP SCHEMA IF EXISTS "{schema}" CASCADE')
        cur.execute(f'CREATE SCHEMA "{schema}"')
        # pgcrypto is created in public, but public stays OUT of the search path
        # so tables another module already created there cannot mask a failure.
        cur.execute("CREATE EXTENSION IF NOT EXISTS pgcrypto")
        cur.execute(f'SET search_path TO "{schema}"')

        # A module's literals are applied as one pool in discovery order, which
        # matches how the module applies them at runtime. Applying each literal
        # on its own would report a false failure for any schema that legitimately
        # depends on a table another literal in the same module creates.
        pool = []
        for path, lit in order_scripts(scripts):
            for stmt in split_statements(strip_comments(lit)):
                pool.append((path, stmt))
        if not pool:
            cur.execute("RESET search_path")
            cur.execute(f'DROP SCHEMA IF EXISTS "{schema}" CASCADE')
            continue
        stmts = [s for _, s in pool]

        if args.retry:
            # Model a runner that retries in passes: only hard dependencies fail.
            dbg = []
            ok, passes, err = apply_script(cur, stmts, dbg, allow_retries=True)
            if not ok:
                origin = next((p for p, s in pool if s in set(dbg)), scripts[0][0])
                bad.append((mod, origin, "missing dependency", err, dbg))
            continue

        # Strict pass: apply in order and stop at the first failure. If the
        # statement succeeds once the rest of the module's statements have run,
        # the problem is statement order; otherwise a dependency is missing.
        first_fail, fail_idx = None, None
        for i, s in enumerate(stmts):
            try:
                cur.execute(s)
            except Exception as e:  # noqa: BLE001
                first_fail, fail_idx, err = s, i, e
                break
        if first_fail is None:
            continue
        for s in stmts[fail_idx + 1 :]:
            try:
                cur.execute(s)
            except Exception:  # noqa: BLE001 - secondary failures are not the subject
                pass
        try:
            cur.execute(first_fail)
            kind = "statement order"
        except Exception as e:  # noqa: BLE001
            kind = "missing dependency"
            err = e
        origin = next((p for p, s in pool if s == first_fail), scripts[0][0])
        bad.append((mod, origin, kind, err, [first_fail]))

        cur.execute("RESET search_path")
        cur.execute(f'DROP SCHEMA IF EXISTS "{schema}" CASCADE')

    if bad:
        print(f"\n{len(bad)} failing script(s):\n")
        for mod, path, kind, err, dbg in bad:
            print(f"  [{kind}] {mod}\n    {path}\n    error={str(err)[:200]}")
            for s in dbg:
                print(f"      STUCK: {' '.join(s.split())[:180]}")
            print()
        return 1

    print(f"all schema scripts applied cleanly across {len(modules)} module(s)")
    return 0


def parse_dsn(dsn: str) -> dict:
    m = re.match(r"postgres(?:ql)?://([^:]+)(?::([^@]*))?@([^:/]+):(\d+)/([^?]+)", dsn)
    if not m:
        raise SystemExit(f"cannot parse dsn: {dsn}")
    user, pwd, host, port, db = m.groups()
    return {"user": user, "password": pwd or "", "host": host, "port": int(port), "database": db}


if __name__ == "__main__":
    sys.exit(main())
