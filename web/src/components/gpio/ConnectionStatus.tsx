import React from 'react';
import './ConnectionStatus.css';

interface ConnectionStatusProps {
  connected: boolean;
  connecting: boolean;
  error: string | null;
}

export const ConnectionStatus: React.FC<ConnectionStatusProps> = ({
  connected,
  connecting,
  error,
}) => {
  const getStatusInfo = () => {
    if (error) {
      return {
        status: 'error',
        icon: '❌',
        text: `Connection Error: ${error}`,
        className: 'error',
      };
    }

    if (connecting) {
      return {
        status: 'connecting',
        icon: '🔄',
        text: 'Connecting to WebSocket...',
        className: 'connecting',
      };
    }

    if (connected) {
      return {
        status: 'connected',
        icon: '🟢',
        text: 'Real-time updates active',
        className: 'connected',
      };
    }

    return {
      status: 'disconnected',
      icon: '🔴',
      text: 'No real-time connection',
      className: 'disconnected',
    };
  };

  const statusInfo = getStatusInfo();

  return (
    <div className={`connection-status ${statusInfo.className}`}>
      <div className="status-indicator">
        <span
          className={`status-icon ${connecting ? 'spinning' : ''}`}
          role="img"
          aria-label={statusInfo.status}
        >
          {statusInfo.icon}
        </span>
        <span className="status-text">{statusInfo.text}</span>
      </div>

      {error && (
        <div className="error-details">
          <details>
            <summary>Error Details</summary>
            <pre>{error}</pre>
          </details>
        </div>
      )}

      {connected && (
        <div className="connection-features">
          <div className="feature-item">
            <span className="feature-icon">⚡</span>
            <span>Live pin updates</span>
          </div>
          <div className="feature-item">
            <span className="feature-icon">🔒</span>
            <span>Reservation sync</span>
          </div>
        </div>
      )}

      {!connected && !connecting && !error && (
        <div className="connection-warning">
          <p>
            <strong>Limited functionality:</strong> Pin states may not reflect real-time changes.
            Check network connection and WebSocket server availability.
          </p>
        </div>
      )}
    </div>
  );
};

export default ConnectionStatus;