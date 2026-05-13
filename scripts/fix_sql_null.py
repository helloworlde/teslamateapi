#!/usr/bin/env python3
"""Restore stdlib `sql.NullX` types that the migration regex
mistakenly rewrote as `sql.nullx.X`."""
import re, glob, os

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
FILES = (
    glob.glob(os.path.join(REPO, 'internal/api/v1/*.go'))
    + glob.glob(os.path.join(REPO, 'internal/api/v2/*.go'))
)

for p in FILES:
    with open(p) as f:
        t = f.read()
    new = t
    new = re.sub(r'sql\.nullx\.Int64', 'sql.NullInt64', new)
    new = re.sub(r'sql\.nullx\.Float64', 'sql.NullFloat64', new)
    new = re.sub(r'sql\.nullx\.Bool', 'sql.NullBool', new)
    new = re.sub(r'sql\.nullx\.String', 'sql.NullString', new)
    if new != t:
        with open(p, 'w') as f:
            f.write(new)
        print('fixed', p)
