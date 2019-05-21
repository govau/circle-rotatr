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
