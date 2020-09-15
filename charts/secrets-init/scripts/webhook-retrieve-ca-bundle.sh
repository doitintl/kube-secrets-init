#!/bin/bash

ROOT=$(
  cd $(dirname $0)/../../
  pwd
)

set -o errexit
set -o nounset
set -o pipefail

export CLUSTER_NAME=$(kubectl config get-contexts $(kubectl config current-context) --no-headers | awk '{print $3}')

echo "Bundle data: $(kubectl config view --raw --flatten -o json | jq -r '.clusters[] | select(.name == "'${CLUSTER_NAME}'") | .cluster."certificate-authority-data"')"
