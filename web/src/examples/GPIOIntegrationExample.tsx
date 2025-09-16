/**
 * GPIO Control Panel Integration Example
 *
 * This file demonstrates how to integrate all GPIO components into a React application.
 * It shows:
 * 1. How to set up routing for the GPIO page
 * 2. How to integrate with the dashboard navigation
 * 3. How to use the components individually
 * 4. Best practices for error handling and state management
 */

import React from 'react';
import { BrowserRouter as Router, Routes, Route, Link } from 'react-router-dom';
import GPIOPage from '../pages/GPIOPage';
import GPIOControlPanel from '../components/gpio/GPIOControlPanel';
import ReservationManager from '../components/gpio/ReservationManager';
import PinControl from '../components/gpio/PinControl';
import { useGPIOWebSocket } from '../hooks/useWebSocket';
import type { Node, GPIODevice } from '../types';

// Example 1: Basic routing setup
export const GPIORoutingExample: React.FC = () => {
  return (
    <Router>
      <div className="app">
        <nav className="app-nav">
          <Link to="/dashboard">Dashboard</Link>
          <Link to="/gpio">GPIO Control</Link>
          <Link to="/nodes">Nodes</Link>
        </nav>

        <main className="app-main">
          <Routes>
            <Route path="/gpio" element={<GPIOPage />} />
            <Route path="/gpio/:nodeId" element={<GPIOPage />} />
            {/* Other routes */}
          </Routes>
        </main>
      </div>
    </Router>
  );
};

// Example 2: Standalone GPIO Control Panel
export const StandaloneGPIOPanel: React.FC = () => {
  const [selectedNodeId, setSelectedNodeId] = React.useState<string>('');

  return (
    <div className="standalone-gpio">
      <h1>GPIO Control</h1>
      <GPIOControlPanel
        selectedNodeId={selectedNodeId}
        onNodeChange={setSelectedNodeId}
      />
    </div>
  );
};

// Example 3: Individual component usage with custom WebSocket handling
export const CustomGPIOComponents: React.FC = () => {
  const [nodes] = React.useState<Node[]>([
    {
      id: 'node-1',
      name: 'Raspberry Pi 4',
      ipAddress: '192.168.1.100',
      macAddress: 'DC:A6:32:XX:XX:XX',
      status: 'online',
      role: 'worker',
      lastSeen: new Date().toISOString(),
    }
  ]);

  const [gpioPins] = React.useState<GPIODevice[]>([
    {
      id: 'gpio-1',
      nodeId: 'node-1',
      pin: 18,
      direction: 'output',
      value: false,
      name: 'LED Control',
      description: 'Controls status LED',
      reserved: false,
    },
    {
      id: 'gpio-2',
      nodeId: 'node-1',
      pin: 24,
      direction: 'input',
      value: true,
      name: 'Button Input',
      description: 'Push button sensor',
      reserved: true,
      reservedBy: 'user@example.com',
      reservedAt: new Date().toISOString(),
    }
  ]);

  const { connected, connecting, error } = useGPIOWebSocket({
    onPinUpdate: (pinState) => {
      console.log('Pin state updated:', pinState);
      // Handle pin state updates
    },
    onReservationChange: (reservation) => {
      console.log('Reservation changed:', reservation);
      // Handle reservation changes
    },
  });

  const handlePinRead = async (pin: number) => {
    console.log(`Reading pin ${pin}`);
    // Implement pin read logic
  };

  const handlePinWrite = async (pin: number, value: boolean) => {
    console.log(`Writing pin ${pin} to ${value}`);
    // Implement pin write logic
  };

  const handlePinReserve = async (pin: number, userId: string) => {
    console.log(`Reserving pin ${pin} for ${userId}`);
    // Implement pin reservation logic
  };

  const handlePinRelease = async (pin: number) => {
    console.log(`Releasing pin ${pin}`);
    // Implement pin release logic
  };

  const handlePinConfigure = async (pin: number, direction: 'input' | 'output', name?: string) => {
    console.log(`Configuring pin ${pin} as ${direction}${name ? ` (${name})` : ''}`);
    // Implement pin configuration logic
  };

  return (
    <div className="custom-gpio-components">
      <div className="connection-status">
        <h2>WebSocket Status</h2>
        <p>
          Status: {connecting ? 'Connecting...' : connected ? 'Connected' : 'Disconnected'}
          {error && <span> - Error: {error}</span>}
        </p>
      </div>

      <div className="individual-pins">
        <h2>Individual Pin Controls</h2>
        {gpioPins.map(pin => (
          <PinControl
            key={pin.id}
            nodeId={pin.nodeId}
            device={pin}
            onRead={() => handlePinRead(pin.pin)}
            onWrite={(value) => handlePinWrite(pin.pin, value)}
            onReserve={(userId) => handlePinReserve(pin.pin, userId)}
            onRelease={() => handlePinRelease(pin.pin)}
            onConfigure={() => handlePinConfigure(pin.pin, pin.direction, pin.name)}
            compact={false}
          />
        ))}
      </div>

      <div className="reservation-management">
        <h2>Reservation Management</h2>
        <ReservationManager
          nodeId="node-1"
          nodes={nodes}
          onNodeChange={(nodeId) => console.log('Node changed to:', nodeId)}
        />
      </div>
    </div>
  );
};

// Example 4: Dashboard integration pattern
export const DashboardWithGPIO: React.FC = () => {
  const [activeView, setActiveView] = React.useState<'overview' | 'gpio' | 'nodes'>('overview');

  const renderContent = () => {
    switch (activeView) {
      case 'gpio':
        return <GPIOPage />;
      case 'nodes':
        return <div>Nodes management content...</div>;
      case 'overview':
      default:
        return (
          <div className="dashboard-overview">
            <h2>System Overview</h2>
            <div className="quick-actions">
              <button
                className="quick-action-btn"
                onClick={() => setActiveView('gpio')}
              >
                <span className="icon">🎛️</span>
                <span>GPIO Control</span>
                <span className="description">Manage GPIO pins and reservations</span>
              </button>

              <button
                className="quick-action-btn"
                onClick={() => setActiveView('nodes')}
              >
                <span className="icon">🖥️</span>
                <span>Node Management</span>
                <span className="description">Monitor and manage cluster nodes</span>
              </button>
            </div>
          </div>
        );
    }
  };

  return (
    <div className="dashboard">
      <nav className="dashboard-nav">
        <button
          className={`nav-btn ${activeView === 'overview' ? 'active' : ''}`}
          onClick={() => setActiveView('overview')}
        >
          Overview
        </button>
        <button
          className={`nav-btn ${activeView === 'gpio' ? 'active' : ''}`}
          onClick={() => setActiveView('gpio')}
        >
          GPIO
        </button>
        <button
          className={`nav-btn ${activeView === 'nodes' ? 'active' : ''}`}
          onClick={() => setActiveView('nodes')}
        >
          Nodes
        </button>
      </nav>

      <main className="dashboard-content">
        {renderContent()}
      </main>
    </div>
  );
};

// Example 5: Error handling and loading states
export const GPIOWithErrorHandling: React.FC = () => {
  const [loading, setLoading] = React.useState(true);
  const [error, setError] = React.useState<string | null>(null);
  const [selectedNodeId, setSelectedNodeId] = React.useState<string>('');

  React.useEffect(() => {
    // Simulate loading and potential errors
    const timer = setTimeout(() => {
      try {
        // Simulate successful load
        setSelectedNodeId('node-1');
        setLoading(false);
      } catch (err) {
        setError('Failed to load GPIO components');
        setLoading(false);
      }
    }, 2000);

    return () => clearTimeout(timer);
  }, []);

  if (loading) {
    return (
      <div className="gpio-loading">
        <div className="loading-spinner"></div>
        <p>Loading GPIO interface...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="gpio-error">
        <h2>Error Loading GPIO Interface</h2>
        <p>{error}</p>
        <button onClick={() => window.location.reload()}>
          Retry
        </button>
      </div>
    );
  }

  return (
    <GPIOControlPanel
      selectedNodeId={selectedNodeId}
      onNodeChange={setSelectedNodeId}
    />
  );
};

// Example styles for the examples (you would put this in a separate CSS file)
const exampleStyles = `
.app {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}

.app-nav {
  display: flex;
  gap: 1rem;
  padding: 1rem;
  background-color: #f8fafc;
  border-bottom: 1px solid #e5e7eb;
}

.app-nav a {
  padding: 0.5rem 1rem;
  text-decoration: none;
  color: #374151;
  border-radius: 6px;
  transition: background-color 0.2s;
}

.app-nav a:hover {
  background-color: #e5e7eb;
}

.app-main {
  flex: 1;
  padding: 2rem;
}

.dashboard {
  display: flex;
  flex-direction: column;
  height: 100vh;
}

.dashboard-nav {
  display: flex;
  gap: 0.5rem;
  padding: 1rem;
  background-color: #ffffff;
  border-bottom: 1px solid #e5e7eb;
}

.nav-btn {
  padding: 0.75rem 1.5rem;
  border: 1px solid #d1d5db;
  border-radius: 6px;
  background-color: #ffffff;
  color: #374151;
  cursor: pointer;
  transition: all 0.2s;
}

.nav-btn.active {
  background-color: #3b82f6;
  border-color: #3b82f6;
  color: white;
}

.dashboard-content {
  flex: 1;
  padding: 2rem;
}

.quick-actions {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
  gap: 1rem;
  margin-top: 2rem;
}

.quick-action-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 0.5rem;
  padding: 2rem;
  border: 1px solid #e5e7eb;
  border-radius: 8px;
  background-color: #ffffff;
  cursor: pointer;
  transition: all 0.2s;
  text-align: center;
}

.quick-action-btn:hover {
  border-color: #3b82f6;
  box-shadow: 0 4px 6px rgba(0, 0, 0, 0.05);
}

.quick-action-btn .icon {
  font-size: 2rem;
}

.quick-action-btn .description {
  font-size: 0.875rem;
  color: #6b7280;
}

.custom-gpio-components {
  display: flex;
  flex-direction: column;
  gap: 2rem;
  max-width: 1200px;
  margin: 0 auto;
}

.individual-pins {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.gpio-loading,
.gpio-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 400px;
  text-align: center;
}

.loading-spinner {
  width: 40px;
  height: 40px;
  border: 3px solid #e5e7eb;
  border-top: 3px solid #3b82f6;
  border-radius: 50%;
  animation: spin 1s linear infinite;
  margin-bottom: 1rem;
}

@keyframes spin {
  0% { transform: rotate(0deg); }
  100% { transform: rotate(360deg); }
}
`;

export default {
  GPIORoutingExample,
  StandaloneGPIOPanel,
  CustomGPIOComponents,
  DashboardWithGPIO,
  GPIOWithErrorHandling,
  exampleStyles,
};