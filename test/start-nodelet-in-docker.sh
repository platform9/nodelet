#!/usr/bin/env bash

export CGROUP_PARENT="$(grep systemd /proc/self/cgroup | cut -d: -f3)/docker"
sudo rm -rf /tmp/containerd 
mkdir /tmp/containerd
echo "Starting nodelet container ..."

docker run -d --name nodelet \
 --privileged \
 --cap-add=ALL \
 -v /sys/fs/cgroup:/sys/fs/cgroup \
 -v `pwd`:/work \
 -v /dev:/dev \
 -v /lib/modules:/lib/modules \
 -v /tmp/containerd:/var/lib/containerd \
jrei/systemd-ubuntu
echo "Container started, sleeping for 10 seconds..."
sleep 2
echo "Checking ip address of nodelet container ..."
master=`docker inspect nodelet --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}'`
echo "IP address of nodelet container is $master"

echo "Starting nodelet ..."
docker exec -it nodelet /bin/bash -c "/work/test/start-nodelet.sh $master"