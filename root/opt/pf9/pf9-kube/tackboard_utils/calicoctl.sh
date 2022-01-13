#!/bin/bash

LD_LIBRARY_PATH="/opt/pf9/python/pf9-lib:/opt/pf9/python/pf9-hostagent-lib:${LD_LIBRARY_PATH}" PYTHONPATH="/opt/pf9/python/lib/python2.7:${PYTHONPATH}" /opt/pf9/pf9-kube/tackboard_utils/calicoctl.py $@
