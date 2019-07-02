#!/usr/bin/env bash
set -euo pipefail

STOP=0
function on_signal() {
    echo "Received signal $1"
    trap - SIGINT SIGTERM # Clear the traps
    STOP=1
}
trap 'on_signal SIGINT' SIGINT
trap 'on_signal SIGTERM' SIGTERM
while [[ $STOP -eq 0 ]]; do
    sleep 1
    echo 'heartbeat'
done
