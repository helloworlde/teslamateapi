#!/usr/bin/env python3
"""Replace bare `db.X(` with `apicommon.DB.X(` in migrated v1 files."""
import re, glob, os

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

V1 = sorted(glob.glob(os.path.join(REPO, 'internal/api/v1/*.go')))

for p in V1:
    with open(p) as f:
        t = f.read()
    new = re.sub(r'\bdb\.(Query|Exec|Prepare|Ping|Close)', r'apicommon.DB.\1', t)
    if new != t:
        with open(p, 'w') as f:
            f.write(new)
        print('updated', p)
