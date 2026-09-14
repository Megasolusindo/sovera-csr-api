/**
 * OpenClaw AI Agent — Telegram Notifier Integration
 * Sends automated HTML alert notifications to Telegram Chat for AI findings & monitoring triggers
 */

const https = require('https');
const dns = require('dns');

const TELEGRAM_BOT_TOKEN = process.env.TELEGRAM_BOT_TOKEN || '8850608348:AAHXkUz7nldwf2WR9dmMm5NI9Yox-WWC2Gw';
const TELEGRAM_CHAT_ID = process.env.TELEGRAM_CHAT_ID || '87116487';

let cachedTelegramIp = null;
let lastDnsResolveTime = 0;

function getTelegramIp() {
  return new Promise((resolve) => {
    const now = Date.now();
    if (cachedTelegramIp && (now - lastDnsResolveTime < 300000)) {
      return resolve(cachedTelegramIp);
    }
    dns.resolve4('api.telegram.org', (err, addresses) => {
      if (!err && addresses && addresses.length > 0) {
        cachedTelegramIp = addresses[0];
        lastDnsResolveTime = now;
        return resolve(cachedTelegramIp);
      }
      resolve('149.154.166.110');
    });
  });
}

function sanitizeTelegramHtml(html) {
  if (!html) return "";

  let clean = html
    .replace(/\*\*([^\*\n]+?)\*\*/g, "<b>$1</b>")
    .replace(/__([^_\n]+?)__/g, "<b>$1</b>")
    .replace(/\*([^\*\n]+?)\*/g, "<i>$1</i>")
    .replace(/_([^_\n]+?)_/g, "<i>$1</i>")
    .replace(/`([^`\n]+?)`/g, "<code>$1</code>")
    .replace(/<strong[^>]*>/gi, "<b>")
    .replace(/<\/strong>/gi, "</b>")
    .replace(/<em[^>]*>/gi, "<i>")
    .replace(/<\/em>/gi, "</i>")
    .replace(/<p[^>]*>/gi, "")
    .replace(/<\/p>/gi, "\n\n")
    .replace(/<br\s*\/?>/gi, "\n")
    .replace(/<ul[^>]*>/gi, "")
    .replace(/<\/ul>/gi, "\n")
    .replace(/<ol[^>]*>/gi, "")
    .replace(/<\/ol>/gi, "\n")
    .replace(/<li[^>]*>/gi, "• ")
    .replace(/<\/li>/gi, "\n")
    .replace(/<h[1-6][^>]*>/gi, "\n<b>")
    .replace(/<\/h[1-6]>/gi, "</b>\n")
    .replace(/<div[^>]*>/gi, "")
    .replace(/<\/div>/gi, "\n");

  clean = clean.replace(/<(?!Content|\/?(?:b|i|u|s|code|pre|a)\b)[^>]+>/gi, "");

  const openB = (clean.match(/<b>/gi) || []).length;
  const closeB = (clean.match(/<\/b>/gi) || []).length;
  if (openB > closeB) {
    clean += "</b>".repeat(openB - closeB);
  }

  const openI = (clean.match(/<i>/gi) || []).length;
  const closeI = (clean.match(/<\/i>/gi) || []).length;
  if (openI > closeI) {
    clean += "</i>".repeat(openI - closeI);
  }

  return clean.trim();
}

async function sendTelegramMessage(rawText, targetChatId = null, isRetry = false) {
  if (typeof targetChatId === 'boolean') {
    isRetry = targetChatId;
    targetChatId = null;
  }

  if (!TELEGRAM_BOT_TOKEN) return false;
  const destChatId = targetChatId || TELEGRAM_CHAT_ID;
  if (!destChatId) return false;

  const targetIp = await getTelegramIp();
  const textToSend = isRetry ? rawText.replace(/<[^>]+>/g, '') : sanitizeTelegramHtml(rawText);

  const postDataPayload = {
    chat_id: destChatId,
    text: textToSend,
    disable_web_page_preview: false
  };

  if (!isRetry && (textToSend.includes('<b>') || textToSend.includes('<i>') || textToSend.includes('<code>'))) {
    postDataPayload.parse_mode = 'HTML';
  }

  const postData = JSON.stringify(postDataPayload);

  return new Promise((resolve) => {
    const options = {
      hostname: targetIp,
      port: 443,
      path: `/bot${TELEGRAM_BOT_TOKEN}/sendMessage`,
      method: 'POST',
      servername: 'api.telegram.org',
      headers: {
        'Host': 'api.telegram.org',
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData)
      }
    };

    const req = https.request(options, (res) => {
      let body = '';
      res.on('data', (chunk) => body += chunk);
      res.on('end', () => {
        if (res.statusCode === 200) {
          console.log(`[OpenClaw Telegram] ✅ Notification delivered to Telegram Chat ID: ${destChatId}`);
          resolve(true);
        } else {
          console.error(`[OpenClaw Telegram] ❌ Telegram API status ${res.statusCode}:`, body);
          if (!isRetry && res.statusCode === 400) {
            console.log('[OpenClaw Telegram] HTML parse error detected. Retrying as plain text...');
            sendTelegramMessage(rawText, destChatId, true).then(resolve);
          } else {
            resolve(false);
          }
        }
      });
    });

    req.on('error', (err) => {
      console.error('[OpenClaw Telegram] Error sending message:', err.message, err.code);
      if (!isRetry) {
        console.log('[OpenClaw Telegram] Retrying message delivery...');
        setTimeout(() => {
          sendTelegramMessage(rawText, destChatId, true).then(resolve);
        }, 1000);
      } else {
        resolve(false);
      }
    });

    req.write(postData);
    req.end();
  });
}

/**
 * Send alert when OpenClaw Research Agent discovers a new CSR finding
 */
async function notifyNewFinding(finding) {
  const isMonitoring = finding.finding_type === 'company_monitoring';
  const icon = isMonitoring ? '📡' : '🤖';
  const title = isMonitoring ? 'OPENCLAW WATCHLIST MONITORING ALERT' : 'OPENCLAW AI CSR RESEARCH FINDING';

  const message = 
    `<b>${icon} ${title}</b>\n\n` +
    `<b>Perusahaan:</b> ${finding.company_name || 'N/A'}\n` +
    `<b>Judul:</b> ${finding.title}\n` +
    `<b>Tipe:</b> <code>${finding.finding_type}</code>\n` +
    `<b>Confidence Score:</b> ${(finding.confidence_score * 100).toFixed(0)}%\n\n` +
    `<b>Ringkasan:</b> ${finding.summary || '-'}\n\n` +
    `<b>Sumber Evidence:</b> <a href="${finding.source_url}">${finding.source_name || 'View Source'}</a>\n` +
    `<b>Status Queue:</b> <code>pending_review</code>\n` +
    `<i>Waktu Deteksi: ${new Date().toLocaleString('id-ID')} UTC</i>`;

  return await sendTelegramMessage(message);
}

/**
 * Send alert when OpenClaw Matching Agent finds high-score corporate prospects
 */
async function notifyMatchingResult(programTitle, topMatches) {
  if (!topMatches || topMatches.length === 0) return false;

  let matchesListText = '';
  topMatches.slice(0, 3).forEach((m, idx) => {
    matchesListText += `<b>${idx + 1}. ${m.company_name}</b> (Score: ${m.match_score}% — ${m.match_grade})\n`;
    matchesListText += `   <i>${m.match_rationale}</i>\n\n`;
  });

  const message = 
    `🎯 <b>OPENCLAW PROGRAM PARTNER MATCHING REPORT</b>\n\n` +
    `<b>Program Proposal:</b> ${programTitle}\n` +
    `<b>Top Recommended Corporate Partners:</b>\n\n` +
    matchesListText +
    `<i>Dihitung secara otomatis oleh OpenClaw Matching Agent</i>`;

  return await sendTelegramMessage(message);
}

/**
 * Register slash commands menu into Telegram Bot API via setMyCommands
 */
async function registerTelegramBotCommands() {
  if (!TELEGRAM_BOT_TOKEN) return false;

  const commandsList = [
    { command: "help", description: "Lihat menu bantuan & daftar perintah" },
    { command: "start", description: "Mulai obrolan dengan OpenClaw AI" },
    { command: "research", description: "Riset program CSR perusahaan (e.g. /research Pertamina)" },
    { command: "monitor", description: "Jalankan monitoring watchlist CSR sekarang" },
    { command: "match", description: "Matching mitra korporasi & program (e.g. /match Kesehatan Jakarta)" },
    { command: "watchlist", description: "Lihat & kelola daftar perusahaan dipantau" },
    { command: "check_linkedin", description: "Verifikasi keaktifan URL LinkedIn" },
    { command: "discover_linkedin", description: "Pencarian empiris LinkedIn perusahaan INVALID" },
    { command: "linkedin_stats", description: "Lihat statistik audit LinkedIn seluruh perusahaan" },
    { command: "check_instagram", description: "Verifikasi keaktifan URL Instagram" },
    { command: "discover_instagram", description: "Pencarian empiris Instagram perusahaan INVALID" },
    { command: "instagram_stats", description: "Lihat statistik audit Instagram seluruh perusahaan" },
    { command: "sync_idx", description: "Sinkronisasi real data emiten resmi dari IDX (BEI)" },
    { command: "stats", description: "Lihat statistik real-time system of record & crawler" }
  ];

  const postData = JSON.stringify({ commands: commandsList });

  return new Promise((resolve) => {
    const options = {
      hostname: 'api.telegram.org',
      port: 443,
      path: `/bot${TELEGRAM_BOT_TOKEN}/setMyCommands`,
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Content-Length': Buffer.byteLength(postData)
      }
    };

    const req = https.request(options, (res) => {
      let body = '';
      res.on('data', (chunk) => body += chunk);
      res.on('end', () => {
        if (res.statusCode === 200) {
          console.log(`[OpenClaw Telegram] ✅ Successfully registered ${commandsList.length} slash commands via setMyCommands!`);
          resolve(true);
        } else {
          console.error(`[OpenClaw Telegram] ❌ setMyCommands failed status ${res.statusCode}:`, body);
          resolve(false);
        }
      });
    });

    req.on('error', (err) => {
      console.error('[OpenClaw Telegram] Error registering commands:', err.message);
      resolve(false);
    });

    req.write(postData);
    req.end();
  });
}

module.exports = {
  sendTelegramMessage,
  notifyNewFinding,
  notifyMatchingResult,
  registerTelegramBotCommands
};

