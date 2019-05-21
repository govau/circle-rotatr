#!/usr/bin/env bash

set -eu
set -o pipefail

: "${CIRCLE_TOKEN:?Need to set CIRCLE_TOKEN}"
: "${ENV_JSON:?Need to set ENV_JSON}"

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

KEYS="$(echo $ENV_JSON | jq -r keys[])"
for KEY in ${KEYS}; do
  VALUE="$(echo $ENV_JSON | jq -r .$KEY)"
  export $KEY=$VALUE
done

CIRCLE_TOKEN="${CIRCLE_TOKEN}" \
torque \
  -config.file "${SCRIPT_DIR}/../../ops/torque/config.yaml"
