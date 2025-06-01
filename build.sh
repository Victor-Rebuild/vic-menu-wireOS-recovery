#!/bin/bash

mkdir -p build

set -e

TC="$PWD/vic-toolchain/arm-linux-gnueabi/bin/arm-linux-gnueabi-"

cd vector-gobot
GCC="${TC}gcc" GPP="${TC}g++" make vector-gobot
cd ..
cp vector-gobot/build/libvector-gobot.so build/

CC=${TC}gcc \
CXX=${TC}g++ \
CGO_CFLAGS="-I$(pwd)/vector-gobot/include" \
CGO_LDFLAGS="-L$(pwd)/build" \
GOARCH=arm \
GOARM=7 \
CGO_ENABLED=1 \
go build \
-ldflags "-s -w" \
-o build/vic-menu
