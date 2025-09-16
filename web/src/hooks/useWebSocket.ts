import { useEffect, useRef, useState, useCallback } from 'react';
import { io, Socket } from 'socket.io-client';
import type { WebSocketMessage, GPIOPinState } from '../types';

interface UseWebSocketOptions {
  url?: string;
  autoConnect?: boolean;
  reconnectAttempts?: number;
  reconnectDelay?: number;
}

interface UseWebSocketReturn {
  socket: Socket | null;
  connected: boolean;
  connecting: boolean;
  error: string | null;
  connect: () => void;
  disconnect: () => void;
  emit: (event: string, data?: any) => void;
  subscribe: (event: string, callback: (data: any) => void) => () => void;
}

export const useWebSocket = (options: UseWebSocketOptions = {}): UseWebSocketReturn => {
  const {
    url = import.meta.env.VITE_WS_URL || 'http://localhost:8080',
    autoConnect = true,
    reconnectAttempts = 5,
    reconnectDelay = 1000,
  } = options;

  const [connected, setConnected] = useState(false);
  const [connecting, setConnecting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const socketRef = useRef<Socket | null>(null);
  const reconnectCountRef = useRef(0);
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  const connect = useCallback(() => {
    if (socketRef.current?.connected) return;

    setConnecting(true);
    setError(null);

    const socket = io(url, {
      transports: ['websocket'],
      timeout: 10000,
      auth: {
        token: localStorage.getItem('authToken'),
      },
    });

    socket.on('connect', () => {
      console.log('WebSocket connected');
      setConnected(true);
      setConnecting(false);
      setError(null);
      reconnectCountRef.current = 0;
    });

    socket.on('disconnect', (reason) => {
      console.log('WebSocket disconnected:', reason);
      setConnected(false);
      setConnecting(false);

      // Auto-reconnect if not manually disconnected
      if (reason !== 'io client disconnect' && reconnectCountRef.current < reconnectAttempts) {
        const delay = reconnectDelay * Math.pow(2, reconnectCountRef.current);
        reconnectTimeoutRef.current = setTimeout(() => {
          reconnectCountRef.current++;
          connect();
        }, delay);
      }
    });

    socket.on('connect_error', (err) => {
      console.error('WebSocket connection error:', err);
      setConnecting(false);
      setError(err.message || 'Connection failed');

      // Attempt reconnect with exponential backoff
      if (reconnectCountRef.current < reconnectAttempts) {
        const delay = reconnectDelay * Math.pow(2, reconnectCountRef.current);
        reconnectTimeoutRef.current = setTimeout(() => {
          reconnectCountRef.current++;
          connect();
        }, delay);
      }
    });

    socketRef.current = socket;
  }, [url, reconnectAttempts, reconnectDelay]);

  const disconnect = useCallback(() => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }

    if (socketRef.current) {
      socketRef.current.disconnect();
      socketRef.current = null;
    }

    setConnected(false);
    setConnecting(false);
    setError(null);
    reconnectCountRef.current = 0;
  }, []);

  const emit = useCallback((event: string, data?: any) => {
    if (socketRef.current?.connected) {
      socketRef.current.emit(event, data);
    } else {
      console.warn('Socket not connected, cannot emit event:', event);
    }
  }, []);

  const subscribe = useCallback((event: string, callback: (data: any) => void) => {
    if (!socketRef.current) {
      console.warn('Socket not initialized, cannot subscribe to event:', event);
      return () => {};
    }

    socketRef.current.on(event, callback);

    return () => {
      if (socketRef.current) {
        socketRef.current.off(event, callback);
      }
    };
  }, []);

  useEffect(() => {
    if (autoConnect) {
      connect();
    }

    return () => {
      disconnect();
    };
  }, [autoConnect, connect, disconnect]);

  return {
    socket: socketRef.current,
    connected,
    connecting,
    error,
    connect,
    disconnect,
    emit,
    subscribe,
  };
};

// Specialized hook for GPIO WebSocket updates
interface UseGPIOWebSocketOptions extends UseWebSocketOptions {
  onPinUpdate?: (pinState: GPIOPinState) => void;
  onReservationChange?: (reservation: any) => void;
}

interface UseGPIOWebSocketReturn extends UseWebSocketReturn {
  subscribeToPinUpdates: (nodeId: string) => () => void;
  subscribeToReservations: (nodeId: string) => () => void;
  requestPinState: (nodeId: string, pin: number) => void;
  reservePin: (nodeId: string, pin: number, userId: string) => void;
  releasePin: (nodeId: string, pin: number) => void;
}

export const useGPIOWebSocket = (options: UseGPIOWebSocketOptions = {}): UseGPIOWebSocketReturn => {
  const { onPinUpdate, onReservationChange, ...wsOptions } = options;
  const webSocket = useWebSocket(wsOptions);

  const subscribeToPinUpdates = useCallback((nodeId: string) => {
    if (!webSocket.connected) {
      console.warn('WebSocket not connected, cannot subscribe to pin updates');
      return () => {};
    }

    return webSocket.subscribe('gpio_pin_update', (data: WebSocketMessage) => {
      try {
        if (data.type === 'gpio_update' && data.payload.nodeId === nodeId) {
          onPinUpdate?.(data.payload as GPIOPinState);
        }
      } catch (error) {
        console.error('Error processing GPIO pin update:', error);
      }
    });
  }, [webSocket.subscribe, webSocket.connected, onPinUpdate]);

  const subscribeToReservations = useCallback((nodeId: string) => {
    if (!webSocket.connected) {
      console.warn('WebSocket not connected, cannot subscribe to reservations');
      return () => {};
    }

    return webSocket.subscribe('gpio_reservation_change', (data: WebSocketMessage) => {
      try {
        if (data.type === 'gpio_reservation' && data.payload.nodeId === nodeId) {
          onReservationChange?.(data.payload);
        }
      } catch (error) {
        console.error('Error processing GPIO reservation change:', error);
      }
    });
  }, [webSocket.subscribe, webSocket.connected, onReservationChange]);

  const requestPinState = useCallback((nodeId: string, pin: number) => {
    if (!webSocket.connected) {
      console.warn('WebSocket not connected, cannot request pin state');
      return;
    }

    try {
      webSocket.emit('request_pin_state', { nodeId, pin });
    } catch (error) {
      console.error('Error requesting pin state:', error);
    }
  }, [webSocket.emit, webSocket.connected]);

  const reservePin = useCallback((nodeId: string, pin: number, userId: string) => {
    if (!webSocket.connected) {
      console.warn('WebSocket not connected, cannot reserve pin');
      return;
    }

    try {
      webSocket.emit('reserve_pin', { nodeId, pin, userId });
    } catch (error) {
      console.error('Error reserving pin:', error);
    }
  }, [webSocket.emit, webSocket.connected]);

  const releasePin = useCallback((nodeId: string, pin: number) => {
    if (!webSocket.connected) {
      console.warn('WebSocket not connected, cannot release pin');
      return;
    }

    try {
      webSocket.emit('release_pin', { nodeId, pin });
    } catch (error) {
      console.error('Error releasing pin:', error);
    }
  }, [webSocket.emit, webSocket.connected]);

  return {
    ...webSocket,
    subscribeToPinUpdates,
    subscribeToReservations,
    requestPinState,
    reservePin,
    releasePin,
  };
};