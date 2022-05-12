#!/usr/bin/env bash
echo "Stopping nodelet container ..."
docker stop nodelet
echo "Remove nodelet container ..."
docker rm nodelet
