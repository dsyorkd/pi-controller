import axios, { type AxiosResponse } from 'axios';
import { getCSRFToken, hasValidSession, refreshTokensSecurely, clearStoredTokensSecurely } from '../utils/secureSession';
import type {
  PaginatedResponse,
  Cluster,
  Node,
  GPIODevice,
  SystemInfo,
  LoginRequest,
  RegisterRequest,
  AuthResponse,
  TokenRefreshResponse,
  User
} from '../types';

// Create axios instance with secure configuration
const secureApi = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1',
  timeout: 10000,
  withCredentials: true, // Essential for httpOnly cookies
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for CSRF protection and session validation
secureApi.interceptors.request.use(
  (config) => {
    // Add CSRF token to all requests
    const csrfToken = getCSRFToken();
    if (csrfToken) {
      config.headers['X-CSRF-Token'] = csrfToken;
    }

    // Check if we have a valid session
    if (!hasValidSession()) {
      console.warn('No valid session found for API request');
    }

    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor for error handling and token refresh
secureApi.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      // Try to refresh token using secure cookie-based refresh
      const refreshSuccess = await refreshTokensSecurely();

      if (refreshSuccess) {
        // Retry the original request
        return secureApi(originalRequest);
      } else {
        // Refresh failed, clear all auth data and redirect to login
        await clearStoredTokensSecurely();

        // Redirect to login page if not already there
        if (window.location.pathname !== '/login') {
          window.location.href = '/login';
        }

        return Promise.reject(error);
      }
    }

    // Handle other auth errors
    if (error.response?.status === 403) {
      console.error('CSRF token validation failed or insufficient permissions');
    }

    return Promise.reject(error);
  }
);

// Secure API service implementation
export const secureApiService = {
  // Authentication endpoints
  auth: {
    login: async (credentials: LoginRequest): Promise<AuthResponse> => {
      const response: AxiosResponse<AuthResponse> = await secureApi.post('/auth/login', credentials);
      return response.data;
    },

    register: async (userData: RegisterRequest): Promise<AuthResponse> => {
      const response: AxiosResponse<AuthResponse> = await secureApi.post('/auth/register', userData);
      return response.data;
    },

    logout: async (): Promise<void> => {
      await secureApi.post('/auth/logout');
      await clearStoredTokensSecurely();
    },

    refresh: async (): Promise<TokenRefreshResponse> => {
      const response: AxiosResponse<TokenRefreshResponse> = await secureApi.post('/auth/refresh');
      return response.data;
    },

    me: async (): Promise<User> => {
      const response: AxiosResponse<User> = await secureApi.get('/auth/me');
      return response.data;
    },

    sessionStatus: async (): Promise<{ isValid: boolean; user?: User }> => {
      try {
        const response = await secureApi.get('/auth/session-status');
        return response.data;
      } catch (error) {
        return { isValid: false };
      }
    },
  },

  // Cluster management
  clusters: {
    getAll: async (): Promise<PaginatedResponse<Cluster>> => {
      const response: AxiosResponse<PaginatedResponse<Cluster>> = await secureApi.get('/clusters');
      return response.data;
    },

    getById: async (id: string): Promise<Cluster> => {
      const response: AxiosResponse<Cluster> = await secureApi.get(`/clusters/${id}`);
      return response.data;
    },

    create: async (cluster: Partial<Cluster>): Promise<Cluster> => {
      const response: AxiosResponse<Cluster> = await secureApi.post('/clusters', cluster);
      return response.data;
    },

    update: async (id: string, cluster: Partial<Cluster>): Promise<Cluster> => {
      const response: AxiosResponse<Cluster> = await secureApi.put(`/clusters/${id}`, cluster);
      return response.data;
    },

    delete: async (id: string): Promise<void> => {
      await secureApi.delete(`/clusters/${id}`);
    },

    getNodes: async (clusterId: string): Promise<Node[]> => {
      const response: AxiosResponse<Node[]> = await secureApi.get(`/clusters/${clusterId}/nodes`);
      return response.data;
    },
  },

  // Node management
  nodes: {
    getAll: async (): Promise<PaginatedResponse<Node>> => {
      const response: AxiosResponse<PaginatedResponse<Node>> = await secureApi.get('/nodes');
      return response.data;
    },

    getById: async (id: string): Promise<Node> => {
      const response: AxiosResponse<Node> = await secureApi.get(`/nodes/${id}`);
      return response.data;
    },

    create: async (node: Partial<Node>): Promise<Node> => {
      const response: AxiosResponse<Node> = await secureApi.post('/nodes', node);
      return response.data;
    },

    update: async (id: string, node: Partial<Node>): Promise<Node> => {
      const response: AxiosResponse<Node> = await secureApi.put(`/nodes/${id}`, node);
      return response.data;
    },

    delete: async (id: string): Promise<void> => {
      await secureApi.delete(`/nodes/${id}`);
    },

    provision: async (nodeId: string, clusterId: string): Promise<void> => {
      await secureApi.post(`/nodes/${nodeId}/provision`, { clusterId });
    },

    deprovision: async (nodeId: string): Promise<void> => {
      await secureApi.post(`/nodes/${nodeId}/deprovision`);
    },
  },

  // GPIO management
  gpio: {
    getAll: async (): Promise<GPIODevice[]> => {
      const response: AxiosResponse<GPIODevice[]> = await secureApi.get('/gpio');
      return response.data;
    },

    getByNodeId: async (nodeId: string): Promise<GPIODevice[]> => {
      const response: AxiosResponse<GPIODevice[]> = await secureApi.get(`/nodes/${nodeId}/gpio`);
      return response.data;
    },

    setPin: async (nodeId: string, pin: number, value: boolean): Promise<void> => {
      await secureApi.post(`/nodes/${nodeId}/gpio/${pin}`, { value });
    },

    getPin: async (nodeId: string, pin: number): Promise<{ value: boolean }> => {
      const response = await secureApi.get(`/nodes/${nodeId}/gpio/${pin}`);
      return response.data;
    },
  },

  // System information
  system: {
    getInfo: async (): Promise<SystemInfo> => {
      const response: AxiosResponse<SystemInfo> = await secureApi.get('/system/info');
      return response.data;
    },

    getHealth: async (): Promise<{ status: string; timestamp: string }> => {
      const response = await secureApi.get('/health');
      return response.data;
    },
  },
};

// Export the secure API service as default
export default secureApiService;