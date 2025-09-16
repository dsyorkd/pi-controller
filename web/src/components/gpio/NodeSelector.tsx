import React from 'react';
import type { Node } from '../../types';
import './NodeSelector.css';

interface NodeSelectorProps {
  nodes: Node[];
  selectedNodeId: string;
  onNodeChange: (nodeId: string) => void;
  disabled?: boolean;
}

export const NodeSelector: React.FC<NodeSelectorProps> = ({
  nodes,
  selectedNodeId,
  onNodeChange,
  disabled = false,
}) => {
  const selectedNode = nodes.find(n => n.id === selectedNodeId);

  return (
    <div className="node-selector">
      <label htmlFor="node-select" className="node-selector-label">
        Select Node:
      </label>
      <div className="node-selector-container">
        <select
          id="node-select"
          value={selectedNodeId}
          onChange={(e) => onNodeChange(e.target.value)}
          disabled={disabled}
          className="node-select"
        >
          {nodes.length === 0 ? (
            <option value="">No nodes available</option>
          ) : (
            nodes.map(node => (
              <option key={node.id} value={node.id}>
                {node.name} ({node.ipAddress}) - {node.status}
              </option>
            ))
          )}
        </select>

        {selectedNode && (
          <div className="selected-node-info">
            <div className="node-status-indicator">
              <span className={`status-dot ${selectedNode.status}`}></span>
              <span className="node-role">{selectedNode.role}</span>
            </div>

            <div className="node-details">
              {selectedNode.architecture && (
                <span className="node-arch">{selectedNode.architecture}</span>
              )}
              {selectedNode.model && (
                <span className="node-model">{selectedNode.model}</span>
              )}
              {selectedNode.cpuCores && (
                <span className="node-cpu">{selectedNode.cpuCores} cores</span>
              )}
              {selectedNode.memory && (
                <span className="node-memory">{Math.round(selectedNode.memory / 1024 / 1024)}MB</span>
              )}
            </div>

            <div className="node-connectivity">
              <span className="last-seen">
                Last seen: {new Date(selectedNode.lastSeen).toLocaleString()}
              </span>
            </div>
          </div>
        )}
      </div>

      {nodes.length === 0 && (
        <div className="no-nodes-message">
          <span className="warning-icon">⚠️</span>
          <span>No online nodes available for GPIO control</span>
        </div>
      )}
    </div>
  );
};

export default NodeSelector;