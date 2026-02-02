import { test, expect } from '@playwright/test';

test.describe('Dashboard Visual Testing', () => {
  test('take screenshot of dashboard', async ({ page }) => {
    // Navigate to dashboard
    await page.goto('http://localhost:3000');

    // Wait for content to load
    await page.waitForTimeout(3000);

    // Take full page screenshot
    await page.screenshot({
      path: 'test-results/dashboard-full-page.png',
      fullPage: true
    });

    // Take viewport screenshot
    await page.screenshot({
      path: 'test-results/dashboard-viewport.png'
    });

    // Print page title and main headings for debugging
    const title = await page.title();
    console.log('Page title:', title);

    const headings = await page.locator('h1, h2, h3').allTextContents();
    console.log('Headings found:', headings);

    // Check if page loaded successfully
    const body = await page.locator('body').textContent();
    expect(body).toBeTruthy();
  });

  test('click create cluster button and take modal screenshot', async ({ page }) => {
    await page.goto('http://localhost:3000');
    await page.waitForTimeout(2000);

    // Find and click any "Create" button
    const createButtons = page.locator('button', { hasText: /create/i });
    const buttonCount = await createButtons.count();
    console.log('Found create buttons:', buttonCount);

    if (buttonCount > 0) {
      await createButtons.first().click();
      await page.waitForTimeout(1000);

      // Take screenshot of modal
      await page.screenshot({
        path: 'test-results/create-cluster-modal.png'
      });
    }
  });
});
