#!/bin/bash
# set -x
numlines="$1"
if [ -z "${numlines}" ]; then
    echo "numlines is missing"
    exit 1
fi
uuid="$2"
if [ -z "${uuid}" ]; then
    echo "request uuid is missing"
    exit 1
fi
LOGFILES=/var/log/pf9/kube/kube.log*
CONTENT_TYPE='text/plain'
URL=http://localhost:9080/tackboard/

# Assumes logs rotate from newest to oldest,
#     kube.log, kube.log.1, ..., kube.log.N
ls -r ${LOGFILES} \
    | xargs cat \
    | tail -n ${numlines} \
    | NO_PROXY=localhost curl -sX POST --data-binary @- -H "${CONTENT_TYPE}" -H "uuid: ${uuid}" ${URL}
