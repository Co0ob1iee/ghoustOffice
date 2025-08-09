#!/usr/bin/env bash
set -euo pipefail
# Usage: ./hibp-build.sh /path/to/pwned-passwords-sha1-ordered-by-hash.txt /output/dir
SRC="${1:-}"; OUT="${2:-/data/hibp_cache}"
if [[ -z "$SRC" || ! -f "$SRC" ]]; then
  echo "Podaj ścieżkę do pwned-passwords (SHA1, ordered-by-hash)."
  exit 1
fi
mkdir -p "$OUT"
# Split by first 5 hex chars (prefix). Each line: HEX:COUNT
# Expects upper-case hex in SRC; if not, we will uppercase on the fly.
awk 'BEGIN{FS=":"} { up=toupper($1); pref=substr(up,1,5); rest=substr(up,6); print rest":"$2 >> (ENVIRON["OUT"] "/" pref ".txt") }' OUT="$OUT" "$SRC"
echo "Zbudowano cache HIBP w: $OUT"
