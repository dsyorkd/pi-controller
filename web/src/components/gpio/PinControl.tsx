import React, { useState, useCallback } from 'react';
import type { GPIODevice } from '../../types';
import './PinControl.css';

interface PinControlProps {
  nodeId: string;
  device: GPIODevice;
  onRead: () => void;
  onWrite: (value: boolean) => void;
  onReserve: (userId: string) => void;
  onRelease: () => void;
  onConfigure: () => void;
  compact?: boolean;
}

export const PinControl: React.FC<PinControlProps> = ({
  device,
  onRead,
  onWrite,
  onReserve,
  onRelease,
  onConfigure,
  compact = false,
}) => {
  const [reserveUserId, setReserveUserId] = useState('');
  const [showReserveInput, setShowReserveInput] = useState(false);
  const [isOperating, setIsOperating] = useState(false);

  const handleRead = useCallback(async () => {
    if (device.reserved && device.reservedBy !== getCurrentUserId()) {
      alert('Pin is reserved by another user');
      return;
    }

    setIsOperating(true);
    try {
      await onRead();
    } finally {
      setIsOperating(false);
    }
  }, [device.reserved, device.reservedBy, onRead]);

  const handleWrite = useCallback(async (value: boolean) => {
    if (device.reserved && device.reservedBy !== getCurrentUserId()) {
      alert('Pin is reserved by another user');
      return;
    }

    if (device.direction !== 'output') {
      alert('Pin must be configured as output to write');
      return;
    }

    setIsOperating(true);
    try {
      await onWrite(value);
    } finally {
      setIsOperating(false);
    }
  }, [device.reserved, device.reservedBy, device.direction, onWrite]);

  const handleReserve = useCallback(() => {
    const userId = reserveUserId.trim() || getCurrentUserId();
    if (!userId) {
      alert('User ID is required for reservation');
      return;
    }

    onReserve(userId);
    setShowReserveInput(false);
    setReserveUserId('');
  }, [reserveUserId, onReserve]);

  const handleRelease = useCallback(() => {
    if (device.reservedBy && device.reservedBy !== getCurrentUserId()) {
      const confirmed = window.confirm(
        `This pin is reserved by ${device.reservedBy}. Force release?`
      );
      if (!confirmed) return;
    }

    onRelease();
  }, [device.reservedBy, onRelease]);

  const getCurrentUserId = () => {
    // In a real app, this would come from authentication context
    return localStorage.getItem('userId') || 'anonymous';
  };

  const isOwner = !device.reserved || device.reservedBy === getCurrentUserId();
  const canControl = isOwner;

  if (compact) {
    return (
      <div className={`pin-control compact ${device.reserved ? 'reserved' : ''}`}>
        <div className="pin-status-indicator">
          <div className={`status-dot ${device.direction} ${device.value ? 'high' : 'low'}`}></div>
          {device.reserved && (
            <div className="reserved-indicator" title={`Reserved by ${device.reservedBy}`}>
              🔒
            </div>
          )}
        </div>

        <div className="pin-direction">{device.direction}</div>

        {device.direction === 'output' && canControl && (
          <div className="pin-value-controls">
            <button
              className={`value-btn ${device.value ? 'active' : ''}`}
              onClick={() => handleWrite(true)}
              disabled={isOperating}
              title="Set HIGH"
            >
              H
            </button>
            <button
              className={`value-btn ${!device.value ? 'active' : ''}`}
              onClick={() => handleWrite(false)}
              disabled={isOperating}
              title="Set LOW"
            >
              L
            </button>
          </div>
        )}

        {device.direction === 'input' && canControl && (
          <button
            className="read-btn"
            onClick={handleRead}
            disabled={isOperating}
            title="Read pin"
          >
            📖
          </button>
        )}
      </div>
    );
  }

  return (
    <div className={`pin-control full ${device.reserved ? 'reserved' : ''}`}>
      <div className="pin-header">
        <div className="pin-info">
          <h4>GPIO {device.pin} - {device.name || 'Unnamed'}</h4>
          <div className="pin-metadata">
            <span className={`direction ${device.direction}`}>{device.direction.toUpperCase()}</span>
            <span className={`value ${device.value ? 'high' : 'low'}`}>
              {device.value ? 'HIGH' : 'LOW'}
            </span>
            {device.lastUpdated && (
              <span className="last-updated">
                Updated: {new Date(device.lastUpdated).toLocaleTimeString()}
              </span>
            )}
          </div>
        </div>

        <div className="pin-actions">
          <button
            className="config-btn"
            onClick={onConfigure}
            title="Configure pin"
          >
            ⚙️
          </button>
        </div>
      </div>

      {device.reserved && (
        <div className="reservation-info">
          <div className="reservation-details">
            <span className="reserved-icon">🔒</span>
            <span>Reserved by: <strong>{device.reservedBy}</strong></span>
            {device.reservedAt && (
              <span>Since: {new Date(device.reservedAt).toLocaleString()}</span>
            )}
          </div>
          <button
            className="release-btn"
            onClick={handleRelease}
            disabled={isOperating}
          >
            Release
          </button>
        </div>
      )}

      <div className="pin-controls">
        {device.direction === 'input' && (
          <div className="input-controls">
            <button
              className="read-btn primary"
              onClick={handleRead}
              disabled={!canControl || isOperating}
            >
              {isOperating ? 'Reading...' : 'Read Pin'}
            </button>
            <div className="value-display">
              <span className="value-label">Current Value:</span>
              <span className={`value-indicator ${device.value ? 'high' : 'low'}`}>
                {device.value ? 'HIGH (1)' : 'LOW (0)'}
              </span>
            </div>
          </div>
        )}

        {device.direction === 'output' && (
          <div className="output-controls">
            <div className="value-buttons">
              <button
                className={`value-btn high ${device.value ? 'active' : ''}`}
                onClick={() => handleWrite(true)}
                disabled={!canControl || isOperating}
              >
                Set HIGH
              </button>
              <button
                className={`value-btn low ${!device.value ? 'active' : ''}`}
                onClick={() => handleWrite(false)}
                disabled={!canControl || isOperating}
              >
                Set LOW
              </button>
            </div>
            <div className="value-display">
              <span className="value-label">Current Output:</span>
              <span className={`value-indicator ${device.value ? 'high' : 'low'}`}>
                {device.value ? 'HIGH (3.3V)' : 'LOW (0V)'}
              </span>
            </div>
          </div>
        )}

        {!device.reserved && (
          <div className="reservation-controls">
            {!showReserveInput ? (
              <button
                className="reserve-btn"
                onClick={() => setShowReserveInput(true)}
              >
                Reserve Pin
              </button>
            ) : (
              <div className="reserve-input-group">
                <input
                  type="text"
                  placeholder="User ID (optional)"
                  value={reserveUserId}
                  onChange={(e) => setReserveUserId(e.target.value)}
                  className="reserve-input"
                />
                <button
                  className="reserve-confirm-btn"
                  onClick={handleReserve}
                >
                  Confirm
                </button>
                <button
                  className="reserve-cancel-btn"
                  onClick={() => {
                    setShowReserveInput(false);
                    setReserveUserId('');
                  }}
                >
                  Cancel
                </button>
              </div>
            )}
          </div>
        )}
      </div>

      {device.description && (
        <div className="pin-description">
          <p>{device.description}</p>
        </div>
      )}

      {!canControl && (
        <div className="access-denied">
          <span className="warning-icon">⚠️</span>
          <span>Pin is reserved. Control disabled.</span>
        </div>
      )}
    </div>
  );
};

export default PinControl;