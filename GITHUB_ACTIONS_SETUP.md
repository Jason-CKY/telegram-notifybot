# GitHub Actions Setup Guide

This document outlines the steps required to enable the Docker build and push workflow for this project.

## Prerequisites

- GitHub repository with GitHub Actions enabled
- Owner/Admin access to the repository

## Steps to Configure

### Step 1: No Secrets Required

This workflow uses `GITHUB_TOKEN` for authentication, which is automatically available in all GitHub Actions workflows. No additional secrets need to be configured.

The `GITHUB_TOKEN` is granted the following permissions by default:
- `contents: read` - Read repository contents
- `packages: write` - Write to GitHub Container Registry

### Step 2: Make the Container Package Public

The workflow includes a step to automatically set the package visibility to public. However, you may need to do this manually the first time if the automatic step fails.

**Option A: Automatic (recommended)**
The workflow attempts to set the package visibility to public after the first successful push. If this fails, use Option B.

**Option B: Manual (if automatic fails)**

1. Navigate to your repository on GitHub
2. Go to **Packages** (the `Packages` link is usually under the repository name on the right side, or visit `https://github.com/orgs/YOUR_ORG/packages`)
3. Find the **telegram-notifybot** package
4. Click on the package, then go to **Package settings** (gear icon on the right)
5. Under **Danger Zone**, click **Change visibility**
6. Select **Public** and confirm

**For personal repositories:**
If the workflow's automatic visibility change fails (because org-level API calls don't work for personal repos), you can manually set visibility at:
- Go to: `https://github.com/users/YOUR_USERNAME/packages/container/telegram-notifybot/settings`
- Click **Change visibility** in the Danger Zone
- Select **Public**

### Step 3: Verify the Workflow

1. **Test with a PR:** Create a pull request targeting the `main` branch. The workflow will run and push an image with the commit SHA as the tag (e.g., `sha-a1b2c3d`).

2. **Test with a push:** Push a commit directly to `main`. The workflow will run and push an image tagged with the short SHA.

3. **Test with a tag:** Create a version tag:
   ```bash
   git tag v1.0.0
   git push origin v1.0.0
   ```
   This will push the image with tags `v1.0.0` and `latest`.

## Image Location

After the workflow runs successfully, images will be available at:

```
ghcr.io/jason-cky/telegram-notifybot
```

### Available Tags

| Event | Tag Example | Description |
|-------|-------------|-------------|
| Push to main | `sha-a1b2c3d` | 7-character commit SHA |
| PR to main | `pr-123` | PR number |
| Tag (v*.*.*) | `v1.0.0`, `latest` | Version tag + latest |

## Pulling the Image

Anyone can pull the public image using:

```bash
# Pull latest
docker pull ghcr.io/jason-cky/telegram-notifybot:latest

# Pull specific version
docker pull ghcr.io/jason-cky/telegram-notifybot:v1.0.0

# Pull by commit SHA
docker pull ghcr.io/jason-cky/telegram-notifybot:sha-a1b2c3d
```

## Environment Variables Required for the Container

When running the container, you need to set the following environment variables:

| Variable | Description | Required |
|----------|-------------|----------|
| `DIRECTUS_HOST` | Directus instance URL | Yes |
| `DIRECTUS_TOKEN` | Directus API token | Yes |
| `TELEGRAM_BOT_TOKEN` | Telegram bot API token | Yes |
| `ALLOWED_USERNAMES` | Comma-separated list of allowed Telegram usernames | Yes |
| `LOG_LEVEL` | Log level (debug, info, warn, error) | No (default: info) |
| `PORT` | Server port | No (default: 8080) |

Example:

```bash
docker run -d \
  -e DIRECTUS_HOST=https://your-directus-instance.com \
  -e DIRECTUS_TOKEN=your-directus-token \
  -e TELEGRAM_BOT_TOKEN=your-telegram-bot-token \
  -e ALLOWED_USERNAMES=username1,username2 \
  ghcr.io/jason-cky/telegram-notifybot:latest
```

## Troubleshooting

### "Package not found" error in visibility step
This is normal for the first run. The package is created on first push, and the visibility step will succeed on subsequent runs.

### Permission denied errors
Ensure the repository has Package write permissions. Go to:
- Repository Settings > Actions > General
- Ensure "Read and write" is selected for Workflow permissions

### Image not appearing as public
Manually set the package visibility as described in Step 2.
