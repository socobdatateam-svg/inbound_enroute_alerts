# Apps Script Port

This folder contains a Google Apps Script version of the Go SeaTalk bot.

## What Is Converted

- Callback challenge response.
- Bot-added and bot-removed group tracking in `bot_config!A2:A`.
- Group ID normalization.
- SeaTalk app access token caching.
- Scheduled report sending at `12MN, 4AM, 6AM, 10AM, 1PM, 3PM, 6PM, 9PM`.
- Interactive card sending with the report link.
- Google Sheets PDF export for `Compliance Tracker!A1:X80`.

## Important Difference

The Go service renders the Sheets PDF into a PNG with `pdftoppm` and ImageMagick. Apps Script cannot run those binaries and does not provide a native API to render a sheet range as a PNG.

Because of that, this port sends:

1. the interactive SeaTalk card, then
2. the exported report as a PDF file attachment.

To preserve the exact image-in-card behavior, keep a small external renderer service or use a third-party PDF-to-image API, then call it from Apps Script.

Apps Script web app handlers also do not expose arbitrary request headers, so the SeaTalk `Signature` header cannot be verified in pure Apps Script. The sample supports an optional `CALLBACK_TOKEN` query parameter as a weaker callback guard.

## Setup

1. Create a new Apps Script project.
2. Paste `Code.gs` into the script editor.
3. Set script properties:

```text
SEATALK_APP_ID=your_app_id
SEATALK_APP_SECRET=your_app_secret
CALLBACK_TOKEN=random_long_secret_optional
```

4. Make sure the Google account that owns/runs the script has access to the spreadsheet.
5. Deploy as a Web App:
   - Execute as: `Me`
   - Who has access: depends on SeaTalk callback requirements; commonly `Anyone`
6. Use the Web App URL as the SeaTalk callback URL. If you set `CALLBACK_TOKEN`, append it as `?token=random_long_secret_optional`.
7. Run `installScheduledSendTriggers()` once from the editor.
8. Run `installDailyGroupSyncTrigger()` once from the editor.
9. Run `testReport()` manually to test sending.

## Required Apps Script Scopes

Apps Script will prompt for these when you run/deploy:

- Read/write Google Sheets.
- External requests through `UrlFetchApp`.
- Script triggers.
- Spreadsheet export through the current user's OAuth token.

## Notes

- Apps Script web apps cannot read SeaTalk's `Signature` header or set arbitrary HTTP status codes through `ContentService`. `doPost` returns text/JSON responses, but SeaTalk should rely mainly on the response body and successful callback handling.
- File messages are subject to SeaTalk's file size limit. If the exported PDF exceeds the limit, reduce the capture range or split the report.
