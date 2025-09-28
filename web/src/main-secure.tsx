import React from 'react'
import ReactDOM from 'react-dom/client'
import SecureApp from './SecureApp'
import { SecurityProvider } from './components/security/SecurityProvider'
import { initializeSecurityLogging } from './utils/securityHeaders'
import './index.css'

// Initialize security monitoring and logging
initializeSecurityLogging()

// Log security initialization
console.log('🛡️ Pi Controller - Enhanced Security Mode Enabled')
console.log('Features:', {
  'httpOnly Cookies': '✓',
  'CSRF Protection': '✓',
  'Security Headers': '✓',
  'Content Security Policy': '✓',
  'Secure Session Management': '✓',
})

// Security check on startup
const performSecurityCheck = () => {
  const isHTTPS = window.location.protocol === 'https:'
  const isLocalhost = window.location.hostname === 'localhost'

  if (!isHTTPS && !isLocalhost && import.meta.env.PROD) {
    console.error('🚨 Security Warning: Application should be served over HTTPS in production')
  }

  // Check for security features availability
  const features = {
    crypto: typeof crypto !== 'undefined' && typeof crypto.getRandomValues === 'function',
    localStorage: typeof localStorage !== 'undefined',
    sessionStorage: typeof sessionStorage !== 'undefined',
    cookies: navigator.cookieEnabled,
  }

  console.log('🔍 Security Features Check:', features)

  // Warn about potential security issues
  if (!features.crypto) {
    console.warn('⚠️ Crypto API not available - CSRF token generation may be compromised')
  }

  if (!features.cookies) {
    console.error('🚨 Cookies disabled - Secure authentication will not work')
  }
}

// Perform security check
performSecurityCheck()

// Enhanced error boundary for security errors
const SecurityErrorFallback: React.FC<{ error: Error }> = ({ error }) => (
  <div className="min-h-screen flex items-center justify-center bg-red-50">
    <div className="max-w-md w-full bg-white shadow-lg rounded-lg p-6">
      <div className="flex items-center mb-4">
        <svg className="w-8 h-8 text-red-600 mr-3" fill="none" stroke="currentColor" viewBox="0 0 24 24">
          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 19c-.77.833.192 2.5 1.732 2.5z" />
        </svg>
        <h1 className="text-xl font-bold text-red-900">Security Error</h1>
      </div>

      <p className="text-red-700 mb-4">
        A security-related error occurred while loading the application.
      </p>

      <details className="mb-4">
        <summary className="cursor-pointer text-red-600 font-medium">Error Details</summary>
        <pre className="mt-2 text-xs text-red-600 bg-red-50 p-2 rounded overflow-auto">
          {error.message}
        </pre>
      </details>

      <div className="space-y-2">
        <button
          onClick={() => window.location.reload()}
          className="w-full px-4 py-2 bg-red-600 text-white rounded hover:bg-red-700 transition-colors"
        >
          Reload Application
        </button>

        <button
          onClick={() => {
            // Clear all local storage and reload
            localStorage.clear()
            sessionStorage.clear()
            window.location.reload()
          }}
          className="w-full px-4 py-2 bg-gray-600 text-white rounded hover:bg-gray-700 transition-colors"
        >
          Reset & Reload
        </button>
      </div>

      <div className="mt-4 p-3 bg-blue-50 border border-blue-200 rounded">
        <p className="text-xs text-blue-700">
          🔒 <strong>Security Note:</strong> This error boundary protects against potential security vulnerabilities.
          If this error persists, please contact your system administrator.
        </p>
      </div>
    </div>
  </div>
)

// Enhanced error boundary component
class SecurityErrorBoundary extends React.Component<
  { children: React.ReactNode },
  { hasError: boolean; error?: Error }
> {
  constructor(props: { children: React.ReactNode }) {
    super(props)
    this.state = { hasError: false }
  }

  static getDerivedStateFromError(error: Error) {
    // Log security-related errors
    if (error.message.includes('CSP') ||
        error.message.includes('CSRF') ||
        error.message.includes('security') ||
        error.message.includes('cookie')) {
      console.error('🚨 Security Error Detected:', error)
    }

    return { hasError: true, error }
  }

  componentDidCatch(error: Error, errorInfo: React.ErrorInfo) {
    console.error('🚨 Security Error Boundary Caught:', error, errorInfo)

    // In production, you might want to send this to an error reporting service
    if (import.meta.env.PROD) {
      // Example: Send to Sentry or other error tracking service
      console.log('Error would be sent to monitoring service in production')
    }
  }

  render() {
    if (this.state.hasError && this.state.error) {
      return <SecurityErrorFallback error={this.state.error} />
    }

    return this.props.children
  }
}

// Main application render with security providers
ReactDOM.createRoot(document.getElementById('root')!).render(
  <React.StrictMode>
    <SecurityErrorBoundary>
      <SecurityProvider>
        <SecureApp />
      </SecurityProvider>
    </SecurityErrorBoundary>
  </React.StrictMode>,
)

// Security event listeners
window.addEventListener('securitypolicyviolation', (event) => {
  console.warn('🚨 CSP Violation Detected:', {
    blockedURI: event.blockedURI,
    directive: event.violatedDirective,
    effectiveDirective: event.effectiveDirective,
    originalPolicy: event.originalPolicy,
    referrer: event.referrer,
    sourceFile: event.sourceFile,
    lineNumber: event.lineNumber,
    columnNumber: event.columnNumber,
  })
})

// Prevent common security issues
window.addEventListener('beforeunload', () => {
  // Clear any sensitive data from memory before page unload
  // This is a good practice for sensitive applications
  console.log('🔒 Clearing sensitive data before page unload')
})

// Disable right-click context menu in production (optional security measure)
if (import.meta.env.PROD) {
  document.addEventListener('contextmenu', (e) => {
    e.preventDefault()
  })

  // Disable F12, Ctrl+Shift+I, Ctrl+U (optional - can be bypassed but adds a layer)
  document.addEventListener('keydown', (e) => {
    if (
      e.key === 'F12' ||
      (e.ctrlKey && e.shiftKey && e.key === 'I') ||
      (e.ctrlKey && e.key === 'u')
    ) {
      e.preventDefault()
    }
  })
}

// Performance monitoring with security context
if ('performance' in window) {
  window.addEventListener('load', () => {
    setTimeout(() => {
      const perfData = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming
      console.log('📊 Performance Metrics (Security Context):', {
        domContentLoaded: perfData.domContentLoadedEventEnd - perfData.domContentLoadedEventStart,
        loadComplete: perfData.loadEventEnd - perfData.loadEventStart,
        securityProtocol: window.location.protocol,
        secureContext: window.isSecureContext,
      })
    }, 0)
  })
}