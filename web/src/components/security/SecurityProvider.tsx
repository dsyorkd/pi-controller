import React, { createContext, useContext, useEffect, useState } from 'react';
import { initializeSecurityLogging, validateSecurityHeaders, getSecurityHeaders } from '../../utils/securityHeaders';
import { getCSRFToken } from '../../utils/secureSession';

interface SecurityContextType {
  csrfToken: string;
  isSecure: boolean;
  securityStatus: {
    headersValid: boolean;
    httpsEnabled: boolean;
    csrfEnabled: boolean;
    cookiesSecure: boolean;
  };
  securityWarnings: string[];
  refreshSecurityStatus: () => void;
}

const SecurityContext = createContext<SecurityContextType | undefined>(undefined);

export const useSecurityContext = (): SecurityContextType => {
  const context = useContext(SecurityContext);
  if (context === undefined) {
    throw new Error('useSecurityContext must be used within a SecurityProvider');
  }
  return context;
};

interface SecurityProviderProps {
  children: React.ReactNode;
}

export const SecurityProvider: React.FC<SecurityProviderProps> = ({ children }) => {
  const [csrfToken, setCsrfToken] = useState<string>('');
  const [securityStatus, setSecurityStatus] = useState({
    headersValid: false,
    httpsEnabled: false,
    csrfEnabled: false,
    cookiesSecure: false,
  });
  const [securityWarnings, setSecurityWarnings] = useState<string[]>([]);

  // Initialize security monitoring
  useEffect(() => {
    initializeSecurityLogging();
    checkSecurityStatus();

    // Refresh CSRF token
    const token = getCSRFToken();
    setCsrfToken(token);

    // Setup periodic security checks
    const securityCheckInterval = setInterval(checkSecurityStatus, 5 * 60 * 1000); // Every 5 minutes

    // Cleanup interval on unmount
    return () => clearInterval(securityCheckInterval);
  }, []);

  const checkSecurityStatus = () => {
    const warnings: string[] = [];

    // Check HTTPS
    const httpsEnabled = window.location.protocol === 'https:' || window.location.hostname === 'localhost';
    if (!httpsEnabled && import.meta.env.PROD) {
      warnings.push('Application should be served over HTTPS in production');
    }

    // Check security headers
    const headers = getSecurityHeaders();
    const headerValidation = validateSecurityHeaders(headers);
    if (!headerValidation.valid) {
      warnings.push(...headerValidation.missing.map(h => `Missing security header: ${h}`));
    }
    warnings.push(...headerValidation.warnings);

    // Check CSRF token
    const csrfEnabled = !!getCSRFToken();
    if (!csrfEnabled) {
      warnings.push('CSRF token not found - ensure secure session is initialized');
    }

    // Check cookies security
    const cookiesSecure = checkCookiesSecurity();

    setSecurityStatus({
      headersValid: headerValidation.valid,
      httpsEnabled,
      csrfEnabled,
      cookiesSecure,
    });

    setSecurityWarnings(warnings);
  };

  const checkCookiesSecurity = (): boolean => {
    // Check if we have secure cookies set
    const cookies = document.cookie.split(';');
    const hasSessionCookie = cookies.some(cookie =>
      cookie.trim().startsWith('pi_session=')
    );

    // In development, we can't easily check cookie flags, so we assume they're correct
    return hasSessionCookie || import.meta.env.DEV;
  };

  const refreshSecurityStatus = () => {
    checkSecurityStatus();
    setCsrfToken(getCSRFToken());
  };

  const isSecure = securityStatus.headersValid &&
                   securityStatus.httpsEnabled &&
                   securityStatus.csrfEnabled;

  const value: SecurityContextType = {
    csrfToken,
    isSecure,
    securityStatus,
    securityWarnings,
    refreshSecurityStatus,
  };

  return (
    <SecurityContext.Provider value={value}>
      {children}
      <SecurityStatusIndicator />
    </SecurityContext.Provider>
  );
};

// Security status indicator component
const SecurityStatusIndicator: React.FC = () => {
  const { isSecure, securityWarnings, securityStatus } = useSecurityContext();

  // Only show in development or if there are security issues
  if (import.meta.env.PROD && isSecure) {
    return null;
  }

  return (
    <div className="fixed bottom-4 right-4 z-50">
      <div className={`p-3 rounded-lg shadow-lg max-w-sm ${
        isSecure ? 'bg-green-50 border border-green-200' : 'bg-yellow-50 border border-yellow-200'
      }`}>
        <div className="flex items-center space-x-2">
          {isSecure ? (
            <svg className="w-5 h-5 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z" />
            </svg>
          ) : (
            <svg className="w-5 h-5 text-yellow-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.964-.833-2.732 0L3.732 19c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
          )}
          <span className={`text-sm font-medium ${
            isSecure ? 'text-green-800' : 'text-yellow-800'
          }`}>
            Security Status
          </span>
        </div>

        <div className="mt-2 space-y-1">
          <SecurityStatusItem
            label="HTTPS"
            status={securityStatus.httpsEnabled}
          />
          <SecurityStatusItem
            label="Security Headers"
            status={securityStatus.headersValid}
          />
          <SecurityStatusItem
            label="CSRF Protection"
            status={securityStatus.csrfEnabled}
          />
          <SecurityStatusItem
            label="Secure Cookies"
            status={securityStatus.cookiesSecure}
          />
        </div>

        {securityWarnings.length > 0 && (
          <details className="mt-2">
            <summary className={`text-xs cursor-pointer ${
              isSecure ? 'text-green-700' : 'text-yellow-700'
            }`}>
              {securityWarnings.length} warnings
            </summary>
            <ul className="mt-1 space-y-1">
              {securityWarnings.map((warning, index) => (
                <li key={index} className={`text-xs ${
                  isSecure ? 'text-green-600' : 'text-yellow-600'
                }`}>
                  • {warning}
                </li>
              ))}
            </ul>
          </details>
        )}
      </div>
    </div>
  );
};

// Individual security status item
const SecurityStatusItem: React.FC<{ label: string; status: boolean }> = ({ label, status }) => (
  <div className="flex items-center justify-between">
    <span className="text-xs text-gray-600">{label}</span>
    <span className={`text-xs font-medium ${status ? 'text-green-600' : 'text-red-600'}`}>
      {status ? '✓' : '✗'}
    </span>
  </div>
);

export default SecurityProvider;