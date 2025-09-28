// Security headers configuration and utilities

export interface SecurityConfig {
  contentSecurityPolicy: {
    directives: Record<string, string[]>;
    reportOnly: boolean;
  };
  strictTransportSecurity: {
    maxAge: number;
    includeSubDomains: boolean;
    preload: boolean;
  };
  referrerPolicy: string;
  permissionsPolicy: Record<string, string[]>;
  crossOriginEmbedderPolicy: string;
  crossOriginOpenerPolicy: string;
  crossOriginResourcePolicy: string;
}

// Default security configuration
export const defaultSecurityConfig: SecurityConfig = {
  contentSecurityPolicy: {
    directives: {
      'default-src': ["'self'"],
      'script-src': [
        "'self'",
        "'unsafe-inline'", // Required for Vite in development
        "'unsafe-eval'", // Required for Vite in development
        "https://js.sentry-cdn.com", // Sentry SDK
      ],
      'style-src': [
        "'self'",
        "'unsafe-inline'", // Required for Tailwind CSS
        "https://fonts.googleapis.com",
      ],
      'font-src': [
        "'self'",
        "https://fonts.gstatic.com",
        "data:", // For icon fonts
      ],
      'img-src': [
        "'self'",
        "data:",
        "blob:",
        "https:", // Allow HTTPS images
      ],
      'connect-src': [
        "'self'",
        "https://api.sentry.io", // Sentry error reporting
        "wss:", // WebSocket connections
        "ws:", // WebSocket connections (dev)
      ],
      'frame-ancestors': ["'none'"],
      'form-action': ["'self'"],
      'upgrade-insecure-requests': [],
      'block-all-mixed-content': [],
    },
    reportOnly: false, // Set to true for testing
  },
  strictTransportSecurity: {
    maxAge: 31536000, // 1 year
    includeSubDomains: true,
    preload: true,
  },
  referrerPolicy: 'strict-origin-when-cross-origin',
  permissionsPolicy: {
    camera: [],
    microphone: [],
    geolocation: [],
    gyroscope: [],
    magnetometer: [],
    payment: [],
    usb: [],
  },
  crossOriginEmbedderPolicy: 'require-corp',
  crossOriginOpenerPolicy: 'same-origin',
  crossOriginResourcePolicy: 'same-site',
};

// Production security configuration (more restrictive)
export const productionSecurityConfig: SecurityConfig = {
  ...defaultSecurityConfig,
  contentSecurityPolicy: {
    ...defaultSecurityConfig.contentSecurityPolicy,
    directives: {
      ...defaultSecurityConfig.contentSecurityPolicy.directives,
      'script-src': [
        "'self'",
        "https://js.sentry-cdn.com",
      ],
      'connect-src': [
        "'self'",
        "https://api.sentry.io",
        "wss:",
      ],
    },
  },
};

/**
 * Generate Content Security Policy header value
 */
export const generateCSPHeader = (config: SecurityConfig): string => {
  const { directives } = config.contentSecurityPolicy;

  return Object.entries(directives)
    .map(([directive, sources]) => {
      if (sources.length === 0) {
        return directive;
      }
      return `${directive} ${sources.join(' ')}`;
    })
    .join('; ');
};

/**
 * Generate Strict Transport Security header value
 */
export const generateHSTSHeader = (config: SecurityConfig): string => {
  const { maxAge, includeSubDomains, preload } = config.strictTransportSecurity;

  let header = `max-age=${maxAge}`;

  if (includeSubDomains) {
    header += '; includeSubDomains';
  }

  if (preload) {
    header += '; preload';
  }

  return header;
};

/**
 * Generate Permissions Policy header value
 */
export const generatePermissionsPolicyHeader = (config: SecurityConfig): string => {
  return Object.entries(config.permissionsPolicy)
    .map(([feature, allowlist]) => {
      if (allowlist.length === 0) {
        return `${feature}=()`;
      }
      return `${feature}=(${allowlist.map(origin => `"${origin}"`).join(' ')})`;
    })
    .join(', ');
};

/**
 * Get all security headers for the current environment
 */
export const getSecurityHeaders = (): Record<string, string> => {
  const isProduction = import.meta.env.PROD;
  const config = isProduction ? productionSecurityConfig : defaultSecurityConfig;

  const headers: Record<string, string> = {
    // Content Security Policy
    'Content-Security-Policy': generateCSPHeader(config),

    // Strict Transport Security (HTTPS only)
    ...(isProduction && {
      'Strict-Transport-Security': generateHSTSHeader(config),
    }),

    // Referrer Policy
    'Referrer-Policy': config.referrerPolicy,

    // Permissions Policy
    'Permissions-Policy': generatePermissionsPolicyHeader(config),

    // Cross-Origin Policies
    'Cross-Origin-Embedder-Policy': config.crossOriginEmbedderPolicy,
    'Cross-Origin-Opener-Policy': config.crossOriginOpenerPolicy,
    'Cross-Origin-Resource-Policy': config.crossOriginResourcePolicy,

    // Additional Security Headers
    'X-Content-Type-Options': 'nosniff',
    'X-Frame-Options': 'DENY',
    'X-XSS-Protection': '1; mode=block',
    'X-DNS-Prefetch-Control': 'off',
    'X-Download-Options': 'noopen',
    'X-Permitted-Cross-Domain-Policies': 'none',
  };

  return headers;
};

/**
 * Apply security headers to fetch requests
 */
export const applySecurityHeaders = (headers: Record<string, string> = {}): Record<string, string> => {
  const securityHeaders = getSecurityHeaders();

  return {
    ...headers,
    ...securityHeaders,
  };
};

/**
 * Security header validation
 */
export const validateSecurityHeaders = (headers: Record<string, string>): {
  valid: boolean;
  missing: string[];
  warnings: string[];
} => {
  const requiredHeaders = [
    'Content-Security-Policy',
    'X-Content-Type-Options',
    'X-Frame-Options',
    'Referrer-Policy',
  ];

  const missing: string[] = [];
  const warnings: string[] = [];

  // Check for required headers
  requiredHeaders.forEach(header => {
    if (!headers[header]) {
      missing.push(header);
    }
  });

  // Check for security warnings
  if (headers['Content-Security-Policy']?.includes("'unsafe-inline'")) {
    warnings.push("CSP contains 'unsafe-inline' directive - consider using nonces");
  }

  if (headers['Content-Security-Policy']?.includes("'unsafe-eval'")) {
    warnings.push("CSP contains 'unsafe-eval' directive - avoid in production");
  }

  if (!headers['Strict-Transport-Security'] && import.meta.env.PROD) {
    warnings.push('HSTS header missing in production environment');
  }

  return {
    valid: missing.length === 0,
    missing,
    warnings,
  };
};

/**
 * Log security header status for debugging
 */
export const logSecurityStatus = (): void => {
  if (import.meta.env.DEV) {
    const headers = getSecurityHeaders();
    const validation = validateSecurityHeaders(headers);

    console.group('🛡️ Security Headers Status');
    console.log('Headers applied:', Object.keys(headers).length);
    console.log('Validation:', validation.valid ? '✅ Valid' : '❌ Issues found');

    if (validation.missing.length > 0) {
      console.warn('Missing headers:', validation.missing);
    }

    if (validation.warnings.length > 0) {
      console.warn('Warnings:', validation.warnings);
    }

    console.log('All headers:', headers);
    console.groupEnd();
  }
};

/**
 * Initialize security headers logging
 */
export const initializeSecurityLogging = (): void => {
  if (import.meta.env.DEV) {
    // Log security status on page load
    logSecurityStatus();

    // Log CSP violations if any
    document.addEventListener('securitypolicyviolation', (event) => {
      console.warn('🚨 CSP Violation:', {
        blockedURI: event.blockedURI,
        directive: event.violatedDirective,
        originalPolicy: event.originalPolicy,
        referrer: event.referrer,
        sourceFile: event.sourceFile,
        lineNumber: event.lineNumber,
      });
    });
  }
};