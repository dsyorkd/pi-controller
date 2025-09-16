// Session storage utilities for secure token management

export interface TokenData {
  token: string;
  refreshToken: string;
  expiresAt: number;
}

// Token storage keys
const TOKEN_KEY = 'authToken';
const REFRESH_TOKEN_KEY = 'refreshToken';
const TOKEN_EXPIRY_KEY = 'authTokenExpiry';

/**
 * Store authentication tokens with expiration
 */
export const storeTokens = (token: string, refreshToken: string, expiresIn: number): void => {
  const expiresAt = Date.now() + (expiresIn * 1000);

  localStorage.setItem(TOKEN_KEY, token);
  localStorage.setItem(REFRESH_TOKEN_KEY, refreshToken);
  localStorage.setItem(TOKEN_EXPIRY_KEY, expiresAt.toString());
};

/**
 * Retrieve stored tokens
 */
export const getStoredTokens = (): TokenData | null => {
  const token = localStorage.getItem(TOKEN_KEY);
  const refreshToken = localStorage.getItem(REFRESH_TOKEN_KEY);
  const expiryStr = localStorage.getItem(TOKEN_EXPIRY_KEY);

  if (!token || !refreshToken || !expiryStr) {
    return null;
  }

  const expiresAt = parseInt(expiryStr, 10);

  return {
    token,
    refreshToken,
    expiresAt
  };
};

/**
 * Check if the current token is expired
 */
export const isTokenExpired = (): boolean => {
  const tokenData = getStoredTokens();

  if (!tokenData) {
    return true;
  }

  // Add 5 minute buffer before actual expiry
  const bufferTime = 5 * 60 * 1000; // 5 minutes in milliseconds
  return Date.now() > (tokenData.expiresAt - bufferTime);
};

/**
 * Clear all stored authentication data
 */
export const clearStoredTokens = (): void => {
  localStorage.removeItem(TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
  localStorage.removeItem(TOKEN_EXPIRY_KEY);
};

/**
 * Check if user has valid authentication tokens
 */
export const hasValidSession = (): boolean => {
  const tokenData = getStoredTokens();
  return tokenData !== null && !isTokenExpired();
};

/**
 * Get the current authentication token
 */
export const getCurrentToken = (): string | null => {
  const tokenData = getStoredTokens();

  if (!tokenData || isTokenExpired()) {
    return null;
  }

  return tokenData.token;
};

/**
 * Get the current refresh token
 */
export const getCurrentRefreshToken = (): string | null => {
  const tokenData = getStoredTokens();
  return tokenData?.refreshToken || null;
};

/**
 * Calculate time until token expiry in milliseconds
 */
export const getTimeUntilExpiry = (): number => {
  const tokenData = getStoredTokens();

  if (!tokenData) {
    return 0;
  }

  return Math.max(0, tokenData.expiresAt - Date.now());
};

/**
 * Setup automatic logout when token expires
 */
export const setupAutoLogout = (onLogout: () => void): (() => void) => {
  const timeUntilExpiry = getTimeUntilExpiry();

  if (timeUntilExpiry <= 0) {
    onLogout();
    return () => {}; // No cleanup needed
  }

  const timeoutId = setTimeout(() => {
    onLogout();
  }, timeUntilExpiry);

  // Return cleanup function
  return () => clearTimeout(timeoutId);
};