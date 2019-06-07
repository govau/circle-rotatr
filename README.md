# Torque

> Torque is a force that causes rotation (of secrets)

Torque will rotate ci user passwords in UAA, and writes them to environment variables in circleci projects.

## Example of what it does

Given this config.yaml:

```
cfs:
  - api_href: https://api.mycloudfoundry.com
    suffix: STAGING
orgs:
  - name: test-org
    spaces:
      - name: test-space
        repos:
          - govau/myrepo1
          - govau/myrepo2
```

1. The app ensures these repos are being built in circleci.

1. The app ensures the following environment variables are set in circleci for each repo:

- CF_API_STAGING=https://api.mycloudfoundry.com
- CF_ORG=test-org
- CF_SPACE=test-space
- CF_USERNAME=ci-test-org-test-space

1. The app changes the password for the user in CloudFoundry UAA called `ci-test-org-test-space`.

1. The app ensures the following environment variable is set in circleci for each repo:

- CF_PASSWORD_STAGING=the-current-password

## Onboarding a new team / space / repo

### 1. Ensure there is a ci user in this space cloud.gov.au

Run the following `cf` commands in prod and/or staging as required for this team.

```bash
CF_ORG=foo
CF_SPACE=bar
CF_USER_TO_CREATE=ci-${CF_ORG}-${CF_SPACE}

# create the user with a random password, torque will later reset it and save it to circle
cf create-user ${CF_USER_TO_CREATE} "$(openssl rand -hex 32)"

# Give the ci user access to deploy to the space
cf set-space-role ${CF_USER_TO_CREATE} ${CF_ORG} ${CF_SPACE} SpaceDeveloper
```

### 2. Ensure the project is being built by CircleCI. 

Go to https://circleci.com/add-projects/gh/govau (for govau), and click "Set Up Project" next to your repo if it is showing. If it is already setup, it will say either "Follow Project" or "Unfollow Project".

### 3. Add the repo into the torque configuration

The cloud.gov.au torque configuration is in the private ops repo at https://github.com/AusDTO/ops/blob/master/torque/config.yaml.

### 4. Wait for torque to run and confirm the project has the env vars set

Torque runs every 24 hours, however you can run it manually.

There should now be the expected env vars at https://circleci.com/gh/govau/project-name/edit#env-vars

## Future plans

1. The app ensures there is a user in CloudFoundry UAA called `ci-test-org-test-space` with the `SpaceDeveloper` role on `test-space` in cf instances.

1. The app waits until there are no builds in progress before rotating the password.

## Configuration

### Create UAA Client

```bash
uaac client add torque \
  --name torque \
  --secret "new-client-secret-password" \
  --authorized_grant_types client_credentials,refresh_token \
  --authorities uaa.admin,password.write
```

(In our CI, this client is created in [set-secrets.sh](ci/set-secrets.sh) and the credentials are added to CI credhub)
