---
layout: default
title: Configuration
nav_order: 10
---

# Pi Controller Configuration Guide
{: .no_toc }

Unified configuration system for all deployment modes
{: .fs-6 .fw-300 }

## Table of contents
{: .no_toc .text-delta }

1. TOC
{:toc}

---

This guide covers the unified configuration system supporting binary, Docker, and Kubernetes deployments.

## Configuration Precedence

1. CLI Flags (highest)
2. Environment Variables  
3. ConfigMap/CRD (K8s only)
4. YAML File
5. Defaults (lowest)

## Quick Start

### Binary Deployment
```bash
pi-controller -c config.yaml --webui-enabled --webui-port 3000
```

### Environment Variables
```bash
export PI_CONTROLLER_WEBUI_PORT=3000
export PI_CONTROLLER_WEBUI_BACKEND_API_URL=http://localhost:8080
```

### Kubernetes
```bash
kubectl apply -f config/crd/picontrollerwebui-crd.yaml
kubectl apply -f config/examples/picontrollerwebui-example.yaml
```

## Configuration Files

- **Example YAML**: `config/pi-controller.example.yaml`
- **CRD Definition**: `config/crd/picontrollerwebui-crd.yaml`  
- **Examples**: `config/examples/picontrollerwebui-example.yaml`

## Key WebUI Settings

- `webui.enabled`: Enable/disable WebUI server
- `webui.port`: WebUI server port (default: 3000)
- `webui.backend.api.url`: Backend API URL
- `webui.features.*`: Feature flag toggles
- `webui.branding.*`: UI customization

See example files for complete reference.
