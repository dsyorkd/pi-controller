import React, { createContext, useContext, useEffect, type ReactNode } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { useAuthState } from '../../hooks/useAuth';
import { setSentryUserContext, clearSentryUserContext, addSentryBreadcrumb } from '../../config/sentry';
import type { User, LoginRequest, RegisterRequest } from '../../types';

interface AuthContextType {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
  login: (credentials: LoginRequest) => Promise<void>;
  register: (userData: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  clearError: () => void;
}

const AuthContext = createContext<AuthContextType | undefined>(undefined);

interface AuthProviderProps {
  children: ReactNode;
}

export const AuthProvider: React.FC<AuthProviderProps> = ({ children }) => {
  const authState = useAuthState();
  const navigate = useNavigate();
  const location = useLocation();

  // Initialize authentication on app start
  useEffect(() => {
    authState.loadUser();
  }, [authState.loadUser]);

  // Update Sentry user context when authentication state changes
  useEffect(() => {
    if (authState.user) {
      setSentryUserContext({
        id: authState.user.id,
        email: authState.user.email,
        username: authState.user.username,
        role: authState.user.role,
      });
      addSentryBreadcrumb('User authenticated', 'auth', 'info');
    } else {
      clearSentryUserContext();
    }
  }, [authState.user]);

  const login = async (credentials: LoginRequest) => {
    try {
      await authState.login(credentials);
      addSentryBreadcrumb('User login successful', 'auth', 'info');

      // Redirect to intended destination or dashboard
      const from = (location.state as any)?.from?.pathname || '/dashboard';
      navigate(from, { replace: true });
    } catch (error) {
      addSentryBreadcrumb('User login failed', 'auth', 'error');
      throw error;
    }
  };

  const register = async (userData: RegisterRequest) => {
    try {
      await authState.register(userData);
      addSentryBreadcrumb('User registration successful', 'auth', 'info');

      // Redirect to dashboard after successful registration
      navigate('/dashboard', { replace: true });
    } catch (error) {
      addSentryBreadcrumb('User registration failed', 'auth', 'error');
      throw error;
    }
  };

  const logout = async () => {
    addSentryBreadcrumb('User logout initiated', 'auth', 'info');
    await authState.logout();
    clearSentryUserContext();
    navigate('/login', { replace: true });
  };

  const contextValue: AuthContextType = {
    user: authState.user,
    isAuthenticated: authState.isAuthenticated,
    isLoading: authState.isLoading,
    error: authState.error,
    login,
    register,
    logout,
    clearError: authState.clearError
  };

  return (
    <AuthContext.Provider value={contextValue}>
      {children}
    </AuthContext.Provider>
  );
};

// Custom hook to use auth context
export const useAuthContext = (): AuthContextType => {
  const context = useContext(AuthContext);
  if (context === undefined) {
    throw new Error('useAuthContext must be used within an AuthProvider');
  }
  return context;
};