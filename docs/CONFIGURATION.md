# Pi-Controller Configuration Guide

This guide provides comprehensive documentation for configuring the Pi-Controller system, including both the control plane and node agents.

## Table of Contents

- [Configuration File Format](#configuration-file-format)
- [Control Plane Configuration](#control-plane-configuration)
- [Node Agent Configuration](#node-agent-configuration)
- [Environment Variables](#environment-variables)
- [TLS/SSL Configuration](#tlsssl-configuration)
- [Database Configuration](#database-configuration)
- [API Configuration](#api-configuration)
- [Service Discovery](#service-discovery)
- [Hardware Configuration](#hardware-configuration)
- [Kubernetes Integration](#kubernetes-integration)
- [Logging and Monitoring](#logging-and-monitoring)
- [Security Configuration](#security-configuration)
- [Advanced Configuration](#advanced-configuration)

## Configuration File Format

Pi-Controller uses YAML format for configuration files. The default configuration file is `config.yaml` in the project root.

### Basic Structure

```yaml
# config.yaml
controller:
  # Control plane configuration

agent:
  # Node agent configuration

common:
  # Shared configuration
```

## Control Plane Configuration

The control plane (`pi-controller`) manages cluster state, node discovery, and provisioning.

### Basic Controller Configuration

```yaml
controller:
  # Server configuration
  server:
    # HTTP/REST API server
    http:
      address: "0.0.0.0"
      port: 8080
      read_timeout: 30s
      write_timeout: 30s
      idle_timeout: 120s
      max_header_bytes: 1048576  # 1MB

    # gRPC server
    grpc:
      address: "0.0.0.0"
      port: 50051
      max_recv_msg_size: 4194304  # 4MB
      max_send_msg_size: 4194304  # 4MB
      keepalive:
        time: 120s
        timeout: 20s
        min_time: 60s

    # WebSocket server
    websocket:
      enabled: true
      port: 8081
      ping_interval: 30s
      pong_wait: 60s
      write_wait: 10s
      max_message_size: 512000  # 512KB

  # Database configuration
  database:
    type: sqlite  # Options: sqlite, postgres (future)
    path: "./data/pi-controller.db"
    max_open_connections: 25
    max_idle_connections: 5
    connection_max_lifetime: 5m

  # Node discovery configuration
  discovery:
    enabled: true
    method: mdns  # Options: mdns, static, kubernetes
    mdns:
      service_name: "_pi-node._tcp"
      domain: "local"
      port: 5353
      ttl: 120
      refresh_interval: 30s
    static:
      nodes:
        - address: "192.168.1.100"
          hostname: "pi-node-1"
        - address: "192.168.1.101"
          hostname: "pi-node-2"
    kubernetes:
      namespace: "pi-controller"
      label_selector: "app=pi-node"
      field_selector: ""

  # Provisioning configuration
  provisioning:
    enabled: true
    ssh:
      user: "pi"
      port: 22
      timeout: 30s
      key_path: "~/.ssh/id_rsa"
      known_hosts_path: "~/.ssh/known_hosts"
      strict_host_checking: false
    k3s:
      version: "latest"  # Or specific version like "v1.28.4+k3s1"
      install_script: "https://get.k3s.io"
      server_args:
        - "--disable=traefik"
        - "--disable=servicelb"
        - "--flannel-backend=wireguard-native"
      agent_args:
        - "--with-node-id"
    retry:
      max_attempts: 3
      initial_delay: 5s
      max_delay: 60s
      multiplier: 2

  # Cluster management
  cluster:
    name: "pi-cluster"
    domain: "cluster.local"
    network:
      pod_subnet: "10.42.0.0/16"
      service_subnet: "10.43.0.0/16"
    storage:
      default_class: "local-path"
      persistent_volume_path: "/var/lib/rancher/k3s/storage"
    registry:
      url: ""  # Optional private registry
      username: ""
      password: ""
```

### API Configuration

```yaml
controller:
  api:
    # REST API configuration
    rest:
      base_path: "/api/v1"
      cors:
        enabled: true
        allowed_origins:
          - "http://localhost:3000"
          - "https://pi-controller.local"
        allowed_methods:
          - GET
          - POST
          - PUT
          - DELETE
          - OPTIONS
        allowed_headers:
          - "Content-Type"
          - "Authorization"
        expose_headers:
          - "X-Total-Count"
        max_age: 3600
        allow_credentials: true

      # Rate limiting
      rate_limit:
        enabled: true
        requests_per_minute: 60
        burst: 10

      # Request validation
      validation:
        max_body_size: 10485760  # 10MB
        strict_mode: false

    # GraphQL configuration (future)
    graphql:
      enabled: false
      endpoint: "/graphql"
      playground: true
      introspection: true
```

## Node Agent Configuration

The node agent (`pi-agent`) runs on each cluster node and provides hardware access and monitoring.

### Basic Agent Configuration

```yaml
agent:
  # Agent identification
  node:
    id: ""  # Auto-generated if empty
    name: ""  # Hostname if empty
    labels:
      location: "rack-1"
      hardware: "rpi-4b"
    annotations:
      owner: "admin"

  # Controller connection
  controller:
    address: "pi-controller.local"
    port: 50051
    tls:
      enabled: true
      ca_cert: "/etc/pi-agent/ca.crt"
      client_cert: "/etc/pi-agent/client.crt"
      client_key: "/etc/pi-agent/client.key"
      server_name: "pi-controller.local"
      insecure_skip_verify: false

    # Connection management
    connection:
      timeout: 30s
      keepalive:
        time: 30s
        timeout: 10s
        permit_without_stream: true
      retry:
        max_attempts: -1  # Infinite retries
        initial_backoff: 1s
        max_backoff: 60s
        backoff_multiplier: 2

  # Hardware access configuration
  hardware:
    gpio:
      enabled: true
      chip: "/dev/gpiochip0"  # GPIO chip device
      permissions: 0660
      available_pins:
        - 2
        - 3
        - 4
        - 17
        - 27
        - 22
        - 10
        - 9
        - 11
        - 5
        - 6
        - 13
        - 19
        - 26
      reserved_pins: []  # Pins reserved by system

    i2c:
      enabled: true
      bus: 1  # I2C bus number
      devices:
        - address: 0x48  # Temperature sensor
          name: "temp_sensor"
        - address: 0x68  # RTC
          name: "rtc"

    spi:
      enabled: false
      bus: 0
      device: 0
      max_speed_hz: 1000000

    pwm:
      enabled: true
      channels:
        - pin: 18
          frequency: 1000
          duty_cycle: 0

  # Monitoring configuration
  monitoring:
    enabled: true
    interval: 30s
    metrics:
      system:
        cpu: true
        memory: true
        disk: true
        network: true
        temperature: true
      custom:
        - name: "gpio_state"
          command: "gpio readall"
          interval: 60s
```

## Environment Variables

Pi-Controller supports configuration via environment variables, which take precedence over config file values.

### Controller Environment Variables

```bash
# Server configuration
PI_CONTROLLER_HTTP_PORT=8080
PI_CONTROLLER_GRPC_PORT=50051
PI_CONTROLLER_WS_PORT=8081

# Database
PI_CONTROLLER_DB_PATH=/data/pi-controller.db
PI_CONTROLLER_DB_MAX_CONNECTIONS=25

# Discovery
PI_CONTROLLER_DISCOVERY_METHOD=mdns
PI_CONTROLLER_DISCOVERY_INTERVAL=30s

# Security
PI_CONTROLLER_TLS_ENABLED=true
PI_CONTROLLER_TLS_CERT_PATH=/certs/server.crt
PI_CONTROLLER_TLS_KEY_PATH=/certs/server.key

# Logging
PI_CONTROLLER_LOG_LEVEL=info
PI_CONTROLLER_LOG_FORMAT=json
```

### Agent Environment Variables

```bash
# Controller connection
PI_AGENT_CONTROLLER_ADDRESS=pi-controller.local
PI_AGENT_CONTROLLER_PORT=50051

# Node identification
PI_AGENT_NODE_ID=node-001
PI_AGENT_NODE_NAME=pi-worker-1

# Hardware
PI_AGENT_GPIO_ENABLED=true
PI_AGENT_I2C_ENABLED=true

# Security
PI_AGENT_TLS_ENABLED=true
PI_AGENT_TLS_CA_CERT=/etc/pi-agent/ca.crt
```

## TLS/SSL Configuration

### Generating Certificates

```bash
# Generate CA certificate
openssl req -x509 -newkey rsa:4096 -keyout ca.key -out ca.crt -days 365 -nodes \
  -subj "/CN=Pi-Controller CA"

# Generate server certificate
openssl req -newkey rsa:4096 -keyout server.key -out server.csr -nodes \
  -subj "/CN=pi-controller.local"

openssl x509 -req -in server.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out server.crt -days 365 \
  -extfile <(printf "subjectAltName=DNS:pi-controller.local,DNS:localhost,IP:127.0.0.1")

# Generate client certificate for agents
openssl req -newkey rsa:4096 -keyout client.key -out client.csr -nodes \
  -subj "/CN=pi-agent"

openssl x509 -req -in client.csr -CA ca.crt -CAkey ca.key -CAcreateserial \
  -out client.crt -days 365
```

### TLS Configuration

```yaml
controller:
  tls:
    enabled: true
    cert_file: "/certs/server.crt"
    key_file: "/certs/server.key"
    ca_file: "/certs/ca.crt"
    client_auth: true  # Require client certificates
    cipher_suites:
      - TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
      - TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256
    min_version: "TLS1.2"
    max_version: "TLS1.3"
```

## Database Configuration

### SQLite Configuration (Default)

```yaml
controller:
  database:
    type: sqlite
    path: "./data/pi-controller.db"
    options:
      journal_mode: "WAL"
      synchronous: "NORMAL"
      cache_size: -64000  # 64MB
      busy_timeout: 5000
      foreign_keys: true
```

### PostgreSQL Configuration (Future)

```yaml
controller:
  database:
    type: postgres
    connection:
      host: "localhost"
      port: 5432
      database: "pi_controller"
      user: "pi_admin"
      password: "${DB_PASSWORD}"  # From environment
      ssl_mode: "require"
    pool:
      max_open: 25
      max_idle: 5
      max_lifetime: 5m
    migrations:
      auto_migrate: true
      path: "./migrations"
```

## Service Discovery

### mDNS Configuration

```yaml
controller:
  discovery:
    mdns:
      interfaces: []  # Empty for all interfaces
      service_name: "_pi-node._tcp"
      service_port: 5353
      txt_records:
        version: "1.0.0"
        platform: "raspberry-pi"
      broadcast_interval: 30s
      cache_duration: 120s
```

### Kubernetes Service Discovery

```yaml
controller:
  discovery:
    kubernetes:
      in_cluster: true  # Use in-cluster config
      config_path: ""  # Or specify kubeconfig path
      namespace: "default"
      label_selector: "pi-controller/managed=true"
      watch: true
      resync_period: 300s
```

## Hardware Configuration

### GPIO Pin Mapping

```yaml
agent:
  hardware:
    gpio:
      pin_mapping:
        # Physical pin : BCM pin
        3: 2
        5: 3
        7: 4
        11: 17
        13: 27
        15: 22
        19: 10
        21: 9
        23: 11
        29: 5
        31: 6
        33: 13
        35: 19
        37: 26

      default_states:
        2: "input_pullup"
        3: "input_pulldown"
        4: "output_low"
```

### I2C Device Configuration

```yaml
agent:
  hardware:
    i2c:
      scan_on_start: true
      devices:
        - address: 0x48
          driver: "ads1115"  # ADC
          config:
            gain: 1
            data_rate: 128

        - address: 0x68
          driver: "ds3231"  # RTC
          config:
            format: "24h"
            timezone: "UTC"

        - address: 0x27
          driver: "lcd1602"  # LCD Display
          config:
            rows: 2
            cols: 16
            backlight: true
```

## Kubernetes Integration

### CRD Configuration

```yaml
controller:
  kubernetes:
    crds:
      enabled: true
      namespace: "pi-controller"
      resources:
        - group: "hardware.pi-controller.io"
          version: "v1alpha1"
          kinds:
            - GPIOPin
            - PWMController
            - I2CDevice
            - SPIDevice

      validation:
        strict: true
        webhook:
          enabled: true
          port: 9443
          cert_dir: "/tmp/k8s-webhook-server/serving-certs"

    rbac:
      create: true
      service_account: "pi-controller"
      cluster_role: "pi-controller-manager"
```

### Operator Configuration

```yaml
controller:
  operator:
    enabled: true
    namespace: "pi-controller-system"
    metrics:
      enabled: true
      port: 8080
      path: "/metrics"
    health:
      enabled: true
      port: 8081
      ready_path: "/readyz"
      live_path: "/healthz"
    leader_election:
      enabled: true
      namespace: "pi-controller-system"
      id: "pi-controller-leader"
```

## Logging and Monitoring

### Logging Configuration

```yaml
common:
  logging:
    level: "info"  # debug, info, warn, error, fatal
    format: "json"  # json, text, pretty
    output:
      - type: "console"
        format: "pretty"
        color: true
      - type: "file"
        path: "/var/log/pi-controller/app.log"
        max_size: 100  # MB
        max_backups: 5
        max_age: 30  # days
        compress: true
      - type: "syslog"
        network: "udp"
        address: "localhost:514"
        tag: "pi-controller"

    fields:
      service: "pi-controller"
      environment: "production"
      version: "${VERSION}"
```

### Metrics Configuration

```yaml
common:
  metrics:
    enabled: true
    provider: "prometheus"

    prometheus:
      port: 9090
      path: "/metrics"
      namespace: "pi_controller"
      subsystem: ""
      buckets: [0.1, 0.3, 1.2, 5.0]

      collectors:
        - go_collector
        - process_collector
        - custom_collector

    custom_metrics:
      - name: "node_discovery_duration"
        type: "histogram"
        help: "Time taken to discover nodes"

      - name: "gpio_operations_total"
        type: "counter"
        help: "Total number of GPIO operations"
        labels: ["pin", "operation"]

      - name: "cluster_nodes_count"
        type: "gauge"
        help: "Current number of nodes in cluster"
```

### Tracing Configuration

```yaml
common:
  tracing:
    enabled: false
    provider: "jaeger"

    jaeger:
      endpoint: "http://localhost:14268/api/traces"
      service_name: "pi-controller"
      sampler:
        type: "probabilistic"
        param: 0.1

      tags:
        environment: "production"
        version: "${VERSION}"
```

## Security Configuration

### Authentication

```yaml
controller:
  auth:
    enabled: true
    type: "token"  # token, oauth2, mtls

    token:
      header: "Authorization"
      prefix: "Bearer"
      secret: "${AUTH_SECRET}"
      expiry: 24h
      refresh: true
      refresh_expiry: 168h  # 7 days

    oauth2:
      provider: "github"
      client_id: "${OAUTH_CLIENT_ID}"
      client_secret: "${OAUTH_CLIENT_SECRET}"
      redirect_url: "https://pi-controller.local/auth/callback"
      scopes:
        - "read:user"
        - "user:email"

    mtls:
      ca_file: "/certs/ca.crt"
      verify_client: true
```

### Authorization

```yaml
controller:
  authorization:
    enabled: true
    type: "rbac"  # rbac, abac

    rbac:
      policy_file: "/config/rbac-policy.yaml"
      default_role: "viewer"
      roles:
        - name: "admin"
          permissions:
            - resource: "*"
              actions: ["*"]

        - name: "operator"
          permissions:
            - resource: "nodes"
              actions: ["get", "list", "update"]
            - resource: "gpio"
              actions: ["*"]

        - name: "viewer"
          permissions:
            - resource: "*"
              actions: ["get", "list"]
```

### Security Headers

```yaml
controller:
  security:
    headers:
      strict_transport_security: "max-age=31536000; includeSubDomains"
      content_security_policy: "default-src 'self'"
      x_frame_options: "DENY"
      x_content_type_options: "nosniff"
      x_xss_protection: "1; mode=block"
      referrer_policy: "strict-origin-when-cross-origin"
```

## Advanced Configuration

### High Availability

```yaml
controller:
  ha:
    enabled: true
    mode: "active-passive"  # active-passive, active-active

    cluster:
      peers:
        - "pi-controller-1.local:50051"
        - "pi-controller-2.local:50051"
        - "pi-controller-3.local:50051"

      consensus:
        algorithm: "raft"
        election_timeout: 1000ms
        heartbeat_timeout: 100ms
        snapshot_interval: 120s

      replication:
        mode: "synchronous"
        max_lag: 10s
```

### Backup Configuration

```yaml
controller:
  backup:
    enabled: true
    schedule: "0 2 * * *"  # Daily at 2 AM

    destinations:
      - type: "local"
        path: "/backups/pi-controller"
        retention: 7  # days

      - type: "s3"
        bucket: "pi-controller-backups"
        prefix: "prod/"
        region: "us-west-2"
        access_key: "${AWS_ACCESS_KEY}"
        secret_key: "${AWS_SECRET_KEY}"
        retention: 30  # days

    compression: true
    encryption:
      enabled: true
      key: "${BACKUP_ENCRYPTION_KEY}"
```

### Performance Tuning

```yaml
controller:
  performance:
    # Connection pooling
    connection_pool:
      max_idle: 10
      max_open: 100
      max_lifetime: 300s

    # Caching
    cache:
      enabled: true
      provider: "memory"  # memory, redis
      ttl: 300s
      max_size: 100MB

      redis:
        address: "localhost:6379"
        password: ""
        db: 0
        pool_size: 10

    # Worker pools
    workers:
      discovery: 5
      provisioning: 3
      api_handlers: 10
      event_processors: 5

    # Resource limits
    limits:
      max_nodes: 1000
      max_concurrent_operations: 50
      max_request_size: 10MB
      max_response_size: 50MB
```

### Feature Flags

```yaml
controller:
  features:
    experimental:
      gpu_support: false
      container_runtime: false
      edge_computing: false

    preview:
      web_ui: true
      graphql_api: false
      plugin_system: false

    deprecated:
      legacy_api_v0: false
      old_auth_system: false
```

## Configuration Validation

Pi-Controller validates configuration on startup. To validate configuration without starting the service:

```bash
# Validate controller configuration
pi-controller validate --config config.yaml

# Validate agent configuration
pi-agent validate --config agent-config.yaml

# Check configuration with verbose output
pi-controller validate --config config.yaml --verbose
```

## Configuration Hot Reload

Some configuration changes can be applied without restarting:

```yaml
controller:
  hot_reload:
    enabled: true
    watch_interval: 10s
    reloadable:
      - logging
      - rate_limits
      - cors
      - feature_flags

    webhook:
      enabled: true
      endpoint: "/admin/reload"
      auth_required: true
```

To trigger a configuration reload:

```bash
# Via API
curl -X POST http://localhost:8080/admin/reload \
  -H "Authorization: Bearer ${TOKEN}"

# Via signal
kill -HUP $(pidof pi-controller)
```

## Best Practices

1. **Use Environment Variables for Secrets**: Never hardcode passwords, API keys, or certificates in configuration files.

2. **Separate Environments**: Maintain different configuration files for development, staging, and production.

3. **Version Control**: Store configuration templates in version control, but exclude files with sensitive data.

4. **Configuration Validation**: Always validate configuration changes before deploying.

5. **Gradual Rollout**: Test configuration changes in development/staging before production.

6. **Backup Configuration**: Keep backups of working configurations before making changes.

7. **Documentation**: Document any custom configuration or deviations from defaults.

8. **Monitoring**: Monitor configuration changes and their impact on system performance.

## Troubleshooting

### Common Configuration Issues

1. **Port Conflicts**
   ```bash
   # Check if ports are in use
   netstat -tulpn | grep -E "8080|50051|8081"
   ```

2. **Permission Issues**
   ```bash
   # Fix file permissions
   chmod 600 config.yaml
   chmod 700 /var/lib/pi-controller
   ```

3. **Certificate Problems**
   ```bash
   # Verify certificate validity
   openssl x509 -in server.crt -text -noout

   # Check certificate chain
   openssl verify -CAfile ca.crt server.crt
   ```

4. **Database Connection**
   ```bash
   # Test SQLite database
   sqlite3 /data/pi-controller.db "SELECT COUNT(*) FROM nodes;"
   ```

### Debug Configuration

Enable debug mode for detailed configuration loading information:

```yaml
common:
  debug:
    enabled: true
    config_dump: true  # Print full configuration at startup
    env_vars: true     # Show environment variable resolution
    validation: true   # Show validation details
```

## Configuration Examples

### Minimal Configuration

```yaml
# Minimal config for development
controller:
  server:
    http:
      port: 8080
  database:
    type: sqlite
    path: "./pi-controller.db"
```

### Production Configuration

See `/examples/config-production.yaml` for a complete production-ready configuration example.

### Kubernetes Deployment

See `/examples/k8s-configmap.yaml` for Kubernetes ConfigMap example.

## References

- [Architecture Documentation](ARCHITECTURE.md)
- [API Documentation](API.md)
- [Security Guide](SECURITY.md)
- [Deployment Guide](DEPLOYMENT.md)
- [Troubleshooting Guide](TROUBLESHOOTING.md)