#!/bin/sh
# Puts the first-boot setup on a freshly flashed Raspberry Pi OS Lite card.
#
#   deploy/rpi/prepare-card.sh /Volumes/bootfs [~/.ssh/id_ed25519_stuen.pub]
set -eu

BOOT="$1"
KEY="${2:-$HOME/.ssh/id_ed25519_stuen.pub}"
HERE="$(cd "$(dirname "$0")" && pwd)"
BIN="${OUT:-$HOME/.cache/speaker-build}/go-librespot"

[ -f "$BOOT/config.txt" ] || { echo "$BOOT does not look like a boot partition"; exit 1; }
[ -f "$BIN" ] || { echo "no binary at $BIN, run build.sh first"; exit 1; }

sed "s|__SSH_PUBKEY__|$(cat "$KEY")|" "$HERE/user-data" > "$BOOT/user-data"
cp "$HERE/network-config" "$BOOT/network-config"
mkdir -p "$BOOT/stuen"
cp "$BIN" "$HERE/config.yml" "$HERE/go-librespot.service" "$BOOT/stuen/"
touch "$BOOT/ssh"

sed -i.bak 's/^dtoverlay=vc4-kms-v3d/#dtoverlay=vc4-kms-v3d/; s/^camera_auto_detect=1/camera_auto_detect=0/' "$BOOT/config.txt"
rm -f "$BOOT/config.txt.bak"
grep -q '^# Stuen Deluxe' "$BOOT/config.txt" || cat "$HERE/config.txt.append" >> "$BOOT/config.txt"

echo "card ready, eject it and boot the Pi"
