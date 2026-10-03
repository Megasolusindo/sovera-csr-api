/**
 * OpenClaw AI Agent — Humanitarian Organization Discovery Skill
 *
 * Flow (triggered by Telegram `/discover <domain>`):
 *   1. Scrape the member directory (JS-rendered, Elementor AJAX pagination) with Playwright.
 *   2. For each organization: enrich website, email, and phone/WhatsApp — VERIFIED ONLY.
 *   3. Upsert into the organizations table via CSR API (skip-if-exists, fill-empty).
 *
 * Quality policy: only store data we can stand behind.
 *   - email: syntactically valid AND its domain matches the org's own website host.
 *   - phone: must appear with an explicit +62 / 62 country code near a contact keyword.
 *   The Go endpoint re-validates the phone with phoneverifier before persisting.
 */

const http = require('http');
const https = require('https');
const { URL } = require('url');

const CSR_API_BASE_URL = process.env.CSR_API_BASE_URL || 'http://api:4000/api/v1/ai';
const OPENCLAW_AGENT_TOKEN = process.env.OPENCLAW_AGENT_TOKEN || 'openclaw_agent_live_key_998877665544';

// ---- small HTTP helper (same shape as other skills) ----
function httpRequest(urlStr, options = {}, postData = null) {
  return new Promise((resolve) => {
    let parsed;
    try { parsed = new URL(urlStr); } catch (e) { return resolve({ statusCode: 0, error: e.message }); }
    const lib = parsed.protocol === 'https:' ? https : http;
    const reqOptions = {
      hostname: parsed.hostname,
      port: parsed.port || (parsed.protocol === 'https:' ? 443 : 80),
      path: parsed.pathname + parsed.search,
      method: options.method || 'GET',
      headers: { 'User-Agent': 'OpenClaw-AI-Agent/1.0', ...options.headers },
    };
    const req = lib.request(reqOptions, (res) => {
      let body = '';
      res.on('data', (c) => (body += c));
      res.on('end', () => {
        try { resolve({ statusCode: res.statusCode, data: JSON.parse(body), raw: body }); }
        catch (e) { resolve({ statusCode: res.statusCode, raw: body }); }
      });
    });
    req.on('error', (e) => resolve({ statusCode: 0, error: e.message }));
    req.setTimeout(20000, () => { req.destroy(); resolve({ statusCode: 0, error: 'timeout' }); });
    if (postData) req.write(typeof postData === 'string' ? postData : JSON.stringify(postData));
    req.end();
  });
}

// ---- contact extraction helpers ----
const EMAIL_RE = /[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}/g;
// Indonesian phone that carries an explicit country code — the only shape we trust.
const PHONE_CC_RE = /(?:\+62|62)[\s\-.()]?\d{2,4}(?:[\s\-.()]?\d{2,4}){1,4}/g;
const CONTACT_KW = ['telp', 'telepon', 'phone', 'whatsapp', 'wa', 'hubungi', 'kontak', 'contact', 'hotline'];

function emailJunk(e) {
  const l = e.toLowerCase();
  return ['example.com', 'sentry', 'wixpress', '@2x', '.png', '.jpg', '.svg', '.gif', '.webp',
    'domain.com', 'yourname', 'u003e', 'react', 'schema.org', 'partner-sovera'].some((b) => l.includes(b));
}
function digits(s) { return (s.match(/\d/g) || []).join(''); }
function normalizeID(raw) {
  let d = digits(raw);
  if (d.startsWith('62')) d = '0' + d.slice(2);
  return d;
}
function looksPlaceholderPhone(norm) {
  let run = 1;
  for (let i = 1; i < norm.length; i++) {
    if (norm[i] === norm[i - 1]) { if (++run >= 5) return true; } else run = 1;
  }
  if (norm.length >= 9) {
    const b = norm.slice(-9);
    if (b.slice(0, 3) === b.slice(3, 6) && b.slice(3, 6) === b.slice(6, 9)) return true;
  }
  return false;
}

function hostOf(u) {
  try { return new URL(u.startsWith('http') ? u : 'https://' + u).hostname.replace(/^www\./, ''); }
  catch (e) { return ''; }
}

// Fetch a page's raw HTML (follows one redirect) for contact scraping.
function fetchHTML(urlStr) {
  return new Promise((resolve) => {
    let parsed;
    try { parsed = new URL(urlStr.startsWith('http') ? urlStr : 'https://' + urlStr); }
    catch (e) { return resolve(''); }
    const lib = parsed.protocol === 'https:' ? https : http;
    const req = lib.request({
      hostname: parsed.hostname, port: parsed.port || 443, path: parsed.pathname + parsed.search,
      method: 'GET', headers: { 'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 Chrome/120 Safari/537.36' },
    }, (res) => {
      if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
        res.resume();
        return resolve(fetchHTML(new URL(res.headers.location, parsed).href));
      }
      if (res.statusCode < 200 || res.statusCode >= 400) { res.resume(); return resolve(''); }
      let body = '';
      let size = 0;
      res.on('data', (c) => { size += c.length; if (size < 900 * 1024) body += c; });
      res.on('end', () => resolve(body));
    });
    req.on('error', () => resolve(''));
    req.setTimeout(15000, () => { req.destroy(); resolve(''); });
    req.end();
  });
}

// Extract a VERIFIED email (domain must match the site host) from a site.
async function discoverEmail(website) {
  if (!website) return '';
  const host = hostOf(website);
  const paths = ['', '/contact', '/kontak', '/contact-us', '/hubungi-kami', '/about', '/tentang-kami'];
  for (const p of paths) {
    const html = await fetchHTML(website.replace(/\/$/, '') + p);
    if (!html) continue;
    const matches = html.match(EMAIL_RE) || [];
    for (let e of matches) {
      e = e.replace(/\.$/, '');
      if (emailJunk(e)) continue;
      const dom = e.split('@')[1].toLowerCase();
      // trust only addresses on the org's own domain
      if (host && (dom === host || dom.endsWith('.' + host))) return e.toLowerCase();
    }
  }
  return '';
}

// Extract a VERIFIED phone (explicit +62, near a contact keyword) from a site.
async function discoverPhone(website) {
  if (!website) return '';
  const paths = ['/contact', '/kontak', '/contact-us', '/hubungi-kami', ''];
  for (const p of paths) {
    const html = await fetchHTML(website.replace(/\/$/, '') + p);
    if (!html) continue;
    const lower = html.toLowerCase();
    let m;
    PHONE_CC_RE.lastIndex = 0;
    while ((m = PHONE_CC_RE.exec(html)) !== null) {
      const start = Math.max(0, m.index - 60);
      const window = lower.slice(start, m.index + 10);
      if (!CONTACT_KW.some((kw) => window.includes(kw))) continue;
      const norm = normalizeID(m[0]);
      if (norm.length < 9 || norm.length > 13) continue;
      if (looksPlaceholderPhone(norm)) continue;
      return norm; // server re-validates with phoneverifier
    }
  }
  return '';
}

const SEARCH_BAD_HOSTS = ['facebook.', 'instagram.', 'linkedin.', 'youtube.', 'twitter.', 'x.com',
  'wikipedia.', 'tokopedia.', 'detik.', 'kompas.', 'tribunnews.', 'duckduckgo.', 'bing.com',
  'google.', 'kitabisa.com', 'tempo.co', 'liputan6.', 'merdeka.com', 'antaranews.'];

function sleep(ms) { return new Promise((r) => setTimeout(r, ms)); }

// Build candidate domain slugs from an org name, most specific first.
function domainSlugs(name) {
  const n = name.toLowerCase()
    .replace(/\([^)]*\)/g, ' ')        // drop parentheticals
    .replace(/[^a-z0-9 ]/g, ' ')
    .replace(/\s+/g, ' ').trim();
  const stop = ['yayasan', 'foundation', 'indonesia', 'the', 'and', 'dan', 'perkumpulan',
    'lembaga', 'pusat', 'nasional', 'institute', 'center', 'centre'];
  const words = n.split(' ').filter((w) => w && !stop.includes(w));
  const cands = [];
  if (words.length) cands.push(words.join(''));          // core words joined
  cands.push(n.replace(/ /g, ''));                        // all words joined (keeps "foundation")
  if (words.length >= 2) cands.push(words.slice(0, 2).join(''));
  if (words.length) cands.push(words[0]);                 // first core word (last resort)
  return [...new Set(cands)].filter((s) => s && s.length >= 4);
}

// Fetch HTML (one redirect) and tell whether the org's name tokens appear on the page.
async function pageMentionsOrg(url, name) {
  const html = await fetchHTML(url);
  if (!html) return false;
  const text = html.toLowerCase();
  const tokens = name.toLowerCase().replace(/\([^)]*\)/g, ' ').replace(/[^a-z0-9 ]/g, ' ')
    .split(/\s+/).filter((w) => w.length >= 4 &&
      !['yayasan', 'foundation', 'indonesia', 'lembaga', 'perkumpulan', 'institute', 'center', 'centre'].includes(w));
  if (tokens.length === 0) return true; // nothing distinctive to check
  // require at least one distinctive token to appear in the page
  return tokens.some((t) => text.includes(t));
}

// Last-resort website discovery with NO search engine: guess domains from the name,
// prefer non-profit TLDs, verify the page actually mentions the org (avoids grabbing
// an unrelated domain that merely resolves). Free, deterministic, no rate limits.
async function discoverWebsiteGuess(name) {
  const slugs = domainSlugs(name);
  const tlds = ['.or.id', '.org', '.id', '.co.id', '.com']; // non-profit TLDs first
  for (const s of slugs) {
    for (const t of tlds) {
      const url = 'https://' + s + t;
      const host = hostOf(url);
      if (SEARCH_BAD_HOSTS.some((b) => host.includes(b))) continue;
      if (await pageMentionsOrg(url, name)) return url;
    }
  }
  return '';
}

// Single DuckDuckGo HTML query. Returns {host, blocked}. blocked=true means DDG
// served an anti-bot challenge (HTTP 202 / no uddg links) and the call should be retried.
function ddgOnce(name) {
  return new Promise((resolve) => {
    const q = encodeURIComponent(`${name} Indonesia`);
    const req = https.request({
      hostname: 'html.duckduckgo.com', path: '/html/?q=' + q, method: 'GET',
      headers: {
        'User-Agent': 'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17 Safari/605.1.15',
        'Accept': 'text/html,application/xhtml+xml',
        'Accept-Language': 'id-ID,id;q=0.9,en;q=0.8',
      },
    }, (r) => {
      let b = '';
      r.on('data', (c) => (b += c));
      r.on('end', () => {
        const links = [...b.matchAll(/uddg=([^&"']+)/g)]
          .map((m) => { try { return decodeURIComponent(m[1]); } catch (e) { return ''; } })
          .filter((u) => u.startsWith('http'));
        if (links.length === 0) return resolve({ host: '', blocked: r.statusCode === 202 || r.statusCode === 429 });
        for (const u of links) {
          const host = hostOf(u);
          if (host && !SEARCH_BAD_HOSTS.some((x) => host.includes(x))) return resolve({ host, blocked: false });
        }
        resolve({ host: '', blocked: false });
      });
    });
    req.on('error', () => resolve({ host: '', blocked: false }));
    req.setTimeout(15000, () => { req.destroy(); resolve({ host: '', blocked: false }); });
    req.end();
  });
}

// Circuit breaker: once DDG has blocked us repeatedly it stays blocked for the whole
// run (IP-level), so stop wasting time on it and go straight to domain-guessing.
let ddgConsecutiveBlocks = 0;
let ddgDisabled = false;

// Free fallback search via DuckDuckGo HTML (no API key, no quota). Retries once with
// a short backoff; trips a circuit breaker after sustained anti-bot challenges.
async function discoverWebsiteDDG(name) {
  if (ddgDisabled) return '';
  for (let attempt = 0; attempt < 2; attempt++) {
    const { host, blocked } = await ddgOnce(name);
    if (host) { ddgConsecutiveBlocks = 0; return 'https://' + host; }
    if (!blocked) return ''; // genuine "no result"
    if (attempt === 0) await sleep(2000);
  }
  // still blocked after retry
  if (++ddgConsecutiveBlocks >= 3) {
    ddgDisabled = true;
    console.log('[OpenClaw Discover] DuckDuckGo appears IP-blocked; disabling it for this run, using domain-guessing only.');
  }
  return '';
}

// Free fallbacks chained: DuckDuckGo search, then deterministic domain-guessing.
async function freeWebsiteFallback(name) {
  const ddg = await discoverWebsiteDDG(name);
  if (ddg) return ddg;
  return discoverWebsiteGuess(name);
}

// Find an official website, cheapest-reliable first:
//   1. Serper (if key present & has credit)
//   2. DuckDuckGo HTML search (free; may be rate-limited)
//   3. Domain-guessing with content verification (free, no search engine)
async function discoverWebsite(name) {
  const key = process.env.SERPER_API_KEY || '';
  if (!key) return freeWebsiteFallback(name);
  const res = await new Promise((resolve) => {
    const payload = JSON.stringify({ q: `${name} situs resmi`, gl: 'id', hl: 'id', num: 5 });
    const req = https.request({
      hostname: 'google.serper.dev', path: '/search', method: 'POST',
      headers: { 'X-API-KEY': key, 'Content-Type': 'application/json', 'Content-Length': Buffer.byteLength(payload) },
    }, (r) => { let b = ''; r.on('data', (c) => (b += c)); r.on('end', () => { try { resolve(JSON.parse(b)); } catch (e) { resolve(null); } }); });
    req.on('error', () => resolve(null));
    req.setTimeout(15000, () => { req.destroy(); resolve(null); });
    req.write(payload); req.end();
  });
  if (!res || !Array.isArray(res.organic) || res.organic.length === 0) {
    return freeWebsiteFallback(name);
  }
  for (const o of res.organic) {
    const host = hostOf(o.link || '');
    if (!host) continue;
    if (SEARCH_BAD_HOSTS.some((b) => host.includes(b))) continue;
    return 'https://' + host;
  }
  return freeWebsiteFallback(name);
}

// Noise strings seen in logo alt / headings that are not organization names.
const NAME_NOISE = [
  'logo filantropi', 'filantropi indonesia', 'id_id', 'en_us', 'daftar anggota',
  'anggota filantropi', 'anggota kami', 'keanggotaan', 'berita', 'program',
  'publikasi', 'tentang', 'kontak', 'beranda', 'menu', 'search', 'pencarian',
  'ikuti kami', 'alamat', 'berlangganan', 'didukung oleh', 'perusahaan',
  'klaster filantropi', 'pilar program', 'some rights reserved', 'perhimpunan filantropi',
  'logo ford foundation', 'hubungi kami', 'copyright', '©',
];
function isOrgName(name) {
  const l = name.toLowerCase().trim();
  if (l.length < 4) return false;
  if (NAME_NOISE.some((n) => l === n || l.startsWith(n) || l.includes('rights reserved'))) return false;
  if (!/[a-z]/i.test(l)) return false;
  // strip a leading "logo " prefix some alts carry
  return true;
}
function cleanName(raw) {
  return raw.replace(/\s+/g, ' ').trim().replace(/^logo\s+/i, '').trim();
}

// ---- scrape the member directory with Playwright ----
// Pagination links (a.page-numbers) are intercepted by JS and load the next grid via
// AJAX (the URL does not really change). So we CLICK each page number in order and wait
// for the grid's first logo to change before collecting. Names come from logo alt text.
async function scrapeMembers(domainOrUrl, sendProgress) {
  const { chromium } = require('playwright');
  const baseList = domainOrUrl.includes('/')
    ? (domainOrUrl.startsWith('http') ? domainOrUrl : 'https://' + domainOrUrl)
    : `https://${domainOrUrl}/keanggotaan/anggota-kami/`;
  const base = baseList.replace(/\/+$/, '') + '/';

  const browser = await chromium.launch({ args: ['--no-sandbox', '--disable-setuid-sandbox'] });
  const names = [];
  const seen = new Set();
  try {
    const page = await browser.newPage({ userAgent: 'Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/120 Safari/537.36' });
    await page.goto(base, { waitUntil: 'networkidle', timeout: 45000 });
    await page.waitForTimeout(1200);

    let lastPage = 1;
    try {
      const nums = await page.$$eval('a.page-numbers, .elementor-pagination a', (els) =>
        els.map((a) => parseInt((a.textContent || '').trim(), 10)).filter((n) => !isNaN(n)));
      if (nums.length) lastPage = Math.max(...nums);
    } catch (e) { /* keep 1 */ }
    if (lastPage < 1 || lastPage > 100) lastPage = 1;

    // first logo alt on the current grid — used as a change sentinel
    const firstLogo = () => page.evaluate(() => {
      const i = [...document.querySelectorAll('img[alt]')].find(
        (x) => x.alt && x.alt.length > 4 && !/logo filantropi|id_ID|en_US/i.test(x.alt));
      return i ? i.alt : null;
    }).catch(() => null);

    const collect = async () => {
      const found = await page.evaluate(() =>
        [...document.querySelectorAll('img[alt]')].map((i) => i.alt).filter(Boolean)
      ).catch(() => []);
      for (const raw of found) {
        const name = cleanName(raw);
        if (!isOrgName(name)) continue;
        const key = name.toLowerCase();
        if (!seen.has(key)) { seen.add(key); names.push(name); }
      }
    };

    await collect(); // page 1
    for (let n = 2; n <= lastPage; n++) {
      const before = await firstLogo();
      const clicked = await page.evaluate((num) => {
        const link = [...document.querySelectorAll('a.page-numbers')].find((a) => a.textContent.trim() === String(num));
        if (link) { link.scrollIntoView(); link.click(); return true; }
        return false;
      }, n).catch(() => false);
      if (!clicked) break;
      // wait until the grid's first logo changes (AJAX swapped content), max ~8s
      for (let w = 0; w < 16; w++) {
        await page.waitForTimeout(500);
        const now = await firstLogo();
        if (now && now !== before) break;
      }
      await collect();
      if (sendProgress && n % 5 === 0) await sendProgress(`📄 Memindai halaman ${n}/${lastPage}… total ditemukan: ${names.length}`);
    }
  } finally {
    await browser.close();
  }
  return names;
}

// ---- main entry ----
async function executeDiscoverTask(domainArg, chatId, sendTelegramMessage) {
  const send = (msg) => (sendTelegramMessage ? sendTelegramMessage(msg, chatId) : Promise.resolve());

  await send(`🔎 <b>OpenClaw Discovery</b> dimulai untuk <code>${domainArg}</code>.\nMemindai direktori lembaga (JS-rendered)…`);

  let names = [];
  try {
    names = await scrapeMembers(domainArg, send);
  } catch (err) {
    await send(`❌ <i>Gagal memindai direktori: ${err.message}</i>`);
    return { discovered: 0, inserted: 0, filled: 0, failed: 0 };
  }

  if (names.length === 0) {
    await send(`⚠️ <i>Tidak ada lembaga yang terdeteksi di halaman tersebut.</i>`);
    return { discovered: 0, inserted: 0, filled: 0, failed: 0 };
  }

  await send(`✅ Ditemukan <b>${names.length}</b> lembaga. Memulai enrichment (website, email, no HP/WA) & menyimpan ke tabel organizations…`);

  let inserted = 0, filled = 0, failed = 0, withEmail = 0, withPhone = 0;
  for (let i = 0; i < names.length; i++) {
    const name = names[i];
    let website = '', email = '', phone = '';
    try {
      website = await discoverWebsite(name);
      if (website) {
        email = await discoverEmail(website);
        phone = await discoverPhone(website);
      }
    } catch (e) { /* enrichment best-effort */ }

    // Space out requests: the free DuckDuckGo fallback blocks bursts aggressively.
    await sleep(7000);

    const res = await httpRequest(`${CSR_API_BASE_URL}/organizations/ingest`, {
      method: 'POST',
      headers: { 'Authorization': `Bearer ${OPENCLAW_AGENT_TOKEN}`, 'Content-Type': 'application/json' },
    }, { name, org_type: 'HUMANITARIAN_NGO', website, email, phone });

    if (res.statusCode >= 200 && res.statusCode < 300 && res.data && res.data.success) {
      if (res.data.action === 'inserted') inserted++; else filled++;
      if (email) withEmail++;
      if (phone) withPhone++;
    } else {
      failed++;
    }

    if ((i + 1) % 25 === 0) {
      await send(`⏳ Progres: <b>${i + 1}/${names.length}</b> diproses — baru: ${inserted}, sudah ada: ${filled}, gagal: ${failed}.`);
    }
  }

  await send(
    `🏁 <b>Discovery Selesai untuk ${domainArg}</b>\n\n` +
    `• Lembaga ditemukan : <b>${names.length}</b>\n` +
    `• Baru disimpan     : <b>${inserted}</b>\n` +
    `• Sudah ada (diisi)  : <b>${filled}</b>\n` +
    `• Gagal simpan      : <b>${failed}</b>\n` +
    `• Dengan email valid : <b>${withEmail}</b>\n` +
    `• Dengan no HP/WA    : <b>${withPhone}</b>\n\n` +
    `<i>Hanya data terverifikasi yang disimpan (email domain cocok, no HP ber-+62 & lolos verifier).</i>`
  );

  return { discovered: names.length, inserted, filled, failed };
}

// Re-enrich organizations already in the DB that still have empty contact fields.
// Does NOT scrape any directory — it pulls the missing list from the API and tries to
// fill website/email/phone via the same verified-only pipeline (Serper -> DDG -> guess).
async function executeEnrichOrganizations(chatId, sendTelegramMessage) {
  const send = (msg) => (sendTelegramMessage ? sendTelegramMessage(msg, chatId) : Promise.resolve());

  const listRes = await httpRequest(`${CSR_API_BASE_URL}/organizations/missing-contacts?limit=500`, {
    headers: { 'Authorization': `Bearer ${OPENCLAW_AGENT_TOKEN}` },
  });
  if (listRes.statusCode < 200 || listRes.statusCode >= 300 || !listRes.data || !Array.isArray(listRes.data.data)) {
    await send(`❌ <i>Gagal mengambil daftar organisasi yang perlu enrichment (HTTP ${listRes.statusCode}).</i>`);
    return { total: 0, enriched: 0 };
  }
  const targets = listRes.data.data;
  if (targets.length === 0) {
    await send(`✅ <i>Semua organisasi sudah memiliki email & no HP. Tidak ada yang perlu di-enrich.</i>`);
    return { total: 0, enriched: 0 };
  }

  await send(`🔧 <b>OpenClaw Enrichment Organisasi</b>\nMelengkapi kontak untuk <b>${targets.length}</b> lembaga yang masih kosong (via tebak-domain, tanpa Serper). Ini perlu waktu…`);

  let gotEmail = 0, gotPhone = 0, touched = 0, failed = 0;
  for (let i = 0; i < targets.length; i++) {
    const name = targets[i].name;
    let website = '', email = '', phone = '';
    try {
      website = await discoverWebsite(name);
      if (website) {
        if (!targets[i].has_email) email = await discoverEmail(website);
        if (!targets[i].has_phone) phone = await discoverPhone(website);
      }
    } catch (e) { /* best-effort */ }

    if (email || phone) {
      const res = await httpRequest(`${CSR_API_BASE_URL}/organizations/ingest`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${OPENCLAW_AGENT_TOKEN}`, 'Content-Type': 'application/json' },
      }, { name, website, email, phone });
      if (res.statusCode >= 200 && res.statusCode < 300 && res.data && res.data.success) {
        touched++;
        if (email) gotEmail++;
        if (phone) gotPhone++;
      } else {
        failed++;
      }
    }

    if ((i + 1) % 25 === 0) {
      await send(`⏳ Progres: <b>${i + 1}/${targets.length}</b> — terisi email: ${gotEmail}, HP: ${gotPhone}.`);
    }
    await sleep(7000); // space out free-fallback requests
  }

  await send(
    `🏁 <b>Enrichment Organisasi Selesai</b>\n\n` +
    `• Diproses          : <b>${targets.length}</b>\n` +
    `• Terisi email baru  : <b>${gotEmail}</b>\n` +
    `• Terisi no HP baru  : <b>${gotPhone}</b>\n` +
    `• Gagal simpan      : <b>${failed}</b>\n\n` +
    `<i>Hanya data terverifikasi yang disimpan; field yang sudah terisi tidak ditimpa.</i>`
  );
  return { total: targets.length, enriched: touched };
}

module.exports = { executeDiscoverTask, executeEnrichOrganizations, scrapeMembers, discoverWebsite, discoverWebsiteGuess, discoverEmail, discoverPhone };
