import axios, { type AxiosResponse } from 'axios';
import type { PaginatedResponse, Cluster, Node, GPIODevice, SystemInfo } from '../types';

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
    const token = localStorage.getItem('authToken');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor for error handling
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response?.status === 401) {
      // Handle unauthorized - redirect to login
      localStorage.removeItem('authToken');
      window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// API service methods
export const apiService = {
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
