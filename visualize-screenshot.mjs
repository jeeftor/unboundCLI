import { chromium } from 'playwright';

const BASE = 'https://caddy-sync.vookie.net';

async function main() {
  const browser = await chromium.launch({ headless: true });
  const page = await browser.newPage({ viewport: { width: 1400, height: 900 } });

  await page.goto(BASE, { waitUntil: 'networkidle' });
  await page.waitForTimeout(3000);

  // Get all hostnames from the entries table
  const hostnames = await page.$$eval('#entries tr[data-hostname]', rows =>
    rows.map(r => r.getAttribute('data-hostname')).filter(Boolean)
  );
  console.log(`Found ${hostnames.length} entries`);

  // Pick interesting hosts — look for ones with different auth setups
  // Let's try: jellyfin, searxng, and a few others
  const preferred = ['jellyfin', 'authentik', 'searxng', 'grafana', 'homeassistant', 'proxmox'];
  const targets = [];
  for (const p of preferred) {
    const match = hostnames.find(h => h.includes(p));
    if (match) targets.push(match);
  }
  // Fill remaining slots
  for (const h of hostnames) {
    if (targets.length >= 6) break;
    if (!targets.includes(h)) targets.push(h);
  }

  console.log('Targets:', targets);

  for (const hostname of targets) {
    console.log(`\n=== Visualizing: ${hostname} ===`);

    // Find the row and click the Flow button
    const row = page.locator(`#entries tr[data-hostname="${hostname}"]`);
    const flowBtn = row.locator('button.row-visualize-btn');
    if (await flowBtn.count() === 0) {
      console.log(`  No Flow button, skipping`);
      continue;
    }

    await flowBtn.click();
    await page.waitForTimeout(3000); // wait for auth inventory fetch

    const safeName = hostname.replace(/[^a-z0-9]/gi, '_');
    const screenshotPath = `/tmp/visualize-${safeName}.png`;
    await page.screenshot({ path: screenshotPath });

    // Extract modal info
    const modal = page.locator('.visualize-modal');
    if (await modal.count() > 0) {
      const verdict = await modal.locator('.visualize-auth-verdict-name').textContent().catch(() => 'N/A');
      const verdictClass = await modal.locator('.visualize-auth-verdict').first().evaluate(el => el.className).catch(() => '');
      const summary = await modal.locator('.visualize-auth-verdict-summary').textContent().catch(() => 'N/A');
      const flowTables = await modal.locator('.flow-table').count();
      const flowNodes = await modal.locator('.auth-flow-node').count();
      const wanRows = await modal.locator('.flow-table').first().locator('tbody tr').count().catch(() => 0);
      const lanRows = await modal.locator('.flow-table').last().locator('tbody tr').count().catch(() => 0);

      console.log(`  Verdict: ${verdict} (${verdictClass})`);
      console.log(`  Summary: ${summary}`);
      console.log(`  Flow nodes: ${flowNodes}, Flow tables: ${flowTables}`);
      console.log(`  WAN rows: ${wanRows}, LAN rows: ${lanRows}`);
      console.log(`  Screenshot: ${screenshotPath}`);
    } else {
      console.log(`  Modal did not open!`);
    }

    // Close modal
    await page.locator('.visualize-modal .modal-close').click().catch(() => {});
    await page.waitForTimeout(500);
  }

  await browser.close();
  console.log('\nDone!');
}

main().catch(e => { console.error(e); process.exit(1); });
