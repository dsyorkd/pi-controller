/// <reference types="vitest" />
import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// Basic security headers for development
const developmentSecurityHeaders = {
  'X-Content-Type-Options': 'nosniff',
  'X-Frame-Options': 'DENY',
  'X-XSS-Protection': '1; mode=block',
  'Referrer-Policy': 'strict-origin-when-cross-origin',
  'X-DNS-Prefetch-Control': 'off',
  'X-Download-Options': 'noopen',
  'X-Permitted-Cross-Domain-Policies': 'none',
  'Content-Security-Policy': "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; font-src 'self' https://fonts.gstatic.com data:; img-src 'self' data: blob: https:; connect-src 'self' http://localhost:8080 ws: wss:; frame-ancestors 'none'; form-action 'self'",
}

// https://vite.dev/config/
export default defineConfig({
  plugins: [
    react(),
    // Custom plugin to add security headers in development
    {
      name: 'security-headers',
      configureServer(server) {
        server.middlewares.use('/', (req, res, next) => {
          // Apply security headers to all responses
          Object.entries(developmentSecurityHeaders).forEach(([key, value]) => {
            res.setHeader(key, value);
          });

          // Additional development-specific headers
          res.setHeader('X-Environment', 'development');
          res.setHeader('X-Security-Mode', 'enhanced');

          next();
        });
      },
    },
  ],
  test: {
    globals: true,
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
  },
  server: {
    port: 3000,
    open: true,
    // Security-focused server configuration
    headers: {
      'X-Content-Type-Options': 'nosniff',
      'X-Frame-Options': 'DENY',
      'X-XSS-Protection': '1; mode=block',
    },
    // Proxy configuration for backend API
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true,
        secure: false, // Set to true in production with valid SSL
        // Add security headers to proxied requests
        configure: (proxy, _options) => {
          proxy.on('proxyReq', (proxyReq, req, _res) => {
            // Add security headers to backend requests
            proxyReq.setHeader('X-Forwarded-Proto', 'http');
            proxyReq.setHeader('X-Forwarded-Host', req.headers.host || 'localhost:3000');
          });
        },
      },
    },
  },
  build: {
    // Security-focused build configuration
    rollupOptions: {
      output: {
        // Add integrity check for built assets
        entryFileNames: 'assets/[name]-[hash].js',
        chunkFileNames: 'assets/[name]-[hash].js',
        assetFileNames: 'assets/[name]-[hash].[ext]',
      },
    },
    // Enable source maps for better debugging (disable in production)
    sourcemap: true,
  },
  // Security environment variables
  define: {
    __SECURITY_MODE__: JSON.stringify('enhanced'),
    __BUILD_TIME__: JSON.stringify(new Date().toISOString()),
  },
})
