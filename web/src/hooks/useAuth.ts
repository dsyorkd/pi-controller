import { useCallback, useEffect } from 'react';
import { useNavigate, useLocation } from 'react-router-dom';
import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import { apiService } from '../services/api';
import { storeTokens, clearStoredTokens, hasValidSession, setupAutoLogout } from '../utils/session';
import type { User, LoginRequest, RegisterRequest } from '../types';

interface AuthState {
  user: User | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;

  // Actions
  login: (credentials: LoginRequest) => Promise<void>;
  register: (userData: RegisterRequest) => Promise<void>;
  logout: () => Promise<void>;
  loadUser: () => Promise<void>;
  clearError: () => void;
}

// Auth store using Zustand
const useAuthStore = create<AuthState>()(
  devtools(
    (set, get) => ({
      user: null,
      isAuthenticated: false,
      isLoading: false,
      error: null,

      login: async (credentials: LoginRequest) => {
        set({ isLoading: true, error: null });
        try {
          const response = await apiService.auth.login(credentials);

          // Store tokens with proper expiration
          storeTokens(response.token, response.refreshToken, response.expiresIn);

          set({
            user: response.user,
            isAuthenticated: true,
            isLoading: false,
            error: null
          });
        } catch (error: any) {
          set({
            user: null,
            isAuthenticated: false,
            isLoading: false,
            error: error.response?.data?.message || 'Login failed'
          });
          throw error;
        }
      },

      register: async (userData: RegisterRequest) => {
        set({ isLoading: true, error: null });
        try {
          const response = await apiService.auth.register(userData);

          // Store tokens with proper expiration
          storeTokens(response.token, response.refreshToken, response.expiresIn);

          set({
            user: response.user,
            isAuthenticated: true,
            isLoading: false,
            error: null
          });
        } catch (error: any) {
          set({
            user: null,
            isAuthenticated: false,
            isLoading: false,
            error: error.response?.data?.message || 'Registration failed'
          });
          throw error;
        }
      },

      logout: async () => {
        set({ isLoading: true, error: null });
        try {
          await apiService.auth.logout();
        } catch (error) {
          console.warn('Logout API call failed:', error);
        } finally {
          // Clear stored tokens
          clearStoredTokens();

          set({
            user: null,
            isAuthenticated: false,
            isLoading: false,
            error: null
          });
        }
      },

      loadUser: async () => {
        // Check if we have a valid session first
        if (!hasValidSession()) {
          clearStoredTokens();
          set({
            user: null,
            isAuthenticated: false,
            isLoading: false
          });
          return;
        }

        set({ isLoading: true, error: null });
        try {
          const user = await apiService.auth.getProfile();
          set({
            user,
            isAuthenticated: true,
            isLoading: false,
            error: null
          });

          // Setup auto-logout when token expires
          setupAutoLogout(() => {
            get().logout();
          });
        } catch (error: any) {
          // If loading user fails, clear authentication
          clearStoredTokens();
          set({
            user: null,
            isAuthenticated: false,
            isLoading: false,
            error: error.response?.data?.message || 'Failed to load user profile'
          });
        }
      },

      clearError: () => set({ error: null })
    }),
    {
      name: 'auth-store'
    }
  )
);

// Custom hook for authentication with navigation
export const useAuth = () => {
  const store = useAuthStore();
  const navigate = useNavigate();
  const location = useLocation();

  // Load user on hook initialization
  useEffect(() => {
    store.loadUser();
  }, [store.loadUser]);

  const loginWithRedirect = useCallback(
    async (credentials: LoginRequest) => {
      await store.login(credentials);

      // Redirect to intended destination or dashboard
      const from = (location.state as any)?.from?.pathname || '/dashboard';
      navigate(from, { replace: true });
    },
    [store.login, navigate, location.state]
  );

  const registerWithRedirect = useCallback(
    async (userData: RegisterRequest) => {
      await store.register(userData);

      // Redirect to dashboard after successful registration
      navigate('/dashboard', { replace: true });
    },
    [store.register, navigate]
  );

  const logoutWithRedirect = useCallback(
    async () => {
      await store.logout();
      navigate('/login', { replace: true });
    },
    [store.logout, navigate]
  );

  return {
    ...store,
    login: loginWithRedirect,
    register: registerWithRedirect,
    logout: logoutWithRedirect
  };
};

// Hook for components that just need auth state without navigation
export const useAuthState = () => {
  return useAuthStore();
};