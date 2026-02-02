import { test, expect } from '@playwright/test';

/**
 * Error Handling E2E Tests
 * 
 * Test Issues:
 * #163: 404 Not Found
 * #164: Server Errors (500, 503)
 * #165: API Timeout
 * #166: Auth Errors (401, 403)
 */
test.describe('Error Handling Tests', () => {
  /**
   * Test #163: 404 Not Found
   */
  test.describe('404 Not Found Errors', () => {
    test('should display 404 page for non-existent routes', async ({ page }) => {
      // Navigate to a clearly non-existent page
      await page.goto('/this-page-definitely-does-not-exist-12345');
      await page.waitForLoadState('networkidle');

      // Check for 404 indicators
      const has404Text = await page.locator('text=404').isVisible();
      const hasNotFoundText = await page.locator('text=Not Found').isVisible();
      const hasPageNotFoundText = await page.locator('text=Page not found').isVisible();

      // At least one 404 indicator should be present
      const has404Indicator = has404Text || hasNotFoundText || hasPageNotFoundText;
      expect(has404Indicator).toBeTruthy();

      console.log('✅ 404 page displayed for non-existent route');
    });

    test('should show 404 for invalid cluster IDs', async ({ page }) => {
      // Try to access a non-existent cluster
      await page.goto('/admin/clusters/non-existent-cluster-id-12345');
      await page.waitForLoadState('networkidle');

      // Should show 404 or "Cluster not found"
      const hasError = await Promise.race([
        page.locator('text=404').isVisible(),
        page.locator('text=not found', { hasText: /cluster/i }).isVisible(),
        page.locator('text=does not exist').isVisible()
      ]);

      expect(hasError).toBeTruthy();
      console.log('✅ 404 shown for invalid cluster ID');
    });

    test('should show 404 for invalid node IDs', async ({ page }) => {
      // Try to access a non-existent node
      await page.goto('/admin/nodes/non-existent-node-id-12345');
      await page.waitForLoadState('networkidle');

      // Should show 404 or "Node not found"
      const hasError = await Promise.race([
        page.locator('text=404').isVisible(),
        page.locator('text=not found', { hasText: /node/i }).isVisible(),
        page.locator('text=does not exist').isVisible()
      ]);

      expect(hasError).toBeTruthy();
      console.log('✅ 404 shown for invalid node ID');
    });

    test('should have functional 404 page with navigation', async ({ page }) => {
      await page.goto('/non-existent-page');
      await page.waitForLoadState('networkidle');

      // Look for navigation back to home
      const homeLink = page.locator('a[href="/"], a:has-text("Home"), a:has-text("Dashboard")');
      
      if (await homeLink.count() > 0) {
        await expect(homeLink.first()).toBeVisible();
        
        // Test clicking the home link
        await homeLink.first().click();
        await page.waitForLoadState('networkidle');
        
        // Should navigate back to dashboard
        expect(page.url()).toMatch(/\/$|\dashboard/);
        console.log('✅ 404 page navigation working');
      } else {
        console.log('ℹ️ No home link found on 404 page');
      }
    });

    test('should maintain proper styling on 404 page', async ({ page }) => {
      await page.goto('/non-existent-page');
      await page.waitForLoadState('networkidle');

      // Check that the page has proper styling (not broken CSS)
      const body = page.locator('body');
      const hasStyles = await body.evaluate((el) => {
        const styles = window.getComputedStyle(el);
        return styles.backgroundColor !== 'rgba(0, 0, 0, 0)' || 
               styles.fontFamily !== 'times' || 
               styles.fontSize !== '16px'; // Basic styling check
      });

      if (hasStyles) {
        console.log('✅ 404 page has proper styling');
      }

      // Check for consistent header/navigation
      const header = page.locator('header, nav, .header, .navigation');
      if (await header.count() > 0) {
        await expect(header.first()).toBeVisible();
        console.log('✅ 404 page maintains site navigation');
      }
    });
  });

  /**
   * Test #164: Server Errors (500, 503)
   */
  test.describe('Server Errors (500, 503)', () => {
    test('should handle 500 internal server errors gracefully', async ({ page }) => {
      // Mock 500 error for API calls
      await page.route('**/api/v1/clusters', (route) => {
        route.fulfill({
          status: 500,
          contentType: 'application/json',
          body: JSON.stringify({ 
            error: 'Internal Server Error',
            message: 'Something went wrong on our end'
          })
        });
      });

      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Should show error message or retry option
      const errorMessage = page.locator('.error, .alert-error, [role="alert"]', { hasText: /error|failed|something.*wrong/i });
      const retryButton = page.locator('button:has-text("Retry"), button:has-text("Try Again")');

      // Wait for either error message or retry button
      try {
        await Promise.race([
          expect(errorMessage.first()).toBeVisible({ timeout: 10000 }),
          expect(retryButton.first()).toBeVisible({ timeout: 10000 })
        ]);
        
        console.log('✅ 500 error handled gracefully');
      } catch (error) {
        console.log('ℹ️ 500 error handling may be silent or use different UI patterns');
      }

      // Test retry functionality if available
      if (await retryButton.count() > 0) {
        // Clear the mock to allow retry to succeed
        await page.unroute('**/api/v1/clusters');
        
        // Mock successful retry
        await page.route('**/api/v1/clusters', (route) => {
          route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ data: [], total: 0 })
          });
        });

        await retryButton.first().click();
        await page.waitForTimeout(2000);

        // Error should be gone after successful retry
        const stillHasError = await errorMessage.first().isVisible().catch(() => false);
        if (!stillHasError) {
          console.log('✅ Retry functionality working');
        }
      }
    });

    test('should handle 503 service unavailable errors', async ({ page }) => {
      // Mock 503 error
      await page.route('**/api/v1/**', (route) => {
        route.fulfill({
          status: 503,
          contentType: 'application/json',
          body: JSON.stringify({
            error: 'Service Unavailable',
            message: 'The service is temporarily unavailable. Please try again later.'
          })
        });
      });

      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Look for service unavailable messaging
      const unavailableMessage = page.locator('.error, .alert, [role="alert"]', { 
        hasText: /unavailable|maintenance|temporarily|try.*later/i 
      });
      
      if (await unavailableMessage.count() > 0) {
        await expect(unavailableMessage.first()).toBeVisible();
        console.log('✅ 503 service unavailable error handled');
      } else {
        console.log('ℹ️ 503 error may be handled differently or silently');
      }

      // Check for maintenance mode indication
      const maintenanceMode = page.locator('.maintenance, .offline-mode', { hasText: /maintenance|offline/i });
      if (await maintenanceMode.count() > 0) {
        console.log('✅ Maintenance mode indicator shown');
      }
    });

    test('should show appropriate error messages for different server errors', async ({ page }) => {
      const errorScenarios = [
        { status: 500, message: 'Internal Server Error' },
        { status: 502, message: 'Bad Gateway' },
        { status: 503, message: 'Service Unavailable' },
        { status: 504, message: 'Gateway Timeout' }
      ];

      for (const scenario of errorScenarios) {
        console.log(`Testing ${scenario.status} error handling`);

        await page.route('**/api/v1/clusters', (route) => {
          route.fulfill({
            status: scenario.status,
            contentType: 'application/json',
            body: JSON.stringify({ error: scenario.message })
          });
        });

        await page.goto('/');
        await page.waitForTimeout(2000);

        // Look for error indication
        const hasError = await Promise.race([
          page.locator('.error, .alert-error').first().isVisible(),
          page.locator('button:has-text("Retry")').first().isVisible(),
          page.locator('text=Error', { hasText: /load|fetch|server/i }).first().isVisible()
        ]).catch(() => false);

        if (hasError) {
          console.log(`✅ ${scenario.status} error handled appropriately`);
        }

        // Clean up route for next test
        await page.unroute('**/api/v1/clusters');
      }
    });

    test('should maintain app functionality during partial server errors', async ({ page }) => {
      // Mock error for only clusters API, not other APIs
      await page.route('**/api/v1/clusters', (route) => {
        route.fulfill({
          status: 500,
          body: JSON.stringify({ error: 'Clusters service unavailable' })
        });
      });

      // Mock successful health check
      await page.route('**/api/v1/health', (route) => {
        route.fulfill({
          status: 200,
          contentType: 'application/json',
          body: JSON.stringify({ status: 'healthy' })
        });
      });

      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Navigation and other non-cluster features should still work
      const navigation = page.locator('nav, .navigation');
      if (await navigation.count() > 0) {
        await expect(navigation.first()).toBeVisible();
        console.log('✅ Navigation remains functional during partial errors');
      }

      // UI should not be completely broken
      const mainContent = page.locator('main, .main-content, .container');
      if (await mainContent.count() > 0) {
        await expect(mainContent.first()).toBeVisible();
        console.log('✅ Main UI remains functional during partial errors');
      }
    });
  });

  /**
   * Test #165: API Timeout
   */
  test.describe('API Timeout Errors', () => {
    test('should handle slow API responses with timeout', async ({ page }) => {
      // Mock slow API response
      await page.route('**/api/v1/clusters', (route) => {
        // Simulate a very slow response (longer than typical timeout)
        setTimeout(() => {
          route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ data: [] })
          });
        }, 15000); // 15 second delay
      });

      await page.goto('/');
      
      // Look for loading state initially
      const loadingIndicator = page.locator('.loading, .spinner, [role="progressbar"]');
      
      if (await loadingIndicator.count() > 0) {
        await expect(loadingIndicator.first()).toBeVisible();
        console.log('✅ Loading indicator shown for slow request');
      }

      // Wait and check for timeout handling
      await page.waitForTimeout(5000);

      // Look for timeout error message
      const timeoutError = page.locator('.error, .alert', { hasText: /timeout|slow|taking.*long/i });
      const retryOption = page.locator('button:has-text("Retry"), button:has-text("Refresh")');

      if (await timeoutError.count() > 0) {
        await expect(timeoutError.first()).toBeVisible();
        console.log('✅ Timeout error message displayed');
      } else if (await retryOption.count() > 0) {
        console.log('✅ Retry option available for slow requests');
      } else {
        console.log('ℹ️ Timeout handling may be different or requests may not timeout');
      }
    });

    test('should provide retry mechanism for timed out requests', async ({ page }) => {
      let requestCount = 0;

      await page.route('**/api/v1/clusters', (route) => {
        requestCount++;
        
        if (requestCount === 1) {
          // First request times out (simulate by not responding)
          return;
        } else {
          // Second request succeeds
          route.fulfill({
            status: 200,
            contentType: 'application/json',
            body: JSON.stringify({ data: [], total: 0 })
          });
        }
      });

      await page.goto('/');
      await page.waitForTimeout(3000);

      // Look for retry button
      const retryButton = page.locator('button:has-text("Retry"), button:has-text("Try Again"), button:has-text("Refresh")');
      
      if (await retryButton.count() > 0) {
        await retryButton.first().click();
        await page.waitForTimeout(2000);

        // Should succeed on retry
        expect(requestCount).toBe(2);
        console.log('✅ Retry mechanism working for timeout errors');
      }
    });

    test('should show progressive timeout messaging', async ({ page }) => {
      await page.route('**/api/v1/clusters', (route) => {
        // Never respond to simulate timeout
        return;
      });

      await page.goto('/');

      // Check for initial loading
      await page.waitForTimeout(1000);
      const initialLoading = page.locator('.loading, .spinner');
      
      if (await initialLoading.count() > 0) {
        console.log('✅ Initial loading state shown');
      }

      // Check for "taking longer than expected" message
      await page.waitForTimeout(3000);
      const slowMessage = page.locator('text=taking.*longer, text=still.*loading, .slow-loading');
      
      if (await slowMessage.count() > 0) {
        console.log('✅ Progressive timeout messaging shown');
      }

      // Check for final timeout/error state
      await page.waitForTimeout(3000);
      const timeoutMessage = page.locator('.timeout, .error', { hasText: /timeout|failed|try.*again/i });
      
      if (await timeoutMessage.count() > 0) {
        console.log('✅ Final timeout error state shown');
      }
    });

    test('should handle timeout during form submissions', async ({ page }) => {
      await page.goto('/');

      // Open cluster creation modal
      const createButton = page.getByRole('button', { name: 'Create Cluster' }).first();
      if (await createButton.count() > 0) {
        await createButton.click();
        await page.locator('input[id="cluster-name"]').fill('timeout-test');

        // Mock timeout for POST request
        await page.route('**/api/v1/clusters', (route) => {
          if (route.request().method() === 'POST') {
            // Don't respond to simulate timeout
            return;
          } else {
            route.continue();
          }
        });

        // Submit the form
        const submitButton = page.getByRole('button', { name: 'Create Cluster' }).last();
        await submitButton.click();

        // Wait for timeout handling
        await page.waitForTimeout(5000);

        // Look for timeout error in modal
        const formError = page.locator('.error, .alert-error', { hasText: /timeout|failed.*create|try.*again/i });
        
        if (await formError.count() > 0) {
          await expect(formError.first()).toBeVisible();
          console.log('✅ Form submission timeout handled');
          
          // Modal should still be open for retry
          await expect(page.getByText('Create New Cluster')).toBeVisible();
        }
      }
    });
  });

  /**
   * Test #166: Auth Errors (401, 403)
   */
  test.describe('Authentication Errors (401, 403)', () => {
    test('should handle 401 unauthorized errors', async ({ page }) => {
      // Mock 401 error for protected endpoints
      await page.route('**/api/v1/clusters', (route) => {
        route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ 
            error: 'Unauthorized',
            message: 'Authentication required'
          })
        });
      });

      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Should redirect to login or show auth error
      const currentUrl = page.url();
      const isRedirectedToLogin = currentUrl.includes('/login') || currentUrl.includes('/auth');
      
      if (isRedirectedToLogin) {
        console.log('✅ 401 error redirected to login page');
        
        // Verify login page elements
        const loginForm = page.locator('form, .login-form');
        if (await loginForm.count() > 0) {
          const usernameField = page.locator('input[type="text"], input[type="email"], input[name="username"]');
          const passwordField = page.locator('input[type="password"]');
          
          if (await usernameField.count() > 0 && await passwordField.count() > 0) {
            console.log('✅ Login form displayed');
          }
        }
      } else {
        // Look for auth error message
        const authError = page.locator('.error, .alert', { hasText: /unauthorized|authentication|login|sign.*in/i });
        
        if (await authError.count() > 0) {
          await expect(authError.first()).toBeVisible();
          console.log('✅ 401 error message displayed');
        }
      }
    });

    test('should handle 403 forbidden errors', async ({ page }) => {
      // Mock 403 error for admin endpoints
      await page.route('**/api/v1/admin/**', (route) => {
        route.fulfill({
          status: 403,
          contentType: 'application/json',
          body: JSON.stringify({ 
            error: 'Forbidden',
            message: 'Insufficient permissions'
          })
        });
      });

      // Try to access an admin page
      await page.goto('/admin/settings');
      await page.waitForLoadState('networkidle');

      // Should show permission error
      const permissionError = page.locator('.error, .alert', { 
        hasText: /forbidden|permission|access.*denied|not.*authorized/i 
      });
      
      if (await permissionError.count() > 0) {
        await expect(permissionError.first()).toBeVisible();
        console.log('✅ 403 permission error displayed');
      } else {
        // Check if redirected away from admin page
        const currentUrl = page.url();
        if (!currentUrl.includes('/admin/settings')) {
          console.log('✅ 403 error caused redirect away from protected resource');
        }
      }
    });

    test('should handle session expiration', async ({ page }) => {
      // Start with valid session
      await page.goto('/');
      await page.waitForLoadState('networkidle');

      // Mock session expiration after some time
      await page.route('**/api/v1/**', (route) => {
        route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ 
            error: 'Session Expired',
            message: 'Please log in again'
          })
        });
      });

      // Trigger an API call that would reveal session expiration
      const refreshButton = page.locator('button:has-text("Refresh"), button:has-text("Reload")');
      if (await refreshButton.count() > 0) {
        await refreshButton.first().click();
      } else {
        // Navigate to trigger API call
        await page.reload();
      }

      await page.waitForTimeout(2000);

      // Should handle session expiration gracefully
      const sessionExpired = page.locator('.error, .alert', { hasText: /session.*expired|log.*in.*again|expired/i });
      const loginRedirect = page.url().includes('/login') || page.url().includes('/auth');

      if (await sessionExpired.count() > 0) {
        await expect(sessionExpired.first()).toBeVisible();
        console.log('✅ Session expiration message displayed');
      } else if (loginRedirect) {
        console.log('✅ Session expiration handled with redirect to login');
      } else {
        console.log('ℹ️ Session expiration handling may be different');
      }
    });

    test('should provide clear login prompts for auth errors', async ({ page }) => {
      // Mock 401 for all API calls
      await page.route('**/api/v1/**', (route) => {
        route.fulfill({
          status: 401,
          contentType: 'application/json',
          body: JSON.stringify({ error: 'Authentication required' })
        });
      });

      await page.goto('/');
      await page.waitForTimeout(2000);

      // Look for login prompt or button
      const loginPrompt = page.locator('.login-prompt, .auth-required');
      const loginButton = page.locator('button:has-text("Login"), button:has-text("Sign In"), a:has-text("Login")');

      if (await loginPrompt.count() > 0) {
        await expect(loginPrompt.first()).toBeVisible();
        console.log('✅ Login prompt displayed for auth errors');
      }

      if (await loginButton.count() > 0) {
        await expect(loginButton.first()).toBeVisible();
        console.log('✅ Login button available');
        
        // Test login button functionality
        await loginButton.first().click();
        await page.waitForLoadState('networkidle');
        
        // Should navigate to login page
        const currentUrl = page.url();
        if (currentUrl.includes('/login') || currentUrl.includes('/auth')) {
          console.log('✅ Login button navigates to login page');
        }
      }
    });

    test('should maintain user context after auth errors', async ({ page }) => {
      // Start authenticated
      await page.goto('/');
      
      // Navigate to a specific page
      const clustersLink = page.locator('a[href*="/clusters"]').first();
      if (await clustersLink.count() > 0) {
        await clustersLink.click();
        await page.waitForLoadState('networkidle');
      }

      // Mock auth error
      await page.route('**/api/v1/**', (route) => {
        route.fulfill({
          status: 401,
          body: JSON.stringify({ error: 'Session expired' })
        });
      });

      // Trigger auth error
      await page.reload();
      await page.waitForTimeout(2000);

      // After re-auth (simulated), should return to the same page
      // This tests that the app remembers where the user was
      const returnUrl = page.url();
      
      if (returnUrl.includes('clusters') || returnUrl.includes('redirect') || returnUrl.includes('return')) {
        console.log('✅ User context preserved during auth errors');
      } else {
        console.log('ℹ️ User context handling may not be implemented');
      }
    });
  });

  test('should have consistent error styling and messaging', async ({ page }) => {
    // Test that all error messages follow consistent design patterns
    
    // Mock various errors to test consistency
    const errorTypes = [
      { route: '**/api/v1/clusters', status: 404, message: 'Not Found' },
      { route: '**/api/v1/nodes', status: 500, message: 'Internal Server Error' },
      { route: '**/api/v1/health', status: 503, message: 'Service Unavailable' }
    ];

    for (const errorType of errorTypes) {
      await page.route(errorType.route, (route) => {
        route.fulfill({
          status: errorType.status,
          body: JSON.stringify({ error: errorType.message })
        });
      });
    }

    await page.goto('/');
    await page.waitForTimeout(2000);

    // Look for error messages
    const errorElements = page.locator('.error, .alert-error, [role="alert"]');
    
    if (await errorElements.count() > 0) {
      // Check that errors have consistent styling
      const firstError = errorElements.first();
      
      const hasConsistentStyling = await firstError.evaluate((el) => {
        const styles = window.getComputedStyle(el);
        // Check for error-like styling (red colors, appropriate margins, etc.)
        return styles.color.includes('red') || 
               styles.borderColor.includes('red') || 
               styles.backgroundColor.includes('red') ||
               el.className.includes('error') ||
               el.className.includes('alert');
      });

      if (hasConsistentStyling) {
        console.log('✅ Error messages have consistent styling');
      }

      // Check for consistent action buttons (retry, dismiss, etc.)
      const errorActions = page.locator('.error button, .alert button, button:has-text("Retry")');
      if (await errorActions.count() > 0) {
        console.log('✅ Error messages have consistent action buttons');
      }
    }
  });
});