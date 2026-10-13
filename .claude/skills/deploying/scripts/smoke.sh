#!/usr/bin/env bash
# Read-only post-deploy smoke test for $pend. Nothing here writes to the API or the database.
# Usage: smoke.sh <api-origin> <web-origin>   e.g. smoke.sh https://x.onrender.com https://y.vercel.app
set -u

if [ $# -ne 2 ]; then
  echo "usage: $0 <api-origin> <web-origin>" >&2
  exit 2
fi
api=${1%/}
web=${2%/}
failed=0
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

pass() { printf 'PASS  %s\n' "$1"; }
fail() { printf 'FAIL  %s\n' "$1"; failed=1; }

# A sleeping free instance needs 30-60s to wake, so the first request gets retries and a long timeout.
code=$(curl -sS -o /dev/null -w '%{http_code}' --max-time 90 --retry 3 --retry-delay 5 "$api/healthz")
[ "$code" = 200 ] && pass "API /healthz -> 200" || fail "API /healthz -> $code (want 200)"

code=$(curl -sS -D "$tmp/refresh.h" -o "$tmp/refresh.b" -w '%{http_code}' --max-time 30 \
  -X POST -H "Origin: $web" "$api/api/v1/refresh")
if [ "$code" = 401 ] && grep -q '"success":false' "$tmp/refresh.b"; then
  pass "API POST /api/v1/refresh without a cookie -> 401 envelope"
else
  fail "API POST /api/v1/refresh without a cookie -> $code $(head -c 200 "$tmp/refresh.b") (want 401 envelope)"
fi
acao=$(grep -i '^access-control-allow-origin:' "$tmp/refresh.h" | tr -d '\r' | cut -d' ' -f2-)
acac=$(grep -i '^access-control-allow-credentials:' "$tmp/refresh.h" | tr -d '\r' | cut -d' ' -f2-)
if [ "$acao" = "$web" ] && [ "$acac" = true ]; then
  pass "CORS allows $web with credentials"
else
  fail "CORS on a real request: allow-origin='$acao' allow-credentials='$acac' (want '$web' / 'true'; check CORS_ALLOWED_ORIGINS)"
fi

code=$(curl -sS -D "$tmp/pre.h" -o /dev/null -w '%{http_code}' --max-time 30 -X OPTIONS \
  -H "Origin: $web" -H 'Access-Control-Request-Method: POST' \
  -H 'Access-Control-Request-Headers: authorization,content-type' "$api/api/v1/login")
acao=$(grep -i '^access-control-allow-origin:' "$tmp/pre.h" | tr -d '\r' | cut -d' ' -f2-)
if { [ "$code" = 204 ] || [ "$code" = 200 ]; } && [ "$acao" = "$web" ]; then
  pass "CORS preflight for POST /api/v1/login -> $code"
else
  fail "CORS preflight for POST /api/v1/login -> $code allow-origin='$acao' (want 204 and '$web')"
fi

code=$(curl -sS -o "$tmp/index.html" -w '%{http_code} %{redirect_url}' --max-time 30 "$web/")
if [ "${code%% *}" = 200 ] && grep -q 'id="root"' "$tmp/index.html"; then
  pass "Web / -> 200 SPA shell"
elif printf '%s' "$code" | grep -q 'vercel.com/sso'; then
  fail "Web / -> redirects to Vercel login: Deployment Protection covers this URL, so the public can't open it (use a production domain, or relax Settings -> Deployment Protection)"
else
  fail "Web / -> $code (want 200 with the React root)"
fi

code=$(curl -sS -o "$tmp/deep.html" -w '%{http_code}' --max-time 30 "$web/transactions")
if [ "$code" = 200 ] && grep -q 'id="root"' "$tmp/deep.html"; then
  pass "Web deep link /transactions -> 200 SPA shell (rewrite works)"
else
  fail "Web deep link /transactions -> $code (want 200; check the rewrite in client/vercel.json)"
fi

bundle=$(grep -o '/assets/index-[^"]*\.js' "$tmp/index.html" | head -n1)
if [ -z "$bundle" ]; then
  fail "Web bundle: no /assets/index-*.js in index.html"
elif curl -sS --max-time 30 "$web$bundle" | grep -qF "$api"; then
  pass "Web bundle $bundle calls $api"
else
  fail "Web bundle $bundle does not contain $api (VITE_API_BASE_URL unset or different; it is inlined at build time, so fix it and redeploy)"
fi

if [ "$failed" = 0 ]; then echo "all checks passed"; else echo "some checks failed"; fi
exit "$failed"
