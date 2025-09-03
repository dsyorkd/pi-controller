import { create } from 'zustand';
import { devtools } from 'zustand/middleware';
import type { Cluster, Node, GPIODevice, SystemInfo } from '../types';

interface AppState {
  // UI state
  isLoading: boolean;
  error: string | null;

  // Data state
  clusters: Cluster[];
  nodes: Node[];
  gpioDevices: GPIODevice[];
  systemInfo: SystemInfo | null;

  // Selected items
  selectedCluster: Cluster | null;
  selectedNode: Node | null;

  // Actions
  setLoading: (loading: boolean) => void;
  setError: (error: string | null) => void;

  // Cluster actions
  setClusters: (clusters: Cluster[]) => void;
  addCluster: (cluster: Cluster) => void;
  updateCluster: (id: string, cluster: Partial<Cluster>) => void;
  removeCluster: (id: string) => void;
  setSelectedCluster: (cluster: Cluster | null) => void;

  // Node actions
  setNodes: (nodes: Node[]) => void;
  addNode: (node: Node) => void;
  updateNode: (id: string, node: Partial<Node>) => void;
  removeNode: (id: string) => void;
  setSelectedNode: (node: Node | null) => void;

  // GPIO actions
  setGpioDevices: (devices: GPIODevice[]) => void;
  addGpioDevice: (device: GPIODevice) => void;
  updateGpioDevice: (id: string, device: Partial<GPIODevice>) => void;
  removeGpioDevice: (id: string) => void;

  // System actions
  setSystemInfo: (info: SystemInfo) => void;

  // Reset actions
  reset: () => void;
}

const initialState = {
  isLoading: false,
  error: null,
  clusters: [],
  nodes: [],
  gpioDevices: [],
  systemInfo: null,
  selectedCluster: null,
  selectedNode: null,
};

export const useAppStore = create<AppState>()(
  devtools(
    (set) => ({
      ...initialState,

      // UI actions
      setLoading: (loading) => set({ isLoading: loading }),
      setError: (error) => set({ error }),

      // Cluster actions
      setClusters: (clusters) => set({ clusters }),
      addCluster: (cluster) =>
        set((state) => ({
          clusters: [...state.clusters, cluster],
        })),
      updateCluster: (id, updatedCluster) =>
        set((state) => ({
          clusters: state.clusters.map((cluster) =>
            cluster.id === id ? { ...cluster, ...updatedCluster } : cluster
          ),
          selectedCluster:
            state.selectedCluster?.id === id
              ? { ...state.selectedCluster, ...updatedCluster }
              : state.selectedCluster,
        })),
      removeCluster: (id) =>
        set((state) => ({
          clusters: state.clusters.filter((cluster) => cluster.id !== id),
          selectedCluster: state.selectedCluster?.id === id ? null : state.selectedCluster,
        })),
      setSelectedCluster: (cluster) => set({ selectedCluster: cluster }),

      // Node actions
      setNodes: (nodes) => set({ nodes }),
      addNode: (node) =>
        set((state) => ({
          nodes: [...state.nodes, node],
        })),
      updateNode: (id, updatedNode) =>
        set((state) => ({
          nodes: state.nodes.map((node) => (node.id === id ? { ...node, ...updatedNode } : node)),
          selectedNode:
            state.selectedNode?.id === id
              ? { ...state.selectedNode, ...updatedNode }
              : state.selectedNode,
        })),
      removeNode: (id) =>
        set((state) => ({
          nodes: state.nodes.filter((node) => node.id !== id),
          selectedNode: state.selectedNode?.id === id ? null : state.selectedNode,
        })),
      setSelectedNode: (node) => set({ selectedNode: node }),

      // GPIO actions
      setGpioDevices: (devices) => set({ gpioDevices: devices }),
      addGpioDevice: (device) =>
        set((state) => ({
          gpioDevices: [...state.gpioDevices, device],
        })),
      updateGpioDevice: (id, updatedDevice) =>
        set((state) => ({
          gpioDevices: state.gpioDevices.map((device) =>
            device.id === id ? { ...device, ...updatedDevice } : device
          ),
        })),
      removeGpioDevice: (id) =>
        set((state) => ({
          gpioDevices: state.gpioDevices.filter((device) => device.id !== id),
        })),

      // System actions
      setSystemInfo: (info) => set({ systemInfo: info }),

      // Reset
      reset: () => set(initialState),
    }),
    {
      name: 'pi-controller-store',
    }
  )
);

// Helper selectors (use directly in components)
// Example: const isLoading = useAppStore((state) => state.isLoading);
