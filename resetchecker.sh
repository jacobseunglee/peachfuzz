#!/usr/bin/env bash
# Reset checker script - tests connectivity to scanned ports using netcat

# add hosts flags argument and teams argument

HOSTS="dcs.txt"
TEAMS="5 5"
 
while true; do
    ./blender.sh --hosts $HOSTS --teams $TEAMS ./portchecker.sh
    sleep 2
done