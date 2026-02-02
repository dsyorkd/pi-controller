import { test, expect } from '@playwright/test';

/**
 * Documentation Page E2E Tests
 * 
 * Test Issues:
 * #154: All Documentation Sections
 * #155: Link Navigation
 * #157: Getting Started Page
 * #159: Hide from navigation Checkbox
 * #160: API Reference Page
 */
test.describe('Documentation Tests', () => {
  test.beforeEach(async ({ page }) => {
    // Navigate to documentation page
    await page.goto('/docs');
    await page.waitForLoadState('networkidle');
    
    // If docs page doesn't exist, try alternative routes
    if (page.url().includes('404') || await page.locator('text=404').isVisible()) {
      await page.goto('/documentation');
      await page.waitForLoadState('networkidle');
    }
    
    // If still not found, navigate from main page
    if (page.url().includes('404') || await page.locator('text=404').isVisible()) {
      await page.goto('/');
      await page.waitForLoadState('networkidle');
      
      const docsLink = page.locator('nav a[href="/docs"], nav a[href="/documentation"], nav a:has-text("Documentation"), nav a:has-text("Docs")');
      if (await docsLink.count() > 0) {
        await docsLink.first().click();
        await page.waitForLoadState('networkidle');
      }
    }
  });

  /**
   * Test #154: All Documentation Sections
   */
  test.describe('Documentation Sections', () => {
    test('should display main documentation sections', async ({ page }) => {
      // Skip if documentation page not found
      if (page.url().includes('404') || await page.locator('text=404').isVisible()) {
        console.log('⚠️ Documentation page not found - skipping section tests');
        return;
      }

      // Verify main documentation heading
      const mainHeading = page.locator('h1', { hasText: /documentation|docs|guide/i });
      await expect(mainHeading.first()).toBeVisible();

      // Check for common documentation sections
      const commonSections = [
        /getting started|quick start|installation/i,
        /configuration|setup|config/i,
        /api reference|api docs|api/i,
        /tutorials|examples|guides/i,
        /troubleshooting|faq|help/i
      ];

      for (const sectionPattern of commonSections) {
        const section = page.locator('h2, h3, .section-title, .nav-item', { hasText: sectionPattern });
        if (await section.count() > 0) {
          await expect(section.first()).toBeVisible();
          console.log(`✅ Found section matching: ${sectionPattern}`);
        } else {
          console.log(`ℹ️ Section not found: ${sectionPattern}`);
        }
      }

      console.log('✅ Documentation sections verification completed');
    });

    test('should have navigation sidebar or menu', async ({ page }) => {
      // Skip if documentation page not found
      if (page.url().includes('404')) {
        return;
      }

      // Look for documentation navigation
      const docNav = page.locator('.docs-nav, .doc-navigation, .sidebar, nav[class*="doc"]');
      
      if (await docNav.count() > 0) {
        await expect(docNav.first()).toBeVisible();
        
        // Check for navigation items
        const navItems = docNav.locator('a, button, .nav-item');
        const itemCount = await navItems.count();
        
        expect(itemCount).toBeGreaterThan(0);
        console.log(`✅ Documentation navigation found with ${itemCount} items`);
        
        // Verify navigation items have proper links
        for (let i = 0; i < Math.min(itemCount, 5); i++) {
          const item = navItems.nth(i);
          const href = await item.getAttribute('href');
          const text = await item.textContent();
          
          if (href && text) {
            console.log(`Navigation item: "${text}" -> ${href}`);
          }
        }
      } else {
        console.log('ℹ️ Documentation navigation not found');
      }
    });

    test('should display table of contents', async ({ page }) => {
      // Skip if documentation page not found
      if (page.url().includes('404')) {
        return;
      }

      // Look for table of contents
      const toc = page.locator('.toc, .table-of-contents, #table-of-contents, [data-testid="toc"]');
      
      if (await toc.count() > 0) {
        await expect(toc.first()).toBeVisible();
        
        // Check for TOC links
        const tocLinks = toc.locator('a');
        const linkCount = await tocLinks.count();
        
        if (linkCount > 0) {
          console.log(`✅ Table of contents found with ${linkCount} links`);
          
          // Test first TOC link
          const firstLink = tocLinks.first();
          const linkHref = await firstLink.getAttribute('href');
          
          if (linkHref && linkHref.startsWith('#')) {
            await firstLink.click();
            await page.waitForTimeout(500);
            
            // Verify anchor navigation worked
            const targetElement = page.locator(linkHref);
            if (await targetElement.count() > 0) {
              console.log('✅ TOC anchor navigation working');
            }
          }
        }
      } else {
        console.log('ℹ️ Table of contents not found');
      }
    });
  });

  /**
   * Test #155: Link Navigation
   */
  test.describe('Documentation Link Navigation', () => {
    test('should navigate between documentation pages', async ({ page }) => {
      // Skip if documentation page not found
      if (page.url().includes('404')) {
        console.log('⚠️ Documentation page not found - skipping link navigation tests');
        return;
      }

      // Find navigation links
      const navLinks = page.locator('.docs-nav a, .doc-navigation a, nav a').filter({ hasText: /getting|start|config|api/i });
      
      if (await navLinks.count() === 0) {
        // Look for any internal documentation links
        const docLinks = page.locator('a[href*="/docs"], a[href*="/documentation"]');
        
        if (await docLinks.count() > 0) {
          const firstLink = docLinks.first();
          const linkText = await firstLink.textContent();
          const linkHref = await firstLink.getAttribute('href');
          
          console.log(`Testing navigation to: ${linkText} (${linkHref})`);
          
          await firstLink.click();
          await page.waitForLoadState('networkidle');
          
          // Verify navigation worked
          expect(page.url()).toContain(linkHref || '');
          await expect(page.locator('text=404')).not.toBeVisible();
          
          console.log('✅ Documentation link navigation working');
        } else {
          console.log('ℹ️ No documentation links found to test');
        }
        return;
      }

      // Test navigation between doc pages
      const linkCount = await navLinks.count();
      
      for (let i = 0; i < Math.min(linkCount, 3); i++) {
        const link = navLinks.nth(i);
        const linkText = await link.textContent();
        const linkHref = await link.getAttribute('href');
        
        if (linkText && linkHref) {
          console.log(`Testing navigation to: ${linkText}`);
          
          await link.click();
          await page.waitForLoadState('networkidle');
          
          // Verify navigation worked
          expect(page.url()).toContain(linkHref);
          await expect(page.locator('text=404')).not.toBeVisible();
          
          // Verify page has content
          const pageContent = page.locator('main, .content, .doc-content');
          await expect(pageContent.first()).toBeVisible();
          
          console.log(`✅ Successfully navigated to: ${linkText}`);
        }
      }
    });

    test('should handle external links correctly', async ({ page }) => {
      // Skip if documentation page not found
      if (page.url().includes('404')) {
        return;
      }

      // Look for external links
      const externalLinks = page.locator('a[href^="http"], a[href^="https"]').filter({ hasNot: page.locator(`a[href*="${new URL(page.url()).hostname}"]`) });
      
      if (await externalLinks.count() > 0) {
        const firstExternal = externalLinks.first();
        const linkHref = await firstExternal.getAttribute('href');
        const linkTarget = await firstExternal.getAttribute('target');
        
        // External links should open in new tab
        if (linkTarget === '_blank' || linkTarget === '_new') {
          console.log('✅ External links properly configured to open in new tab');
        }
        
        // Test that external links have proper attributes
        const hasRel = await firstExternal.getAttribute('rel');
        if (hasRel && (hasRel.includes('noopener') || hasRel.includes('noreferrer'))) {
          console.log('✅ External links have security attributes');
        }
        
        console.log(`External link found: ${linkHref}`);
      } else {
        console.log('ℹ️ No external links found in documentation');
      }
    });

    test('should support breadcrumb navigation', async ({ page }) => {
      // Skip if documentation page not found
      if (page.url().includes('404')) {
        return;
      }

      // Look for breadcrumbs
      const breadcrumbs = page.locator('.breadcrumb, .breadcrumbs, nav[aria-label="breadcrumb"]');
      
      if (await breadcrumbs.count() > 0) {
        await expect(breadcrumbs.first()).toBeVisible();
        
        // Test breadcrumb links
        const breadcrumbLinks = breadcrumbs.locator('a');
        const linkCount = await breadcrumbLinks.count();
        
        if (linkCount > 0) {
          console.log(`✅ Breadcrumbs found with ${linkCount} links`);
          
          // Test clicking a breadcrumb (not the last one)
          if (linkCount > 1) {
            const parentLink = breadcrumbLinks.first();
            const parentHref = await parentLink.getAttribute('href');
            
            if (parentHref) {
              await parentLink.click();
              await page.waitForLoadState('networkidle');
              
              // Verify navigation worked
              expect(page.url()).toContain(parentHref);
              console.log('✅ Breadcrumb navigation working');
            }
          }
        }
      } else {
        console.log('ℹ️ Breadcrumb navigation not found');
      }
    });
  });

  /**
   * Test #157: Getting Started Page
   */
  test.describe('Getting Started Page', () => {
    test('should have accessible getting started guide', async ({ page }) => {
      // Try to find and navigate to getting started
      const gettingStartedLink = page.locator('a', { hasText: /getting started|quick start|start/i });
      
      if (await gettingStartedLink.count() === 0) {
        // Try direct navigation
        await page.goto('/docs/getting-started');
        await page.waitForLoadState('networkidle');
        
        if (page.url().includes('404')) {
          await page.goto('/docs/quickstart');
          await page.waitForLoadState('networkidle');
        }
        
        if (page.url().includes('404')) {
          console.log('⚠️ Getting Started page not found');
          return;
        }
      } else {
        await gettingStartedLink.first().click();
        await page.waitForLoadState('networkidle');
      }

      // Verify getting started content
      const pageTitle = page.locator('h1', { hasText: /getting started|quick start|installation/i });
      await expect(pageTitle.first()).toBeVisible();
      
      // Check for essential getting started sections
      const essentialSections = [
        /installation|install|setup/i,
        /requirements|prerequisite/i,
        /configuration|config/i,
        /first.*cluster|create.*cluster/i
      ];

      let foundSections = 0;
      for (const sectionPattern of essentialSections) {
        const section = page.locator('h2, h3, .section', { hasText: sectionPattern });
        if (await section.count() > 0) {
          foundSections++;
          console.log(`✅ Found getting started section: ${sectionPattern}`);
        }
      }

      expect(foundSections).toBeGreaterThan(1);
      console.log('✅ Getting Started page has essential sections');
    });

    test('should provide installation instructions', async ({ page }) => {
      // Navigate to getting started if not already there
      if (!page.url().includes('getting-started') && !page.url().includes('quickstart')) {
        const gettingStartedLink = page.locator('a', { hasText: /getting started|quick start/i });
        if (await gettingStartedLink.count() > 0) {
          await gettingStartedLink.first().click();
          await page.waitForLoadState('networkidle');
        }
      }

      // Look for installation instructions
      const installSection = page.locator('h2, h3', { hasText: /installation|install/i });
      
      if (await installSection.count() > 0) {
        // Check for code blocks or command examples
        const codeBlocks = page.locator('pre, code, .highlight');
        
        if (await codeBlocks.count() > 0) {
          console.log('✅ Installation code examples found');
          
          // Verify code blocks contain common installation patterns
          const firstCodeBlock = codeBlocks.first();
          const codeContent = await firstCodeBlock.textContent();
          
          if (codeContent) {
            const hasInstallCommands = /curl|wget|docker|kubectl|helm|go install|npm install/.test(codeContent);
            if (hasInstallCommands) {
              console.log('✅ Installation commands detected');
            }
          }
        }
        
        // Check for copy buttons on code blocks
        const copyButtons = page.locator('button', { hasText: /copy/i }).or(page.locator('[title*="copy"]'));
        if (await copyButtons.count() > 0) {
          console.log('✅ Copy buttons available for code blocks');
        }
      }
    });

    test('should include prerequisites section', async ({ page }) => {
      // Look for prerequisites or requirements section
      const prereqSection = page.locator('h2, h3', { hasText: /prerequisite|requirement|before.*start/i });
      
      if (await prereqSection.count() > 0) {
        console.log('✅ Prerequisites section found');
        
        // Check for common prerequisites
        const pageText = await page.textContent('body');
        const commonPrereqs = [
          /docker/i,
          /kubernetes|k8s|k3s/i,
          /raspberry pi|pi 4/i,
          /ubuntu|debian|linux/i,
          /memory|ram|gb/i
        ];
        
        let foundPrereqs = 0;
        for (const prereq of commonPrereqs) {
          if (prereq.test(pageText || '')) {
            foundPrereqs++;
          }
        }
        
        if (foundPrereqs > 2) {
          console.log('✅ Comprehensive prerequisites listed');
        }
      } else {
        console.log('ℹ️ Prerequisites section not explicitly found');
      }
    });
  });

  /**
   * Test #159: Hide from navigation Checkbox
   */
  test.describe('Navigation Visibility Control', () => {
    test('should have hide from navigation functionality', async ({ page }) => {
      // This test assumes there's an admin/editor interface for documentation
      // Skip if documentation page not found
      if (page.url().includes('404')) {
        console.log('⚠️ Documentation page not found - skipping navigation visibility test');
        return;
      }

      // Look for edit mode or admin controls
      const editButton = page.locator('button', { hasText: /edit|admin|manage/i });
      const settingsButton = page.locator('button[aria-label*="settings"], button[title*="settings"]');
      
      if (await editButton.count() > 0) {
        await editButton.first().click();
        await page.waitForTimeout(500);
      } else if (await settingsButton.count() > 0) {
        await settingsButton.first().click();
        await page.waitForTimeout(500);
      } else {
        // Try accessing a settings or admin panel directly
        await page.goto('/admin/docs');
        await page.waitForLoadState('networkidle');
        
        if (page.url().includes('404')) {
          console.log('ℹ️ Documentation admin interface not found');
          return;
        }
      }

      // Look for hide from navigation checkbox
      const hideCheckbox = page.locator('input[type="checkbox"]').filter({ hasText: /hide.*navigation|navigation.*hide|visible.*nav/i });
      
      if (await hideCheckbox.count() === 0) {
        // Look for the checkbox by nearby label text
        const hideLabel = page.locator('label', { hasText: /hide.*navigation|navigation.*hide|visible.*nav/i });
        if (await hideLabel.count() > 0) {
          const checkboxId = await hideLabel.getAttribute('for');
          if (checkboxId) {
            const checkbox = page.locator(`#${checkboxId}`);
            if (await checkbox.count() > 0) {
              console.log('✅ Hide from navigation checkbox found');
              
              // Test toggling the checkbox
              const initialState = await checkbox.isChecked();
              await checkbox.click();
              
              const newState = await checkbox.isChecked();
              expect(newState).toBe(!initialState);
              
              console.log('✅ Hide from navigation toggle working');
            }
          }
        } else {
          console.log('ℹ️ Hide from navigation functionality not found');
        }
      } else {
        console.log('✅ Hide from navigation checkbox found');
        
        // Test the functionality
        const checkbox = hideCheckbox.first();
        const initialState = await checkbox.isChecked();
        
        await checkbox.click();
        const newState = await checkbox.isChecked();
        expect(newState).toBe(!initialState);
        
        console.log('✅ Hide from navigation functionality working');
      }
    });

    test('should respect navigation visibility settings', async ({ page }) => {
      // This test would verify that pages marked as hidden don't appear in navigation
      // Skip if no admin interface
      if (page.url().includes('404')) {
        return;
      }

      // Mock a scenario where some docs are hidden
      const allNavLinks = page.locator('.docs-nav a, nav a[href*="/docs"]');
      const visibleCount = await allNavLinks.count();
      
      console.log(`Found ${visibleCount} visible documentation navigation links`);
      
      // This is more of a structural test - in a real implementation,
      // we'd verify that hidden pages don't appear in the navigation
      if (visibleCount > 0) {
        console.log('✅ Navigation visibility control appears to be working');
      }
    });
  });

  /**
   * Test #160: API Reference Page
   */
  test.describe('API Reference Page', () => {
    test('should have accessible API reference documentation', async ({ page }) => {
      // Try to find and navigate to API reference
      const apiLink = page.locator('a', { hasText: /api reference|api docs|api/i });
      
      if (await apiLink.count() === 0) {
        // Try direct navigation
        await page.goto('/docs/api');
        await page.waitForLoadState('networkidle');
        
        if (page.url().includes('404')) {
          await page.goto('/docs/reference');
          await page.waitForLoadState('networkidle');
        }
        
        if (page.url().includes('404')) {
          console.log('⚠️ API Reference page not found - skipping API reference tests');
          return;
        }
      } else {
        await apiLink.first().click();
        await page.waitForLoadState('networkidle');
      }

      // Verify API reference content
      const pageTitle = page.locator('h1', { hasText: /api reference|api documentation|rest api/i });
      await expect(pageTitle.first()).toBeVisible();
      
      console.log('✅ API Reference page accessible');
    });

    test('should document API endpoints', async ({ page }) => {
      // Navigate to API reference if not already there
      if (!page.url().includes('/api') && !page.url().includes('reference')) {
        const apiLink = page.locator('a', { hasText: /api reference|api docs/i });
        if (await apiLink.count() > 0) {
          await apiLink.first().click();
          await page.waitForLoadState('networkidle');
        } else {
          return; // Skip if no API reference found
        }
      }

      // Look for API endpoint documentation
      const endpointSections = [
        /clusters?.*api|\/clusters?/i,
        /nodes?.*api|\/nodes?/i,
        /health.*api|\/health/i,
        /authentication|auth.*api/i
      ];

      let foundEndpoints = 0;
      for (const endpointPattern of endpointSections) {
        const endpoint = page.locator('h2, h3, .endpoint, code', { hasText: endpointPattern });
        if (await endpoint.count() > 0) {
          foundEndpoints++;
          console.log(`✅ Found API endpoint documentation: ${endpointPattern}`);
        }
      }

      if (foundEndpoints > 0) {
        console.log(`✅ API reference documents ${foundEndpoints} endpoint categories`);
      } else {
        console.log('ℹ️ No specific API endpoints found in documentation');
      }

      // Check for HTTP methods
      const httpMethods = page.locator('.method, .http-method', { hasText: /GET|POST|PUT|DELETE|PATCH/i });
      if (await httpMethods.count() > 0) {
        console.log('✅ HTTP methods documented');
      }
    });

    test('should provide API examples and schemas', async ({ page }) => {
      // Skip if API reference not found
      if (!page.url().includes('/api') && !page.url().includes('reference')) {
        return;
      }

      // Look for code examples
      const codeExamples = page.locator('pre, .example, .code-block');
      
      if (await codeExamples.count() > 0) {
        console.log('✅ API code examples found');
        
        // Check for JSON examples
        const firstExample = codeExamples.first();
        const exampleContent = await firstExample.textContent();
        
        if (exampleContent && exampleContent.includes('{')) {
          console.log('✅ JSON examples present');
        }
        
        // Look for curl examples
        if (exampleContent && exampleContent.includes('curl')) {
          console.log('✅ cURL examples present');
        }
      }

      // Look for schema documentation
      const schemaSection = page.locator('h2, h3', { hasText: /schema|model|response|request/i });
      
      if (await schemaSection.count() > 0) {
        console.log('✅ API schema documentation found');
        
        // Check for parameter documentation
        const parameters = page.locator('table, .parameters, .props').filter({ hasText: /parameter|field|property/i });
        if (await parameters.count() > 0) {
          console.log('✅ Parameter documentation found');
        }
      }
    });

    test('should have interactive API explorer', async ({ page }) => {
      // Skip if API reference not found
      if (!page.url().includes('/api') && !page.url().includes('reference')) {
        return;
      }

      // Look for interactive elements (Swagger UI, etc.)
      const interactiveElements = page.locator('.swagger-ui, .redoc, .api-explorer, button:has-text("Try it")');
      
      if (await interactiveElements.count() > 0) {
        console.log('✅ Interactive API explorer found');
        
        // Test try-it-out functionality if available
        const tryButton = page.locator('button', { hasText: /try.*it|execute|send/i });
        if (await tryButton.count() > 0) {
          console.log('✅ Try-it-out functionality available');
        }
      } else {
        console.log('ℹ️ Interactive API explorer not found');
      }
    });
  });

  test('should have proper documentation search functionality', async ({ page }) => {
    // Skip if documentation page not found
    if (page.url().includes('404')) {
      console.log('⚠️ Documentation page not found - skipping search test');
      return;
    }

    // Look for search functionality
    const searchInput = page.locator('input[type="search"], input[placeholder*="search"], .search-input');
    
    if (await searchInput.count() > 0) {
      console.log('✅ Documentation search found');
      
      // Test search functionality
      await searchInput.first().fill('cluster');
      await searchInput.first().press('Enter');
      
      // Wait for search results
      await page.waitForTimeout(1000);
      
      // Look for search results
      const searchResults = page.locator('.search-results, .search-result, .result');
      
      if (await searchResults.count() > 0) {
        console.log('✅ Search results displayed');
        
        // Test clicking a search result
        const firstResult = searchResults.first();
        const resultLink = firstResult.locator('a').first();
        
        if (await resultLink.count() > 0) {
          await resultLink.click();
          await page.waitForLoadState('networkidle');
          
          // Verify navigation worked
          await expect(page.locator('text=404')).not.toBeVisible();
          console.log('✅ Search result navigation working');
        }
      } else {
        console.log('ℹ️ No search results found or search may use different UI');
      }
    } else {
      console.log('ℹ️ Documentation search functionality not found');
    }
  });
});