# Pi Controller Custom Resource Definitions (CRDs)

This directory contains the Kubernetes Custom Resource Definitions (CRDs) for the Pi Controller project, enabling GPIO-as-a-Service through Kubernetes native resources.

## Overview

The Pi Controller project extends Kubernetes with custom resources for managing Raspberry Pi hardware:

- **GPIOPin**: Manages individual GPIO pins (input, output, PWM modes)
- **PWMController**: Manages multiple PWM channels with synchronized control
- **I2CDevice**: Manages I2C devices with register-level configuration

## Installation

### Apply CRDs to Kubernetes Cluster

```bash
# Apply all CRDs
kubectl apply -k config/crd/

# Or apply individually
kubectl apply -f config/crd/gpiopin-crd.yaml
kubectl apply -f config/crd/pwmcontroller-crd.yaml
kubectl apply -f config/crd/i2cdevice-crd.yaml
```

### Verify Installation

```bash
# Check if CRDs are installed
kubectl get crd | grep pi-controller

# Expected output:
# gpiopins.gpio.pi-controller.io          2023-XX-XX
# pwmcontrollers.gpio.pi-controller.io    2023-XX-XX
# i2cdevices.gpio.pi-controller.io        2023-XX-XX
```

## GPIOPin Resource

Manages individual GPIO pins on Raspberry Pi nodes.

### Supported Modes

- **input**: Read digital values from pins
- **output**: Control digital pin states
- **pwm**: Generate PWM signals

### Key Features

- Node selection via labels
- Pin mode configuration (input/output/PWM)
- Pull-up/pull-down resistor configuration
- Debouncing for input pins
- PWM frequency and duty cycle control
- Real-time status reporting

### Example Usage

```bash
# Create a simple LED controller
kubectl apply -f config/examples/gpiopin-examples.yaml

# List GPIO pins
kubectl get gpiopins

# Get detailed status
kubectl describe gpiopin led-gpio18

# Monitor pin status
kubectl get gpiopin led-gpio18 -w
```

## PWMController Resource

Manages multiple PWM channels with coordinated control, ideal for robotics and lighting applications.

### Key Features

- Multi-channel PWM control
- Phase offset support for synchronized motors
- Global clock and range settings
- Per-channel enable/disable control
- Comprehensive status reporting

### Example Usage

```bash
# Create RGB LED controller
kubectl apply -f config/examples/pwmcontroller-examples.yaml

# List PWM controllers
kubectl get pwmcontrollers

# Check controller status
kubectl describe pwmcontroller rgb-led-controller
```

## I2CDevice Resource

Manages I2C devices with register-level configuration and monitoring.

### Supported Device Types

- **sensor**: Temperature, humidity, pressure sensors
- **display**: OLED, LCD displays
- **actuator**: Motor drivers, relays
- **gpio-expander**: I/O expansion chips
- **adc/dac**: Analog-to-digital/digital-to-analog converters
- **rtc**: Real-time clocks
- **eeprom**: Memory devices
- **generic**: General-purpose devices

### Key Features

- Register-level configuration
- Automatic polling of read-only registers
- Data type conversion (uint8, uint16, float32, etc.)
- Device discovery and identification
- Connection monitoring and error reporting

### Example Usage

```bash
# Deploy temperature sensor
kubectl apply -f config/examples/i2cdevice-examples.yaml

# List I2C devices
kubectl get i2cdevices

# Monitor sensor readings
kubectl describe i2cdevice sht30-temp-humidity

# Watch device status
kubectl get i2cdevice sht30-temp-humidity -w
```

## Node Selection

All resources support flexible node selection using Kubernetes label selectors:

```yaml
# Select by hostname
nodeSelector:
  kubernetes.io/hostname: pi-node-01

# Select by architecture
nodeSelector:
  kubernetes.io/arch: arm64

# Select by custom labels
nodeSelector:
  zone: outdoor
  application: robotics
```

## Status and Monitoring

Each resource provides comprehensive status information:

### Common Status Fields

- **phase**: Current lifecycle phase (Pending, Configuring, Ready, Failed)
- **nodeId**: ID of the managing Pi Controller node
- **lastUpdated**: Timestamp of last status update
- **conditions**: Detailed condition information
- **message**: Human-readable status message

### Monitoring Commands

```bash
# Watch all GPIO pins
kubectl get gpiopins -w

# Monitor specific resource
kubectl describe gpiopin my-sensor-pin

# View resource events
kubectl get events --field-selector involvedObject.name=my-device

# Check resource status across namespaces
kubectl get gpiopins -A
```

## Namespacing and Organization

Resources can be organized across namespaces for better management:

```bash
# Create namespaces for different applications
kubectl create namespace sensors
kubectl create namespace automation
kubectl create namespace robotics

# Deploy resources to specific namespaces
kubectl apply -f sensor-config.yaml -n sensors
kubectl apply -f automation-config.yaml -n automation
```

## Troubleshooting

### Common Issues

1. **CRD Not Recognized**
   ```bash
   # Ensure CRDs are properly installed
   kubectl get crd | grep pi-controller
   ```

2. **Resource Stuck in Pending**
   ```bash
   # Check if Pi Controller is running on target node
   kubectl get pods -l app=pi-controller
   
   # Verify node labels match selectors
   kubectl get nodes --show-labels
   ```

3. **I2C Device Connection Issues**
   ```bash
   # Check device status and error messages
   kubectl describe i2cdevice device-name
   
   # Verify I2C bus availability on node
   kubectl exec -it pi-agent-pod -- i2cdetect -y 1
   ```

### Debug Commands

```bash
# View Pi Controller logs
kubectl logs -l app=pi-controller -f

# Check Pi Agent status
kubectl get pods -l app=pi-agent

# Examine resource definitions
kubectl explain gpiopin.spec
kubectl explain pwmcontroller.status
kubectl explain i2cdevice.spec.registers
```

## Integration with Existing Systems

These CRDs integrate seamlessly with standard Kubernetes tooling:

- **GitOps**: Use ArgoCD, Flux for declarative hardware management
- **Monitoring**: Prometheus metrics from Pi Controller
- **Alerting**: AlertManager rules for hardware failures
- **Automation**: Use with Kubernetes Jobs, CronJobs
- **Service Mesh**: Istio integration for secure communication

## Security Considerations

- Resources are namespaced for multi-tenant isolation
- RBAC controls access to hardware resources
- Node selection prevents cross-node resource conflicts
- TLS encryption for all Pi Controller communication

## Next Steps

1. Deploy the Pi Controller operator (Task 24)
2. Integrate with Kubernetes client-go (Task 25)  
3. Set up monitoring and alerting
4. Implement GitOps workflows for hardware management

For more information, see the main Pi Controller documentation.