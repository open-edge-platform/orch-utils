#!/usr/bin/env bash
# SPDX-FileCopyrightText: 2026 Intel Corporation
# SPDX-License-Identifier: Apache-2.0
#
# Integration test runner for tenancy-manager.
#
# Starts a throwaway Postgres container, builds the binary, runs the server,
# executes curl-based smoke tests, then tears everything down.
#
# Usage:
#   ./run.sh [--skip-build]   # --skip-build reuses an existing binary in /tmp
#
# Requirements: docker, asdf (golang plugin), curl

set -euo pipefail

# Source asdf so its shims are on PATH and .tool-versions is respected.
# shellcheck source=/dev/null
[[ -f "${HOME}/.asdf/asdf.sh" ]] && source "${HOME}/.asdf/asdf.sh"

# ── Colours ────────────────────────────────────────────────────────────────────
RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[1;33m'; CYAN='\033[0;36m'; NC='\033[0m'
log()  { echo -e "${GREEN}[$(date +%H:%M:%S)]${NC} $*"; }
warn() { echo -e "${YELLOW}[$(date +%H:%M:%S)] WARN:${NC} $*"; }
err()  { echo -e "${RED}[$(date +%H:%M:%S)] ERROR:${NC} $*" >&2; }
step() { echo -e "\n${CYAN}━━━ $* ━━━${NC}"; }

# ── Config ─────────────────────────────────────────────────────────────────────
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"

PG_CONTAINER="tm-integration-postgres"
PG_PORT=15432
PG_USER=tenancy
PG_PASS=testpass
PG_DB=tenancy
DB_URL="postgres://${PG_USER}:${PG_PASS}@localhost:${PG_PORT}/${PG_DB}?sslmode=disable"

TM_PORT=18080
TM_BIN="${REPO_DIR}/bin/tenancy-manager"
TM_PID=""

SKIP_BUILD=false
[[ "${1:-}" == "--skip-build" ]] && SKIP_BUILD=true

PASS=0
FAIL=0

# ── Cleanup ────────────────────────────────────────────────────────────────────
cleanup() {
    step "Cleanup"
    if [[ -n "${TM_PID}" ]] && kill -0 "${TM_PID}" 2>/dev/null; then
        log "Stopping tenancy-manager (pid ${TM_PID})"
        kill "${TM_PID}" 2>/dev/null || true
        wait "${TM_PID}" 2>/dev/null || true
    fi
    if docker inspect "${PG_CONTAINER}" &>/dev/null; then
        log "Removing Postgres container"
        docker rm -f "${PG_CONTAINER}" >/dev/null
    fi
    echo ""
    if (( FAIL == 0 )); then
        echo -e "${GREEN}✓ ${PASS} passed, ${FAIL} failed${NC}"
    else
        echo -e "${RED}✗ ${PASS} passed, ${FAIL} failed${NC}"
    fi
}
trap cleanup EXIT

# ── Helpers ────────────────────────────────────────────────────────────────────
check() {
    local desc="$1" expected="$2" got="$3"
    if [[ "${got}" == *"${expected}"* ]]; then
        echo -e "  ${GREEN}PASS${NC}: ${desc}"
        PASS=$(( PASS + 1 ))
    else
        echo -e "  ${RED}FAIL${NC}: ${desc}"
        echo    "        want: ${expected}"
        echo    "         got: ${got}"
        FAIL=$(( FAIL + 1 ))
    fi
}

wait_for() {
    local desc="$1" url="$2" retries="${3:-20}"
    for i in $(seq 1 "${retries}"); do
        if curl -sf "${url}" >/dev/null 2>&1; then
            log "${desc} is ready"
            return 0
        fi
        sleep 1
    done
    err "${desc} did not become ready after ${retries}s"
    return 1
}

TM="http://localhost:${TM_PORT}"

# ── Prerequisites ──────────────────────────────────────────────────────────────
step "Prerequisites"
for cmd in docker mage curl; do
    command -v "${cmd}" >/dev/null 2>&1 || { err "Missing: ${cmd}"; exit 1; }
done
log "All prerequisites present (mage: $(mage --version 2>&1 | head -1))"

# ── Build ──────────────────────────────────────────────────────────────────────
step "Build"
if [[ "${SKIP_BUILD}" == true ]] && [[ -x "${TM_BIN}" ]]; then
    warn "Skipping build (--skip-build), using existing binary"
else
    log "Building tenancy-manager..."
    ( cd "${REPO_DIR}" && mage Binary:Build )
    log "Build OK → ${TM_BIN}"
fi

# ── Postgres ───────────────────────────────────────────────────────────────────
step "Postgres"
if docker inspect "${PG_CONTAINER}" &>/dev/null; then
    log "Removing stale container"
    docker rm -f "${PG_CONTAINER}" >/dev/null
fi

log "Starting Postgres on port ${PG_PORT}..."
docker run -d --rm \
    --name "${PG_CONTAINER}" \
    -e POSTGRES_USER="${PG_USER}" \
    -e POSTGRES_PASSWORD="${PG_PASS}" \
    -e POSTGRES_DB="${PG_DB}" \
    -p "${PG_PORT}:5432" \
    postgres:15-alpine >/dev/null

log "Waiting for Postgres to accept connections..."
PG_READY=false
for i in $(seq 1 30); do
    if docker exec "${PG_CONTAINER}" pg_isready -U "${PG_USER}" >/dev/null 2>&1; then
        PG_READY=true
        break
    fi
    sleep 1
done
[[ "${PG_READY}" == true ]] || { err "Postgres did not become ready after 30s"; exit 1; }
log "Postgres ready"

# ── Start server ───────────────────────────────────────────────────────────────
step "Start tenancy-manager"
log "Launching server on :${TM_PORT} (auth disabled — no OIDC_SERVER_URL)"
DATABASE_URL="${DB_URL}" "${TM_BIN}" \
    -listen ":${TM_PORT}" \
    -log-level debug \
    > /tmp/tm-integration.log 2>&1 &
TM_PID=$!

wait_for "tenancy-manager" "${TM}/healthz" 20
log "Server ready (pid ${TM_PID})"

# ── Tests ──────────────────────────────────────────────────────────────────────
step "Health"
check "GET /healthz → 200" \
    "200" "$(curl -so /dev/null -w '%{http_code}' "${TM}/healthz")"

step "Org CRUD"
check "PUT /v1/orgs/test-org → 200 (create)" \
    "200" "$(curl -so /dev/null -w '%{http_code}' \
        -X PUT -H 'Content-Type: application/json' \
        -d '{"description":"integration test org"}' \
        "${TM}/v1/orgs/test-org")"

check "GET /v1/orgs/test-org → 200" \
    "200" "$(curl -so /dev/null -w '%{http_code}' "${TM}/v1/orgs/test-org")"

check "GET /v1/orgs/test-org body contains name" \
    '"name":"test-org"' "$(curl -s "${TM}/v1/orgs/test-org")"

check "GET /v1/orgs/test-org body contains description" \
    '"description":"integration test org"' "$(curl -s "${TM}/v1/orgs/test-org")"

check "GET /v1/orgs → list includes test-org" \
    "test-org" "$(curl -s "${TM}/v1/orgs")"

check "PUT /v1/orgs/test-org → 200 (update description)" \
    "200" "$(curl -so /dev/null -w '%{http_code}' \
        -X PUT -H 'Content-Type: application/json' \
        -d '{"description":"updated"}' \
        "${TM}/v1/orgs/test-org")"

check "GET /v1/orgs/test-org body has updated description" \
    '"description":"updated"' "$(curl -s "${TM}/v1/orgs/test-org")"

step "Project CRUD"
check "PUT /v1/projects/test-proj?org=test-org → 200 (create)" \
    "200" "$(curl -so /dev/null -w '%{http_code}' \
        -X PUT -H 'Content-Type: application/json' \
        -d '{"description":"integration test project"}' \
        "${TM}/v1/projects/test-proj?org=test-org")"

check "GET /v1/projects/test-proj?org=test-org → 200" \
    "200" "$(curl -so /dev/null -w '%{http_code}' "${TM}/v1/projects/test-proj?org=test-org")"

check "GET /v1/projects/test-proj body contains name" \
    '"name":"test-proj"' "$(curl -s "${TM}/v1/projects/test-proj?org=test-org")"

check "GET /v1/projects → list includes test-proj" \
    "test-proj" "$(curl -s "${TM}/v1/projects")"

check "DELETE /v1/projects/test-proj?org=test-org → 200" \
    "200" "$(curl -so /dev/null -w '%{http_code}' \
        -X DELETE "${TM}/v1/projects/test-proj?org=test-org")"

step "Error handling"
check "GET /v1/orgs/does-not-exist → 404" \
    "404" "$(curl -so /dev/null -w '%{http_code}' "${TM}/v1/orgs/does-not-exist")"

NOT_FOUND_BODY=$(curl -s "${TM}/v1/orgs/does-not-exist")
check "404 body is safe (no raw DB error)" \
    '"error":"org not found"' "${NOT_FOUND_BODY}"
check "404 body contains no internal details (no 'ent:' / 'pq:')" \
    "1" "$(echo "${NOT_FOUND_BODY}" | grep -cv 'ent:\|pq:\|sql:')"

step "Request body limits"
OVERSIZED=$(head -c 70000 /dev/urandom | base64 | tr -d '\n')
check "PUT with body > 64 KB → 400" \
    "400" "$(curl -so /dev/null -w '%{http_code}' \
        -X PUT -H 'Content-Type: application/json' \
        -d "{\"description\":\"${OVERSIZED}\"}" \
        "${TM}/v1/orgs/too-big")"

step "Query param validation"
check "GET /v1/events?controller=x&after=notanumber → 400" \
    "400" "$(curl -so /dev/null -w '%{http_code}' \
        "${TM}/v1/events?controller=x&after=notanumber")"

check "GET /v1/events?controller=x&limit=0 → 400" \
    "400" "$(curl -so /dev/null -w '%{http_code}' \
        "${TM}/v1/events?controller=x&limit=0")"

check "GET /v1/events?controller=x&limit=9999 → 400" \
    "400" "$(curl -so /dev/null -w '%{http_code}' \
        "${TM}/v1/events?controller=x&limit=9999")"

check "GET /v1/events?controller=x → 200 (valid)" \
    "200" "$(curl -so /dev/null -w '%{http_code}' \
        "${TM}/v1/events?controller=x")"

step "Cleanup test data"
check "DELETE /v1/orgs/test-org → 200" \
    "200" "$(curl -so /dev/null -w '%{http_code}' \
        -X DELETE "${TM}/v1/orgs/test-org")"

# Final result is printed by the cleanup trap.
[[ ${FAIL} -eq 0 ]]
