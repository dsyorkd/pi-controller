import React, { useState, useEffect } from 'react';
import { useSearchParams, useNavigate } from 'react-router-dom';
import GPIOControlPanel from '../components/gpio/GPIOControlPanel';
import ReservationManager from '../components/gpio/ReservationManager';
import { apiService } from '../services/api';
import type { Node } from '../types';
import './GPIOPage.css';

export const GPIOPage: React.FC = () => {
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();

  const [selectedNodeId, setSelectedNodeId] = useState<string>(
    searchParams.get('nodeId') || ''
  );
  const [activeView, setActiveView] = useState<'control' | 'reservations'>(
    (searchParams.get('view') as 'control' | 'reservations') || 'control'
  );
  const [nodes, setNodes] = useState<Node[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Load available nodes on component mount
  useEffect(() => {
    const loadNodes = async () => {
      try {
        const response = await apiService.nodes.getAll();
        const onlineNodes = response.data.filter(node => node.status === 'online');
        setNodes(onlineNodes);

        // Set default node if none selected
        if (!selectedNodeId && onlineNodes.length > 0) {
          setSelectedNodeId(onlineNodes[0].id);
        }
      } catch (err) {
        console.error('Failed to load nodes:', err);
        setError('Failed to load available nodes');
      } finally {
        setLoading(false);
      }
    };

    loadNodes();
  }, [selectedNodeId]);

  // Update URL when node or view changes
  useEffect(() => {
    const params = new URLSearchParams();
    if (selectedNodeId) {
      params.set('nodeId', selectedNodeId);
    }
    if (activeView !== 'control') {
      params.set('view', activeView);
    }
    setSearchParams(params);
  }, [selectedNodeId, activeView, setSearchParams]);

  const handleNodeChange = (nodeId: string) => {
    setSelectedNodeId(nodeId);
  };

  const handleViewChange = (view: 'control' | 'reservations') => {
    setActiveView(view);
  };

  const handleBackToDashboard = () => {
    navigate('/dashboard');
  };

  if (loading) {
    return (
      <div className="gpio-page loading">
        <div className="loading-spinner-large"></div>
        <p>Loading GPIO interface...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="gpio-page error">
        <div className="error-container">
          <div className="error-icon">⚠️</div>
          <h2>GPIO Interface Error</h2>
          <p>{error}</p>
          <div className="error-actions">
            <button
              className="btn-primary"
              onClick={() => window.location.reload()}
            >
              Retry
            </button>
            <button
              className="btn-secondary"
              onClick={handleBackToDashboard}
            >
              Back to Dashboard
            </button>
          </div>
        </div>
      </div>
    );
  }

  if (nodes.length === 0) {
    return (
      <div className="gpio-page no-nodes">
        <div className="no-nodes-container">
          <div className="no-nodes-icon">🔌</div>
          <h2>No GPIO Nodes Available</h2>
          <p>
            No online nodes with GPIO capabilities are currently available.
            Please ensure your Raspberry Pi nodes are connected and running.
          </p>
          <div className="no-nodes-actions">
            <button
              className="btn-primary"
              onClick={() => window.location.reload()}
            >
              Refresh
            </button>
            <button
              className="btn-secondary"
              onClick={handleBackToDashboard}
            >
              Back to Dashboard
            </button>
          </div>
        </div>
      </div>
    );
  }

  const selectedNode = nodes.find(n => n.id === selectedNodeId);

  return (
    <div className="gpio-page">
      <div className="gpio-page-header">
        <div className="page-title">
          <button
            className="back-button"
            onClick={handleBackToDashboard}
            aria-label="Back to Dashboard"
          >
            ←
          </button>
          <div className="title-content">
            <h1>GPIO Control Center</h1>
            {selectedNode && (
              <p className="page-subtitle">
                Managing {selectedNode.name} ({selectedNode.ipAddress})
              </p>
            )}
          </div>
        </div>

        <div className="page-actions">
          <div className="view-tabs">
            <button
              className={`tab-button ${activeView === 'control' ? 'active' : ''}`}
              onClick={() => handleViewChange('control')}
            >
              <span className="tab-icon">🎛️</span>
              <span>Pin Control</span>
            </button>
            <button
              className={`tab-button ${activeView === 'reservations' ? 'active' : ''}`}
              onClick={() => handleViewChange('reservations')}
            >
              <span className="tab-icon">🔒</span>
              <span>Reservations</span>
            </button>
          </div>
        </div>
      </div>

      <div className="gpio-page-content">
        {activeView === 'control' && (
          <div className="control-view">
            <GPIOControlPanel
              selectedNodeId={selectedNodeId}
              onNodeChange={handleNodeChange}
            />
          </div>
        )}

        {activeView === 'reservations' && (
          <div className="reservations-view">
            <ReservationManager
              nodeId={selectedNodeId}
              nodes={nodes}
              onNodeChange={handleNodeChange}
            />
          </div>
        )}
      </div>

      <div className="gpio-page-footer">
        <div className="footer-info">
          <div className="safety-reminders">
            <div className="reminder-item">
              <span className="reminder-icon">⚠️</span>
              <span>Always verify pin configurations before connecting hardware</span>
            </div>
            <div className="reminder-item">
              <span className="reminder-icon">⚡</span>
              <span>GPIO pins output 3.3V - do not connect 5V devices directly</span>
            </div>
            <div className="reminder-item">
              <span className="reminder-icon">🔒</span>
              <span>Reserve pins when performing critical operations</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

export default GPIOPage;