/**
 * OpenClaw AI Agent — Instagram URL Verification Skill
 * Verifies Instagram URL validity, entity types, canonical URLs, and health status via Core API
 */

const http = require('http');
const https = require('https');
const { URL } = require('url');

const rawBaseURL = process.env.CSR_API_BASE_URL || 'http://api:4000/api/v1';
const CSR_API_BASE_URL = rawBaseURL.replace(/\/ai\/?$/, '');

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

async function verifyInstagramURL(rawURL) {
  const endpointsToTry = [
    `${CSR_API_BASE_URL}/url/verify-instagram`,
    'http://api:4000/api/v1/url/verify-instagram',
    'http://localhost:4000/api/v1/url/verify-instagram',
  ];

  for (const verifyEndpoint of endpointsToTry) {
    try {
      const res = await httpRequest(
        verifyEndpoint,
        {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
        },
        { url: rawURL }
      );

      if (res.statusCode >= 200 && res.statusCode < 300 && res.data && res.data.data) {
        return res.data.data;
      }
    } catch (err) {
      // Try next endpoint fallback
    }
  }
  return null;
}

async function triggerBatchCompanyInstagramVerification() {
  const endpointsToTry = [
    `${CSR_API_BASE_URL}/url/verify-instagram-batch`,
    'http://api:4000/api/v1/url/verify-instagram-batch',
    'http://localhost:4000/api/v1/url/verify-instagram-batch',
  ];

  for (const endpoint of endpointsToTry) {
    try {
      const res = await httpRequest(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      if (res.statusCode >= 200 && res.statusCode < 300 && res.data && res.data.success) {
        return res.data;
      }
    } catch (err) {
      // Try next endpoint fallback
    }
  }
  return null;
}

async function getCompanyInstagramStats() {
  const endpointsToTry = [
    `${CSR_API_BASE_URL}/url/instagram-stats`,
    'http://api:4000/api/v1/url/instagram-stats',
    'http://localhost:4000/api/v1/url/instagram-stats',
  ];

  for (const endpoint of endpointsToTry) {
    try {
      const res = await httpRequest(endpoint, { method: 'GET' });
      if (res.statusCode >= 200 && res.statusCode < 300 && res.data && res.data.data) {
        return res.data.data;
      }
    } catch (err) {
      // Try next endpoint fallback
    }
  }
  return null;
}

async function triggerBatchCompanyInstagramDiscovery() {
  const endpointsToTry = [
    `${CSR_API_BASE_URL}/url/discover-instagram-batch`,
    'http://api:4000/api/v1/url/discover-instagram-batch',
    'http://localhost:4000/api/v1/url/discover-instagram-batch',
  ];

  for (const endpoint of endpointsToTry) {
    try {
      const res = await httpRequest(endpoint, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
      });
      if (res.statusCode >= 200 && res.statusCode < 300 && res.data && res.data.success) {
        return res.data;
      }
    } catch (err) {
      // Try next endpoint fallback
    }
  }
  return null;
}

module.exports = {
  verifyInstagramURL,
  triggerBatchCompanyInstagramVerification,
  triggerBatchCompanyInstagramDiscovery,
  getCompanyInstagramStats,
};
