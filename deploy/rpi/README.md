# Stuen Deluxe on a Raspberry Pi 3

Headless speaker: Raspberry Pi OS Lite (Trixie, arm64), wired Ethernet, HDMI
audio into the receiver, web interface at `http://stuen-deluxe.local/ui/`.

1. `deploy/rpi/build.sh` builds the arm64 binary (needs Docker).
2. Flash Raspberry Pi OS Lite 64-bit to the card, e.g.
   `sudo dd if=raspios-trixie-arm64-lite.img of=/dev/rdiskN bs=4m && sync`.
3. `deploy/rpi/prepare-card.sh /Volumes/bootfs` adds the first-boot setup.
4. Boot the Pi. cloud-init installs the libraries and the service (~5 min).

Then `ssh -i ~/.ssh/id_ed25519_stuen peter@stuen-deluxe.local`.

Notes:
- HDMI audio goes through the firmware driver (`plughw:CARD=b1,DEV=0`), the
  KMS overlay is off. The KMS driver (`vc4-hdmi`) gave gaps on this receiver
  even with plain `aplay`.
- `hdmi_ignore_cec_init=1` keeps the Pi from waking the TV.
- The daemon's version must be a release number, see `build.sh`.
- Updating later: `build.sh`, then
  `scp` the binary to the Pi, `sudo install -m 0755 go-librespot /usr/local/bin/`
  and `sudo systemctl restart go-librespot`.
