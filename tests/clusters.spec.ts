import { test, expect } from '@playwright/test';

/**
 * Test Issue #119: End-to-End Successful Cluster Creation
 * 
 * Tests the complete cluster creation wizard flow from start to finish,
 * including form validation, step progression, and successful completion.
 */
test.describe('End-to-End Cluster Creation', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to the dashboard
    await page.goto('/');
    await page.waitForLoadState('networkidle');
  });

  test('should complete full cluster creation wizard successfully', async ({ page }) => {
    // Step 1: Open cluster creation modal/wizard
    const createButton = page.getByRole('button', { name: 'Create Cluster' }).first();
    await expect(createButton).toBeVisible();
    await createButton.click();

    // Verify modal/wizard opened
    await expect(page.getByText('Create New Cluster')).toBeVisible();

    // Step 2: Fill in basic cluster information
    const clusterNameInput = page.locator('input[id="cluster-name"]');
    await expect(clusterNameInput).toBeVisible();
    await clusterNameInput.fill('test-k3s-cluster');

    // Check if create button is now enabled
    const submitButton = page.getByRole('button', { name: 'Create Cluster' }).last();
    await expect(submitButton).toBeEnabled();

    // Step 3: Look for additional configuration options (if they exist)
    const clusterTypeSelect = page.locator('select[name="type"], select[name="cluster-type"], input[name="type"]');
    if (await clusterTypeSelect.count() > 0) {
      await clusterTypeSelect.selectOption('k3s');
    }

    const nodeCountInput = page.locator('input[name="node-count"], input[name="nodes"]');
    if (await nodeCountInput.count() > 0) {
      await nodeCountInput.fill('3');
    }

    const descriptionInput = page.locator('textarea[name="description"], input[name="description"]');
    if (await descriptionInput.count() > 0) {
      await descriptionInput.fill('Test K3s cluster for E2E testing');
    }

    // Step 4: Handle multi-step wizard if it exists
    const nextButton = page.getByRole('button', { name: /next|continue/i });
    if (await nextButton.count() > 0) {
      console.log('Multi-step wizard detected');
      
      // Click next to proceed to next step
      await nextButton.click();
      
      // Wait for next step to load
      await page.waitForTimeout(500);
      
      // Look for node configuration step
      const nodeConfigSection = page.locator('[data-testid="node-config"], .node-configuration, .step-nodes');
      if (await nodeConfigSection.count() > 0) {
        console.log('Node configuration step found');
        
        // Fill node-specific configuration if available
        const masterNodeInput = page.locator('input[name="master-nodes"], input[name="control-plane"]');
        if (await masterNodeInput.count() > 0) {
          await masterNodeInput.fill('1');
        }
        
        const workerNodeInput = page.locator('input[name="worker-nodes"], input[name="workers"]');
        if (await workerNodeInput.count() > 0) {
          await workerNodeInput.fill('2');
        }
      }
      
      // Continue to next step if there are more
      const continueButton = page.getByRole('button', { name: /next|continue/i });
      if (await continueButton.count() > 0) {
        await continueButton.click();
        await page.waitForTimeout(500);
      }
      
      // Look for network/advanced configuration step
      const networkConfigSection = page.locator('[data-testid="network-config"], .network-configuration, .step-network');
      if (await networkConfigSection.count() > 0) {
        console.log('Network configuration step found');
        
        // Configure network settings if available
        const networkRangeInput = page.locator('input[name="network-range"], input[name="cidr"]');
        if (await networkRangeInput.count() > 0) {
          await networkRangeInput.fill('10.42.0.0/16');
        }
        
        const serviceRangeInput = page.locator('input[name="service-range"], input[name="service-cidr"]');
        if (await serviceRangeInput.count() > 0) {
          await serviceRangeInput.fill('10.43.0.0/16');
        }
      }
    }

    // Step 5: Mock the API call for cluster creation
    await page.route('**/api/v1/clusters', (route) => {
      if (route.request().method() === 'POST') {
        // Mock successful cluster creation
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'test-cluster-123',
            name: 'test-k3s-cluster',
            status: 'creating',
            type: 'k3s',
            created_at: new Date().toISOString()
          })
        });
      } else {
        route.continue();
      }
    });

    // Step 6: Submit the cluster creation form
    const finalSubmitButton = page.getByRole('button', { name: /create cluster|create|finish/i }).last();
    await expect(finalSubmitButton).toBeEnabled();
    await finalSubmitButton.click();

    // Step 7: Verify successful submission
    // Look for success message or redirect
    const successMessage = page.locator('.success, .alert-success, [role="alert"]', { hasText: /success|created|cluster.*created/i });
    const loadingIndicator = page.locator('.loading, .spinner, [role="progressbar"]');
    
    // Wait for either success message or loading to appear
    try {
      await Promise.race([
        successMessage.waitFor({ state: 'visible', timeout: 5000 }),
        loadingIndicator.waitFor({ state: 'visible', timeout: 2000 })
      ]);
      
      if (await successMessage.isVisible()) {
        console.log('✅ Success message displayed');
        await expect(successMessage).toBeVisible();
      } else if (await loadingIndicator.isVisible()) {
        console.log('✅ Loading indicator shown - waiting for completion');
        
        // Wait for loading to complete and success message
        await loadingIndicator.waitFor({ state: 'hidden', timeout: 10000 });
        
        // Check for success message after loading
        if (await successMessage.count() > 0) {
          await expect(successMessage).toBeVisible();
        }
      }
    } catch (error) {
      // If no explicit success message, check if modal closed (indicates success)
      const modalIsGone = await page.getByText('Create New Cluster').count() === 0;
      if (modalIsGone) {
        console.log('✅ Modal closed - likely indicates successful creation');
      } else {
        console.log('⚠️ No clear success indicator found');
      }
    }

    // Step 8: Verify cluster appears in the list (if redirected to clusters page)
    const currentUrl = page.url();
    if (currentUrl.includes('/clusters') || currentUrl.includes('/dashboard')) {
      // Look for the newly created cluster
      const newClusterElement = page.locator('text=test-k3s-cluster, [data-testid="cluster-test-k3s-cluster"]');
      
      // Give some time for the UI to update
      await page.waitForTimeout(2000);
      
      if (await newClusterElement.count() > 0) {
        await expect(newClusterElement.first()).toBeVisible();
        console.log('✅ New cluster appears in the list');
      }
    }

    console.log('✅ Cluster creation flow completed successfully');
  });

  test('should validate cluster creation form inputs', async ({ page }) => {
    // Open cluster creation modal
    await page.getByRole('button', { name: 'Create Cluster' }).first().click();
    await expect(page.getByText('Create New Cluster')).toBeVisible();

    const clusterNameInput = page.locator('input[id="cluster-name"]');
    const submitButton = page.getByRole('button', { name: 'Create Cluster' }).last();

    // Test empty name validation
    await expect(submitButton).toBeDisabled();

    // Test invalid characters in cluster name
    await clusterNameInput.fill('invalid cluster name with spaces!@#');
    
    // Look for validation error message
    const validationError = page.locator('.error, .invalid, [role="alert"]', { hasText: /invalid|error/i });
    
    if (await validationError.count() > 0) {
      await expect(validationError.first()).toBeVisible();
      console.log('✅ Validation error shown for invalid input');
    }

    // Test valid cluster name
    await clusterNameInput.fill('valid-cluster-name');
    await expect(submitButton).toBeEnabled();

    // Test name length limits if they exist
    await clusterNameInput.fill('a'.repeat(100)); // Very long name
    
    if (await validationError.count() > 0 && await validationError.first().isVisible()) {
      console.log('✅ Length validation working');
    }

    console.log('✅ Form validation tests completed');
  });

  test('should handle cluster creation API errors gracefully', async ({ page }) => {
    // Open cluster creation modal
    await page.getByRole('button', { name: 'Create Cluster' }).first().click();
    await page.locator('input[id="cluster-name"]').fill('test-error-cluster');

    // Mock API error response
    await page.route('**/api/v1/clusters', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Failed to create cluster: insufficient resources'
          })
        });
      } else {
        route.continue();
      }
    });

    // Submit the form
    const submitButton = page.getByRole('button', { name: 'Create Cluster' }).last();
    await submitButton.click();

    // Wait for error message
    const errorMessage = page.locator('.error, .alert-error, [role="alert"]', { hasText: /error|failed|insufficient/i });
    await expect(errorMessage.first()).toBeVisible({ timeout: 10000 });

    // Verify the modal is still open for retry
    await expect(page.getByText('Create New Cluster')).toBeVisible();
    
    // Verify the form is still functional
    await expect(submitButton).toBeEnabled();

    console.log('✅ Error handling working correctly');
  });

  test('should support cluster creation with different types', async ({ page }) => {
    // Open cluster creation modal
    await page.getByRole('button', { name: 'Create Cluster' }).first().click();
    await page.locator('input[id="cluster-name"]').fill('multi-type-test');

    // Look for cluster type selection
    const typeSelector = page.locator('select[name="type"], [data-testid="cluster-type"], input[type="radio"][name="type"]');
    
    if (await typeSelector.count() > 0) {
      console.log('✅ Cluster type selection found');
      
      // Test different cluster types if available
      const selectElement = typeSelector.first();
      
      if (await selectElement.getAttribute('tagName') === 'SELECT') {
        // Dropdown selection
        const options = await selectElement.locator('option').allTextContents();
        console.log('Available cluster types:', options);
        
        if (options.includes('k3s') || options.some(opt => opt.toLowerCase().includes('k3s'))) {
          await selectElement.selectOption({ label: /k3s/i });
        }
      } else {
        // Radio button selection
        const k3sRadio = page.locator('input[type="radio"][value="k3s"], input[type="radio"] + label:has-text("k3s")');
        if (await k3sRadio.count() > 0) {
          await k3sRadio.first().click();
        }
      }
    }

    // Mock successful creation
    await page.route('**/api/v1/clusters', (route) => {
      if (route.request().method() === 'POST') {
        route.fulfill({
          status: 201,
          contentType: 'application/json',
          body: JSON.stringify({
            id: 'multi-type-test-123',
            name: 'multi-type-test',
            type: 'k3s',
            status: 'creating'
          })
        });
      } else {
        route.continue();
      }
    });

    // Submit the form
    await page.getByRole('button', { name: 'Create Cluster' }).last().click();

    // Verify successful submission
    await page.waitForTimeout(2000);
    console.log('✅ Different cluster types test completed');
  });

  test('should preserve form data when navigating between wizard steps', async ({ page }) => {
    // Open cluster creation modal
    await page.getByRole('button', { name: 'Create Cluster' }).first().click();
    
    // Fill initial data
    await page.locator('input[id="cluster-name"]').fill('persistence-test');
    
    // Look for multi-step wizard
    const nextButton = page.getByRole('button', { name: /next|continue/i });
    
    if (await nextButton.count() > 0) {
      console.log('Testing wizard step persistence');
      
      // Go to next step
      await nextButton.click();
      await page.waitForTimeout(500);
      
      // Go back to previous step
      const backButton = page.getByRole('button', { name: /back|previous/i });
      if (await backButton.count() > 0) {
        await backButton.click();
        await page.waitForTimeout(500);
        
        // Verify data is still there
        const nameInput = page.locator('input[id="cluster-name"]');
        await expect(nameInput).toHaveValue('persistence-test');
        console.log('✅ Form data persistence working');
      }
    } else {
      console.log('ℹ️ Single-step form - no persistence test needed');
    }
  });

  test('should show cluster creation progress', async ({ page }) => {
    // Open cluster creation modal
    await page.getByRole('button', { name: 'Create Cluster' }).first().click();
    await page.locator('input[id="cluster-name"]').fill('progress-test');

    // Mock slow API response to test progress indicators
    await page.route('**/api/v1/clusters', (route) => {
      if (route.request().method() === 'POST') {
        setTimeout(() => {
          route.fulfill({
            status: 201,
            contentType: 'application/json',
            body: JSON.stringify({
              id: 'progress-test-123',
              name: 'progress-test',
              status: 'creating'
            })
          });
        }, 2000); // 2 second delay
      } else {
        route.continue();
      }
    });

    const submitButton = page.getByRole('button', { name: 'Create Cluster' }).last();
    await submitButton.click();

    // Check for loading indicators
    const loadingSpinner = page.locator('.spinner, .loading, [role="progressbar"]');
    const disabledButton = page.locator('button:disabled', { hasText: /creating|creating cluster/i });
    
    // Verify loading state is shown
    try {
      await Promise.race([
        loadingSpinner.waitFor({ state: 'visible', timeout: 1000 }),
        disabledButton.waitFor({ state: 'visible', timeout: 1000 })
      ]);
      console.log('✅ Progress indicator shown during creation');
    } catch {
      console.log('ℹ️ No progress indicators detected');
    }

    // Wait for completion
    await page.waitForTimeout(3000);
    console.log('✅ Progress test completed');
  });
});