#!/bin/bash

socketpath="$1"
if [ -z "${socketpath}" ]; then
    echo "socket path is missing"
    exit 1
fi

if [ ! -S "${socketpath}" ]; then
    echo "path does not refer to a unix domain socket"
    exit 2
fi

logfile=/var/log/pf9/kube/unixsocket_forwarding_client.log
echo unixsocket_forwarding_client started on `date` >> $logfile
/opt/pf9/comms/nodejs/bin/node /opt/pf9/pf9-kube/unixsocket_forwarding_client.js localhost:7391 $socketpath >> $logfile 2>&1 &

