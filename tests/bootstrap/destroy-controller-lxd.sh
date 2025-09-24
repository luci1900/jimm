#!/bin/bash

# This script creates a jimm lxd controller to then call jaas destroy-controller on

set -euo pipefail
source "local/jimm/detect-jaas.sh"

export CONTROLLER_NAME="butterfly"

echo
echo "Adding lxd to juju and bootstrapping controller"
local/jimm/setup-controller.sh

echo
echo "Adding controller to jimm"
local/jimm/add-controller.sh

echo
echo "Destroying controller"
$JAAS destroy-controller "${CONTROLLER_NAME}" --no-prompt
