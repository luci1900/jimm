#!/bin/bash
set -e

# generate.sh
# This script extracts the current Juju controller's configuration
# and writes them to a controller.yaml file. It retrieves the controller's UUID,
# API endpoints, username, password, and CA certificate.
#
# This enables the developer to quickly set up configuration for
# running E2E tests that require a backing Juju controller.
#
# Usage:
#   ./generate.sh [--controller=<name>]
#
# If --controller is not provided, it uses the current active controller.

CONTROLLER=""
for arg in "$@"; do
    case $arg in
        --controller=*)
            CONTROLLER="${arg#*=}"
            shift
            ;;
        *)
            ;;
    esac
done

# Output path (project root - script should be run from there)
CONFIG_FILE="controllers.yaml"

# Get current controller if not specified
if [ -z "$CONTROLLER" ]; then
    CONTROLLER=$(juju whoami | yq -r '.Controller')
fi

echo "Extracting controller configuration for: ${CONTROLLER}"

# Get controller UUID
CONTROLLER_UUID=$(juju show-controller "${CONTROLLER}" | yq -r ".${CONTROLLER}.details.controller-uuid")

# Get all controller API addresses as a JSON array
CONTROLLER_ADDRS=$(juju show-controller "${CONTROLLER}" | yq -r ".${CONTROLLER}.details.\"api-endpoints\" | @json")

# Get username and password from accounts.yaml
CONTROLLER_USERNAME=$(cat ~/.local/share/juju/accounts.yaml | yq -r ".controllers.${CONTROLLER}.user")
CONTROLLER_PASSWORD=$(cat ~/.local/share/juju/accounts.yaml | yq -r ".controllers.${CONTROLLER}.password")

# Get CA certificate
CONTROLLER_CACERT=$(juju show-controller "${CONTROLLER}" | yq -r ".${CONTROLLER}.details.\"ca-cert\"")

# Build the YAML using yq to ensure proper formatting
export CONTROLLER CONTROLLER_UUID CONTROLLER_ADDRS CONTROLLER_USERNAME CONTROLLER_PASSWORD CONTROLLER_CACERT
yq -n '
  .controllers.[strenv(CONTROLLER)].uuid = strenv(CONTROLLER_UUID) |
  .controllers.[strenv(CONTROLLER)].addrs = (strenv(CONTROLLER_ADDRS) | fromjson) |
  .controllers.[strenv(CONTROLLER)].username = strenv(CONTROLLER_USERNAME) |
  .controllers.[strenv(CONTROLLER)].password = strenv(CONTROLLER_PASSWORD) |
  .controllers.[strenv(CONTROLLER)]."ca-cert" = strenv(CONTROLLER_CACERT)
' | tee "$CONFIG_FILE" > /dev/null

echo "Controller configuration written to ${CONFIG_FILE}"
