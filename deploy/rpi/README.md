# Spotify speaker on a Raspberry Pi 3 / 3B+ over HDMI

This turns a Raspberry Pi 3 or 3B+ into a headless Spotify Connect speaker:

- Raspberry Pi OS Lite (Trixie, 64-bit), wired Ethernet, no Wi-Fi or Bluetooth
- audio over HDMI into a receiver or TV
- never wakes the TV over HDMI-CEC
- a small web interface at `http://<hostname>.local/ui/` (now playing, playlists, albums)
- shows up in the Spotify app on your network, no login on the Pi needed

Everything is set up on first boot by cloud-init, so you never need a screen
or keyboard on the Pi.

## What you need

- Raspberry Pi 3 or 3B+, power supply, Ethernet cable
- microSD card (8 GB is plenty)
- HDMI cable to the receiver
- a Mac or Linux machine with Docker (to build the arm64 binary)
- Spotify Premium

## 1. Make it yours

The files in this folder are set up for my speaker ("Stuen Deluxe"). Change
these before you build the card:

| File | What to change |
| --- | --- |
| `config.yml` | `device_name`, the name you see in Spotify |
| `user-data` | `hostname`, `timezone` and the user `name` (`peter`) |
| `go-librespot.service` | the `Description` line (cosmetic) |
| `prepare-card.sh` | the default SSH key path, or pass your key as the 2nd argument |

You need an SSH key pair. If you don't have one for the Pi:

```sh
ssh-keygen -t ed25519 -f ~/.ssh/id_ed25519_speaker
```

## 2. Build the binary

```sh
deploy/rpi/build.sh
```

This builds `go-librespot` for arm64 in a `golang:1.26-trixie` container and
puts it in `~/.cache/speaker-build/` (change it with `OUT=...`).

Keep `VERSION` a real release number like `0.10.5` (the default). Spotify's
client token endpoint rejects anything else, a commit hash included, and then
every connect fails with `clienttoken: 400`.

On macOS with colima, only your home folder is mounted into the container,
which is why the output goes under `$HOME`.

## 3. Flash the card

Download **Raspberry Pi OS Lite (64-bit)**, Trixie, from
<https://www.raspberrypi.com/software/operating-systems/> and write it to
the card. With Raspberry Pi Imager, skip its "customise settings" step,
cloud-init does that here. Or by hand on macOS:

```sh
diskutil list                      # find the card, e.g. /dev/disk4
diskutil unmountDisk /dev/disk4
xz -dc 2026-10-06-raspios-trixie-arm64-lite.img.xz | sudo dd of=/dev/rdisk4 bs=4m
sync
```

Double check the disk number, `dd` overwrites whatever you point it at.

## 4. Add the first-boot setup

Re-insert the card so the `bootfs` partition mounts, then:

```sh
deploy/rpi/prepare-card.sh /Volumes/bootfs ~/.ssh/id_ed25519_speaker.pub
```

It:

- writes `user-data` (with your public key) and `network-config` (eth0, DHCP)
- copies the binary, `config.yml` and the systemd unit to `bootfs/stuen/`
- comments out `dtoverlay=vc4-kms-v3d` in `config.txt` (see "HDMI audio" below)
- appends `config.txt.append` once

Eject the card.

## 5. Boot it

Put the card in the Pi, plug in Ethernet and HDMI, power it on. The first boot
takes about 5 minutes: cloud-init installs the audio libraries and avahi,
creates a `golibrespot` system user and starts the service.

Then:

```sh
ssh -i ~/.ssh/id_ed25519_speaker <user>@<hostname>.local
systemctl status go-librespot
```

Open Spotify on your phone or computer, tap the devices button and pick the
speaker. The login sticks (`persist_credentials`), so after that it just
reconnects, also after a reboot.

The web interface is at `http://<hostname>.local/ui/`. Add `?kiosk` to hide
the cursor, meant for a touchscreen on the Pi itself.

## HDMI audio

This is the part that took the longest, so here is what the config does and
why.

**Firmware audio driver, not KMS.** Current Raspberry Pi OS uses the KMS
graphics driver (`dtoverlay=vc4-kms-v3d`), and with it HDMI audio goes through
`vc4-hdmi`. On my receiver that gave short gaps every few seconds, getting
worse over time, even with plain `aplay` and nothing else running. With the
KMS overlay commented out, the firmware driver (`snd_bcm2835`) plays clean.
The audio device in `config.yml` is that card:

```yaml
audio_backend: alsa
audio_device: "plughw:CARD=b1,DEV=0"
```

Check the card name on your Pi with `aplay -l`. It should list `bcm2835 HDMI 1`
with the id `b1`. If audio gaps come back after an OS update, check that the
KMS line is still commented out in `/boot/firmware/config.txt`.

**Stereo PCM.** Plain 2-channel output. I tried a 5.1 upmix to stop the
receiver switching modes, and it gave silence, so leave it stereo.

**`config.txt.append`:**

| Setting | Why |
| --- | --- |
| `hdmi_ignore_cec_init=1` | don't send the CEC "active source" at boot, so the TV stays off |
| `hdmi_force_hotplug=1` | use HDMI even if the receiver/TV is off when the Pi boots |
| `hdmi_drive=2` | HDMI mode (with audio), not DVI |
| `hdmi_force_edid_audio=1` | assume the sink takes audio even if its EDID says otherwise |
| `hdmi_blanking=1` | let the display sleep, there's nothing to show |
| `disable_splash=1` | no rainbow screen at boot |
| `dtoverlay=disable-wifi`, `disable-bt` | wired only, radios off |

cloud-init also disables `wpa_supplicant`, `bluetooth` and `hciuart` and runs
`rfkill block all`.

## Updating

```sh
deploy/rpi/build.sh
ssh <user>@<hostname>.local 'cat > /tmp/go-librespot' < ~/.cache/speaker-build/go-librespot
ssh <user>@<hostname>.local 'sudo install -m 0755 /tmp/go-librespot /usr/local/bin/go-librespot && sudo systemctl restart go-librespot'
```

A restart stops playback, pick the speaker in Spotify again after.

## Troubleshooting

- **Speaker doesn't show up in Spotify:** `systemctl status avahi-daemon go-librespot`,
  and make sure the phone is on the same network/VLAN (mDNS doesn't cross subnets).
- **Shows up but connecting fails:** `journalctl -u go-librespot -n 50`. A
  `clienttoken: 400` means the binary was built with a non-release version.
- **Track list in the web UI says "needs metadata.enabled":** set
  `metadata.enabled: true` in `/var/lib/go-librespot/config.yml` and restart.
- **Gaps or dropouts in the audio:** check the KMS overlay (above) and look for
  `alsa underrun` in the log.
- **Config lives in** `/var/lib/go-librespot/config.yml` on the Pi, together
  with the saved credentials.
