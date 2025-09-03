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
