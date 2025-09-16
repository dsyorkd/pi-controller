import React, { useState, useEffect, useCallback } from 'react';
import { apiService } from '../../services/api';
import { useGPIOWebSocket } from '../../hooks/useWebSocket';
import type { Node, GPIODevice, GPIOPinState } from '../../types';
import GPIOPinGrid from './GPIOPinGrid';
import NodeSelector from './NodeSelector';
import ConnectionStatus from './ConnectionStatus';
import './GPIOControlPanel.css';

interface GPIOControlPanelProps {
  selectedNodeId?: string;
  onNodeChange?: (nodeId: string) => void;
}

export const GPIOControlPanel: React.FC<GPIOControlPanelProps> = ({
  selectedNodeId,
  onNodeChange,
}) => {
  const [nodes, setNodes] = useState<Node[]>([]);
  const [currentNodeId, setCurrentNodeId] = useState<string>(selectedNodeId || '');
  const [gpioPins, setGpioPins] = useState<Map<number, GPIODevice>>(new Map());
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [refreshing, setRefreshing] = useState(false);

  // WebSocket connection for real-time updates
  const {
    connected,
    connecting,
    error: wsError,
    subscribeToPinUpdates,
    subscribeToReservations,
    requestPinState,
    reservePin,
    releasePin,
  } = useGPIOWebSocket({
    onPinUpdate: useCallback((pinState: GPIOPinState) => {
      if (pinState.nodeId === currentNodeId) {
        setGpioPins(prev => {
          const updated = new Map(prev);
          const existing = updated.get(pinState.pin);
          if (existing) {
            updated.set(pinState.pin, {
              ...existing,
              value: pinState.value,
              reserved: pinState.reserved,
              reservedBy: pinState.reservedBy,
              lastUpdated: pinState.lastUpdated,
            });
          }
          return updated;
        });
      }
    }, [currentNodeId]),
    onReservationChange: useCallback((reservation: any) => {
      if (reservation.nodeId === currentNodeId) {
        setGpioPins(prev => {
          const updated = new Map(prev);
          const existing = updated.get(reservation.pin);
          if (existing) {
            updated.set(reservation.pin, {
              ...existing,
              reserved: reservation.reserved,
              reservedBy: reservation.reservedBy,
              reservedAt: reservation.reservedAt,
            });
          }
          return updated;
        });
      }
    }, [currentNodeId]),
  });

  // Load available nodes
  useEffect(() => {
    const loadNodes = async () => {
      try {
        const response = await apiService.nodes.getAll();
        const availableNodes = response.data.filter(node => node.status === 'online');
        setNodes(availableNodes);

        if (!currentNodeId && availableNodes.length > 0) {
          setCurrentNodeId(availableNodes[0].id);
        }
      } catch (err) {
        console.error('Failed to load nodes:', err);
        setError('Failed to load nodes');
      }
    };

    loadNodes();
  }, [currentNodeId]);

  // Load GPIO pins for selected node
  useEffect(() => {
    if (!currentNodeId) return;

    const loadGPIOPins = async () => {
      setLoading(true);
      setError(null);

      try {
        const pins = await apiService.nodes.getGPIO(currentNodeId);
        const pinMap = new Map<number, GPIODevice>();
        pins.forEach(pin => pinMap.set(pin.pin, pin));
        setGpioPins(pinMap);
      } catch (err) {
        console.error('Failed to load GPIO pins:', err);
        setError('Failed to load GPIO pins for selected node');
      } finally {
        setLoading(false);
      }
    };

    loadGPIOPins();
  }, [currentNodeId]);

  // Subscribe to WebSocket updates for current node
  useEffect(() => {
    if (!currentNodeId || !connected) return;

    const unsubscribePins = subscribeToPinUpdates(currentNodeId);
    const unsubscribeReservations = subscribeToReservations(currentNodeId);

    return () => {
      unsubscribePins();
      unsubscribeReservations();
    };
  }, [currentNodeId, connected, subscribeToPinUpdates, subscribeToReservations]);

  const handleNodeChange = useCallback((nodeId: string) => {
    setCurrentNodeId(nodeId);
    onNodeChange?.(nodeId);
  }, [onNodeChange]);

  const handlePinRead = useCallback(async (pin: number) => {
    if (!currentNodeId) return;

    try {
      setRefreshing(true);
      const result = await apiService.gpio.readPin(currentNodeId, pin);

      // Update local state
      setGpioPins(prev => {
        const updated = new Map(prev);
        const existing = updated.get(pin);
        if (existing) {
          updated.set(pin, {
            ...existing,
            value: result.value,
            lastUpdated: new Date().toISOString(),
          });
        }
        return updated;
      });

      // Also request fresh state via WebSocket
      requestPinState(currentNodeId, pin);
    } catch (err) {
      console.error('Failed to read pin:', err);
      setError(`Failed to read pin ${pin}`);
    } finally {
      setRefreshing(false);
    }
  }, [currentNodeId, requestPinState]);

  const handlePinWrite = useCallback(async (pin: number, value: boolean) => {
    if (!currentNodeId) return;

    try {
      await apiService.gpio.writePin(currentNodeId, pin, value);

      // Optimistically update local state
      setGpioPins(prev => {
        const updated = new Map(prev);
        const existing = updated.get(pin);
        if (existing && existing.direction === 'output') {
          updated.set(pin, {
            ...existing,
            value,
            lastUpdated: new Date().toISOString(),
          });
        }
        return updated;
      });
    } catch (err) {
      console.error('Failed to write pin:', err);
      setError(`Failed to write to pin ${pin}`);
    }
  }, [currentNodeId]);

  const handlePinConfigure = useCallback(async (pin: number, direction: 'input' | 'output', name?: string) => {
    if (!currentNodeId) return;

    try {
      const updated = await apiService.gpio.configurePin(currentNodeId, pin, direction, name);

      setGpioPins(prev => {
        const newMap = new Map(prev);
        newMap.set(pin, updated);
        return newMap;
      });
    } catch (err) {
      console.error('Failed to configure pin:', err);
      setError(`Failed to configure pin ${pin}`);
    }
  }, [currentNodeId]);

  const handlePinReserve = useCallback(async (pin: number, userId: string) => {
    if (!currentNodeId) return;

    try {
      await apiService.gpio.reservePin(currentNodeId, pin, userId);

      // Also notify via WebSocket
      reservePin(currentNodeId, pin, userId);

      // Update local state
      setGpioPins(prev => {
        const updated = new Map(prev);
        const existing = updated.get(pin);
        if (existing) {
          updated.set(pin, {
            ...existing,
            reserved: true,
            reservedBy: userId,
            reservedAt: new Date().toISOString(),
          });
        }
        return updated;
      });
    } catch (err) {
      console.error('Failed to reserve pin:', err);
      setError(`Failed to reserve pin ${pin}`);
    }
  }, [currentNodeId, reservePin]);

  const handlePinRelease = useCallback(async (pin: number) => {
    if (!currentNodeId) return;

    try {
      await apiService.gpio.releasePin(currentNodeId, pin);

      // Also notify via WebSocket
      releasePin(currentNodeId, pin);

      // Update local state
      setGpioPins(prev => {
        const updated = new Map(prev);
        const existing = updated.get(pin);
        if (existing) {
          updated.set(pin, {
            ...existing,
            reserved: false,
            reservedBy: undefined,
            reservedAt: undefined,
          });
        }
        return updated;
      });
    } catch (err) {
      console.error('Failed to release pin:', err);
      setError(`Failed to release pin ${pin}`);
    }
  }, [currentNodeId, releasePin]);

  const currentNode = nodes.find(n => n.id === currentNodeId);

  return (
    <div className="gpio-control-panel">
      <div className="gpio-panel-header">
        <div className="gpio-panel-title">
          <h2>GPIO Control Panel</h2>
          <ConnectionStatus
            connected={connected}
            connecting={connecting}
            error={wsError || error}
          />
        </div>

        <NodeSelector
          nodes={nodes}
          selectedNodeId={currentNodeId}
          onNodeChange={handleNodeChange}
          disabled={loading}
        />
      </div>

      {error && (
        <div className="gpio-error-banner">
          <span className="error-icon">⚠️</span>
          <span>{error}</span>
          <button
            className="error-dismiss"
            onClick={() => setError(null)}
          >
            ×
          </button>
        </div>
      )}

      {loading ? (
        <div className="gpio-loading">
          <div className="loading-spinner"></div>
          <p>Loading GPIO pins...</p>
        </div>
      ) : currentNode ? (
        <div className="gpio-panel-content">
          <div className="gpio-node-info">
            <h3>{currentNode.name} - {currentNode.ipAddress}</h3>
            <p>Architecture: {currentNode.architecture || 'Unknown'}</p>
            <p>Status: <span className={`status ${currentNode.status}`}>{currentNode.status}</span></p>
          </div>

          <GPIOPinGrid
            nodeId={currentNodeId}
            pins={Array.from(gpioPins.values())}
            onPinRead={handlePinRead}
            onPinWrite={handlePinWrite}
            onPinConfigure={handlePinConfigure}
            onPinReserve={handlePinReserve}
            onPinRelease={handlePinRelease}
            refreshing={refreshing}
          />
        </div>
      ) : (
        <div className="gpio-no-node">
          <p>No online nodes available for GPIO control</p>
        </div>
      )}
    </div>
  );
};

export default GPIOControlPanel;