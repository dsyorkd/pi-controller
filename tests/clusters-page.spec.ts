import { test, expect } from '@playwright/test';

test.describe('Clusters Page Testing', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the main page first
    await page.goto('http://localhost:3003/');
  });

  test('should navigate to clusters page directly without 404', async ({ page }) => {
    // Navigate directly to clusters page
    await page.goto('http://localhost:3003/admin/clusters');

    // Should not show 404 error
    await expect(page.locator('text=404')).not.toBeVisible();
    await expect(page.locator('text=Page not found')).not.toBeVisible();

    // Should show clusters page content
    await expect(page.locator('h1')).toContainText(/cluster/i);
  });

  test('should navigate to clusters page via navigation link', async ({ page }) => {
    // Navigate to main page
    await page.goto('http://localhost:3003/');

    // Find and click the Clusters navigation link
    const clustersLink = page.locator('nav a[href="/admin/clusters"], nav a:has-text("Clusters")');
    await expect(clustersLink).toBeVisible();
    await clustersLink.click();

    // Should navigate to clusters page
    await expect(page).toHaveURL(/.*\/admin\/clusters/);

    // Should not show 404 error
    await expect(page.locator('text=404')).not.toBeVisible();
    await expect(page.locator('text=Page not found')).not.toBeVisible();
  });

  test('should display clusters page content correctly', async ({ page }) => {
    // Navigate to clusters page
    await page.goto('http://localhost:3003/admin/clusters');

    // Check for cluster management interface elements
    // This might be cluster cards, a "New Cluster" button, or empty state
    const hasClusterContent = await page.locator('text=/cluster/i, button:has-text("New Cluster"), [data-testid="cluster-card"], text=/no clusters/i').first().isVisible();
    expect(hasClusterContent).toBeTruthy();
  });

  test('should handle cluster actions if available', async ({ page }) => {
    // Navigate to clusters page
    await page.goto('http://localhost:3003/admin/clusters');

    // Check if "New Cluster" button exists and is clickable
    const newClusterButton = page.locator('button:has-text("New Cluster"), a:has-text("New Cluster")');
    const buttonExists = await newClusterButton.count() > 0;

    if (buttonExists) {
      await expect(newClusterButton.first()).toBeVisible();
      // We won't actually click it to avoid side effects, just verify it's there
    }
  });

  test('should take screenshot of clusters page', async ({ page }) => {
    // Navigate to clusters page
    await page.goto('http://localhost:3003/admin/clusters');

    // Wait for any loading to complete
    await page.waitForLoadState('networkidle');

    // Take screenshot
    await page.screenshot({
      path: 'test-results/clusters-page-screenshot.png',
      fullPage: true
    });
  });
});
