#!/usr/bin/env bash

if [ "$1" = "" ] || [ "$1" = " " ] || [ "$2" = "" ] || [ "$2" = " " ]; then
echo "usage: ./build-and-deploy.sh <path to ssh key> <robot ip>"
exit

else

sudo rm -rf build
./build.sh
ssh -i "$1" root@"$2" 'systemctl stop vic-menu && rm /data/vic-menu'
ssh -i "$1" root@"$2" 'systemctl stop vic-menu && rm /data/vic-menu/vic-menu'
scp -i "$1" -O build/vic-menu root@"$2":/data/vic-menu/vic-menu
scp -i "$1" -O build/libvector-gobot.so root@"$2":/lib
scp -i "$1" -O export-gpio root@"$2":/sbin
scp -i "$1" -O ota-list.json root@"$2":/data/vic-menu/ota-list.json
scp -i "$1" -O vic-menu.service root@"$2":/lib/systemd/system
ssh -i "$1" root@"$2" 'systemctl daemon-reload && systemctl enable vic-menu && /sbin/reboot recovery'

exit

fi
