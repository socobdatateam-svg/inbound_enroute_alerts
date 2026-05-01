# Bot Workstation

Go SeaTalk bot that watches `Summary Sheet (In progress)!AE6`. When the value changes, it waits 7 seconds for dependent sheet data to settle, then captures `Compliance Tracker!A1:X80`, renders it as an image, and sends an interactive SeaTalk card to every known group.

## Requirements

- SeaTalk app with bot capability, event callback, and group message permission enabled.
- Google service account with access to spreadsheet `1APnTQXUQvWpTwmOLIC9U17kwjQcWX0BPYZvlfUPOJrU`.
- The spreadsheet must have:
  - `Compliance Tracker`
  - `bot_config`
  - group IDs stored in `bot_config!A2:A`

## Configure

Copy `.env.example` to `.env` and fill in the secrets:

```env
SEATALK_APP_ID=
SEATALK_APP_SECRET=
SEATALK_SIGNING_SECRET=
ADMIN_TOKEN=
GOOGLE_APPLICATION_CREDENTIALS=/run/secrets/google-service-account.json
```

The server loads `.env` automatically for local runs. Environment variables already set outside the file take precedence.

For Render, set `GOOGLE_CREDENTIALS_JSON` instead of mounting a credential file. The bot name is controlled by:

```env
BOT_NAME=Bot Workstation
```

Set the SeaTalk callback URL to:

```text
https://your-public-host/seatalk/callback
```

## Change-Triggered Sends

The first read of the watched cell is treated as the baseline and does not send. Later value changes trigger a delayed send.

```env
ENABLE_CHANGE_SENDS=true
WATCH_TAB=Summary Sheet (In progress)
WATCH_CELL=AE6
WATCH_POLL_SECONDS=5
CHANGE_SETTLE_SECONDS=7
```

Google Sheets limits service-account reads to 60 requests per minute per user. A 5-second poll interval keeps the watcher at about 12 reads per minute before report sends and callback updates.

Image render defaults:

```env
PNG_DPI=300
PNG_MAX_WIDTH=2400
```

Scheduled sends are optional and disabled by default:

```env
ENABLE_SCHEDULED_SENDS=false
```

## Run

```bash
docker compose up --build
```

Health check:

```text
GET /healthz
```

Manual report test, enabled only when `ADMIN_TOKEN` is set:

```bash
curl -X POST https://your-public-host/admin/test-report \
  -H "Authorization: Bearer your-admin-token"
```

## Deploy On Render

This repo includes [render.yaml](render.yaml) for a Docker web service.

Set these secret environment variables in Render:

```env
SEATALK_APP_ID=
SEATALK_APP_SECRET=
SEATALK_SIGNING_SECRET=
ADMIN_TOKEN=
GOOGLE_CREDENTIALS_JSON=
```

After deployment, use the Render service URL:

```text
https://your-render-service.onrender.com/healthz
https://your-render-service.onrender.com/seatalk/callback
https://your-render-service.onrender.com/admin/test-report
```

## Group ID Handling

When the bot is added to a SeaTalk group, the callback handler stores the `group_id` in `bot_config!A2:A`. When the bot is removed, it removes that ID. A daily sync normalizes the sheet list by sorting and deduplicating known IDs.
