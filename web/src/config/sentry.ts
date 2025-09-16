import * as Sentry from '@sentry/react';
import { BrowserTracing } from '@sentry/tracing';

// Sentry configuration interface
interface SentryConfig {
  dsn: string;
  environment: string;
  tracesSampleRate: number;
  enableTracing: boolean;
  debug: boolean;
}

// Default configuration
const defaultConfig: SentryConfig = {
  dsn: '',
  environment: 'development',
  tracesSampleRate: 0.1, // 10% of transactions
  enableTracing: true,
  debug: import.meta.env.DEV,
};

// Get Sentry configuration from environment variables
const getSentryConfig = (): SentryConfig => {
  return {
    dsn: import.meta.env.VITE_SENTRY_DSN || defaultConfig.dsn,
    environment: import.meta.env.VITE_APP_ENV || import.meta.env.MODE || defaultConfig.environment,
    tracesSampleRate: parseFloat(import.meta.env.VITE_SENTRY_TRACES_SAMPLE_RATE || String(defaultConfig.tracesSampleRate)),
    enableTracing: import.meta.env.VITE_SENTRY_ENABLE_TRACING !== 'false',
    debug: import.meta.env.VITE_SENTRY_DEBUG === 'true' || defaultConfig.debug,
  };
};

// Initialize Sentry
export const initializeSentry = (): void => {
  const config = getSentryConfig();

  // Only initialize Sentry if DSN is provided
  if (!config.dsn) {
    console.warn('Sentry DSN not found. Sentry will not be initialized.');
    return;
  }

  try {
    Sentry.init({
      dsn: config.dsn,
      environment: config.environment,
      debug: config.debug,

      // Performance monitoring
      tracesSampleRate: config.tracesSampleRate,

      // Integrations
      integrations: config.enableTracing ? [
        new BrowserTracing({
          // Enable capturing of long tasks
          enableLongTask: true,
        }) as any
      ] : [],

      // Error filtering
      beforeSend(event, hint) {
        // Filter out errors from browser extensions
        if (event.exception) {
          const error = hint.originalException;
          if (error && error instanceof Error) {
            if (error.message && error.message.includes('Non-Error promise rejection')) {
              return null;
            }
            // Filter out network errors that are not actionable
            if (error.message && (
              error.message.includes('NetworkError') ||
              error.message.includes('Failed to fetch') ||
              error.message.includes('Load failed')
            )) {
              return null;
            }
          }
        }

        // Filter out errors from development environment that are not useful
        if (config.environment === 'development') {
          // You might want to filter out certain development-only errors
          console.log('Sentry event (dev):', event);
        }

        return event;
      },

      // Performance event filtering
      beforeSendTransaction(event) {
        // Filter out certain transactions if needed
        if (config.environment === 'development') {
          console.log('Sentry transaction (dev):', event);
        }
        return event;
      },

      // Set release version if available
      release: import.meta.env.VITE_APP_VERSION || undefined,
    });

    console.log(`Sentry initialized successfully for environment: ${config.environment}`);
  } catch (error) {
    console.error('Failed to initialize Sentry:', error);
  }
};

// Helper function to set user context
export const setSentryUserContext = (user: {
  id: string;
  email?: string;
  username?: string;
  role?: string;
}): void => {
  Sentry.setUser({
    id: user.id,
    email: user.email,
    username: user.username,
    role: user.role,
  });
};

// Helper function to clear user context (on logout)
export const clearSentryUserContext = (): void => {
  Sentry.setUser(null);
};

// Helper function to add breadcrumb
export const addSentryBreadcrumb = (message: string, category: string, level: Sentry.SeverityLevel = 'info'): void => {
  Sentry.addBreadcrumb({
    message,
    category,
    level,
    timestamp: Date.now() / 1000,
  });
};

// Helper function to capture custom error
export const captureSentryError = (error: Error, context?: Record<string, any>): void => {
  if (context) {
    Sentry.withScope((scope) => {
      Object.keys(context).forEach(key => {
        scope.setContext(key, context[key]);
      });
      Sentry.captureException(error);
    });
  } else {
    Sentry.captureException(error);
  }
};

// Export Sentry for direct use
export { Sentry };