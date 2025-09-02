# System Architecture Overview

Pi-Controller is a comprehensive Kubernetes management platform designed specifically for Raspberry Pi clusters. The system provides automated discovery, provisioning, and lifecycle management of K3s clusters while offering GPIO-as-a-Service capabilities through Kubernetes Custom Resources.

## High-Level Diagram

This diagram shows the main actors and components of the Pi-Controller ecosystem.

```mermaid
graph TD
    subgraph User
        direction LR
        A[Web UI]
        B[CLI]
        C[Kubernetes API]
    end

    subgraph "Pi-Controller Control Plane"
        direction LR
        CP[Single Go Binary]
        DB[(SQLite)]
        CP -- Manages --> K8s
        CP -- Stores State --> DB
    end

    subgraph "Raspberry Pi Cluster"
        K8s[K3s Kubernetes Cluster]
        subgraph "Master Nodes"
            M1[Pi Master 1]
            M2[Pi Master 2]
        end
        subgraph "Worker Nodes"
            W1[Pi Worker 1]
            W2[Pi Worker 2]
            Wn[Pi Worker N]
        end
        M1 & M2 -- K3s Server --> K8s
        W1 & W2 & Wn -- K3s Agent --> K8s
    end

    subgraph "Pi-Controller Node Agents"
        A1[Agent on M1]
        A2[Agent on M2]
        AW1[Agent on W1]
        AW2[Agent on W2]
        AWn[Agent on Wn]
    end

    User -- Interacts with --> CP
    C -- Interacts with --> K8s

    CP -- Deploys/Manages --> K8s
    CP -- Communicates with --> A1 & A2 & AW1 & AW2 & AWn

    A1 -- Runs on --> M1
    A2 -- Runs on --> M2
    AW1 -- Runs on --> W1
    AW2 -- Runs on --> W2
    AWn -- Runs on --> Wn

    A1 & A2 & AW1 & AW2 & AWn -- Control Hardware --> RPi[Raspberry Pi Hardware/GPIO]

    style CP fill:#cde4f7,stroke:#367d91,stroke-width:2px
    style DB fill:#f9f,stroke:#333,stroke-width:2px
    style K8s fill:#d5e8d4,stroke:#82b366,stroke-width:2px
```

## Core Philosophy
-   **Single Binary Deployment**: The control plane runs as one Go binary with an embedded web UI for simplicity.
-   **Zero Dependencies**: No external automation tools like Ansible or Terraform are required.
-   **Pi-Native**: Optimized for the hardware constraints of ARM64/ARMv7 devices.
-   **Kubernetes-First**: All hardware control, including GPIO, is managed through standard Kubernetes APIs and Custom Resources.
-   **Homelab-Friendly**: Designed for simple deployment while being scalable enough for enterprise use cases.