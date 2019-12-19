#!/usr/bin/env bash

# Sets the secrets needed by this pipeline.
# Where possible, credentials are rotated each time this script is run.

PIPELINE=torque

set -euo pipefail

: "${PATH_TO_OPS:?Need to set PATH_TO_OPS}"

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

set_credhub_value() {
  KEY="$1"
  VALUE="$2"
  https_proxy=socks5://localhost:8112 \
  credhub set -n "/concourse/main/$PIPELINE/$KEY" -t value -v "${VALUE}"
}

assert_credhub_value() {
  KEY="$1"
  if ! https_proxy=socks5://localhost:8112 credhub get -n "/concourse/apps/${PIPELINE}/${KEY}" > /dev/null 2>&1 ; then
    echo "${KEY} not set in credhub. Add it to your environment (e.g. use .envrc) and re-run this script"
    exit 1
  fi
}

echo "Ensuring logged in to credhub"
if ! https_proxy=socks5://localhost:8112 credhub find > /dev/null; then
  https_proxy=socks5://localhost:8112 credhub login --sso
fi

# UAA secrets - get the UAAs from torque's config.yaml, and then create
# a user in uaa for torque to use.
# We have to save the uaa secrets as JSON so it can be passed to the concourse job.
UAA_CLIENT_ID=torque
# Annoyingly, uaac seemingly needs a non-empty uaac.yml valid yaml file if it already exists
# TODO - Fix - The uaac commands will print this message, which can be ignored:
# "Unknown key: Max-Age = 86400"
cat << EOF > ${SCRIPT_DIR}/.uaac.yml
notused: {}
EOF

UAAC_CMD="docker run -it --rm -v ${SCRIPT_DIR}/.uaac.yml:/root/.uaac.yml govau/cf-uaac:4.1.0 "
UAA_ORIGIN="$(yq -r '.uaa_origin' ${PATH_TO_OPS}/torque/config.yaml)"

UAA_IDS="$(yq -r '.cfs[].id' ${PATH_TO_OPS}/torque/config.yaml)"
ENV_JSON="{"
for UAA_ID in ${UAA_IDS}; do
  echo "Ensuring UAA user exists for $UAA_ID"

  API_HREF="$(yq -r ".cfs[] | select (.id == \"${UAA_ID}\") | .api_href" ${PATH_TO_OPS}/torque/config.yaml)"
  UAA_HREF="$(curl -s "${API_HREF}" | jq -r .links.uaa.href)"

  regex="([a-z]+).cld.gov.au"
  [[ $API_HREF =~ $regex ]]
  ENV_NAME="${BASH_REMATCH[1]}"
  JUMPBOX="bosh-jumpbox.${ENV_NAME}.cld.gov.au"
  
  # Target our uaa
  ${UAAC_CMD} target "${UAA_HREF}"
  
  # Get the admin password from the jumpbox
  CLIENT_SECRET="$(ssh ${JUMPBOX} credhub get -n /main/cf/uaa_admin_client_secret --output-json | jq -r .value)"
  
  # Login
  ${UAAC_CMD} token client get admin -s "${CLIENT_SECRET}"

  #Generate a new password
  NEW_UAA_CLIENT_SECRET="$(openssl rand -hex 32)"

  echo "Checking if UAA client exists"
  EXISTING="$(${UAAC_CMD} client get ${UAA_CLIENT_ID} || true)"
  if [[ $EXISTING != *"CF::UAA::NotFound"* ]]; then
    echo "Rotating secret"
    ${UAAC_CMD} secret set torque -s "${NEW_UAA_CLIENT_SECRET}"
  else
    echo "Creating new user"
    ${UAAC_CMD} client add "${UAA_CLIENT_ID}" \
      --secret "${NEW_UAA_CLIENT_SECRET}" \
      --authorized_grant_types client_credentials,refresh_token \
      --authorities uaa.admin,password.write \
      --no-interactive
  fi

  #Save the password into ENV_JSON
  if [[ $ENV_JSON != "{" ]]; then
    ENV_JSON="${ENV_JSON},"
  fi
  ENV_JSON="${ENV_JSON}\"UAA_CLIENT_ID_${UAA_ID}\":\"${UAA_CLIENT_ID}\""
  ENV_JSON="${ENV_JSON},\"UAA_CLIENT_SECRET_${UAA_ID}\":\"${NEW_UAA_CLIENT_SECRET}\""
done
ENV_JSON="${ENV_JSON}}"
set_credhub_value ENV_JSON "${ENV_JSON}"

# lets not leave uaac secrets lying around 
rm ${SCRIPT_DIR}/.uaac.yml

if [ -v CIRCLE_TOKEN ]; then
  set_credhub_value CIRCLE_TOKEN "${CIRCLE_TOKEN}"
else
  assert_credhub_value CIRCLE_TOKEN
fi

if [ -v DOCKER_HUB_EMAIL ]; then
  set_credhub_value DOCKER_HUB_EMAIL "${DOCKER_HUB_EMAIL}"
  set_credhub_value DOCKER_HUB_USERNAME "${DOCKER_HUB_USERNAME}"
  set_credhub_value DOCKER_HUB_PASSWORD "${DOCKER_HUB_PASSWORD}"
else
  assert_credhub_value DOCKER_HUB_EMAIL
  assert_credhub_value DOCKER_HUB_USERNAME
  assert_credhub_value DOCKER_HUB_PASSWORD
fi

