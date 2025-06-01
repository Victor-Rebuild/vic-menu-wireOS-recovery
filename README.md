# Vic-menu

## Building
Build with ./build.sh

## Deploying
- `scp` build/vic-menu over to /data/vic-menu. If you want to change the OTA list, modify ota-list.json then `scp` that over to /data/ota-list.json.
- You should also `scp` build/libvector-gobot.so to /lib/, and export-gpio to /sbin/.
