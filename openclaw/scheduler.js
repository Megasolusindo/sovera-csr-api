/**
 * OpenClaw AI Agent Scheduler & Telegram Command Listener
 * Executes periodic automated CSR Research, Watchlist Monitoring, and listens for Inbound Telegram commands
 */

const cron = require('node-cron');
const { executeResearchTask } = require('./skills/research_skill');
const { executeMonitoringTask } = require('./skills/monitoring_skill');
const { triggerLiveIDXSync } = require('./skills/idx_sync_skill');
const { startTelegramReceiver } = require('./telegram_receiver');

const targetCompanies = [
  'PT Pertamina Patra Niaga',
  'PT Bank Mandiri Tbk',
  'PT Astra International Tbk',
  'PT Telkom Indonesia Tbk'
];

console.log(`⏱️ [OpenClaw Cron Scheduler] Starting periodic research & monitoring scheduler...`);

// Run research task every 6 hours
cron.schedule('0 */6 * * *', async () => {
  console.log(`\n⏰ [Cron Trigger - Research] Running scheduled research cycle at ${new Date().toISOString()}`);
  for (const company of targetCompanies) {
    try {
      await executeResearchTask(company);
    } catch (err) {
      console.error(`Error executing scheduled research for ${company}:`, err.message);
    }
  }
});

// Run watchlist monitoring task every 12 hours
cron.schedule('0 */12 * * *', async () => {
  console.log(`\n⏰ [Cron Trigger - Monitoring] Running scheduled watchlist monitoring cycle at ${new Date().toISOString()}`);
  try {
    await executeMonitoringTask();
  } catch (err) {
    console.error(`Error executing scheduled monitoring run:`, err.message);
  }
});

// Run live IDX emiten sync every Monday at 02:00 AM ('0 2 * * 1')
cron.schedule('0 2 * * 1', async () => {
  console.log(`\n⏰ [Cron Trigger - IDX Sync] Running scheduled live IDX emiten sync at ${new Date().toISOString()}`);
  try {
    await triggerLiveIDXSync();
  } catch (err) {
    console.error(`Error executing scheduled IDX sync:`, err.message);
  }
});

// Start Inbound Telegram Command Listener & initial startup checks
(async () => {
  console.log(`🚀 [Startup Trigger] Starting OpenClaw Inbound Telegram Receiver & Initial Cycles...`);
  // Start Telegram bot receiver in non-blocking background loop
  startTelegramReceiver().catch((err) => {
    console.error(`Telegram Receiver Error:`, err.message);
  });

  try {
    await executeResearchTask(targetCompanies[0]);
    await executeMonitoringTask();
  } catch (err) {
    console.error(`Initial startup run error:`, err.message);
  }
})();
