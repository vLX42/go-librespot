#!/bin/sh
# Builds the arm64 daemon for Raspberry Pi OS Trixie in a matching container.
#
# The version must look like a release (0.10.5): Spotify's clienttoken
# endpoint answers 400 to anything else, a commit hash included, and then
# every connect fails.
set -eu

VERSION="${VERSION:-0.10.5}"
OUT="${OUT:-$HOME/.cache/speaker-build}"
REPO="$(cd "$(dirname "$0")/../.." && pwd)"

mkdir -p "$OUT"
docker run --rm --platform linux/arm64 -v "$REPO":/src:ro -v "$OUT":/out golang:1.26-trixie sh -c "
  apt-get update -qq >/dev/null &&
  apt-get install -y -qq libasound2-dev libogg-dev libvorbis-dev libflac-dev libmpg123-dev pkg-config >/dev/null &&
  cp -r /src /build && cd /build &&
  CGO_ENABLED=1 go build -trimpath -ldflags '-s -w -X github.com/devgianlu/go-librespot.version=$VERSION' -o /out/go-librespot ./cmd/daemon"

echo "built $OUT/go-librespot ($VERSION)"
