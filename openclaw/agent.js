/**
 * OpenClaw AI Agent Runner & Orchestrator
 */

const { executeResearchTask } = require('./skills/research_skill');
const { executeEnrichmentTask } = require('./skills/enrichment_skill');
const { executeMatchingTask } = require('./skills/matching_skill');

const args = process.argv.slice(2);
const targetCompanyArg = args.find(a => a.startsWith('--company='))?.split('=')[1] || 'PT Pertamina Patra Niaga';

async function main() {
  console.log(`🤖 [OpenClaw AI Agent Service v1.0] Starting...`);
  console.log(`Timestamp: ${new Date().toISOString()}`);

  try {
    if (targetCompanyArg.startsWith('ENRICH:')) {
      const realCompany = targetCompanyArg.replace('ENRICH:', '').trim();
      await executeEnrichmentTask(realCompany);
    } else if (targetCompanyArg.startsWith('MATCH:')) {
      const category = targetCompanyArg.replace('MATCH:', '').trim() || 'Pendidikan';
      await executeMatchingTask(category);
    } else {
      const count = await executeResearchTask(targetCompanyArg);
      console.log(`✅ [OpenClaw AI Agent Service] Run completed successfully. Submitted ${count} findings.`);
    }
    process.exit(0);
  } catch (err) {
    console.error(`❌ [OpenClaw AI Agent Service] Fatal Error:`, err);
    process.exit(1);
  }
}

main();
