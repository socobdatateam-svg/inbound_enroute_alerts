const CONFIG = {
  SHEET_ID: '1LiSwe5XABNPSPIhdK-Hu7S8VjWEsjPmUHmnU3toGXhc',
  TAB_NAME: 'Compliance Tracker',
  CAPTURE_RANGE: 'A1:X80',
  BOT_CONFIG_TAB: 'bot_config',
  BOT_NAME: 'Bot Workstation',
  REPORT_LINK: 'https://docs.google.com/spreadsheets/d/1hYCkLL9Z4UR3WeKFuCDsOYch5v1FxmTJLGtk8UG_yyI/edit?gid=2001886446#gid=2001886446',
  TIMEZONE: 'Asia/Manila',
  SEATALK_TOKEN_URL: 'https://openapi.seatalk.io/auth/app_access_token',
  SEATALK_GROUP_MESSAGE_URL: 'https://openapi.seatalk.io/messaging/v2/group_chat',
};

const EVENT_VERIFICATION = 'event_verification';
const EVENT_BOT_ADDED_TO_GROUP_CHAT = 'bot_added_to_group_chat';
const EVENT_BOT_REMOVED_FROM_GROUP_CHAT = 'bot_removed_from_group_chat';

/**
 * SeaTalk event callback entry point.
 *
 * Configure the Apps Script deployment URL as the SeaTalk callback URL.
 */
function doPost(e) {
  const body = e && e.postData && e.postData.contents ? e.postData.contents : '';

  if (!validCallbackToken_(e)) {
    console.warn('SeaTalk callback rejected: invalid callback token');
    return textResponse_('unauthorized');
  }

  const event = JSON.parse(body);
  console.log(`SeaTalk callback received: event_type=${event.event_type} event_id=${event.event_id}`);

  if (event.event_type === EVENT_VERIFICATION) {
    return jsonResponse_({
      seatalk_challenge: event.event && event.event.seatalk_challenge || '',
    });
  }

  handleSeaTalkEvent_(event);
  return textResponse_('ok', 200);
}

/**
 * Lightweight health endpoint.
 */
function doGet() {
  return textResponse_('ok', 200);
}

/**
 * Run this manually from the Apps Script editor to test report sending.
 */
function testReport() {
  sendReportNow();
}

/**
 * Run this from an Apps Script time trigger.
 */
function sendReportNow() {
  const lock = LockService.getScriptLock();
  if (!lock.tryLock(1000)) {
    throw new Error('report send already running');
  }

  try {
    const groupIds = groupIDs_();
    if (groupIds.length === 0) {
      throw new Error(`no SeaTalk group IDs found in ${CONFIG.BOT_CONFIG_TAB}!A2:A`);
    }

    const card = buildAlertCard_();
    const pdf = exportReportPdf_();

    groupIds.forEach(groupId => {
      sendInteractiveAlert_(groupId, card);
      sendFile_(groupId, pdf.filename, pdf.base64);
      console.log(`sent interactive card and PDF report to ${groupId}`);
    });
  } finally {
    lock.releaseLock();
  }
}

/**
 * Run daily from a time trigger to sort and deduplicate bot_config!A2:A.
 */
function normalizeGroupIDs() {
  writeGroupIDs_(groupIDs_());
}

/**
 * One-time helper. Run manually to create daily triggers matching the Go app schedule.
 */
function installScheduledSendTriggers() {
  deleteTriggers_('sendReportNow');
  [0, 4, 6, 10, 13, 15, 18, 21].forEach(hour => {
    ScriptApp.newTrigger('sendReportNow')
      .timeBased()
      .atHour(hour)
      .everyDays(1)
      .inTimezone(CONFIG.TIMEZONE)
      .create();
  });
}

/**
 * One-time helper. Run manually to create the daily group normalization trigger.
 */
function installDailyGroupSyncTrigger() {
  deleteTriggers_('normalizeGroupIDs');
  ScriptApp.newTrigger('normalizeGroupIDs')
    .timeBased()
    .atHour(2)
    .everyDays(1)
    .inTimezone(CONFIG.TIMEZONE)
    .create();
}

function handleSeaTalkEvent_(event) {
  const group = event.event && event.event.group || {};
  const groupId = group.group_id || '';

  if (event.event_type === EVENT_BOT_ADDED_TO_GROUP_CHAT) {
    if (!groupId) {
      console.warn('bot_added_to_group_chat received without group_id');
      return;
    }
    upsertGroupID_(groupId);
    console.log(`stored group id ${groupId} in ${CONFIG.BOT_CONFIG_TAB}`);
    return;
  }

  if (event.event_type === EVENT_BOT_REMOVED_FROM_GROUP_CHAT) {
    if (!groupId) {
      console.warn('bot_removed_from_group_chat received without group_id');
      return;
    }
    removeGroupID_(groupId);
    console.log(`removed group id ${groupId} from ${CONFIG.BOT_CONFIG_TAB}`);
    return;
  }

  console.log(`ignored SeaTalk event type ${event.event_type}`);
}

function buildAlertCard_() {
  const sheet = spreadsheet_().getSheetByName(CONFIG.TAB_NAME);
  const controlTowerUpdate = sheet.getRange('E1').getDisplayValue() || '-';
  return {
    updatedAt: Utilities.formatDate(new Date(), CONFIG.TIMEZONE, 'h:mma MMM-dd'),
    controlTowerUpdate: controlTowerUpdate,
    reportLink: CONFIG.REPORT_LINK,
  };
}

function sendInteractiveAlert_(groupId, card) {
  const description = [
    '----------------------------------',
    'OTP Control Tower',
    `Latest Update: ${card.controlTowerUpdate || '-'}`,
    '----------------------------------',
  ].join('\n');

  const message = {
    tag: 'interactive_message',
    interactive_message: {
      elements: [
        {
          element_type: 'title',
          title: {
            text: `${CONFIG.BOT_NAME} Compliance as of ${card.updatedAt}`,
          },
        },
        {
          element_type: 'description',
          description: {
            format: 1,
            text: description,
          },
        },
        {
          element_type: 'button',
          button: {
            button_type: 'redirect',
            text: 'View Report Link',
            mobile_link: {
              type: 'web',
              path: card.reportLink,
            },
            desktop_link: {
              type: 'web',
              path: card.reportLink,
            },
          },
        },
      ],
    },
  };

  sendGroupMessage_(groupId, message);
}

function sendFile_(groupId, filename, base64Content) {
  sendGroupMessage_(groupId, {
    tag: 'file',
    file: {
      filename: filename,
      content: base64Content,
    },
  });
}

function sendGroupMessage_(groupId, message) {
  postSeaTalkAuthed_(CONFIG.SEATALK_GROUP_MESSAGE_URL, {
    group_id: groupId,
    message: message,
  });
}

function postSeaTalkAuthed_(url, payload) {
  const response = UrlFetchApp.fetch(url, {
    method: 'post',
    contentType: 'application/json',
    headers: {
      Authorization: `Bearer ${appAccessToken_()}`,
    },
    payload: JSON.stringify(payload),
    muteHttpExceptions: true,
  });

  const status = response.getResponseCode();
  const text = response.getContentText();
  if (status < 200 || status >= 300) {
    throw new Error(`SeaTalk status ${status}: ${text}`);
  }

  const parsed = JSON.parse(text);
  if (parsed.code !== 0) {
    throw new Error(`SeaTalk code ${parsed.code}: ${parsed.msg || text}`);
  }
  return parsed;
}

function appAccessToken_() {
  const cache = CacheService.getScriptCache();
  const cached = cache.get('seatalk_app_access_token');
  if (cached) {
    return cached;
  }

  const props = PropertiesService.getScriptProperties();
  const response = UrlFetchApp.fetch(CONFIG.SEATALK_TOKEN_URL, {
    method: 'post',
    contentType: 'application/json',
    payload: JSON.stringify({
      app_id: requiredProperty_('SEATALK_APP_ID'),
      app_secret: requiredProperty_('SEATALK_APP_SECRET'),
    }),
    muteHttpExceptions: true,
  });

  const status = response.getResponseCode();
  const text = response.getContentText();
  if (status < 200 || status >= 300) {
    throw new Error(`SeaTalk token status ${status}: ${text}`);
  }

  const parsed = JSON.parse(text);
  if (parsed.code !== 0 || !parsed.app_access_token) {
    throw new Error(`SeaTalk token code ${parsed.code}: ${text}`);
  }

  const ttlSeconds = Math.max(60, Math.min(6900, Number(parsed.expire) - Math.floor(Date.now() / 1000) - 300));
  cache.put('seatalk_app_access_token', parsed.app_access_token, ttlSeconds);
  return parsed.app_access_token;
}

function exportReportPdf_() {
  const gid = sheetGid_(CONFIG.TAB_NAME);
  const params = {
    format: 'pdf',
    gid: String(gid),
    range: CONFIG.CAPTURE_RANGE,
    size: 'A4',
    portrait: 'false',
    fitw: 'true',
    sheetnames: 'false',
    printtitle: 'false',
    pagenumbers: 'false',
    gridlines: 'false',
    fzr: 'false',
  };
  const query = Object.keys(params)
    .map(key => `${encodeURIComponent(key)}=${encodeURIComponent(params[key])}`)
    .join('&');
  const url = `https://docs.google.com/spreadsheets/d/${CONFIG.SHEET_ID}/export?${query}`;
  const response = UrlFetchApp.fetch(url, {
    headers: {
      Authorization: `Bearer ${ScriptApp.getOAuthToken()}`,
    },
    muteHttpExceptions: true,
  });

  const status = response.getResponseCode();
  if (status < 200 || status >= 300) {
    throw new Error(`sheet export status ${status}: ${response.getContentText()}`);
  }

  const filename = `bot-workstation-${Utilities.formatDate(new Date(), CONFIG.TIMEZONE, 'yyyyMMdd-HHmmss')}.pdf`;
  return {
    filename: filename,
    base64: Utilities.base64Encode(response.getBlob().getBytes()),
  };
}

function groupIDs_() {
  const sheet = botConfigSheet_();
  const lastRow = sheet.getLastRow();
  if (lastRow < 2) {
    return [];
  }

  const values = sheet.getRange(2, 1, lastRow - 1, 1).getDisplayValues();
  const seen = {};
  const ids = [];
  values.forEach(row => {
    const id = String(row[0] || '').trim();
    if (id && !seen[id]) {
      seen[id] = true;
      ids.push(id);
    }
  });
  return ids;
}

function upsertGroupID_(groupId) {
  const ids = groupIDs_();
  if (ids.indexOf(groupId) === -1) {
    ids.push(groupId);
  }
  writeGroupIDs_(ids);
}

function removeGroupID_(groupId) {
  writeGroupIDs_(groupIDs_().filter(id => id !== groupId));
}

function writeGroupIDs_(ids) {
  const sheet = botConfigSheet_();
  const lastRow = Math.max(2, sheet.getLastRow());
  sheet.getRange(2, 1, lastRow - 1, 1).clearContent();

  const sorted = Array.from(new Set(ids.filter(Boolean))).sort();
  if (sorted.length > 0) {
    sheet.getRange(2, 1, sorted.length, 1).setValues(sorted.map(id => [id]));
  }
}

function sheetGid_(tabName) {
  const sheet = spreadsheet_().getSheetByName(tabName);
  if (!sheet) {
    throw new Error(`tab "${tabName}" not found`);
  }
  return sheet.getSheetId();
}

function spreadsheet_() {
  return SpreadsheetApp.openById(CONFIG.SHEET_ID);
}

function botConfigSheet_() {
  const sheet = spreadsheet_().getSheetByName(CONFIG.BOT_CONFIG_TAB);
  if (!sheet) {
    throw new Error(`tab "${CONFIG.BOT_CONFIG_TAB}" not found`);
  }
  return sheet;
}

function validCallbackToken_(e) {
  const expected = PropertiesService.getScriptProperties().getProperty('CALLBACK_TOKEN');
  if (!expected) {
    return true;
  }
  return e && e.parameter && e.parameter.token === expected;
}

function requiredProperty_(key) {
  const value = PropertiesService.getScriptProperties().getProperty(key);
  if (!value) {
    throw new Error(`missing script property ${key}`);
  }
  return value;
}

function jsonResponse_(value) {
  return ContentService
    .createTextOutput(JSON.stringify(value))
    .setMimeType(ContentService.MimeType.JSON);
}

function textResponse_(value) {
  return ContentService
    .createTextOutput(value)
    .setMimeType(ContentService.MimeType.TEXT);
}

function deleteTriggers_(handlerName) {
  ScriptApp.getProjectTriggers()
    .filter(trigger => trigger.getHandlerFunction() === handlerName)
    .forEach(trigger => ScriptApp.deleteTrigger(trigger));
}
