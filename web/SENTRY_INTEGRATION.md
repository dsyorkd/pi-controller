# Sentry Integration

This document describes the Sentry integration implementation for error monitoring and performance tracking in the Pi Controller web application.

## Overview

The Sentry integration provides:
- Automatic error capture and reporting
- Performance monitoring and transaction tracking
- User context tracking for authenticated users
- Custom error boundaries with fallback UI
- Integration with React Router for page navigation tracking

## Configuration

### Environment Variables

Copy `.env.example` to `.env` and configure the following variables:

```bash
# Required: Your Sentry DSN
VITE_SENTRY_DSN=https://your-dsn@your-sentry-project.ingest.sentry.io/your-project-id

# Optional configurations
VITE_APP_ENV=development
VITE_SENTRY_TRACES_SAMPLE_RATE=0.1
VITE_SENTRY_ENABLE_TRACING=true
VITE_SENTRY_DEBUG=false
VITE_APP_VERSION=1.0.0
```

### Sample Rates

- **Development**: 100% error capture, 10% performance monitoring
- **Production**: 100% error capture, 1% performance monitoring (recommended)

## Architecture

### Core Components

1. **Sentry Configuration** (`src/config/sentry.ts`)
   - Centralized configuration management
   - Environment-specific settings
   - Helper functions for user context and breadcrumbs

2. **Error Boundary** (`src/components/error/SentryErrorBoundary.tsx`)
   - Wraps the entire application
   - Provides fallback UI for unhandled errors
   - Captures error context and user information

3. **Authentication Integration**
   - Automatic user context setting on login
   - Context clearing on logout
   - Authentication event breadcrumbs

### Performance Monitoring

The integration includes:
- **Page Load Monitoring**: Automatic tracking of initial page loads
- **Navigation Tracking**: React Router integration for SPA navigation
- **Core Web Vitals**: FCP, LCP, CLS, FID tracking
- **Long Task Detection**: Identification of blocking main thread tasks
- **User Interaction Tracking**: Click and navigation events

### Error Filtering

The following errors are automatically filtered:
- Browser extension errors
- Non-actionable network errors (in production)
- Development-only console errors

## Usage

### Manual Error Reporting

```typescript
import { captureSentryError, addSentryBreadcrumb } from '../config/sentry';

// Capture an error with context
try {
  // Some operation
} catch (error) {
  captureSentryError(error, {
    operation: 'user-registration',
    additionalContext: { userId: '123' }
  });
}

// Add a breadcrumb
addSentryBreadcrumb('User clicked save button', 'ui', 'info');
```

### Component-Level Error Boundaries

```typescript
import { withSentryErrorBoundary } from '../components/error/SentryErrorBoundary';

const MyComponent = () => {
  // Component implementation
};

export default withSentryErrorBoundary(MyComponent);
```

### User Context

User context is automatically managed through the authentication system:
- Set on successful login
- Updated when user data changes
- Cleared on logout

## Development vs Production

### Development Mode
- Debug logging enabled
- Full error details shown in UI
- Console logging for Sentry events
- Higher sample rates for testing

### Production Mode
- Error filtering enabled
- User-friendly error messages
- Performance-optimized sample rates
- Automatic error reporting without user intervention

## Error Recovery

The error boundary provides multiple recovery options:
1. **Try Again**: Attempts to re-render the component
2. **Go to Dashboard**: Safe navigation to main application
3. **Report Issue**: Opens Sentry user feedback dialog (if event ID available)

## Security Considerations

- Sensitive data filtering in `beforeSend` hooks
- User context limited to non-sensitive fields (ID, email, username, role)
- Error messages sanitized in production
- No source code exposure in error traces

## Monitoring and Alerts

Configure Sentry alerts for:
- Error rate thresholds
- Performance degradation
- Critical errors (authentication failures, API errors)
- User-reported issues

## Testing

### Quick Testing with SentryTestComponent

In development mode, you can import and use the `SentryTestComponent`:

```typescript
import { SentryTestComponent } from '../components/debug/SentryTestComponent';

// Add to any page for testing
<SentryTestComponent />
```

This component provides buttons to:
- Test manual error capture
- Test breadcrumb creation
- Test error boundary (throws an error)

### Manual Testing Steps

1. **Error Boundary**: Add a component that throws an error
2. **Performance**: Check transaction traces in Sentry dashboard
3. **User Context**: Verify user information appears in error reports
4. **Breadcrumbs**: Confirm authentication events are tracked

### Integration Testing

1. Set up a test Sentry project
2. Configure `VITE_SENTRY_DSN` with test project DSN
3. Run the application and trigger various scenarios:
   - Login/logout to test user context
   - Navigate between pages to test performance monitoring
   - Trigger errors to test error boundary
   - Check breadcrumbs for user actions

## Troubleshooting

### Common Issues

1. **Events not appearing in Sentry**
   - Check DSN configuration
   - Verify network connectivity
   - Check console for initialization errors

2. **Performance data missing**
   - Ensure `VITE_SENTRY_ENABLE_TRACING=true`
   - Check sample rate settings
   - Verify BrowserTracing integration

3. **User context not updating**
   - Check authentication state management
   - Verify Sentry user context calls in auth provider

### Debug Mode

Enable debug mode for troubleshooting:
```bash
VITE_SENTRY_DEBUG=true
```

This will log all Sentry events to the console.

## Dependencies

- `@sentry/react`: Core Sentry React integration
- `@sentry/tracing`: Performance monitoring and tracing
- React Router v7: For navigation tracking
- React 19: For error boundary integration

## Best Practices

1. **Error Context**: Always include relevant context when manually capturing errors
2. **User Privacy**: Never include sensitive data in error reports
3. **Performance Impact**: Monitor the performance impact of Sentry itself
4. **Error Grouping**: Use consistent error messages for proper grouping
5. **Release Tracking**: Set release versions for better error tracking across deployments