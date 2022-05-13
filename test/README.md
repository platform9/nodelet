This directory has a couple of simple bash scripts to start nodelet in a container:


## Pre-requsites

The top level directory should have:

* pf9-kube.tar.gz which should contain the pf9-kube-1.21.3-pmk0.x86_64.rpm and debian inside the tar.gz
* nodeletctl should be built and present in the nodeletctl directory


# Start Nodelet

```
bash ./test/start-nodelete-in-docker.sh
```

This script creates a docker container named "nodelet" and starts the nodelet inside the
container using nodeletctl.


# Stop Nodelet
```
bash ./test/delete-nodelet
```

# Docker

Once the nodelet is started you can just exec into it to take a look at the environments

```
docker exec -it nodelet bash
```