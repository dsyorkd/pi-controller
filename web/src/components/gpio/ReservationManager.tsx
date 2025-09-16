import React, { useState, useEffect, useCallback } from 'react';
import { apiService } from '../../services/api';
import { useGPIOWebSocket } from '../../hooks/useWebSocket';
import NodeSelector from './NodeSelector';
import type { Node, GPIODevice, GPIOReservation } from '../../types';
import './ReservationManager.css';

interface ReservationManagerProps {
  nodeId: string;
  nodes: Node[];
  onNodeChange: (nodeId: string) => void;
}

interface ReservationWithPin extends GPIOReservation {
  pinName?: string;
  pinDirection?: 'input' | 'output';
}

export const ReservationManager: React.FC<ReservationManagerProps> = ({
  nodeId,
  nodes,
  onNodeChange,
}) => {
  const [reservations, setReservations] = useState<ReservationWithPin[]>([]);
  const [gpioPins, setGpioPins] = useState<GPIODevice[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [filter, setFilter] = useState<'all' | 'my' | 'others'>('all');
  const [newReservation, setNewReservation] = useState({
    pin: '',
    userId: '',
    duration: 60, // minutes
    showForm: false,
  });

  const currentUserId = localStorage.getItem('userId') || 'anonymous';

  // WebSocket for real-time reservation updates
  const { connected, subscribeToReservations } = useGPIOWebSocket({
    onReservationChange: useCallback((reservation: any) => {
      if (reservation.nodeId === nodeId) {
        loadReservations();
      }
    }, [nodeId]),
  });

  // Load reservations and pin data
  const loadReservations = useCallback(async () => {
    if (!nodeId) return;

    setLoading(true);
    setError(null);

    try {
      const [reservationsData, pinsData] = await Promise.all([
        apiService.gpio.getReservations(nodeId),
        apiService.nodes.getGPIO(nodeId),
      ]);

      // Enhance reservations with pin information
      const enhancedReservations = reservationsData.map(reservation => {
        const pin = pinsData.find(p => p.pin === reservation.pin);
        return {
          ...reservation,
          pinName: pin?.name,
          pinDirection: pin?.direction,
        };
      });

      setReservations(enhancedReservations);
      setGpioPins(pinsData);
    } catch (err) {
      console.error('Failed to load reservations:', err);
      setError('Failed to load reservation data');
    } finally {
      setLoading(false);
    }
  }, [nodeId]);

  // Load data when node changes
  useEffect(() => {
    loadReservations();
  }, [loadReservations]);

  // Subscribe to WebSocket updates
  useEffect(() => {
    if (!nodeId || !connected) return;

    const unsubscribe = subscribeToReservations(nodeId);
    return unsubscribe;
  }, [nodeId, connected, subscribeToReservations]);

  // Filter reservations based on current filter
  const filteredReservations = reservations.filter(reservation => {
    switch (filter) {
      case 'my':
        return reservation.reservedBy === currentUserId;
      case 'others':
        return reservation.reservedBy !== currentUserId;
      case 'all':
      default:
        return true;
    }
  });

  const availablePins = gpioPins.filter(pin => !pin.reserved);

  const handleReservePin = async () => {
    if (!newReservation.pin || !newReservation.userId.trim()) {
      setError('Pin and User ID are required');
      return;
    }

    try {
      await apiService.gpio.reservePin(
        nodeId,
        parseInt(newReservation.pin),
        newReservation.userId.trim(),
        newReservation.duration * 60 // Convert minutes to seconds
      );

      setNewReservation({
        pin: '',
        userId: '',
        duration: 60,
        showForm: false,
      });

      loadReservations();
    } catch (err) {
      console.error('Failed to reserve pin:', err);
      setError('Failed to reserve pin');
    }
  };

  const handleReleaseReservation = async (pin: number) => {
    const reservation = reservations.find(r => r.pin === pin);

    if (reservation && reservation.reservedBy !== currentUserId) {
      const confirmed = window.confirm(
        `This pin is reserved by ${reservation.reservedBy}. Force release?`
      );
      if (!confirmed) return;
    }

    try {
      await apiService.gpio.releasePin(nodeId, pin);
      loadReservations();
    } catch (err) {
      console.error('Failed to release reservation:', err);
      setError('Failed to release reservation');
    }
  };

  const formatDuration = (reservedAt: string, expires?: string) => {
    const start = new Date(reservedAt);
    const now = new Date();
    const elapsed = Math.floor((now.getTime() - start.getTime()) / 1000 / 60);

    if (expires) {
      const expiry = new Date(expires);
      const remaining = Math.floor((expiry.getTime() - now.getTime()) / 1000 / 60);

      if (remaining <= 0) {
        return { text: 'Expired', className: 'expired' };
      } else if (remaining <= 10) {
        return { text: `${remaining}m remaining`, className: 'expiring' };
      } else {
        return { text: `${remaining}m remaining`, className: 'active' };
      }
    }

    return { text: `${elapsed}m ago`, className: 'permanent' };
  };

  return (
    <div className="reservation-manager">
      <div className="reservation-header">
        <div className="header-title">
          <h2>Pin Reservations</h2>
          <p>Manage GPIO pin access control</p>
        </div>

        <NodeSelector
          nodes={nodes}
          selectedNodeId={nodeId}
          onNodeChange={onNodeChange}
          disabled={loading}
        />
      </div>

      {error && (
        <div className="error-banner">
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

      <div className="reservation-controls">
        <div className="filter-controls">
          <label>Show:</label>
          <select
            value={filter}
            onChange={(e) => setFilter(e.target.value as any)}
            className="filter-select"
          >
            <option value="all">All Reservations</option>
            <option value="my">My Reservations</option>
            <option value="others">Others' Reservations</option>
          </select>
        </div>

        <div className="action-controls">
          <button
            className="btn-primary"
            onClick={() => setNewReservation(prev => ({ ...prev, showForm: !prev.showForm }))}
            disabled={availablePins.length === 0}
          >
            {newReservation.showForm ? 'Cancel' : 'Reserve Pin'}
          </button>
          <button
            className="btn-secondary"
            onClick={loadReservations}
            disabled={loading}
          >
            {loading ? 'Refreshing...' : 'Refresh'}
          </button>
        </div>
      </div>

      {newReservation.showForm && (
        <div className="new-reservation-form">
          <h3>Reserve a Pin</h3>
          <div className="form-row">
            <div className="form-group">
              <label>Pin:</label>
              <select
                value={newReservation.pin}
                onChange={(e) => setNewReservation(prev => ({ ...prev, pin: e.target.value }))}
                className="form-select"
              >
                <option value="">Select a pin...</option>
                {availablePins.map(pin => (
                  <option key={pin.pin} value={pin.pin}>
                    GPIO {pin.pin} - {pin.name || 'Unnamed'} ({pin.direction})
                  </option>
                ))}
              </select>
            </div>

            <div className="form-group">
              <label>User ID:</label>
              <input
                type="text"
                value={newReservation.userId}
                onChange={(e) => setNewReservation(prev => ({ ...prev, userId: e.target.value }))}
                placeholder={currentUserId}
                className="form-input"
              />
            </div>

            <div className="form-group">
              <label>Duration (minutes):</label>
              <input
                type="number"
                value={newReservation.duration}
                onChange={(e) => setNewReservation(prev => ({ ...prev, duration: parseInt(e.target.value) || 60 }))}
                min="1"
                max="1440"
                className="form-input"
              />
            </div>

            <button
              className="btn-primary"
              onClick={handleReservePin}
            >
              Reserve
            </button>
          </div>
        </div>
      )}

      <div className="reservations-list">
        {loading ? (
          <div className="loading-state">
            <div className="loading-spinner"></div>
            <p>Loading reservations...</p>
          </div>
        ) : filteredReservations.length === 0 ? (
          <div className="empty-state">
            <div className="empty-icon">🔓</div>
            <h3>No Reservations</h3>
            <p>
              {filter === 'all'
                ? 'No pins are currently reserved on this node.'
                : filter === 'my'
                ? 'You have no active reservations on this node.'
                : 'No other users have reservations on this node.'
              }
            </p>
          </div>
        ) : (
          <div className="reservations-grid">
            {filteredReservations.map(reservation => {
              const duration = formatDuration(reservation.reservedAt, reservation.expires);
              const isOwner = reservation.reservedBy === currentUserId;

              return (
                <div
                  key={`${reservation.nodeId}-${reservation.pin}`}
                  className={`reservation-card ${duration.className} ${isOwner ? 'owned' : ''}`}
                >
                  <div className="reservation-header">
                    <div className="pin-info">
                      <span className="pin-number">GPIO {reservation.pin}</span>
                      {reservation.pinName && (
                        <span className="pin-name">{reservation.pinName}</span>
                      )}
                    </div>
                    <div className="reservation-status">
                      <span className={`status-badge ${duration.className}`}>
                        {duration.text}
                      </span>
                    </div>
                  </div>

                  <div className="reservation-details">
                    <div className="detail-row">
                      <span className="detail-label">Reserved by:</span>
                      <span className={`detail-value ${isOwner ? 'owner' : ''}`}>
                        {reservation.reservedBy}
                        {isOwner && <span className="owner-indicator"> (You)</span>}
                      </span>
                    </div>

                    <div className="detail-row">
                      <span className="detail-label">Since:</span>
                      <span className="detail-value">
                        {new Date(reservation.reservedAt).toLocaleString()}
                      </span>
                    </div>

                    {reservation.expires && (
                      <div className="detail-row">
                        <span className="detail-label">Expires:</span>
                        <span className="detail-value">
                          {new Date(reservation.expires).toLocaleString()}
                        </span>
                      </div>
                    )}

                    {reservation.pinDirection && (
                      <div className="detail-row">
                        <span className="detail-label">Direction:</span>
                        <span className={`detail-value direction ${reservation.pinDirection}`}>
                          {reservation.pinDirection.toUpperCase()}
                        </span>
                      </div>
                    )}
                  </div>

                  <div className="reservation-actions">
                    <button
                      className={`release-btn ${isOwner ? 'primary' : 'danger'}`}
                      onClick={() => handleReleaseReservation(reservation.pin)}
                    >
                      {isOwner ? 'Release' : 'Force Release'}
                    </button>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      <div className="reservations-summary">
        <div className="summary-stats">
          <div className="stat-item">
            <span className="stat-value">{reservations.length}</span>
            <span className="stat-label">Total Reserved</span>
          </div>
          <div className="stat-item">
            <span className="stat-value">{reservations.filter(r => r.reservedBy === currentUserId).length}</span>
            <span className="stat-label">Your Reservations</span>
          </div>
          <div className="stat-item">
            <span className="stat-value">{availablePins.length}</span>
            <span className="stat-label">Available Pins</span>
          </div>
          <div className="stat-item">
            <span className="stat-value">{gpioPins.length}</span>
            <span className="stat-label">Total Configured</span>
          </div>
        </div>
      </div>
    </div>
  );
};

export default ReservationManager;