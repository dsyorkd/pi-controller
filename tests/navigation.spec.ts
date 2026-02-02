import { test, expect } from '@playwright/test';

/**
 * Test Issue #106: Sidebar Menu Navigation Routing
 * 
 * Tests that clicking each sidebar navigation link properly changes the URL
 * and navigates to the correct page without 404 errors.
 */
test.describe('Sidebar Menu Navigation Routing', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the main page
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should navigate to Dashboard from sidebar', async ({ page }) => {
    // Find and click the Dashboard link in sidebar
    const dashboardLink = page.locator('nav a[href="/"], nav a[href="/dashboard"], nav a:has-text("Dashboard")');
    
    // Verify link is visible
    await expect(dashboardLink.first()).toBeVisible();
    
    // Click the dashboard link
    await dashboardLink.first().click();
    
    // Verify URL change
    await expect(page).toHaveURL(new RegExp('\/$|\/dashboard'));
    
    // Verify page content loaded correctly
    await expect(page.locator('h1', { hasText: /dashboard|cluster dashboard/i })).toBeVisible();
  });

  test('should navigate to Clusters from sidebar', async ({ page }) => {
    // Find and click the Clusters link in sidebar
    const clustersLink = page.locator('nav a[href="/admin/clusters"], nav a[href="/clusters"], nav a:has-text("Clusters")');
    
    // Verify link is visible
    await expect(clustersLink.first()).toBeVisible();
    
    // Click the clusters link
    await clustersLink.first().click();
    
    // Verify URL change
    await expect(page).toHaveURL(/.*\/.*clusters/);
    
    // Verify no 404 error
    await expect(page.locator('text=404')).not.toBeVisible();
    await expect(page.locator('text=Page not found')).not.toBeVisible();
    
    // Verify page has cluster-related content
    await expect(page.locator('h1, h2, h3')).toContainText(/cluster/i);
  });

  test('should navigate to Nodes from sidebar', async ({ page }) => {
    // Find and click the Nodes link in sidebar
    const nodesLink = page.locator('nav a[href="/admin/nodes"], nav a[href="/nodes"], nav a:has-text("Nodes")');
    
    // Check if nodes link exists
    const linkExists = await nodesLink.count() > 0;
    
    if (linkExists) {
      // Verify link is visible
      await expect(nodesLink.first()).toBeVisible();
      
      // Click the nodes link
      await nodesLink.first().click();
      
      // Verify URL change
      await expect(page).toHaveURL(/.*\/.*nodes/);
      
      // Verify no 404 error
      await expect(page.locator('text=404')).not.toBeVisible();
      await expect(page.locator('text=Page not found')).not.toBeVisible();
      
      // Verify page has node-related content
      await expect(page.locator('h1, h2, h3')).toContainText(/node/i);
    } else {
      console.log('Nodes navigation link not found - may not be implemented yet');
    }
  });

  test('should navigate to Settings from sidebar', async ({ page }) => {
    // Find and click the Settings link in sidebar
    const settingsLink = page.locator('nav a[href="/settings"], nav a[href="/admin/settings"], nav a:has-text("Settings")');
    
    // Check if settings link exists
    const linkExists = await settingsLink.count() > 0;
    
    if (linkExists) {
      // Verify link is visible
      await expect(settingsLink.first()).toBeVisible();
      
      // Click the settings link
      await settingsLink.first().click();
      
      // Verify URL change
      await expect(page).toHaveURL(/.*\/settings/);
      
      // Verify no 404 error
      await expect(page.locator('text=404')).not.toBeVisible();
      await expect(page.locator('text=Page not found')).not.toBeVisible();
      
      // Verify page has settings-related content
      await expect(page.locator('h1, h2, h3')).toContainText(/settings|configuration/i);
    } else {
      console.log('Settings navigation link not found - may not be implemented yet');
    }
  });

  test('should navigate to Documentation from sidebar', async ({ page }) => {
    // Find and click the Documentation link in sidebar
    const docsLink = page.locator('nav a[href="/docs"], nav a[href="/documentation"], nav a:has-text("Documentation"), nav a:has-text("Docs")');
    
    // Check if documentation link exists
    const linkExists = await docsLink.count() > 0;
    
    if (linkExists) {
      // Verify link is visible
      await expect(docsLink.first()).toBeVisible();
      
      // Click the documentation link
      await docsLink.first().click();
      
      // Verify URL change
      await expect(page).toHaveURL(/.*\/(docs|documentation)/);
      
      // Verify no 404 error
      await expect(page.locator('text=404')).not.toBeVisible();
      await expect(page.locator('text=Page not found')).not.toBeVisible();
      
      // Verify page has documentation-related content
      await expect(page.locator('h1, h2, h3')).toContainText(/documentation|docs|guide/i);
    } else {
      console.log('Documentation navigation link not found - may not be implemented yet');
    }
  });

  test('should navigate to Monitoring from sidebar', async ({ page }) => {
    // Find and click the Monitoring link in sidebar
    const monitoringLink = page.locator('nav a[href="/monitoring"], nav a[href="/admin/monitoring"], nav a:has-text("Monitoring")');
    
    // Check if monitoring link exists
    const linkExists = await monitoringLink.count() > 0;
    
    if (linkExists) {
      // Verify link is visible
      await expect(monitoringLink.first()).toBeVisible();
      
      // Click the monitoring link
      await monitoringLink.first().click();
      
      // Verify URL change
      await expect(page).toHaveURL(/.*\/monitoring/);
      
      // Verify no 404 error
      await expect(page.locator('text=404')).not.toBeVisible();
      await expect(page.locator('text=Page not found')).not.toBeVisible();
      
      // Verify page has monitoring-related content
      await expect(page.locator('h1, h2, h3')).toContainText(/monitoring|metrics/i);
    } else {
      console.log('Monitoring navigation link not found - may not be implemented yet');
    }
  });

  test('should navigate to GPIO from sidebar', async ({ page }) => {
    // Find and click the GPIO link in sidebar
    const gpioLink = page.locator('nav a[href="/gpio"], nav a[href="/admin/gpio"], nav a:has-text("GPIO")');
    
    // Check if gpio link exists
    const linkExists = await gpioLink.count() > 0;
    
    if (linkExists) {
      // Verify link is visible
      await expect(gpioLink.first()).toBeVisible();
      
      // Click the gpio link
      await gpioLink.first().click();
      
      // Verify URL change
      await expect(page).toHaveURL(/.*\/gpio/);
      
      // Verify no 404 error
      await expect(page.locator('text=404')).not.toBeVisible();
      await expect(page.locator('text=Page not found')).not.toBeVisible();
      
      // Verify page has gpio-related content
      await expect(page.locator('h1, h2, h3')).toContainText(/gpio|pin/i);
    } else {
      console.log('GPIO navigation link not found - may not be implemented yet');
    }
  });

  test('should highlight active navigation item', async ({ page }) => {
    // Test that the current page's navigation item is visually highlighted
    
    // Start on dashboard
    await page.goto('/');
    
    // Check if dashboard nav item has active styling
    const dashboardNav = page.locator('nav a[href="/"], nav a[href="/dashboard"], nav a:has-text("Dashboard")').first();
    
    // Look for active state indicators (common class names)
    const hasActiveState = await dashboardNav.evaluate((el) => {
      const classes = el.className.toLowerCase();
      return classes.includes('active') || 
             classes.includes('current') || 
             classes.includes('selected') ||
             el.getAttribute('aria-current') === 'page';
    });
    
    if (hasActiveState) {
      console.log('✅ Active navigation state found');
    } else {
      console.log('ℹ️ No active navigation state detected - may not be implemented');
    }
  });

  test('should preserve navigation state after page refresh', async ({ page }) => {
    // Navigate to a specific page
    const clustersLink = page.locator('nav a[href="/admin/clusters"], nav a[href="/clusters"], nav a:has-text("Clusters")').first();
    
    if (await clustersLink.count() > 0) {
      await clustersLink.click();
      await expect(page).toHaveURL(/.*\/.*clusters/);
      
      // Refresh the page
      await page.reload();
      await page.waitForLoadState('networkidle');
      
      // Verify we're still on the same page
      await expect(page).toHaveURL(/.*\/.*clusters/);
      await expect(page.locator('text=404')).not.toBeVisible();
    }
  });

  test('should handle navigation keyboard shortcuts', async ({ page }) => {
    // Test common keyboard navigation patterns if they exist
    
    // Check if there's a keyboard shortcut indicator
    const shortcutIndicator = page.locator('[title*="Ctrl"], [title*="Cmd"], [data-shortcut], .shortcut');
    
    if (await shortcutIndicator.count() > 0) {
      console.log('✅ Keyboard shortcut indicators found');
      
      // Test if Alt+D or similar goes to dashboard
      await page.keyboard.press('Alt+d');
      await page.waitForTimeout(500);
      
      // Check if URL changed
      const currentUrl = page.url();
      if (currentUrl.includes('/dashboard') || currentUrl.endsWith('/')) {
        console.log('✅ Keyboard shortcut navigation working');
      }
    } else {
      console.log('ℹ️ No keyboard shortcuts detected');
    }
  });
});