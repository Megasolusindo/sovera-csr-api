/**
 * OpenClaw AI Agent — Company Monitoring Skill Module
 * Continuously monitors active watchlist companies for new CSR disclosures and policy shifts
 */

const http = require('http');
const https = require('https');
const { URL } = require('url');

const CSR_API_BASE_URL = process.env.CSR_API_BASE_URL || 'http://api:4000/api/v1/ai';
const OPENCLAW_AGENT_TOKEN = process.env.OPENCLAW_AGENT_TOKEN || 'openclaw_agent_live_key_998877665544';
const SERPER_API_KEY = process.env.SERPER_API_KEY || '';

function httpRequest(urlStr, options = {}, postData = null) {
  return new Promise((resolve, reject) => {
    const parsedUrl = new URL(urlStr);
    const lib = parsedUrl.protocol === 'https:' ? https : http;

    const reqOptions = {
      hostname: parsedUrl.hostname,
      port: parsedUrl.port || (parsedUrl.protocol === 'https:' ? 443 : 80),
      path: parsedUrl.pathname + parsedUrl.search,
      method: options.method || 'GET',
      headers: {
        'User-Agent': 'OpenClaw-AI-Agent/1.0',
        ...options.headers,
      },
    };

    const req = lib.request(reqOptions, (res) => {
      let body = '';
      res.on('data', (chunk) => (body += chunk));
      res.on('end', () => {
        try {
          const json = JSON.parse(body);
          resolve({ statusCode: res.statusCode, data: json, raw: body });
        } catch (e) {
          resolve({ statusCode: res.statusCode, raw: body });
        }
      });
    });

    req.on('error', (err) => reject(err));

    if (postData) {
      req.write(typeof postData === 'string' ? postData : JSON.stringify(postData));
    }
    req.end();
  });
}

async function fetchActiveWatchlist() {
  try {
    const url = `${CSR_API_BASE_URL}/watchlist`;
    const res = await httpRequest(url, {
      headers: {
        'Authorization': `Bearer ${OPENCLAW_AGENT_TOKEN}`
      }
    });

    if (res.statusCode === 200 && res.data && res.data.data) {
      return res.data.data;
    }
  } catch (err) {
    console.error('[OpenClaw Monitoring Skill] Error fetching watchlist:', err.message);
  }
  return [];
}

async function searchWebViaSerper(query) {
  if (!SERPER_API_KEY) {
    const encoded = encodeURIComponent(query);
    const rssUrl = `https://news.google.com/rss/search?q=${encoded}&hl=id&gl=ID&ceid=ID:id`;
    return [{
      title: `Monitoring Alert RSS for ${query}`,
      snippet: `Deteksi berkala OpenClaw Agent untuk kata kunci ${query}`,
      link: rssUrl,
      source: 'Google News RSS'
    }];
  }

  try {
    const res = await httpRequest('https://google.serper.dev/search', {
      method: 'POST',
      headers: {
        'X-API-KEY': SERPER_API_KEY,
        'Content-Type': 'application/json'
      }
    }, { q: query, gl: 'id', hl: 'id', num: 5 });

    if (res.data && res.data.organic) {
      return res.data.organic.map(item => ({
        title: item.title,
        snippet: item.snippet,
        link: item.link,
        source: item.domain || 'Google Search'
      }));
    }
  } catch (err) {
    console.error('[OpenClaw Monitoring Skill] Serper API error:', err.message);
  }
  return [];
}

const { notifyNewFinding } = require('../telegram');

async function submitMonitoringFinding(findingPayload) {
  try {
    const submitUrl = `${CSR_API_BASE_URL}/research/findings`;
    const res = await httpRequest(submitUrl, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${OPENCLAW_AGENT_TOKEN}`,
        'Content-Type': 'application/json'
      }
    }, findingPayload);

    if (res.statusCode === 201 || res.statusCode === 200) {
      console.log(`[OpenClaw Monitoring Skill] ✅ Submitted Watchlist Finding: "${findingPayload.title}"`);
      // Trigger Telegram Alert
      await notifyNewFinding(findingPayload);
      return res.data;
    }
  } catch (err) {
    console.error('[OpenClaw Monitoring Skill] Submit finding error:', err.message);
  }
  return null;
}

async function executeMonitoringTask() {
  console.log(`\n===============================================================`);
  console.log(`[OpenClaw Company Monitoring Agent] Initiating Monitoring Run`);
  console.log(`Timestamp: ${new Date().toISOString()}`);
  console.log(`===============================================================\n`);

  const watchlist = await fetchActiveWatchlist();
  console.log(`[OpenClaw Monitoring Skill] Retrieved ${watchlist.length} active companies from Watchlist.`);

  if (watchlist.length === 0) {
    console.log(`[OpenClaw Monitoring Skill] Watchlist is currently empty.`);
    return 0;
  }

  let totalFindingsSubmitted = 0;

  for (const item of watchlist) {
    console.log(`\n🔍 Monitoring company: ${item.company_name} (ID: ${item.company_id})`);
    const keywords = (item.monitoring_keywords || ['CSR', 'TJSL', 'Keberlanjutan']).join(' OR ');
    const query = `"${item.company_name}" ${keywords}`;

    const searchResults = await searchWebViaSerper(query);
    console.log(`  Found ${searchResults.length} items for query: ${query}`);

    for (const searchItem of searchResults) {
      if (!searchItem.link || !searchItem.title) continue;

      const idempotencyKey = `openclaw_mon_${Buffer.from(item.company_id + searchItem.title + searchItem.link).toString('hex').substring(0, 24)}`;

      const payload = {
        company_id: item.company_id,
        company_name: item.company_name,
        finding_type: 'company_monitoring',
        title: searchItem.title,
        summary: searchItem.snippet || `Laporan monitoring berkala OpenClaw Agent untuk ${item.company_name}`,
        source_url: searchItem.link,
        source_name: searchItem.source || 'Web Search',
        source_type: 'news',
        confidence_score: 0.95,
        evidence_data: {
          agent_name: 'openclaw-monitoring-agent',
          query_used: query,
          watchlist_id: item.id,
          execution_time: new Date().toISOString()
        },
        idempotency_key: idempotencyKey
      };

      const res = await submitMonitoringFinding(payload);
      if (res) totalFindingsSubmitted++;
    }
  }

  console.log(`\n[OpenClaw Monitoring Agent] Monitoring Run Completed. Total findings queued: ${totalFindingsSubmitted}`);
  return totalFindingsSubmitted;
}

module.exports = {
  executeMonitoringTask
};
