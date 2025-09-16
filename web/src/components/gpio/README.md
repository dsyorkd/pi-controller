# GPIO Control Panel Components

This directory contains a comprehensive set of React components for managing GPIO pins on Raspberry Pi nodes through the pi-controller web interface.

## Components Overview

### Core Components

#### 1. `GPIOControlPanel`
**Main container component that orchestrates all GPIO functionality.**

```tsx
import GPIOControlPanel from './components/gpio/GPIOControlPanel';

<GPIOControlPanel
  selectedNodeId="node-1"
  onNodeChange={(nodeId) => setSelectedNodeId(nodeId)}
/>
```

**Features:**
- Node selection and management
- Real-time WebSocket integration
- GPIO pin loading and state management
- Error handling and loading states
- Automatic pin state synchronization

#### 2. `GPIOPinGrid`
**Visual representation of the Raspberry Pi GPIO pin layout.**

```tsx
import GPIOPinGrid from './components/gpio/GPIOPinGrid';

<GPIOPinGrid
  nodeId="node-1"
  pins={gpioPins}
  onPinRead={(pin) => handlePinRead(pin)}
  onPinWrite={(pin, value) => handlePinWrite(pin, value)}
  onPinConfigure={(pin, direction, name) => handleConfigure(pin, direction, name)}
  onPinReserve={(pin, userId) => handleReserve(pin, userId)}
  onPinRelease={(pin) => handleRelease(pin)}
  refreshing={false}
/>
```

**Features:**
- Authentic Raspberry Pi 40-pin layout
- Color-coded pin types (GPIO, Power, Ground)
- Interactive pin controls
- Filter options (all, configured, available, reserved)
- Visual status indicators
- Pin configuration modal integration

#### 3. `PinControl`
**Individual pin control widget with full functionality.**

```tsx
import PinControl from './components/gpio/PinControl';

<PinControl
  nodeId="node-1"
  device={gpioDevice}
  onRead={() => handleRead()}
  onWrite={(value) => handleWrite(value)}
  onReserve={(userId) => handleReserve(userId)}
  onRelease={() => handleRelease()}
  onConfigure={() => handleConfigure()}
  compact={false} // Set to true for grid view
/>
```

**Features:**
- Compact and full view modes
- Input/Output value controls
- Real-time status indicators
- Pin reservation management
- Safety warnings and validations
- User permission checks

#### 4. `ReservationManager`
**Centralized pin reservation management interface.**

```tsx
import ReservationManager from './components/gpio/ReservationManager';

<ReservationManager
  nodeId="node-1"
  nodes={availableNodes}
  onNodeChange={(nodeId) => setSelectedNode(nodeId)}
/>
```

**Features:**
- View all pin reservations
- Filter by user (my, others, all)
- Create new reservations with duration
- Release reservations (with force option)
- Real-time reservation updates
- Reservation statistics

#### 5. `PinConfigModal`
**Modal dialog for configuring GPIO pin settings.**

```tsx
import PinConfigModal from './components/gpio/PinConfigModal';

<PinConfigModal
  nodeId="node-1"
  pin={18}
  currentDevice={existingDevice} // Optional for reconfiguration
  onConfigure={(pin, direction, name) => handleConfigure(pin, direction, name)}
  onClose={() => setModalOpen(false)}
/>
```

**Features:**
- Pin direction configuration (Input/Output)
- Custom pin naming
- Description and metadata
- Form validation
- Safety warnings
- Existing configuration editing

### Utility Components

#### 6. `NodeSelector`
**Dropdown component for selecting target nodes.**

```tsx
import NodeSelector from './components/gpio/NodeSelector';

<NodeSelector
  nodes={availableNodes}
  selectedNodeId="node-1"
  onNodeChange={(nodeId) => setSelectedNode(nodeId)}
  disabled={false}
/>
```

#### 7. `ConnectionStatus`
**WebSocket connection status indicator.**

```tsx
import ConnectionStatus from './components/gpio/ConnectionStatus';

<ConnectionStatus
  connected={wsConnected}
  connecting={wsConnecting}
  error={wsError}
/>
```

### Page Component

#### 8. `GPIOPage`
**Complete page implementation with navigation and view switching.**

```tsx
import GPIOPage from './pages/GPIOPage';

// Use in router
<Route path="/gpio" element={<GPIOPage />} />
<Route path="/gpio/:nodeId" element={<GPIOPage />} />
```

**Features:**
- Full-page GPIO interface
- Tab navigation (Pin Control / Reservations)
- URL parameter support for node selection
- Error boundaries and loading states
- Safety reminders footer

## WebSocket Integration

### useGPIOWebSocket Hook

```tsx
import { useGPIOWebSocket } from './hooks/useWebSocket';

const {
  connected,
  connecting,
  error,
  subscribeToPinUpdates,
  subscribeToReservations,
  requestPinState,
  reservePin,
  releasePin,
} = useGPIOWebSocket({
  onPinUpdate: (pinState) => {
    // Handle real-time pin state updates
    console.log('Pin updated:', pinState);
  },
  onReservationChange: (reservation) => {
    // Handle reservation changes
    console.log('Reservation changed:', reservation);
  },
});
```

**WebSocket Events:**
- `gpio_pin_update` - Real-time pin value changes
- `gpio_reservation_change` - Pin reservation updates
- `request_pin_state` - Request fresh pin state
- `reserve_pin` - Reserve a pin
- `release_pin` - Release a pin reservation

## API Integration

### GPIO API Services

The components integrate with the following API endpoints:

```tsx
import { apiService } from './services/api';

// Node operations
const nodes = await apiService.nodes.getAll();
const gpioPins = await apiService.nodes.getGPIO(nodeId);

// Pin operations
await apiService.gpio.readPin(nodeId, pin);
await apiService.gpio.writePin(nodeId, pin, value);
await apiService.gpio.configurePin(nodeId, pin, direction, name);

// Reservation operations
await apiService.gpio.reservePin(nodeId, pin, userId, duration);
await apiService.gpio.releasePin(nodeId, pin);
const reservations = await apiService.gpio.getReservations(nodeId);
```

## Styling and Theming

All components include comprehensive CSS with:
- **Responsive Design** - Mobile-first approach
- **Dark Mode Support** - CSS custom properties
- **Accessibility** - ARIA labels and keyboard navigation
- **Animations** - Smooth transitions and loading states
- **Visual Hierarchy** - Clear status indicators and grouping

### CSS Custom Properties

```css
:root {
  --color-bg-primary: #ffffff;
  --color-bg-secondary: #f9fafb;
  --color-text-primary: #111827;
  --color-text-secondary: #6b7280;
  --color-border: #d1d5db;
  --color-primary: #3b82f6;
  --color-success: #10b981;
  --color-warning: #f59e0b;
  --color-error: #ef4444;
}
```

## Usage Examples

### Basic Integration

```tsx
import React from 'react';
import GPIOControlPanel from './components/gpio/GPIOControlPanel';

export const BasicGPIOExample = () => {
  const [selectedNodeId, setSelectedNodeId] = React.useState('');

  return (
    <div className="app">
      <h1>GPIO Control Center</h1>
      <GPIOControlPanel
        selectedNodeId={selectedNodeId}
        onNodeChange={setSelectedNodeId}
      />
    </div>
  );
};
```

### Router Integration

```tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import GPIOPage from './pages/GPIOPage';

export const App = () => (
  <BrowserRouter>
    <Routes>
      <Route path="/gpio" element={<GPIOPage />} />
      <Route path="/gpio/:nodeId" element={<GPIOPage />} />
    </Routes>
  </BrowserRouter>
);
```

### Custom WebSocket Handling

```tsx
import { useGPIOWebSocket } from './hooks/useWebSocket';

export const CustomGPIOComponent = () => {
  const { connected, subscribeToPinUpdates } = useGPIOWebSocket({
    onPinUpdate: (pinState) => {
      // Custom pin update handling
      updateLocalState(pinState);
    },
  });

  React.useEffect(() => {
    if (connected) {
      const unsubscribe = subscribeToPinUpdates('node-1');
      return unsubscribe;
    }
  }, [connected, subscribeToPinUpdates]);

  // Component render...
};
```

## Best Practices

### 1. Error Handling
- Always wrap GPIO operations in try-catch blocks
- Provide user-friendly error messages
- Implement retry mechanisms for failed operations

### 2. State Management
- Use the WebSocket hooks for real-time updates
- Implement optimistic updates for better UX
- Handle connection states gracefully

### 3. Security
- Validate user permissions before pin operations
- Respect pin reservations
- Sanitize user inputs

### 4. Performance
- Use React.memo for expensive components
- Implement virtualization for large pin lists
- Debounce frequent operations

### 5. Accessibility
- Provide keyboard navigation
- Include ARIA labels
- Ensure color contrast compliance
- Support screen readers

## Troubleshooting

### Common Issues

**WebSocket Connection Fails**
- Check VITE_WS_URL environment variable
- Verify WebSocket server is running
- Check authentication tokens

**Pin Operations Timeout**
- Verify node connectivity
- Check GPIO service status
- Validate pin permissions

**Components Not Rendering**
- Ensure all CSS files are imported
- Check for TypeScript errors
- Verify API service configuration

### Debug Mode

Enable debug logging:

```tsx
// Set in environment or localStorage
localStorage.setItem('gpio-debug', 'true');
```

## Environment Variables

```env
# WebSocket connection
VITE_WS_URL=ws://localhost:8080

# API base URL
VITE_API_BASE_URL=http://localhost:8080/api/v1

# Debug mode
VITE_GPIO_DEBUG=false
```

## Browser Support

- **Chrome/Edge**: Full support
- **Firefox**: Full support
- **Safari**: Full support (iOS 14+)
- **Mobile**: Responsive design optimized

## Contributing

When contributing to GPIO components:

1. Follow existing TypeScript patterns
2. Include comprehensive tests
3. Update CSS for new features
4. Document new props and methods
5. Test WebSocket integration
6. Verify accessibility compliance

## License

Part of the pi-controller project. See main project LICENSE file.