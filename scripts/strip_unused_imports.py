#!/usr/bin/env python3
"""Strip imports that are not referenced by any qualified identifier in the file.

Targets the four internal packages we added (apicommon, config, nullx, conv) plus
common stdlib packages frequently rendered unused by the migration regex.

Heuristic: an import path "x/y/foo" is used if `foo.` appears in the file body
outside the import block. For aliased imports `bar "..."`, we use the alias.
"""
import re, glob, os, sys

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

CHECK_PACKAGES = {
    'apicommon', 'config', 'nullx', 'conv',
    'http', 'fmt', 'strconv', 'strings', 'time', 'json', 'log',
    'embed', 'errors', 'sql', 'driver', 'context', 'os', 'os/signal',
    'gin', 'pq', 'gzip', 'base64', 'filepath', 'slices', 'atomic',
    'scalar', 'scalargin',
}

IMPORT_BLOCK_RE = re.compile(r'^import \(\s*\n((?:[^\)]|\n)*?)\n\)\s*$', re.M)

def trim_file(path):
    with open(path) as f:
        text = f.read()

    m = IMPORT_BLOCK_RE.search(text)
    if not m:
        return False
    block = m.group(1)
    body_start = m.end()
    body = text[body_start:]

    new_lines = []
    changed = False
    for line in block.split('\n'):
        s = line.strip()
        if not s or s.startswith('//'):
            new_lines.append(line)
            continue
        # parse `alias "path"` or `_ "path"` or `"path"`
        match_imp = re.match(r'(?:(\w+|_|\.)\s+)?"([^"]+)"', s)
        if not match_imp:
            new_lines.append(line)
            continue
        alias, path_str = match_imp.group(1), match_imp.group(2)
        # skip blank-import (`_`) and dot-import (`.`)
        if alias in ('_', '.'):
            new_lines.append(line)
            continue
        ident = alias if alias else path_str.rsplit('/', 1)[-1]
        # only check our list of well-known short package names
        if ident not in CHECK_PACKAGES:
            new_lines.append(line)
            continue
        # search for `ident.` in the body (outside imports)
        if re.search(r'\b' + re.escape(ident) + r'\.', body):
            new_lines.append(line)
        else:
            changed = True

    if not changed:
        return False

    new_block = '\n'.join(new_lines).rstrip()
    new_text = text[:m.start()] + 'import (\n' + new_block + '\n)' + body
    with open(path, 'w') as f:
        f.write(new_text)
    return True

def main():
    files = (
        glob.glob(os.path.join(REPO, 'internal/api/v1/*.go'))
        + glob.glob(os.path.join(REPO, 'internal/api/v2/*.go'))
        + glob.glob(os.path.join(REPO, 'internal/docs/*.go'))
    )
    n = 0
    for p in files:
        if trim_file(p):
            n += 1
            print('trimmed', p)
    print(f'{n} files trimmed')

if __name__ == '__main__':
    main()
