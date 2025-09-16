import React, { useState, useMemo } from 'react';
import type { GPIODevice } from '../../types';
import PinControl from './PinControl';
import PinConfigModal from './PinConfigModal';
import './GPIOPinGrid.css';

interface GPIOPinGridProps {
  nodeId: string;
  pins: GPIODevice[];
  onPinRead: (pin: number) => void;
  onPinWrite: (pin: number, value: boolean) => void;
  onPinConfigure: (pin: number, direction: 'input' | 'output', name?: string) => void;
  onPinReserve: (pin: number, userId: string) => void;
  onPinRelease: (pin: number) => void;
  refreshing: boolean;
}

// Raspberry Pi 4 GPIO pin layout (40 pins)
const RASPBERRY_PI_PINOUT: PinLayout[] = [
  { pin: 1, type: 'power', label: '3.3V', gpio: null },
  { pin: 2, type: 'power', label: '5V', gpio: null },
  { pin: 3, type: 'gpio', label: 'GPIO2', gpio: 2 },
  { pin: 4, type: 'power', label: '5V', gpio: null },
  { pin: 5, type: 'gpio', label: 'GPIO3', gpio: 3 },
  { pin: 6, type: 'ground', label: 'GND', gpio: null },
  { pin: 7, type: 'gpio', label: 'GPIO4', gpio: 4 },
  { pin: 8, type: 'gpio', label: 'GPIO14', gpio: 14 },
  { pin: 9, type: 'ground', label: 'GND', gpio: null },
  { pin: 10, type: 'gpio', label: 'GPIO15', gpio: 15 },
  { pin: 11, type: 'gpio', label: 'GPIO17', gpio: 17 },
  { pin: 12, type: 'gpio', label: 'GPIO18', gpio: 18 },
  { pin: 13, type: 'gpio', label: 'GPIO27', gpio: 27 },
  { pin: 14, type: 'ground', label: 'GND', gpio: null },
  { pin: 15, type: 'gpio', label: 'GPIO22', gpio: 22 },
  { pin: 16, type: 'gpio', label: 'GPIO23', gpio: 23 },
  { pin: 17, type: 'power', label: '3.3V', gpio: null },
  { pin: 18, type: 'gpio', label: 'GPIO24', gpio: 24 },
  { pin: 19, type: 'gpio', label: 'GPIO10', gpio: 10 },
  { pin: 20, type: 'ground', label: 'GND', gpio: null },
  { pin: 21, type: 'gpio', label: 'GPIO9', gpio: 9 },
  { pin: 22, type: 'gpio', label: 'GPIO25', gpio: 25 },
  { pin: 23, type: 'gpio', label: 'GPIO11', gpio: 11 },
  { pin: 24, type: 'gpio', label: 'GPIO8', gpio: 8 },
  { pin: 25, type: 'ground', label: 'GND', gpio: null },
  { pin: 26, type: 'gpio', label: 'GPIO7', gpio: 7 },
  { pin: 27, type: 'gpio', label: 'GPIO0', gpio: 0 },
  { pin: 28, type: 'gpio', label: 'GPIO1', gpio: 1 },
  { pin: 29, type: 'gpio', label: 'GPIO5', gpio: 5 },
  { pin: 30, type: 'ground', label: 'GND', gpio: null },
  { pin: 31, type: 'gpio', label: 'GPIO6', gpio: 6 },
  { pin: 32, type: 'gpio', label: 'GPIO12', gpio: 12 },
  { pin: 33, type: 'gpio', label: 'GPIO13', gpio: 13 },
  { pin: 34, type: 'ground', label: 'GND', gpio: null },
  { pin: 35, type: 'gpio', label: 'GPIO19', gpio: 19 },
  { pin: 36, type: 'gpio', label: 'GPIO16', gpio: 16 },
  { pin: 37, type: 'gpio', label: 'GPIO26', gpio: 26 },
  { pin: 38, type: 'gpio', label: 'GPIO20', gpio: 20 },
  { pin: 39, type: 'ground', label: 'GND', gpio: null },
  { pin: 40, type: 'gpio', label: 'GPIO21', gpio: 21 },
];

interface PinLayout {
  pin: number;
  type: 'gpio' | 'power' | 'ground';
  label: string;
  gpio: number | null;
}

export const GPIOPinGrid: React.FC<GPIOPinGridProps> = ({
  nodeId,
  pins,
  onPinRead,
  onPinWrite,
  onPinConfigure,
  onPinReserve,
  onPinRelease,
  refreshing,
}) => {
  const [selectedPin, setSelectedPin] = useState<number | null>(null);
  const [configModalOpen, setConfigModalOpen] = useState(false);
  const [filter, setFilter] = useState<'all' | 'configured' | 'available' | 'reserved'>('all');

  // Create a map of GPIO pin numbers to device data
  const pinDeviceMap = useMemo(() => {
    const map = new Map<number, GPIODevice>();
    pins.forEach(pin => {
      map.set(pin.pin, pin);
    });
    return map;
  }, [pins]);

  // Filter pins based on current filter
  const filteredPinout = useMemo(() => {
    return RASPBERRY_PI_PINOUT.filter(pinInfo => {
      if (pinInfo.type !== 'gpio') {
        return filter === 'all'; // Only show non-GPIO pins in 'all' view
      }

      const device = pinInfo.gpio ? pinDeviceMap.get(pinInfo.gpio) : null;

      switch (filter) {
        case 'configured':
          return device !== undefined;
        case 'available':
          return !device || !device.reserved;
        case 'reserved':
          return device?.reserved === true;
        case 'all':
        default:
          return true;
      }
    });
  }, [filter, pinDeviceMap]);

  const handlePinClick = (pinInfo: PinLayout) => {
    if (pinInfo.type !== 'gpio' || pinInfo.gpio === null) return;

    setSelectedPin(pinInfo.gpio);
    const device = pinDeviceMap.get(pinInfo.gpio);

    if (!device) {
      // Pin not configured, open configuration modal
      setConfigModalOpen(true);
    }
  };

  const handlePinConfigure = (pin: number, direction: 'input' | 'output', name?: string) => {
    onPinConfigure(pin, direction, name);
    setConfigModalOpen(false);
    setSelectedPin(null);
  };

  const renderPin = (pinInfo: PinLayout, index: number) => {
    const device = pinInfo.gpio ? pinDeviceMap.get(pinInfo.gpio) : null;
    const isLeftColumn = index % 2 === 0;

    if (pinInfo.type !== 'gpio') {
      return (
        <div
          key={`${pinInfo.pin}-${pinInfo.type}`}
          className={`pin-layout ${pinInfo.type} ${isLeftColumn ? 'left' : 'right'}`}
        >
          <div className="pin-number">{pinInfo.pin}</div>
          <div className="pin-label">{pinInfo.label}</div>
        </div>
      );
    }

    return (
      <div
        key={`${pinInfo.pin}-gpio`}
        className={`pin-layout gpio ${isLeftColumn ? 'left' : 'right'} ${!device ? 'unconfigured' : ''}`}
        onClick={() => handlePinClick(pinInfo)}
      >
        <div className="pin-number">{pinInfo.pin}</div>
        <div className="pin-label">{pinInfo.label}</div>

        {device && (
          <PinControl
            nodeId={nodeId}
            device={device}
            onRead={() => onPinRead(device.pin)}
            onWrite={(value) => onPinWrite(device.pin, value)}
            onReserve={(userId) => onPinReserve(device.pin, userId)}
            onRelease={() => onPinRelease(device.pin)}
            onConfigure={() => {
              setSelectedPin(device.pin);
              setConfigModalOpen(true);
            }}
            compact={true}
          />
        )}

        {!device && (
          <div className="pin-unconfigured">
            <span>Click to configure</span>
          </div>
        )}
      </div>
    );
  };

  // Split pins into two columns for the visual layout
  const leftColumnPins = filteredPinout.filter((_, index) => index % 2 === 0);
  const rightColumnPins = filteredPinout.filter((_, index) => index % 2 === 1);

  return (
    <div className="gpio-pin-grid">
      <div className="gpio-grid-header">
        <h3>GPIO Pin Layout</h3>
        <div className="gpio-grid-filters">
          <label>Filter:</label>
          <select
            value={filter}
            onChange={(e) => setFilter(e.target.value as any)}
            className="filter-select"
          >
            <option value="all">All Pins</option>
            <option value="configured">Configured</option>
            <option value="available">Available</option>
            <option value="reserved">Reserved</option>
          </select>
        </div>
      </div>

      <div className="gpio-grid-legend">
        <div className="legend-item">
          <div className="legend-color power"></div>
          <span>Power</span>
        </div>
        <div className="legend-item">
          <div className="legend-color ground"></div>
          <span>Ground</span>
        </div>
        <div className="legend-item">
          <div className="legend-color gpio"></div>
          <span>GPIO</span>
        </div>
        <div className="legend-item">
          <div className="legend-color reserved"></div>
          <span>Reserved</span>
        </div>
      </div>

      <div className="gpio-grid-container">
        <div className="gpio-grid-board">
          <div className="gpio-column left">
            {leftColumnPins.map((pinInfo, index) => renderPin(pinInfo, index * 2))}
          </div>
          <div className="gpio-column right">
            {rightColumnPins.map((pinInfo, index) => renderPin(pinInfo, index * 2 + 1))}
          </div>
        </div>
      </div>

      <div className="gpio-grid-stats">
        <div className="stat-item">
          <span className="stat-value">{pins.length}</span>
          <span className="stat-label">Configured</span>
        </div>
        <div className="stat-item">
          <span className="stat-value">{pins.filter(p => p.reserved).length}</span>
          <span className="stat-label">Reserved</span>
        </div>
        <div className="stat-item">
          <span className="stat-value">{pins.filter(p => p.direction === 'input').length}</span>
          <span className="stat-label">Inputs</span>
        </div>
        <div className="stat-item">
          <span className="stat-value">{pins.filter(p => p.direction === 'output').length}</span>
          <span className="stat-label">Outputs</span>
        </div>
      </div>

      {configModalOpen && selectedPin !== null && (
        <PinConfigModal
          nodeId={nodeId}
          pin={selectedPin}
          currentDevice={pinDeviceMap.get(selectedPin)}
          onConfigure={handlePinConfigure}
          onClose={() => {
            setConfigModalOpen(false);
            setSelectedPin(null);
          }}
        />
      )}

      {refreshing && (
        <div className="gpio-refreshing-overlay">
          <div className="loading-spinner"></div>
          <span>Refreshing pin states...</span>
        </div>
      )}
    </div>
  );
};

export default GPIOPinGrid;