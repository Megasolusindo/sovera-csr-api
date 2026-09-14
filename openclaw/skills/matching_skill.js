/**
 * OpenClaw AI Agent — Program & Institutional Matching Skill Module
 * Evaluates institution CSR proposals and computes ranked corporate partner recommendations
 */

const http = require('http');
const https = require('https');
const { URL } = require('url');
const { notifyMatchingResult } = require('../telegram');

const CSR_API_BASE_URL = process.env.CSR_API_BASE_URL || 'http://api:4000/api/v1/ai';
const OPENCLAW_AGENT_TOKEN = process.env.OPENCLAW_AGENT_TOKEN || 'openclaw_agent_live_key_998877665544';

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

async function executeMatchingTask(category = 'Pendidikan', location = 'Jawa Barat') {
  console.log(`\n===============================================================`);
  console.log(`[OpenClaw Program Matching Agent] Initiating Proposal Matching`);
  console.log(`Target Focus Category: ${category}`);
  console.log(`Target Geographic Footprint: ${location}`);
  console.log(`===============================================================\n`);

  const payload = {
    program_title: `Program CSR Pemberdayaan ${category} Terpadu`,
    csr_category: category,
    target_location: location,
    target_beneficiaries: 'Komunitas & Siswa Berprestasi',
    estimated_budget: 500000000,
    keywords: [category, location, 'CSR', 'TJSL', 'Beasiswa'],
    limit: 5
  };

  try {
    const url = `${CSR_API_BASE_URL}/matching`;
    const res = await httpRequest(url, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${OPENCLAW_AGENT_TOKEN}`,
        'Content-Type': 'application/json'
      }
    }, payload);

    if (res.statusCode === 200 && res.data && res.data.data) {
      const matches = res.data.data.corporate_matches || [];
      console.log(`[OpenClaw Matching Skill] ✅ Evaluated proposal. Found ${matches.length} matching corporate prospects:\n`);

      matches.forEach((m, idx) => {
        console.log(`  ${idx + 1}. ${m.company_name} — Score: ${m.match_score}% (${m.match_grade})`);
        console.log(`     Sektor: ${m.industry_sector} | Tier: ${m.priority_tier}`);
        console.log(`     Rationale: ${m.match_rationale}`);
        console.log(`     Website: ${m.website || 'N/A'}\n`);
      });

      // Trigger Telegram Alert
      await notifyMatchingResult(payload.program_title, matches);

      return res.data.data;
    } else {
      console.error(`[OpenClaw Matching Skill] ❌ Failed to run matching task. Status: ${res.statusCode}`, res.data);
    }
  } catch (err) {
    console.error('[OpenClaw Matching Skill] API error:', err.message);
  }

  return null;
}

module.exports = {
  executeMatchingTask
};
