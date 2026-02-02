---
layout: default
title: Quick Start Guide
nav_order: 2
description: Get from zero to a running Pi-Controller cluster in minutes
---

This guide will get you from zero to a running Pi-Controller cluster in minutes.

## Prerequisites

- **Hardware**: At least one Raspberry Pi (3, 4, 5, or Zero 2 W) running Raspberry Pi OS (64-bit recommended) or a Linux/macOS workstation for portable mode.
- **Network**: All devices must be on the same local network.

## Step 1: Installation

You can install Pi-Controller using our automated script.

### On a Raspberry Pi (Node)

This installs Pi-Controller as a system service.

```bash
curl -sSL https://pi-controller.io/install.sh | bash
sudo systemctl enable --now pi-controller
```

### On a Laptop/Workstation (Controller)

This installs the binary for remote management ("Portable Mode").

```bash
PORTABLE_MODE=true curl -sSL https://pi-controller.io/install.sh | bash
```

## Step 2: Form a Cluster

### Initialize the First Node (Bootstrap)

On your first Raspberry Pi, you need to tell it to start a new cluster.

1. Edit the configuration file:

    ```bash
    sudo nano /etc/pi-controller/config.yaml
    ```

2. Set `bootstrap: true` and ensure `node_id` is unique (e.g., `pi-1`):

    ```yaml
    cluster:
      enabled: true
      node_id: "pi-1"
      bootstrap: true
    ```

3. Restart the service:

    ```bash
    sudo systemctl restart pi-controller
    ```

### Join Additional Nodes

On your second Pi (e.g., `pi-2`), configure it to join the first one.

1. Edit `/etc/pi-controller/config.yaml`:

    ```yaml
    cluster:
      enabled: true
      node_id: "pi-2"
      bootstrap: false
      join_addresses:
        - "192.168.1.10:9091"  # Replace with IP of pi-1
    ```

2. Restart the service:

    ```bash
    sudo systemctl restart pi-controller
    ```

## Step 3: Verify the Cluster

From any node (or your laptop if configured), check the status:

```bash
pi-controller cluster status
```

You should see a list of members with one leader and followers.

## Step 4: Blink an LED (Hello World)

Let's test the hardware control. Connect an LED to GPIO Pin 17 on `pi-1`.

1. **Set the pin to Output/High**:

    ```bash
    pi-controller gpio set --node=pi-1 --pin=17 --value=high
    ```

    *The LED should turn ON.*

2. **Turn it Off**:

    ```bash
    pi-controller gpio set --node=pi-1 --pin=17 --value=low
    ```

    *The LED should turn OFF.*

## Step 5: Explore the UI

By default, the Web UI is available on port 8080 of any node.

Open `http://<node-ip>:8080` in your browser to view your cluster dashboard, manage nodes, and control pins visually.

## Next Steps

- **[Architecture](./architecture/index.html)**: Learn how it works under the hood.
- **[Configuration](./reference/configuration.html)**: Customize your setup.
- **[Kubernetes](./operations/installation.html#kubernetes-integration-optional)**: Deploy K3s on your cluster.
