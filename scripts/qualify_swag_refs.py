#!/usr/bin/env python3
"""Qualify cross-package swag refs.

swag must see fully qualified type names (`apicommon.APIErrorResponse`) to
resolve a type defined in another package. Run this in-place over all
non-test Go files under internal/.

The normalize script (normalize_swagger_main_prefix.py) strips the package
prefix back out of the generated swagger.json afterwards.
"""
from __future__ import annotations

import pathlib
import re
import sys

ROOT = pathlib.Path("internal")

PATTERN = re.compile(
    r"(// @(?:Success|Failure)[^\n]*\{object\}\s+)(APIError(?:Response|Body))\b"
)


def fix(path: pathlib.Path) -> bool:
    text = path.read_text()
    new_text = PATTERN.sub(r"\1apicommon.\2", text)
    if text != new_text:
        path.write_text(new_text)
        return True
    return False


def main() -> int:
    changed = 0
    for p in ROOT.rglob("*.go"):
        if p.name.endswith("_test.go"):
            continue
        if fix(p):
            changed += 1
            print(f"updated {p}")
    print(f"done: {changed} file(s) updated")
    return 0


if __name__ == "__main__":
    sys.exit(main())
