#!/bin/bash -x

if [[ "$OSTYPE" == "linux"* ]]; then
    SRC=$HOME/go/bin
    DST=/usr/local/bin
    if [[ -f "$DST/go-dock-ss" ]]; then
        sudo systemctl stop GoDockMs
        sudo $DST/go-dock-ss -service uninstall
        sleep 3
    fi
    go install
    (cd go-dock-ss; go install)
    (cd go-dock-sh; go install)
    sudo cp $SRC/go-dock-ms $DST
    sudo cp $SRC/go-dock-ss $DST
    sudo $DST/go-dock-ss -service install
    sudo systemctl restart GoDockMs
fi
