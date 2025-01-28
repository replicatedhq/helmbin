import { test, expect, Page, FrameLocator } from '@playwright/test';
import { login } from '../shared';

test('deploy upgrade', async ({ page }) => {
  test.setTimeout(15 * 60 * 1000); // 15 minutes
  await login(page);
  await initiateUpgrade(page);
  const iframe = page.frameLocator('#upgrade-service-iframe');
  await fillConfigForm(iframe);
  await handlePreflightChecks(iframe);
  await deployUpgrade(iframe);
  await waitForClusterUpdate(page);
  await verifyUpgradeSuccess(page);
});

async function initiateUpgrade(page: Page) {
  await page.getByRole('link', { name: 'Version history', exact: true }).click();
  await page.locator('.available-update-row', { hasText: process.env.APP_UPGRADE_VERSION }).getByRole('button', { name: 'Deploy', exact: true }).click();
}

async function fillConfigForm(iframe: FrameLocator) {
  await expect(iframe.locator('h3')).toContainText('The First Config Group', { timeout: 60 * 1000 }); // can take time to download the kots binary

  const hostnameInput = iframe.locator('#hostname-group').locator('input[type="text"]');
  // the hostname can be either 'initial-hostname.com' or 'updated-hostname.com' if we have run the upgrade multiple times
  await expect(hostnameInput).toHaveValue(process.env.APP_INITIAL_HOSTNAME ? process.env.APP_INITIAL_HOSTNAME : /(initial|updated)-hostname\.com/);
  await hostnameInput.click();
  await hostnameInput.fill('updated-hostname.com');

  await iframe.locator('input[type="password"]').click();
  await iframe.locator('input[type="password"]').fill('updated password');

  await iframe.getByRole('button', { name: 'Next', exact: true }).click();
}

async function handlePreflightChecks(iframe: FrameLocator) {
  await expect(iframe.getByText('Preflight checks', { exact: true })).toBeVisible({ timeout: 10 * 1000 });
  await expect(iframe.getByRole('button', { name: 'Rerun' })).toBeVisible({ timeout: 10 * 1000 });
  await expect(iframe.locator('#app')).toContainText('The Volume Snapshots CRD exists');
  await expect(iframe.getByRole('button', { name: 'Back: Config' })).toBeVisible();
  await iframe.getByRole('button', { name: 'Next: Confirm and deploy' }).click();
}

async function deployUpgrade(iframe: FrameLocator) {
  await expect(iframe.locator('#app')).toContainText('All preflight checks passed');
  await expect(iframe.getByRole('button', { name: 'Back: Preflight checks' })).toBeVisible();
  await iframe.getByRole('button', { name: 'Deploy', exact: true }).click();
}

async function waitForClusterUpdate(page: Page) {
  if (process.env.SKIP_CLUSTER_UPGRADE_CHECK !== 'true') {
    await expect(page.locator('.Modal-body')).toContainText('Cluster update in progress');
    await expect(page.locator('.Modal-body').getByText('Cluster update in progress')).not.toBeVisible({ timeout: 20 * 60 * 1000 });
  }
}

async function verifyUpgradeSuccess(page: Page) {
  await expect(page.locator('.available-update-row', { hasText: process.env.APP_UPGRADE_VERSION })).not.toBeVisible({ timeout: 10 * 1000 });
  await expect(page.locator('.VersionHistoryRow', { hasText: process.env.APP_UPGRADE_VERSION })).toContainText('Currently deployed version', { timeout: 90 * 1000 });
  await page.getByRole('link', { name: 'Dashboard', exact: true }).click();
  await expect(page.locator('.VersionCard-content--wrapper')).toContainText(process.env.APP_UPGRADE_VERSION);
  await expect(page.locator('#app')).toContainText('Currently deployed version');
  await expect(page.locator('#app')).toContainText('Ready', { timeout: 30 * 1000 });
  await expect(page.locator('#app')).toContainText('Up to date');
}
