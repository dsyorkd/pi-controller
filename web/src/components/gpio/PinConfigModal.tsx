import React, { useState, useEffect } from 'react';
import type { GPIODevice } from '../../types';
import './PinConfigModal.css';

interface PinConfigModalProps {
  nodeId: string;
  pin: number;
  currentDevice?: GPIODevice;
  onConfigure: (pin: number, direction: 'input' | 'output', name?: string) => void;
  onClose: () => void;
}

export const PinConfigModal: React.FC<PinConfigModalProps> = ({
  nodeId,
  pin,
  currentDevice,
  onConfigure,
  onClose,
}) => {
  const [direction, setDirection] = useState<'input' | 'output'>(
    currentDevice?.direction || 'input'
  );
  const [name, setName] = useState(currentDevice?.name || '');
  const [description, setDescription] = useState(currentDevice?.description || '');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [errors, setErrors] = useState<Record<string, string>>({});

  useEffect(() => {
    if (currentDevice) {
      setDirection(currentDevice.direction);
      setName(currentDevice.name || '');
      setDescription(currentDevice.description || '');
    }
  }, [currentDevice]);

  const validateForm = () => {
    const newErrors: Record<string, string> = {};

    if (name.trim().length === 0) {
      newErrors.name = 'Pin name is required';
    } else if (name.trim().length > 50) {
      newErrors.name = 'Pin name must be 50 characters or less';
    }

    if (description.length > 200) {
      newErrors.description = 'Description must be 200 characters or less';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();

    if (!validateForm()) {
      return;
    }

    setIsSubmitting(true);

    try {
      onConfigure(pin, direction, name.trim() || undefined);
    } catch (error) {
      console.error('Failed to configure pin:', error);
      setErrors({ submit: 'Failed to configure pin. Please try again.' });
    } finally {
      setIsSubmitting(false);
    }
  };

  const handleOverlayClick = (e: React.MouseEvent) => {
    if (e.target === e.currentTarget) {
      onClose();
    }
  };

  const isExistingPin = !!currentDevice;
  const title = isExistingPin ? `Reconfigure GPIO ${pin}` : `Configure GPIO ${pin}`;

  return (
    <div className="pin-config-modal-overlay" onClick={handleOverlayClick}>
      <div className="pin-config-modal">
        <div className="modal-header">
          <h3>{title}</h3>
          <button
            className="modal-close-btn"
            onClick={onClose}
            aria-label="Close modal"
          >
            ×
          </button>
        </div>

        <div className="modal-content">
          <div className="pin-info">
            <div className="pin-details">
              <span className="pin-number">Pin {pin}</span>
              <span className="node-id">Node: {nodeId}</span>
            </div>
            {isExistingPin && (
              <div className="current-status">
                <span className={`status-indicator ${currentDevice.reserved ? 'reserved' : 'available'}`}>
                  {currentDevice.reserved ? 'Reserved' : 'Available'}
                </span>
                {currentDevice.reservedBy && (
                  <span className="reserved-by">by {currentDevice.reservedBy}</span>
                )}
              </div>
            )}
          </div>

          <form onSubmit={handleSubmit} className="config-form">
            <div className="form-group">
              <label htmlFor="pin-name" className="form-label">
                Pin Name *
              </label>
              <input
                id="pin-name"
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                placeholder="e.g., LED Control, Sensor Input"
                className={`form-input ${errors.name ? 'error' : ''}`}
                disabled={isSubmitting}
                maxLength={50}
              />
              {errors.name && (
                <span className="form-error">{errors.name}</span>
              )}
              <div className="form-help">
                Give this pin a descriptive name for easy identification
              </div>
            </div>

            <div className="form-group">
              <label className="form-label">
                Direction *
              </label>
              <div className="direction-options">
                <label className="radio-option">
                  <input
                    type="radio"
                    name="direction"
                    value="input"
                    checked={direction === 'input'}
                    onChange={(e) => setDirection(e.target.value as 'input')}
                    disabled={isSubmitting}
                  />
                  <div className="radio-label">
                    <span className="radio-title">Input</span>
                    <span className="radio-description">
                      Read values from sensors, buttons, etc.
                    </span>
                  </div>
                </label>

                <label className="radio-option">
                  <input
                    type="radio"
                    name="direction"
                    value="output"
                    checked={direction === 'output'}
                    onChange={(e) => setDirection(e.target.value as 'output')}
                    disabled={isSubmitting}
                  />
                  <div className="radio-label">
                    <span className="radio-title">Output</span>
                    <span className="radio-description">
                      Control LEDs, relays, motors, etc.
                    </span>
                  </div>
                </label>
              </div>
            </div>

            <div className="form-group">
              <label htmlFor="pin-description" className="form-label">
                Description (Optional)
              </label>
              <textarea
                id="pin-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                placeholder="Additional details about this pin's purpose or configuration..."
                className={`form-textarea ${errors.description ? 'error' : ''}`}
                disabled={isSubmitting}
                rows={3}
                maxLength={200}
              />
              {errors.description && (
                <span className="form-error">{errors.description}</span>
              )}
              <div className="form-help">
                {description.length}/200 characters
              </div>
            </div>

            {errors.submit && (
              <div className="form-error-banner">
                <span className="error-icon">⚠️</span>
                <span>{errors.submit}</span>
              </div>
            )}

            <div className="form-actions">
              <button
                type="button"
                className="btn-secondary"
                onClick={onClose}
                disabled={isSubmitting}
              >
                Cancel
              </button>
              <button
                type="submit"
                className="btn-primary"
                disabled={isSubmitting}
              >
                {isSubmitting ? 'Configuring...' : (isExistingPin ? 'Update Pin' : 'Configure Pin')}
              </button>
            </div>
          </form>

          <div className="configuration-warnings">
            <div className="warning-item">
              <span className="warning-icon">⚠️</span>
              <span>
                Changing pin configuration may affect connected hardware.
                Ensure no critical devices are connected before proceeding.
              </span>
            </div>
            {direction === 'output' && (
              <div className="warning-item">
                <span className="warning-icon">⚡</span>
                <span>
                  Output pins can source up to 16mA. Do not exceed this limit
                  to avoid damaging the GPIO controller.
                </span>
              </div>
            )}
          </div>
        </div>
      </div>
    </div>
  );
};

export default PinConfigModal;