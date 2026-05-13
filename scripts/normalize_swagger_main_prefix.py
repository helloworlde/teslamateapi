#!/usr/bin/env python3
"""Strip stray Go package prefixes (e.g. `main.`, `apicommon.`, `v2.`) from
swag-generated swagger.json / swagger.yaml definition keys & $ref values.

swag may emit definitions like `main.APIErrorResponse` when models are declared
in the package containing the @generalInfo file, or `apicommon.APIErrorResponse`
when the model is defined in another package. Scalar / OpenAPI tooling treats
those as opaque names; we keep only the bare type name so refs match.

Usage: normalize_swagger_main_prefix.py <docs_out_dir>
"""
from __future__ import annotations

import json
import re
import sys
from pathlib import Path

# Package prefixes we want to strip when they appear before a definition name.
# Order matters: longer / more-specific first is irrelevant here because all
# alternatives are case-sensitive package identifiers without overlap.
_PKG_PREFIXES = (
    "main",
    "apicommon",
    "v1",
    "v2",
    "docs",
    "server",
    "config",
    "conv",
    "nullx",
)
_PREFIX_RE = re.compile(r"\b(?:" + "|".join(_PKG_PREFIXES) + r")\.(?=[A-Z])")


def _strip(text: str) -> str:
    return _PREFIX_RE.sub("", text)


def _rewrite_json(path: Path) -> None:
    raw = path.read_text(encoding="utf-8")
    cleaned = _strip(raw)
    # ensure the JSON still parses round-trip
    json.loads(cleaned)
    path.write_text(cleaned, encoding="utf-8")


def _rewrite_text(path: Path) -> None:
    raw = path.read_text(encoding="utf-8")
    cleaned = _strip(raw)
    path.write_text(cleaned, encoding="utf-8")


def main(argv: list[str]) -> int:
    if len(argv) != 2:
        print("usage: normalize_swagger_main_prefix.py <docs_out_dir>", file=sys.stderr)
        return 2
    out = Path(argv[1])
    json_path = out / "swagger.json"
    yaml_path = out / "swagger.yaml"
    if json_path.exists():
        _rewrite_json(json_path)
        print(f"[normalize] {json_path}")
    if yaml_path.exists():
        _rewrite_text(yaml_path)
        print(f"[normalize] {yaml_path}")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
