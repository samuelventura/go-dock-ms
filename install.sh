#!/bin/bash -x

#curl -X POST http://127.0.0.1:31600/api/daemon/stop/dock
#curl -X POST http://127.0.0.1:31600/api/daemon/disable/dock
#curl -X POST http://127.0.0.1:31600/api/daemon/uninstall/dock
if [[ "$OSTYPE" == "linux"* ]]; then
    DAEMON=dock
    BINARY=go-dock-ms
    SRC=$HOME/go/bin
    DST=/usr/local/bin
    go install
    curl -X POST http://127.0.0.1:31600/api/daemon/stop/$DAEMON
    curl -X POST http://127.0.0.1:31600/api/daemon/uninstall/$DAEMON
    sudo cp $SRC/$BINARY $DST
    curl -X POST http://127.0.0.1:31600/api/daemon/install/$DAEMON?path=$DST/$BINARY
    # do not touch working environment
    # curl -X POST http://127.0.0.1:31600/api/daemon/env/$DAEMON \
    #     -H "DaemonEnviron: DOCK_HOSTKEY=$HOME/.ssh/id_rsa"
    curl -X POST http://127.0.0.1:31600/api/daemon/enable/$DAEMON
    curl -X POST http://127.0.0.1:31600/api/daemon/start/$DAEMON
    curl -X GET http://127.0.0.1:31600/api/daemon/info/$DAEMON
    curl -X GET http://127.0.0.1:31600/api/daemon/env/$DAEMON
fi
