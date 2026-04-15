# portctl

A CLI for managing [Portainer EE](https://www.portainer.io/) stacks. Deploy git-backed stacks, check image freshness, redeploy with updated images, and tail logs — without touching the Portainer UI.

## Install

```sh
go install github.com/tlvenn/portctl/cmd/portctl@latest
```

Or build from source:

```sh
git clone https://github.com/tlvenn/portctl.git
cd portctl
go build -o portctl ./cmd/portctl
```

## Configuration

All configuration is via environment variables. Use a `.envrc` (with [direnv](https://direnv.net/)) or export them in your shell profile.

### Required

| Variable | Description |
|----------|-------------|
| `PORTAINER_URL` | Portainer instance URL (e.g., `https://portainer.example.com`) |
| `PORTAINER_API_KEY` | API key from Portainer (User Settings > API Keys) |

### Optional

| Variable | Default | Description |
|----------|---------|-------------|
| `PORTCTL_GIT_REPO` | — | Git repo URL for stack creation |
| `PORTCTL_GIT_BRANCH` | `main` | Git branch to deploy from |
| `PORTCTL_GIT_CREDENTIAL_ID` | — | Portainer git credential ID (run `portctl credentials` to find it) |
| `PORTCTL_ENDPOINT_ID` | auto-detected | Portainer endpoint ID (auto-discovers first local Docker endpoint) |
| `PORTCTL_REPO_PATH` | — | Local path to the stacks repo (for webhook registration in `portainer-webhooks.json`) |

### Example `.envrc`

```sh
export PORTAINER_URL="https://portainer.example.com"
export PORTAINER_API_KEY="ptr_your_api_key_here"
export PORTCTL_GIT_REPO="https://github.com/you/your-stacks.git"
export PORTCTL_GIT_BRANCH="main"
export PORTCTL_GIT_CREDENTIAL_ID="2"
export PORTCTL_REPO_PATH="/path/to/your-stacks"
```

## Usage

### List stacks

Show all stacks with status, image freshness, container states, and deployed git version:

```sh
portctl list
```

```
NAME            STATUS  IMAGES      CONTAINERS                       UPDATED           VERSION
traefik         active  outdated    1 running / 0 stopped / 0 error  2025-12-03 09:22  e039da3
jellyfin        active  outdated    1 running / 1 stopped / 0 error  2026-02-05 17:54  1b4b727
sonarr          active  outdated    1 running / 1 stopped / 0 error  2025-12-03 16:46  2add468
radarr          active  up to date  1 running / 1 stopped / 0 error  -                 02fd750
```

The **IMAGES** column compares local image digests against remote registries to detect when a newer `:latest` image has been pushed.

Skip the image check for faster output:

```sh
portctl list --no-images
```

### Deploy a new stack

Create a git-backed stack in Portainer with automatic webhook registration:

```sh
portctl deploy radarr
```

```
Stack 'radarr' deployed successfully (ID: 30).
Webhook ID: 1b1a2f93-a1d4-4033-9eca-085a27f3c600
Webhook registered in portainer-webhooks.json
```

This will:

1. Validate the stack name and that `<stack-name>/docker-compose.yml` exists locally
2. Create a git-backed stack in Portainer pointing to your repo
3. Generate a webhook UUID for CI-triggered redeployments
4. Register the webhook in `portainer-webhooks.json`

The deploy command expects stacks to follow a convention: each stack is a subdirectory containing a `docker-compose.yml`:

```
your-stacks/
  sonarr/docker-compose.yml
  radarr/docker-compose.yml
  prowlarr/docker-compose.yml
  portainer-webhooks.json
```

### Redeploy a stack

Pull the latest git changes and redeploy, forcing fresh image pulls:

```sh
portctl redeploy sonarr
```

Use this after pushing compose changes or when `portctl list` shows images as **outdated**.

### View logs

Tail logs for all containers in a stack:

```sh
portctl logs prowlarr
```

Filter by service name:

```sh
portctl logs prowlarr flaresolverr
```

Follow logs in real-time:

```sh
portctl logs prowlarr --follow
```

Show more or fewer lines:

```sh
portctl logs prowlarr --tail 50
```

### List git credentials

Find the credential ID to set `PORTCTL_GIT_CREDENTIAL_ID`:

```sh
portctl credentials
```

```
ID  NAME
2   truenas-stack
```

## CI Integration

The deploy command generates webhooks compatible with Portainer's webhook API. A GitHub Actions workflow can trigger redeployments on push:

```yaml
- name: Redeploy stack
  run: |
    WEBHOOK_ID=$(jq -r '."${{ matrix.stack }}"[0]' portainer-webhooks.json)
    curl -sSf -X POST "https://portainer.example.com/api/stacks/webhooks/$WEBHOOK_ID"
```

## Requirements

- Portainer Enterprise Edition (tested with 2.33.6)
- Go 1.24+ (for building from source)
- Git credentials configured in Portainer for private repos

## Architecture

Single binary, one external dependency ([cobra](https://github.com/spf13/cobra)). All Portainer and Docker API interactions go through the Portainer REST API — no direct Docker socket access or SSH required.

```
portctl/
  cmd/portctl/main.go          # Entry point
  internal/
    cmd/                        # Cobra commands
    client/                     # Portainer API client
    config/                     # Environment variable config
    webhooks/                   # portainer-webhooks.json read/write
```
