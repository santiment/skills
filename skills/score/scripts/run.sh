#!/usr/bin/env bash
# Self-contained launcher for the `score` CLI bundled inside this skill.
#
# The binary ships next to this script as a gzip+base64 text blob
# (score-<os>-<arch>.gz.b64) so it survives `npx skills add` transport
# (both the skills.sh blob snapshot and the git-clone fallback handle text
# reliably; raw binaries do not). On first run we materialize the binary for
# the host OS/arch into ./.bin/, mark it executable, and exec it. Subsequent
# runs reuse the cached binary.
#
# Usage: bash scripts/run.sh <score args...>   (e.g. health --json)
set -euo pipefail

here="$(cd "$(dirname "${BASH_SOURCE[0]:-$0}")" && pwd)"

os="$(uname -s | tr '[:upper:]' '[:lower:]')"
arch="$(uname -m)"
case "$arch" in
  x86_64 | amd64) arch=amd64 ;;
  arm64 | aarch64) arch=arm64 ;;
esac

bin="$here/.bin/score-$os-$arch"
src="$here/score-$os-$arch.gz.b64"

if [ ! -x "$bin" ]; then
  if [ ! -f "$src" ]; then
    echo "score: no bundled binary for $os/$arch" >&2
    echo "score: rebuild it from the repo with: task skill-bundle (add $os/$arch to SKILL_PLATFORMS)" >&2
    exit 7
  fi
  mkdir -p "$here/.bin"
  # Portable base64 decode: GNU coreutils accepts --decode; BSD/macOS does not,
  # so fall back to openssl (present on both). Pipe through gunzip.
  if base64 --decode </dev/null >/dev/null 2>&1; then
    base64 --decode "$src" | gunzip > "$bin"
  else
    openssl base64 -d -in "$src" | gunzip > "$bin"
  fi
  chmod +x "$bin"
fi

exec "$bin" "$@"
