import React, { createContext, useContext, useEffect, useState, useCallback } from 'react';
import {
  loginSecurely,
  logoutSecurely,
  hasValidSession,
  setupAutoLogoutSecure,
  getCSRFToken
} from '../../utils/secureSession';
import { secureApiService } from '../../services/secureApi';
import type { User, LoginRequest, RegisterRequest } from '../../types';

interface SecureAuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  csrfToken: string;
  login: (credentials: LoginRequest) => Promise<void>;
  register: (userData: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  clearError: () => void;
  refreshSession: () => Promise<void>;
}

const SecureAuthContext = createContext<SecureAuthContextType | undefined>(undefined);

export const useSecureAuthContext = (): SecureAuthContextType => {
  const context = useContext(SecureAuthContext);
  if (context === undefined) {
    throw new Error('useSecureAuthContext must be used within a SecureAuthProvider');
  }
  return context;
};

interface SecureAuthProviderProps {
  children: React.ReactNode;
}

export const SecureAuthProvider: React.FC<SecureAuthProviderProps> = ({ children }) => {
  const [user, setUser] = useState<User | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [csrfToken, setCsrfToken] = useState<string>('');

  // Initialize CSRF token
  useEffect(() => {
    const token = getCSRFToken();
    setCsrfToken(token);
  }, []);

  // Check for existing session on mount
  const checkSession = useCallback(async () => {
    try {
      setIsLoading(true);
      setError(null);

      if (hasValidSession()) {
        // Check session status with server
        const sessionData = await secureApiService.auth.sessionStatus();

        if (sessionData.isValid && sessionData.user) {
          setUser(sessionData.user);
        } else {
          // Invalid session, clear everything
          await logoutSecurely();
          setUser(null);
        }
      } else {
        setUser(null);
      }
    } catch (err) {
      console.error('Failed to check session:', err);
      setError('Failed to verify session');
      await logoutSecurely();
      setUser(null);
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Initialize session check
  useEffect(() => {
    checkSession();
  }, [checkSession]);

  // Setup automatic logout monitoring
  useEffect(() => {
    const cleanup = setupAutoLogoutSecure(async () => {
      console.log('Session expired, logging out...');
      await logout();
    });

    return cleanup;
  }, []);

  // Login function with secure cookie-based authentication
  const login = useCallback(async (credentials: LoginRequest): Promise<void> => {
    try {
      setIsLoading(true);
      setError(null);

      const response = await loginSecurely(credentials.email, credentials.password);

      if (response.success) {
        setUser(response.user);
        // Refresh CSRF token after login
        const newCsrfToken = getCSRFToken();
        setCsrfToken(newCsrfToken);
      } else {
        throw new Error('Login failed');
      }
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Login failed';
      setError(errorMessage);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Register function
  const register = useCallback(async (userData: RegisterRequest): Promise<void> => {
    try {
      setIsLoading(true);
      setError(null);

      const response = await secureApiService.auth.register(userData);
      setUser(response.user);

      // Refresh CSRF token after registration
      const newCsrfToken = getCSRFToken();
      setCsrfToken(newCsrfToken);
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Registration failed';
      setError(errorMessage);
      throw err;
    } finally {
      setIsLoading(false);
    }
  }, []);

  // Logout function with secure cleanup
  const logout = useCallback(async (): Promise<void> => {
    try {
      setIsLoading(true);
      setError(null);

      // Call backend logout endpoint
      await secureApiService.auth.logout();
    } catch (err) {
      console.error('Logout error:', err);
      // Continue with local cleanup even if server call fails
    } finally {
      // Always clear local state
      setUser(null);
      setIsLoading(false);

      // Redirect to login page
      window.location.href = '/login';
    }
  }, []);

  // Refresh session function
  const refreshSession = useCallback(async (): Promise<void> => {
    await checkSession();
  }, [checkSession]);

  // Clear error function
  const clearError = useCallback(() => {
    setError(null);
  }, []);

  // Handle window focus to check session validity
  useEffect(() => {
    const handleFocus = () => {
      if (user && hasValidSession()) {
        checkSession();
      }
    };

    window.addEventListener('focus', handleFocus);
    return () => window.removeEventListener('focus', handleFocus);
  }, [user, checkSession]);

  const value: SecureAuthContextType = {
    user,
    isAuthenticated: !!user,
    isLoading,
    error,
    csrfToken,
    login,
    register,
    logout,
    clearError,
    refreshSession,
  };

  return (
    <SecureAuthContext.Provider value={value}>
      {children}
    </SecureAuthContext.Provider>
  );
};

export default SecureAuthProvider;