#!/usr/bin/env python3
"""Mass-migrate src/v1_*.go and src/v2_*.go files into internal/api/{v1,v2}.

Performs:
  - package rename: main -> v1 / v2
  - identifier replacements: NullString/NullInt64/NullBool/NullFloat64 -> nullx.X
  - helper renames: getEnv -> config.Env, getEnvAsInt -> config.EnvAsInt,
    getEnvAsBool -> config.EnvAsBool, kilometersToMiles -> conv.KmToMi, etc.
  - response helper renames: TeslaMateAPIHandle{Error,Success,Other}Response ->
    apicommon.Handle{Error,Success,Other}Response
  - v2 helpers: v2Error/v2BadRequest -> apicommon.V2Error/V2BadRequest,
    APIErrorResponse/APIErrorBody -> apicommon.APIErrorResponse/APIErrorBody (only at type position)
  - global db/appUsersTimezone -> apicommon.DB / apicommon.AppUsersTimezone
  - apiVersion -> config.APIVersion
  - dbTimestampFormat / getTimeInTimeZone / parseDateParam -> apicommon.X
  - convertStringToBool/Float/Integer -> apicommon.ConvertStringTo{Bool,Float,Integer}
  - inserts the appropriate imports

This is a regex-based syntactic rewrite; it relies on Go's `goimports`/`gofmt`
not being applied yet — we add imports unconditionally and rely on gofmt later
to remove unused ones (or we drop unused ones manually).
"""
import os, re, sys, shutil

REPO = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# (pattern, replacement)
# Order matters: longer / more specific patterns first.
REPLACEMENTS = [
    # response helpers
    (r'\bTeslaMateAPIHandleErrorResponse\b', 'apicommon.HandleErrorResponse'),
    (r'\bTeslaMateAPIHandleSuccessResponse\b', 'apicommon.HandleSuccessResponse'),
    (r'\bTeslaMateAPIHandleOtherResponse\b', 'apicommon.HandleOtherResponse'),

    # v2 error helpers
    (r'\bv2Error\b', 'apicommon.V2Error'),
    (r'\bv2BadRequest\b', 'apicommon.V2BadRequest'),
    # APIErrorResponse / APIErrorBody as type usages -> apicommon.APIErrorResponse
    # Avoid the doc comment lines which contain `APIErrorResponse` in plain text
    # by only rewriting in code positions. We'll rewrite all and then keep doc
    # mentions as plain text — swag still resolves the @name reference.
    (r'\bAPIErrorResponse\b', 'apicommon.APIErrorResponse'),
    (r'\bAPIErrorBody\b', 'apicommon.APIErrorBody'),

    # global state
    (r'\bappUsersTimezone\b', 'apicommon.AppUsersTimezone'),
    (r'(?<!\.)(?<!\w)apiVersion\b', 'config.APIVersion'),
    # `db` variable is too short and clashes with method receivers/fields. Skip it,
    # handle case-by-case in the v2_handler.

    # config helpers
    (r'\bgetEnvAsBool\b', 'config.EnvAsBool'),
    (r'\bgetEnvAsInt\b', 'config.EnvAsInt'),
    (r'\bgetEnv\b', 'config.Env'),

    # convert helpers
    (r'\bconvertStringToBool\b', 'apicommon.ConvertStringToBool'),
    (r'\bconvertStringToFloat\b', 'apicommon.ConvertStringToFloat'),
    (r'\bconvertStringToInteger\b', 'apicommon.ConvertStringToInteger'),

    # time helpers
    (r'\bgetTimeInTimeZone\b', 'apicommon.GetTimeInTimeZone'),
    (r'\bparseDateParam\b', 'apicommon.ParseDateParam'),
    (r'\bdbTimestampFormat\b', 'apicommon.DBTimestampFormat'),

    # nullx types
    (r'\bNullInt64\b', 'nullx.Int64'),
    (r'\bNullBool\b', 'nullx.Bool'),
    (r'\bNullFloat64\b', 'nullx.Float64'),
    (r'\bNullString\b', 'nullx.String'),

    # conv helpers
    (r'\bkilometersToMilesNilSupport\b', 'conv.KmToMiNullable'),
    (r'\bkilometersToMilesInteger\b', 'conv.KmToMiInt'),
    (r'\bkilometersToMiles\b', 'conv.KmToMi'),
    (r'\bmilesToKilometers\b', 'conv.MiToKm'),
    (r'\bbarToPsi\b', 'conv.BarToPsi'),
    (r'\bcelsiusToFahrenheitNilSupport\b', 'conv.CelsiusToFahrenheitNullable'),
    (r'\bcelsiusToFahrenheit\b', 'conv.CelsiusToFahrenheit'),
    (r'\bslopeAdjustedConsumption\b', 'conv.SlopeAdjustedConsumption'),
]

# packages a transformed file may need to import
IMPORT_HINTS = {
    'apicommon.': 'github.com/tobiasehlert/teslamateapi/internal/apicommon',
    'config.':    'github.com/tobiasehlert/teslamateapi/internal/config',
    'nullx.':     'github.com/tobiasehlert/teslamateapi/internal/nullx',
    'conv.':      'github.com/tobiasehlert/teslamateapi/internal/conv',
}


def transform(text: str, target_pkg: str) -> tuple[str, set[str]]:
    """Apply replacements; return new text and set of new imports needed."""
    # rename package
    text = re.sub(r'^package main\b', f'package {target_pkg}', text, count=1, flags=re.M)

    # apply identifier replacements
    for pattern, repl in REPLACEMENTS:
        text = re.sub(pattern, repl, text)

    # determine imports needed
    needed = set()
    for prefix, import_path in IMPORT_HINTS.items():
        if prefix in text:
            needed.add(import_path)

    # swag annotation lines reference type names like `apicommon.APIErrorResponse`.
    # For schema cross-references we want them to resolve to `APIErrorResponse`
    # (the @name) — strip the `apicommon.` prefix from `// @Failure ... apicommon.APIError…`,
    # `// @Success ... apicommon.APIError…` lines but keep code-level references.
    text = re.sub(
        r'(// @(?:Success|Failure)[^\n]*?)apicommon\.(APIError(?:Response|Body))',
        r'\1\2',
        text,
    )

    return text, needed


def insert_imports(text: str, needed: set[str]) -> str:
    """Insert needed import paths into the file's import block."""
    if not needed:
        return text

    # Find an existing import block. Two cases:
    #   1) `import (` … `)`
    #   2) single-line `import "x"`
    m = re.search(r'^import \(\s*$', text, flags=re.M)
    if m:
        # multi-line block: insert before the closing `)`
        end_match = re.search(r'^\)\s*$', text[m.end():], flags=re.M)
        if not end_match:
            raise RuntimeError('could not find closing ) for import block')
        end_pos = m.end() + end_match.start()
        # extract existing imports to dedupe
        existing_block = text[m.end():end_pos]
        existing_paths = set(re.findall(r'"([^"]+)"', existing_block))
        to_add = sorted(needed - existing_paths)
        if not to_add:
            return text
        insertion = '\n'.join(f'\t"{p}"' for p in to_add) + '\n'
        return text[:end_pos] + insertion + text[end_pos:]
    else:
        # try single-line
        m = re.search(r'^import "([^"]+)"\s*$', text, flags=re.M)
        if m:
            existing = m.group(1)
            block = 'import (\n\t"' + existing + '"\n'
            for p in sorted(needed):
                block += f'\t"{p}"\n'
            block += ')'
            return text[:m.start()] + block + text[m.end():]
        # no import block at all (e.g. only `package x`); add one after package line
        m = re.search(r'^package \w+\s*$', text, flags=re.M)
        if m:
            block = '\n\nimport (\n'
            for p in sorted(needed):
                block += f'\t"{p}"\n'
            block += ')'
            return text[:m.end()] + block + text[m.end():]
    return text


def migrate(src_files, target_dir, target_pkg):
    os.makedirs(target_dir, exist_ok=True)
    for src in src_files:
        with open(src, 'r', encoding='utf-8') as f:
            text = f.read()
        new_text, needed = transform(text, target_pkg)
        new_text = insert_imports(new_text, needed)
        dst = os.path.join(target_dir, os.path.basename(src))
        with open(dst, 'w', encoding='utf-8') as f:
            f.write(new_text)
        print(f'migrated: {src} -> {dst}')


def main():
    import glob
    v1_files = sorted(glob.glob(os.path.join(REPO, 'src', 'v1_*.go')))
    v1_files.append(os.path.join(REPO, 'src', 'swagger_query_params_reference.go'))
    migrate(v1_files, os.path.join(REPO, 'internal', 'api', 'v1'), 'v1')

    v2_files = sorted(glob.glob(os.path.join(REPO, 'src', 'v2_*.go')))
    migrate(v2_files, os.path.join(REPO, 'internal', 'api', 'v2'), 'v2')

    print('migration complete')


if __name__ == '__main__':
    main()
