#!/bin/bash
logfile=/var/log/pf9/kube/forward_kubernetes.log
echo forward_kubernetes.sh started on `date` >> $logfile
/opt/pf9/comms/nodejs/bin/node /opt/pf9/comms/utils/forwarding_client.js localhost:7393 localhost:8080 >> $logfile 2>&1 &
