#!/bin/bash

TC=/home/thommomc/Documents/Vector/vector-gobot/vic-toolchain/arm-linux-gnueabi/bin/arm-linux-gnueabi-

CC=${TC}gcc \
CXX=${TC}g++ \
CGO_CFLAGS="-I$(pwd)/include" \
CGO_LDFLAGS="-L$(pwd)/lib" \
GOARCH=arm \
GOARM=7 \
CGO_ENABLED=1 \
go build \
-ldflags "-s -w" \
-o main
