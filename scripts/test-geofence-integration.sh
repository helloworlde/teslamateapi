#!/usr/bin/env bash
set -euo pipefail
umask 077

# An owned, Unix-socket-only PostgreSQL cluster. Never use DATABASE_* or an
# existing server, and never install packages or start system services here.
repo_dir=$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)
pg_bin=${PG_BIN:-}
if [[ -z "$pg_bin" ]] && command -v pg_config >/dev/null 2>&1; then
    pg_bin=$(pg_config --bindir)
fi
for tool in initdb pg_ctl createdb; do
    if [[ -z "$pg_bin" || ! -x "$pg_bin/$tool" ]]; then
        printf 'PostgreSQL server tools are required; set PG_BIN to their directory (missing %s).\n' "$tool" >&2
        exit 1
    fi
done
if [[ $(id -u) == 0 ]]; then
    printf 'Run integration tests as a non-root user.\n' >&2
    exit 1
fi

# A short path also fits macOS's Unix-domain socket path limit.
work_dir=$(mktemp -d /tmp/teslamateapi-geofence.XXXXXX)
readonly work_dir
cluster_dir="$work_dir/data"
cleanup() {
    local result=$?
    trap - EXIT INT TERM
    if [[ -s "$cluster_dir/postmaster.pid" ]]; then
        if ! "$pg_bin/pg_ctl" -D "$cluster_dir" -m fast -w stop >>"$work_dir/lifecycle.log" 2>&1; then
            printf 'Could not stop owned test cluster; inspect %s\n' "$work_dir" >&2
            cat "$work_dir/lifecycle.log" >&2
            exit 1
        fi
    fi
    if [[ $result == 0 ]]; then
        rm -rf -- "$work_dir"
    else
        printf 'Integration test failed; logs retained at %s (cluster stopped).\n' "$work_dir" >&2
        if [[ -f "$work_dir/postgres.log" ]]; then
            tail -40 "$work_dir/postgres.log" >&2
        elif [[ -f "$work_dir/initdb.log" ]]; then
            tail -40 "$work_dir/initdb.log" >&2
        fi
    fi
    exit "$result"
}
trap cleanup EXIT
trap 'exit 130' INT
trap 'exit 143' TERM

"$pg_bin/initdb" -D "$cluster_dir" --username=geofence_test \
    --auth-local=trust --auth-host=reject --encoding=UTF8 --no-locale >"$work_dir/initdb.log" 2>&1
cat >>"$cluster_dir/postgresql.conf" <<CONFIG
listen_addresses = ''
unix_socket_directories = '$work_dir'
port = 5432
jit = off
CONFIG
"$pg_bin/pg_ctl" -D "$cluster_dir" -l "$work_dir/postgres.log" -w start >"$work_dir/lifecycle.log" 2>&1
"$pg_bin/createdb" -h "$work_dir" -p 5432 -U geofence_test \
    --maintenance-db=postgres teslamateapi_geofence_test

cd "$repo_dir"
TESLAMATEAPI_GEOFENCE_TEST_SOCKET="$work_dir" \
    go test -tags=integration -count=1 -v ./internal/httpapi/handlers/v2 \
    -run '^TestStatsByGeofenceIntegration' "$@"
