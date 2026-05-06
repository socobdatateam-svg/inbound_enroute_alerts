# Fly.io Failover Setup

Automatic failover from Render to Fly.io when Render free tier limits are exceeded.

## Quick Start

### 1. Install Fly.io CLI locally (one-time setup)

```bash
# Windows (PowerShell)
iwr https://fly.io/install.ps1 -useb | iex

# Mac/Linux
curl -L https://fly.io/install.sh | sh
```

### 2. Create Fly.io App

```bash
fly auth login
fly apps create bot-workstation
```

### 3. Set Secrets in Fly.io

```bash
fly secrets set SEATALK_APP_ID=your_app_id
fly secrets set SEATALK_APP_SECRET=your_secret
fly secrets set SEATALK_SIGNING_SECRET=your_signing_secret
fly secrets set ADMIN_TOKEN=your_admin_token
fly secrets set GOOGLE_CREDENTIALS_JSON='your_json_credentials'
```

### 4. Deploy to Fly.io

```bash
fly deploy
```

## GitHub Actions Auto-Failover

### Required GitHub Secrets

Go to **Settings → Secrets and variables → Actions** and add:

| Secret | Description | How to get |
|--------|-------------|------------|
| `FLY_API_TOKEN` | Fly.io API token | `fly auth token` or create at [fly.io/user/personal_access_tokens](https://fly.io/user/personal_access_tokens) |
| `RENDER_HEALTH_URL` | Your Render health endpoint | `https://your-service.onrender.com/healthz` |

### Workflows

1. **deploy-flyio.yml** - Deploys to Fly.io (manual or on push)
2. **render-failover.yml** - Checks Render health every 15 min, auto-deploys to Fly.io if Render is down
3. **switch-to-fly.yml** - Manual switch with deploy/destroy/status options

### Trigger Failover Manually

Go to **Actions → Manual Switch to Fly.io → Run workflow**

### Auto-Failover Behavior

- Checks Render `/healthz` every 15 minutes
- If Render returns non-200 for 15+ minutes → triggers Fly.io deployment
- Only deploys if Fly.io isn't already running (prevents duplicate deployments)

## Cost on Fly.io

**Free Tier:**
- 3 shared-cpu-1x VMs (256MB RAM each)
- 3GB persistent storage
- 160GB outbound data transfer

**If you exceed free tier:** ~$1.94/month per 256MB VM

## Update SeaTalk Webhook

When switching to Fly.io, update your SeaTalk callback URL:

```
https://bot-workstation.fly.dev/seatalk/callback
```

Or use both (SeaTalk supports multiple webhooks):
1. `https://your-render-service.onrender.com/seatalk/callback`
2. `https://bot-workstation.fly.dev/seatalk/callback`

## Monitoring

Check Fly.io status:
```bash
fly status
fly logs
```

Health check:
```bash
curl https://bot-workstation.fly.dev/healthz
```

## Switch Back to Render

1. Go to **Actions → Manual Switch to Fly.io**
2. Select action: `destroy`
3. Type `yes` to confirm
4. Your bot is now back on Render only

