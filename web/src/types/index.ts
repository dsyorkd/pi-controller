// Pi Controller Web UI Types

export interface Cluster {
  id: string;
  name: string;
  status: 'active' | 'inactive' | 'error';
  nodes: Node[];
  createdAt: string;
  updatedAt: string;
}

export interface Node {
  id: string;
  name: string;
  ipAddress: string;
  macAddress: string;
  status: 'online' | 'offline' | 'provisioning' | 'error';
  role: 'master' | 'worker';
  clusterId?: string;
  architecture?: string;
  model?: string;
  cpuCores?: number;
  memory?: number;
  lastSeen: string;
}

export interface GPIODevice {
  id: string;
  nodeId: string;
  pin: number;
  direction: 'input' | 'output';
  value: boolean;
  name?: string;
  description?: string;
  reserved?: boolean;
  reservedBy?: string;
  reservedAt?: string;
  lastUpdated?: string;
}

export interface GPIOPinState {
  pin: number;
  nodeId: string;
  direction: 'input' | 'output';
  value: boolean;
  reserved: boolean;
  reservedBy?: string;
  name?: string;
  description?: string;
  lastUpdated: string;
}

export interface GPIOReservation {
  pin: number;
  nodeId: string;
  reservedBy: string;
  reservedAt: string;
  expires?: string;
}

export interface WebSocketMessage {
  type: 'gpio_update' | 'gpio_reservation' | 'error' | 'connection_status';
  payload: any;
  timestamp: string;
}

export interface SystemInfo {
  version: string;
  uptime: string;
  memory: {
    total: number;
    used: number;
    free: number;
  };
  cpu: {
    cores: number;
    usage: number;
  };
}

export interface ApiResponse<T> {
  data: T;
  message?: string;
  error?: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  count: number;
  limit: number;
  offset: number;
}

// Authentication types
export interface User {
  id: string;
  username: string;
  email: string;
  role: 'admin' | 'user' | 'readonly';
  createdAt: string;
  updatedAt: string;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  email: string;
  password: string;
}

export interface AuthResponse {
  token: string;
  refreshToken: string;
  user: User;
  expiresIn: number;
}

export interface TokenRefreshRequest {
  refreshToken: string;
}

export interface TokenRefreshResponse {
  token: string;
  refreshToken: string;
  expiresIn: number;
}
