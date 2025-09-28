// Secure session storage utilities using httpOnly cookies

export interface TokenData {
  token: string;
  refreshToken: string;
  expiresAt: number;
}

// Cookie names
const SESSION_COOKIE = 'pi_session';
const CSRF_TOKEN_KEY = 'pi_csrf_token';

/**
 * Generate a random CSRF token
 */
const generateCSRFToken = (): string => {
  const array = new Uint8Array(32);
  crypto.getRandomValues(array);
  return Array.from(array, byte => byte.toString(16).padStart(2, '0')).join('');
};

/**
 * Get CSRF token from localStorage (only the CSRF token, not auth tokens)
 */
export const getCSRFToken = (): string => {
  let token = localStorage.getItem(CSRF_TOKEN_KEY);
  if (!token) {
    token = generateCSRFToken();
    localStorage.setItem(CSRF_TOKEN_KEY, token);
  }
  return token;
};

/**
 * Clear CSRF token
 */
export const clearCSRFToken = (): void => {
  localStorage.removeItem(CSRF_TOKEN_KEY);
};

/**
 * Store authentication tokens using secure httpOnly cookies
 * This function makes a request to the backend to set secure cookies
 */
export const storeTokensSecurely = async (token: string, refreshToken: string, expiresIn: number): Promise<void> => {
  const csrfToken = getCSRFToken();

  try {
    const response = await fetch('/api/v1/auth/set-cookies', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
      },
      credentials: 'include', // Include cookies in request
      body: JSON.stringify({
        token,
        refreshToken,
        expiresIn,
        csrfToken,
      }),
    });

    if (!response.ok) {
      throw new Error('Failed to set secure cookies');
    }
  } catch (error) {
    console.error('Failed to store tokens securely:', error);
    throw error;
  }
};

/**
 * Check if user has a valid session by checking cookie presence
 * We can't directly read httpOnly cookies, so we check for session indicators
 */
export const hasValidSession = (): boolean => {
  // Check if session cookie exists (we can't read httpOnly cookies directly)
  return document.cookie.split(';').some(cookie =>
    cookie.trim().startsWith(`${SESSION_COOKIE}=`)
  );
};

/**
 * Clear all stored authentication data by calling backend endpoint
 */
export const clearStoredTokensSecurely = async (): Promise<void> => {
  const csrfToken = getCSRFToken();

  try {
    await fetch('/api/v1/auth/clear-cookies', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
      },
      credentials: 'include',
    });
  } catch (error) {
    console.error('Failed to clear tokens securely:', error);
  }

  // Clear CSRF token as well
  clearCSRFToken();
};

/**
 * Get the current authentication token from httpOnly cookie
 * Since we can't read httpOnly cookies directly, we make a request to get token status
 */
export const getCurrentTokenSecurely = async (): Promise<string | null> => {
  if (!hasValidSession()) {
    return null;
  }

  try {
    const response = await fetch('/api/v1/auth/token-status', {
      method: 'GET',
      credentials: 'include',
      headers: {
        'X-CSRF-Token': getCSRFToken(),
      },
    });

    if (response.ok) {
      const data = await response.json();
      return data.hasValidToken ? 'valid' : null;
    }
    return null;
  } catch (error) {
    console.error('Failed to check token status:', error);
    return null;
  }
};

/**
 * Refresh authentication tokens using secure httpOnly cookies
 */
export const refreshTokensSecurely = async (): Promise<boolean> => {
  const csrfToken = getCSRFToken();

  try {
    const response = await fetch('/api/v1/auth/refresh', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
      },
      credentials: 'include',
    });

    return response.ok;
  } catch (error) {
    console.error('Failed to refresh tokens:', error);
    return false;
  }
};

/**
 * Setup automatic logout when session expires
 * Since we can't directly monitor httpOnly cookies, we periodically check session status
 */
export const setupAutoLogoutSecure = (onLogout: () => void): (() => void) => {
  const checkInterval = 5 * 60 * 1000; // Check every 5 minutes

  const intervalId = setInterval(async () => {
    const isValid = await getCurrentTokenSecurely();
    if (!isValid) {
      onLogout();
    }
  }, checkInterval);

  // Return cleanup function
  return () => clearInterval(intervalId);
};

/**
 * Login with secure token storage
 */
export const loginSecurely = async (email: string, password: string): Promise<{ user: any; success: boolean }> => {
  const csrfToken = getCSRFToken();

  try {
    const response = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-CSRF-Token': csrfToken,
      },
      credentials: 'include',
      body: JSON.stringify({ email, password }),
    });

    if (response.ok) {
      const data = await response.json();
      return { user: data.user, success: true };
    } else {
      const error = await response.json();
      throw new Error(error.message || 'Login failed');
    }
  } catch (error) {
    console.error('Login failed:', error);
    throw error;
  }
};

/**
 * Logout with secure token clearing
 */
export const logoutSecurely = async (): Promise<void> => {
  await clearStoredTokensSecurely();
};

/**
 * Fallback functions for backward compatibility during migration
 */
export const storeTokens = storeTokensSecurely;
export const clearStoredTokens = clearStoredTokensSecurely;
export const getCurrentToken = getCurrentTokenSecurely;
export const getCurrentRefreshToken = async (): Promise<string | null> => {
  // With httpOnly cookies, we can't directly access refresh tokens
  // The server handles refresh token logic internally
  return hasValidSession() ? 'server-managed' : null;
};

// For migration: these functions will throw errors to help identify usage
export const getStoredTokens = (): never => {
  throw new Error('getStoredTokens is deprecated. Use secure cookie-based authentication.');
};

export const isTokenExpired = (): never => {
  throw new Error('isTokenExpired is deprecated. Use hasValidSession() with secure cookies.');
};

export const getTimeUntilExpiry = (): never => {
  throw new Error('getTimeUntilExpiry is deprecated. Server manages token expiry with secure cookies.');
};

export const setupAutoLogout = setupAutoLogoutSecure;