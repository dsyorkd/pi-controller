import axios, { type AxiosResponse } from 'axios';
import { storeTokens, clearStoredTokens, getCurrentToken, getCurrentRefreshToken } from '../utils/session';
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

// Create axios instance with default configuration
const api = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api/v1',
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
  },
});

// Request interceptor for authentication
api.interceptors.request.use(
  (config) => {
    // Add auth token if available
    const token = getCurrentToken();
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor for error handling and token refresh
api.interceptors.response.use(
  (response) => response,
  async (error) => {
    const originalRequest = error.config;

    if (error.response?.status === 401 && !originalRequest._retry) {
      originalRequest._retry = true;

      // Try to refresh token
      const refreshToken = getCurrentRefreshToken();
      if (refreshToken) {
        try {
          const response = await api.post<TokenRefreshResponse>('/auth/refresh', {
            refreshToken
          });

          const { token: newToken, refreshToken: newRefreshToken, expiresIn } = response.data;

          // Store new tokens with proper expiration
          storeTokens(newToken, newRefreshToken, expiresIn);

          // Retry original request with new token
          originalRequest.headers.Authorization = `Bearer ${newToken}`;
          return api(originalRequest);
        } catch (refreshError) {
          // Refresh failed, clear tokens and redirect to login
          clearStoredTokens();
          window.location.href = '/login';
          return Promise.reject(refreshError);
        }
      } else {
        // No refresh token, clear tokens and redirect to login
        clearStoredTokens();
        window.location.href = '/login';
      }
    }

    return Promise.reject(error);
  }
);

// API service methods
export const apiService = {
  // Authentication operations
  auth: {
    login: async (credentials: LoginRequest): Promise<AuthResponse> => {
      const response: AxiosResponse<AuthResponse> = await api.post('/auth/login', credentials);

      // Store tokens with proper expiration using session utility
      storeTokens(response.data.token, response.data.refreshToken, response.data.expiresIn);

      return response.data;
    },

    register: async (userData: RegisterRequest): Promise<AuthResponse> => {
      const response: AxiosResponse<AuthResponse> = await api.post('/auth/register', userData);

      // Store tokens with proper expiration using session utility
      storeTokens(response.data.token, response.data.refreshToken, response.data.expiresIn);

      return response.data;
    },

    logout: async (): Promise<void> => {
      try {
        // Call logout endpoint to invalidate token on server
        await api.post('/auth/logout');
      } catch (error) {
        // Continue with logout even if server request fails
        console.warn('Logout request failed, continuing with local cleanup:', error);
      } finally {
        // Always clear stored tokens using session utility
        clearStoredTokens();
      }
    },

    refreshToken: async (refreshToken: string): Promise<TokenRefreshResponse> => {
      const response: AxiosResponse<TokenRefreshResponse> = await api.post('/auth/refresh', {
        refreshToken
      });

      // Update stored tokens with proper expiration using session utility
      storeTokens(response.data.token, response.data.refreshToken, response.data.expiresIn);

      return response.data;
    },

    getProfile: async (): Promise<User> => {
      const response: AxiosResponse<User> = await api.get('/auth/profile');
      return response.data;
    },

    // Check if user is currently authenticated
    isAuthenticated: (): boolean => {
      return getCurrentToken() !== null;
    },

    // Get current auth token
    getToken: (): string | null => {
      return getCurrentToken();
    }
  },

  // Cluster operations
  clusters: {
    getAll: async (): Promise<PaginatedResponse<Cluster>> => {
      const response: AxiosResponse<PaginatedResponse<Cluster>> = await api.get('/clusters');
      return response.data;
    },

    getById: async (id: string): Promise<Cluster> => {
      const response: AxiosResponse<Cluster> = await api.get(`/clusters/${id}`);
      return response.data;
    },

    create: async (cluster: Partial<Cluster>): Promise<Cluster> => {
      const response: AxiosResponse<Cluster> = await api.post('/clusters', cluster);
      return response.data;
    },

    update: async (id: string, cluster: Partial<Cluster>): Promise<Cluster> => {
      const response: AxiosResponse<Cluster> = await api.put(`/clusters/${id}`, cluster);
      return response.data;
    },

    delete: async (id: string): Promise<void> => {
      await api.delete(`/clusters/${id}`);
    },

    getNodes: async (id: string): Promise<Node[]> => {
      const response: AxiosResponse<{ nodes: Node[] }> = await api.get(`/clusters/${id}/nodes`);
      return response.data.nodes;
    },

    getStatus: async (id: string): Promise<unknown> => {
      const response: AxiosResponse<unknown> = await api.get(`/clusters/${id}/status`);
      return response.data;
    },
  },

  // Node operations
  nodes: {
    getAll: async (): Promise<PaginatedResponse<Node>> => {
      const response: AxiosResponse<PaginatedResponse<Node>> = await api.get('/nodes');
      return response.data;
    },

    getById: async (id: string): Promise<Node> => {
      const response: AxiosResponse<Node> = await api.get(`/nodes/${id}`);
      return response.data;
    },

    create: async (node: Partial<Node>): Promise<Node> => {
      const response: AxiosResponse<Node> = await api.post('/nodes', node);
      return response.data;
    },

    update: async (id: string, node: Partial<Node>): Promise<Node> => {
      const response: AxiosResponse<Node> = await api.put(`/nodes/${id}`, node);
      return response.data;
    },

    delete: async (id: string): Promise<void> => {
      await api.delete(`/nodes/${id}`);
    },

    getGPIO: async (id: string): Promise<GPIODevice[]> => {
      const response: AxiosResponse<GPIODevice[]> = await api.get(`/nodes/${id}/gpio`);
      return response.data;
    },

    provision: async (id: string, clusterId: string): Promise<void> => {
      await api.post(`/nodes/${id}/provision`, { clusterId });
    },

    deprovision: async (id: string): Promise<void> => {
      await api.post(`/nodes/${id}/deprovision`);
    },
  },

  // GPIO operations
  gpio: {
    getAll: async (): Promise<PaginatedResponse<GPIODevice>> => {
      const response: AxiosResponse<PaginatedResponse<GPIODevice>> = await api.get('/gpio');
      return response.data;
    },

    getById: async (id: string): Promise<GPIODevice> => {
      const response: AxiosResponse<GPIODevice> = await api.get(`/gpio/${id}`);
      return response.data;
    },

    create: async (device: Partial<GPIODevice>): Promise<GPIODevice> => {
      const response: AxiosResponse<GPIODevice> = await api.post('/gpio', device);
      return response.data;
    },

    update: async (id: string, device: Partial<GPIODevice>): Promise<GPIODevice> => {
      const response: AxiosResponse<GPIODevice> = await api.put(`/gpio/${id}`, device);
      return response.data;
    },

    delete: async (id: string): Promise<void> => {
      await api.delete(`/gpio/${id}`);
    },

    read: async (id: string): Promise<{ value: boolean }> => {
      const response: AxiosResponse<{ value: boolean }> = await api.post(`/gpio/${id}/read`);
      return response.data;
    },

    write: async (id: string, value: boolean): Promise<void> => {
      await api.post(`/gpio/${id}/write`, { value });
    },

    // Pin management by node and pin number
    getPin: async (nodeId: string, pin: number): Promise<GPIODevice> => {
      const response: AxiosResponse<GPIODevice> = await api.get(`/nodes/${nodeId}/gpio/${pin}`);
      return response.data;
    },

    configurePin: async (nodeId: string, pin: number, direction: 'input' | 'output', name?: string): Promise<GPIODevice> => {
      const response: AxiosResponse<GPIODevice> = await api.post(`/nodes/${nodeId}/gpio/${pin}/configure`, {
        direction,
        name,
      });
      return response.data;
    },

    readPin: async (nodeId: string, pin: number): Promise<{ value: boolean }> => {
      const response: AxiosResponse<{ value: boolean }> = await api.get(`/nodes/${nodeId}/gpio/${pin}/read`);
      return response.data;
    },

    writePin: async (nodeId: string, pin: number, value: boolean): Promise<void> => {
      await api.post(`/nodes/${nodeId}/gpio/${pin}/write`, { value });
    },

    // Pin reservation operations
    reservePin: async (nodeId: string, pin: number, userId: string, duration?: number): Promise<void> => {
      await api.post(`/nodes/${nodeId}/gpio/${pin}/reserve`, { userId, duration });
    },

    releasePin: async (nodeId: string, pin: number): Promise<void> => {
      await api.delete(`/nodes/${nodeId}/gpio/${pin}/reserve`);
    },

    getReservations: async (nodeId: string): Promise<any[]> => {
      const response: AxiosResponse<{ reservations: any[] }> = await api.get(`/nodes/${nodeId}/gpio/reservations`);
      return response.data.reservations;
    },

    checkReservation: async (nodeId: string, pin: number): Promise<any> => {
      const response: AxiosResponse<any> = await api.get(`/nodes/${nodeId}/gpio/${pin}/reservation`);
      return response.data;
    },
  },

  // System operations
  system: {
    getInfo: async (): Promise<SystemInfo> => {
      const response: AxiosResponse<SystemInfo> = await api.get('/system/info');
      return response.data;
    },

    getMetrics: async (): Promise<unknown> => {
      const response: AxiosResponse<unknown> = await api.get('/system/metrics');
      return response.data;
    },
  },

  // Health checks
  health: {
    check: async (): Promise<{ status: string }> => {
      const response: AxiosResponse<{ status: string }> = await api.get('/health');
      return response.data;
    },

    ready: async (): Promise<{ status: string }> => {
      const response: AxiosResponse<{ status: string }> = await api.get('/ready');
      return response.data;
    },
  },
};

export default api;
