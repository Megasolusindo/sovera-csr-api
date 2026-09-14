/**
 * OpenClaw AI Agent — Real Live IDX Emiten Sync Skill
 * Fetches 100% real live listed company data directly from official Bursa Efek Indonesia (IDX) API
 * and updates the master companies database in sovera.
 */

const { exec } = require('child_process');
const path = require('path');

function triggerLiveIDXSync() {
  return new Promise((resolve, reject) => {
    const scriptPath = path.resolve(__dirname, '../../scripts/sync_idx_live.py');
    console.log(`[OpenClaw IDX Skill] Executing real live IDX ingestion script: ${scriptPath}`);

    exec(`python3 ${scriptPath}`, { timeout: 120000 }, (error, stdout, stderr) => {
      if (error) {
        console.error(`[OpenClaw IDX Skill] Sync execution error: ${error.message}`);
        return resolve({
          success: false,
          error: error.message,
          output: stdout || stderr,
        });
      }

      console.log(`[OpenClaw IDX Skill] Live IDX Sync Completed successfully: ${stdout}`);
      return resolve({
        success: true,
        output: stdout.trim(),
      });
    });
  });
}

module.exports = {
  triggerLiveIDXSync,
};
