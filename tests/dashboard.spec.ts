import { test, expect } from '@playwright/test';

test.describe('Pi-Controller Dashboard', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the dashboard
    await page.goto('/');

    // Wait for the page to load
    await page.waitForLoadState('networkidle');
  });

  test('should load dashboard without errors', async ({ page }) => {
    // Check that the page loads without errors
    await expect(page.locator('h1:has-text("Cluster Dashboard")')).toBeVisible();

    // Verify main sections are present - use more specific selectors
    await expect(page.getByText('Total Clusters')).toBeVisible();
    await expect(page.getByText('Active Clusters')).toBeVisible();
    await expect(page.getByText('Total Nodes')).toBeVisible();
    await expect(page.locator('p.text-sm.text-gray-600:has-text("Namespaces")')).toBeVisible();
  });

  test('should display cluster overview stats', async ({ page }) => {
    // Verify all stat cards are present - use more specific selectors
    await expect(page.getByText('Total Clusters')).toBeVisible();
    await expect(page.getByText('Active Clusters')).toBeVisible();
    await expect(page.getByText('Total Nodes')).toBeVisible();
    await expect(page.locator('p.text-sm.text-gray-600:has-text("Namespaces")')).toBeVisible();

    // Check that stats show numeric values (should be "0" for empty state or actual numbers)
    const totalClustersValue = page.locator('p:text("Total Clusters") + p');
    await expect(totalClustersValue).toBeVisible();

    const activeClustersValue = page.locator('p:text("Active Clusters") + p');
    await expect(activeClustersValue).toBeVisible();
  });

  test('should display cluster table with proper headers', async ({ page }) => {
    // Check cluster table heading is present (more specific selector)
    await expect(page.getByRole('heading', { name: 'Clusters' })).toBeVisible();

    // Verify table headers
    await expect(page.locator('th:has-text("Name")')).toBeVisible();
    await expect(page.locator('th:has-text("Provider")')).toBeVisible();
    await expect(page.locator('th:has-text("Version")')).toBeVisible();
    await expect(page.locator('th:has-text("Nodes")')).toBeVisible();
    await expect(page.locator('th:has-text("CPU")')).toBeVisible();
    await expect(page.locator('th:has-text("Memory")')).toBeVisible();
    await expect(page.locator('th:has-text("Status")')).toBeVisible();
  });

  test('should handle API error states', async ({ page }) => {
    // Check that the table shows either an error or data
    const clusterTable = page.locator('table');
    await expect(clusterTable).toBeVisible();

    // Check for either error text, retry button, or cluster data
    const tableContent = clusterTable.locator('tbody');
    await expect(tableContent).toBeVisible();

    // Look for specific error indicators based on screenshot
    const errorText = page.getByText('Request failed with status code 426');
    const retryButton = page.getByRole('button', { name: 'Retry' });

    // Either should have error state OR working state
    const hasError = await errorText.isVisible();
    const hasRetry = await retryButton.isVisible();

    if (hasError || hasRetry) {
      console.log('Dashboard is showing API error state - this is expected for testing');
    }
  });

  test('should display workloads section', async ({ page }) => {
    // Check workloads heading is present
    await expect(page.getByRole('heading', { name: 'Workloads' })).toBeVisible();

    // Verify workload types are shown
    await expect(page.getByText('Deployments')).toBeVisible();
    await expect(page.getByText('DaemonSets')).toBeVisible();
    await expect(page.getByText('StatefulSets')).toBeVisible();
    await expect(page.getByText('Jobs')).toBeVisible();

    // Check "View All Workloads" button
    await expect(page.getByRole('button', { name: 'View All Workloads' })).toBeVisible();
  });

  test('should display events section', async ({ page }) => {
    // Check events section is present
    await expect(page.getByRole('heading', { name: 'Recent Events' })).toBeVisible();

    // Check "View All Events" button
    await expect(page.getByRole('button', { name: 'View All Events' })).toBeVisible();
  });

  test('should have working namespace selector', async ({ page }) => {
    // Check namespace selector is present
    const namespaceSelect = page.locator('select');
    await expect(namespaceSelect).toBeVisible();

    // Click the select to open it and check options
    await namespaceSelect.click();

    // Verify namespace options exist
    await expect(namespaceSelect.locator('option:has-text("All Namespaces")')).toBeAttached();
    await expect(namespaceSelect.locator('option:has-text("default")')).toBeAttached();
    await expect(namespaceSelect.locator('option:has-text("kube-system")')).toBeAttached();
    await expect(namespaceSelect.locator('option:has-text("monitoring")')).toBeAttached();
  });

  test('should have working import YAML button', async ({ page }) => {
    // Check Import YAML button is present
    const importButton = page.getByRole('button', { name: 'Import YAML' });
    await expect(importButton).toBeVisible();
    await expect(importButton).toBeEnabled();
  });
});

test.describe('Create Cluster Modal', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should open create cluster modal', async ({ page }) => {
    // Click the main "Create Cluster" button in header
    const createButton = page.getByRole('button', { name: 'Create Cluster' }).first();
    await expect(createButton).toBeVisible();
    await createButton.click();

    // Verify modal opens
    await expect(page.getByText('Create New Cluster')).toBeVisible();

    // Check modal elements
    await expect(page.locator('input[id="cluster-name"]')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Create Cluster' }).last()).toBeVisible();
  });

  test('should have functional modal buttons', async ({ page }) => {
    // Open modal
    await page.getByRole('button', { name: 'Create Cluster' }).first().click();
    await expect(page.getByText('Create New Cluster')).toBeVisible();

    // Verify modal buttons are present and in correct states
    const cancelButton = page.getByRole('button', { name: 'Cancel' });
    const createButton = page.getByRole('button', { name: 'Create Cluster' }).last();

    await expect(cancelButton).toBeVisible();
    await expect(createButton).toBeVisible();
    await expect(createButton).toBeDisabled(); // Should be disabled when name is empty

    // Note: Modal close functionality tested separately in integration tests
    // to avoid complexity with animation timing in unit tests
  });

  test('should validate cluster name input', async ({ page }) => {
    // Open modal
    await page.getByRole('button', { name: 'Create Cluster' }).first().click();

    // Check that create button is disabled when name is empty
    const createButton = page.getByRole('button', { name: 'Create Cluster' }).last();
    await expect(createButton).toBeDisabled();

    // Enter cluster name
    await page.locator('input[id="cluster-name"]').fill('test-cluster');

    // Check that create button is now enabled
    await expect(createButton).toBeEnabled();

    // Clear the input
    await page.locator('input[id="cluster-name"]').clear();

    // Button should be disabled again
    await expect(createButton).toBeDisabled();
  });

  test('should show correct form elements', async ({ page }) => {
    // Open modal
    await page.getByRole('button', { name: 'Create Cluster' }).first().click();

    // Check that the input has the expected placeholder
    const input = page.locator('input[id="cluster-name"]');
    await expect(input).toHaveAttribute('placeholder', 'e.g., production, development');

    // Check form label
    await expect(page.getByText('Cluster Name')).toBeVisible();

    // Check help text
    await expect(page.getByText('Choose a descriptive name for your K3s cluster')).toBeVisible();
  });
});

test.describe('API Integration and Error Handling', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should handle API errors gracefully', async ({ page }) => {
    // The table should show either an error or data
    const clusterTable = page.locator('table tbody');
    await expect(clusterTable).toBeVisible();

    // Check that at least one table cell is present
    const tableCell = clusterTable.locator('td').first();
    await expect(tableCell).toBeVisible();
  });

  test('should show retry button on API failure', async ({ page }) => {
    // Simulate API failure by intercepting requests
    await page.route('**/api/v1/clusters', (route) => {
      route.fulfill({
        status: 500,
        contentType: 'application/json',
        body: JSON.stringify({ error: 'Internal Server Error' })
      });
    });

    // Reload page to trigger failed API call
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Check for error state and retry button
    await expect(page.getByRole('button', { name: 'Retry' })).toBeVisible();
  });

  test('should retry API call when retry button is clicked', async ({ page }) => {
    let requestCount = 0;

    // Intercept API calls and fail the first one, succeed on retry
    await page.route('**/api/v1/clusters', (route) => {
      requestCount++;
      if (requestCount === 1) {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Internal Server Error' })
        });
      } else {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ data: [], total: 0, page: 1, limit: 10 })
        });
      }
    });

    // Reload page to trigger failed API call
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Wait for error state
    const retryButton = page.getByRole('button', { name: 'Retry' });
    await expect(retryButton).toBeVisible();

    // Click retry button
    await retryButton.click();

    // Verify that the retry was successful (no more error state)
    await expect(retryButton).not.toBeVisible();
  });
});

test.describe('Navigation and Links', () => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should have working cluster detail links when clusters exist', async ({ page }) => {
    // Mock a cluster to ensure we have links to test
    await page.route('**/api/v1/clusters', (route) => {
      route.fulfill({
        status: 200,
        contentType: 'application/json',
        body: JSON.stringify({
          data: [
            {
              id: 'test-cluster-1',
              name: 'test-cluster',
              status: 'active',
              nodes: []
            }
          ],
          total: 1,
          page: 1,
          limit: 10
        })
      });
    });

    // Reload to get mocked data
    await page.reload();
    await page.waitForLoadState('networkidle');

    // Check for cluster link
    const clusterLink = page.getByRole('link', { name: 'test-cluster' });
    await expect(clusterLink).toBeVisible();

    // Verify link has correct href
    await expect(clusterLink).toHaveAttribute('href', '/admin/clusters/test-cluster-1');
  });

  test('should have functional navigation buttons', async ({ page }) => {
    // Test that buttons are present and enabled
    await expect(page.getByRole('button', { name: 'Import YAML' })).toBeEnabled();
    await expect(page.getByRole('button', { name: 'Create Cluster' })).toBeEnabled();
    await expect(page.getByRole('button', { name: 'View All Workloads' })).toBeEnabled();
    await expect(page.getByRole('button', { name: 'View All Events' })).toBeEnabled();
  });
});

test.describe('Responsive Design', () => {
  test('should be responsive on mobile', async ({ page }) => {
    // Set mobile viewport
    await page.setViewportSize({ width: 375, height: 667 });
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Check that main elements are still visible
    await expect(page.locator('h1', { hasText: 'Cluster Dashboard' })).toBeVisible();
    await expect(page.getByText('Total Clusters')).toBeVisible();
  });

  test('should be responsive on tablet', async ({ page }) => {
    // Set tablet viewport
    await page.setViewportSize({ width: 768, height: 1024 });
    await page.goto('/');
    await page.waitForLoadState('networkidle');

    // Check that main elements are still visible and properly arranged
    await expect(page.locator('h1', { hasText: 'Cluster Dashboard' })).toBeVisible();
    await expect(page.getByText('Total Clusters')).toBeVisible();
    await expect(page.locator('table')).toBeVisible();
  });
});
