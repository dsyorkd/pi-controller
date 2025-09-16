import React from 'react';
import * as Sentry from '@sentry/react';

// Fallback component when an error occurs
interface ErrorFallbackProps {
  error: Error;
  resetError: () => void;
  eventId: string | null;
}

const ErrorFallback: React.FC<ErrorFallbackProps> = ({ error, resetError, eventId }) => {
  const reportError = () => {
    if (eventId) {
      // Open Sentry user feedback dialog
      Sentry.showReportDialog({ eventId });
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 py-12 px-4 sm:px-6 lg:px-8">
      <div className="max-w-md w-full space-y-8">
        <div className="text-center">
          {/* Error icon */}
          <div className="mx-auto h-16 w-16 text-red-500 mb-4">
            <svg
              className="h-16 w-16"
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={2}
                d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L4.268 16.5c-.77.833.192 2.5 1.732 2.5z"
              />
            </svg>
          </div>

          {/* Error title */}
          <h1 className="text-3xl font-bold text-gray-900 mb-4">
            Oops! Something went wrong
          </h1>

          {/* Error description */}
          <p className="text-gray-600 mb-8">
            We're sorry, but something unexpected happened. Our team has been notified
            and we're working to fix the issue.
          </p>

          {/* Error details (only in development) */}
          {import.meta.env.DEV && (
            <div className="mb-6 p-4 bg-red-50 border border-red-200 rounded-lg text-left">
              <h3 className="text-sm font-medium text-red-800 mb-2">
                Error Details (Development Mode):
              </h3>
              <pre className="text-xs text-red-700 whitespace-pre-wrap overflow-auto max-h-32">
                {error.message}
                {error.stack && '\n\nStack Trace:\n' + error.stack}
              </pre>
            </div>
          )}

          {/* Event ID (for support) */}
          {eventId && (
            <div className="mb-6 p-3 bg-gray-100 border rounded-lg">
              <p className="text-sm text-gray-700">
                <span className="font-medium">Error ID:</span> {eventId}
              </p>
              <p className="text-xs text-gray-500 mt-1">
                Reference this ID when contacting support
              </p>
            </div>
          )}

          {/* Action buttons */}
          <div className="flex flex-col sm:flex-row gap-3 justify-center">
            <button
              onClick={resetError}
              className="inline-flex items-center px-4 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              Try Again
            </button>

            <button
              onClick={() => window.location.href = '/'}
              className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              Go to Dashboard
            </button>

            {eventId && (
              <button
                onClick={reportError}
                className="inline-flex items-center px-4 py-2 border border-gray-300 text-sm font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                Report Issue
              </button>
            )}
          </div>

          {/* Help text */}
          <div className="mt-8 text-sm text-gray-500">
            <p>
              If this problem persists, please contact our support team or
              check our status page for any ongoing issues.
            </p>
          </div>
        </div>
      </div>
    </div>
  );
};

// Create a simpler Sentry Error Boundary
export const SentryErrorBoundary: React.FC<{ children: React.ReactNode }> = ({ children }) => {
  return (
    <Sentry.ErrorBoundary
      fallback={({ error, resetError, eventId }) => {
        const errorObj = error instanceof Error ? error : new Error(String(error));
        return (
          <ErrorFallback
            error={errorObj}
            resetError={resetError}
            eventId={eventId}
          />
        );
      }}
      beforeCapture={(scope, error, errorInfo) => {
        // Add additional context to the error
        scope.setTag('errorBoundary', true);
        scope.setExtra('errorInfo', errorInfo);

        // Add user agent and viewport info
        scope.setExtra('browser', {
          userAgent: navigator.userAgent,
          viewport: {
            width: window.innerWidth,
            height: window.innerHeight,
          },
          url: window.location.href,
        });

        console.error('Error caught by Sentry Error Boundary:', error, errorInfo);
      }}
    >
      {children}
    </Sentry.ErrorBoundary>
  );
};

// Higher-order component for wrapping individual components
export const withSentryErrorBoundary = <P extends object>(
  Component: React.ComponentType<P>
) => {
  const WrappedComponent: React.FC<P> = (props) => (
    <Sentry.ErrorBoundary
      fallback={({ error, resetError, eventId }) => {
        const errorObj = error instanceof Error ? error : new Error(String(error));
        return (
          <ErrorFallback
            error={errorObj}
            resetError={resetError}
            eventId={eventId}
          />
        );
      }}
    >
      <Component {...props} />
    </Sentry.ErrorBoundary>
  );

  WrappedComponent.displayName = `withSentryErrorBoundary(${Component.displayName || Component.name || 'Component'})`;

  return WrappedComponent;
};