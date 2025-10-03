# TLS/HTTPS Setup Guide

This guide covers TLS/HTTPS configuration for the pi-controller project.

## Table of Contents
- [Overview](#overview)
- [Development Setup](#development-setup)
- [Production Setup](#production-setup)
- [Certificate Management](#certificate-management)
- [Troubleshooting](#troubleshooting)

---

## Overview

Pi-controller supports TLS/HTTPS encryption for secure API communication. The implementation includes:

- **Auto-generation** of self-signed certificates for development
- **Production certificate support** for Let's Encrypt, corporate CAs, or custom certificates
- **TLS 1.2+ minimum** with secure cipher suites
- **Automatic certificate validation** and expiry checking
- **ECDSA key generation** (more efficient than RSA)

**Security Note:** Auto-generated certificates are for development only. Production deployments MUST use properly signed certificates.

---

## Development Setup

### Automatic Certificate Generation

The simplest way to use TLS in development is to let pi-controller auto-generate certificates:

```bash
# Set environment to development
export PI_CONTROLLER_ENVIRONMENT=development

# Start the controller
./pi-controller

# Certificates are auto-generated at:
# - ./data/tls/server.crt (certificate)
# - ./data/tls/server.key (private key)
```

The server will start on `https://localhost:8080` with a self-signed certificate valid for 1 year.

**Browser Warning:** Your browser will show a security warning because the certificate is self-signed. This is expected in development.

### Custom Development Certificates

If you need custom hostnames for development:

```yaml
# config.yaml
app:
  environment: development
  data_dir: ./data

api:
  host: 0.0.0.0
  port: 8443
  # Leave cert files empty for auto-generation
  # But specify custom hostnames if needed
```

Or via environment:

```bash
export PI_CONTROLLER_ENVIRONMENT=development
export API_HOST=0.0.0.0
export API_PORT=8443
./pi-controller
```

---

## Production Setup

Production deployments require properly signed TLS certificates from a trusted Certificate Authority (CA).

### Option 1: Let's Encrypt (Recommended)

Let's Encrypt provides free, automated SSL/TLS certificates.

#### Using Certbot (Standalone)

```bash
# Install certbot
apt-get install certbot  # Debian/Ubuntu
yum install certbot      # RHEL/CentOS

# Obtain certificate (requires port 80/443 to be available)
certbot certonly --standalone -d your-domain.com

# Certificates are installed at:
# /etc/letsencrypt/live/your-domain.com/fullchain.pem
# /etc/letsencrypt/live/your-domain.com/privkey.pem

# Configure pi-controller
export PI_CONTROLLER_ENVIRONMENT=production
export TLS_CERT_FILE=/etc/letsencrypt/live/your-domain.com/fullchain.pem
export TLS_KEY_FILE=/etc/letsencrypt/live/your-domain.com/privkey.pem

./pi-controller
```

#### Using Certbot (Webroot)

If you have an existing web server:

```bash
# Obtain certificate using webroot method
certbot certonly --webroot -w /var/www/html -d your-domain.com

# Configure as above
```

#### Automatic Renewal

Let's Encrypt certificates expire after 90 days. Setup automatic renewal:

```bash
# Add to crontab
crontab -e

# Renew certificates twice daily (certbot will only renew if needed)
0 0,12 * * * certbot renew --quiet --deploy-hook "systemctl restart pi-controller"
```

### Option 2: Corporate/Internal CA

If you have an internal Certificate Authority:

```bash
# Request certificate from your CA (process varies)
# You should receive:
# - certificate.pem (your certificate)
# - private-key.pem (your private key)
# - ca-bundle.pem (optional, CA chain)

# Combine certificate with CA chain if needed
cat certificate.pem ca-bundle.pem > fullchain.pem

# Configure pi-controller
export PI_CONTROLLER_ENVIRONMENT=production
export TLS_CERT_FILE=/etc/pi-controller/tls/fullchain.pem
export TLS_KEY_FILE=/etc/pi-controller/tls/private-key.pem

# Secure the key file
chmod 600 /etc/pi-controller/tls/private-key.pem

./pi-controller
```

### Option 3: Custom Self-Signed (NOT Recommended for Production)

Only use self-signed certificates for internal testing environments:

```bash
# Generate self-signed certificate
openssl req -x509 -newkey rsa:4096 -nodes \
  -keyout server.key -out server.crt \
  -days 365 \
  -subj "/CN=your-domain.com" \
  -addext "subjectAltName=DNS:your-domain.com,DNS:localhost,IP:127.0.0.1"

# Configure pi-controller
export PI_CONTROLLER_ENVIRONMENT=production
export TLS_CERT_FILE=/path/to/server.crt
export TLS_KEY_FILE=/path/to/server.key

./pi-controller
```

---

## Certificate Management

### Certificate Validation

Pi-controller automatically validates certificates on startup:

- Checks if certificate file exists and is readable
- Verifies certificate is not expired
- Warns if certificate expires within 30 days
- Validates that private key matches certificate

**Example startup logs:**

```json
{"level":"info","msg":"TLS/HTTPS configured successfully","cert_file":"/etc/letsencrypt/live/domain.com/fullchain.pem","key_file":"/etc/letsencrypt/live/domain.com/privkey.pem"}
{"level":"info","msg":"Starting HTTPS server with TLS enabled"}
```

### Certificate Expiry Monitoring

Pi-controller logs warnings when certificates are about to expire:

```json
{"level":"warn","msg":"Certificate will expire within 30 days","expiry":"2025-02-28T10:30:00Z"}
```

Monitor these logs and renew certificates before expiry.

### Certificate Rotation

To rotate certificates without downtime:

1. Obtain new certificate (using same method as initial setup)
2. Update certificate files in place
3. Restart pi-controller gracefully:

```bash
# Systemd
systemctl reload pi-controller

# Docker
docker exec pi-controller kill -HUP 1

# Kubernetes
kubectl rollout restart deployment/pi-controller
```

---

## Configuration Reference

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `PI_CONTROLLER_ENVIRONMENT` | Runtime environment | `production` or `development` |
| `TLS_CERT_FILE` | Path to TLS certificate | `/etc/letsencrypt/live/domain.com/fullchain.pem` |
| `TLS_KEY_FILE` | Path to TLS private key | `/etc/letsencrypt/live/domain.com/privkey.pem` |
| `API_PORT` | HTTPS port (default: 8443) | `8443` |
| `API_HOST` | Bind address | `0.0.0.0` |

### YAML Configuration

```yaml
app:
  environment: production  # or development
  data_dir: /var/lib/pi-controller

api:
  host: 0.0.0.0
  port: 8443
  tls_cert_file: /etc/pi-controller/tls/cert.pem
  tls_key_file: /etc/pi-controller/tls/key.pem
```

### Docker Secrets

```yaml
# docker-compose.yml
services:
  pi-controller:
    image: pi-controller:latest
    secrets:
      - tls_cert
      - tls_key
    environment:
      - TLS_CERT_FILE=/run/secrets/tls_cert
      - TLS_KEY_FILE=/run/secrets/tls_key
      - PI_CONTROLLER_ENVIRONMENT=production

secrets:
  tls_cert:
    file: ./tls/cert.pem
  tls_key:
    file: ./tls/key.pem
```

### Kubernetes Secrets

```yaml
# Create secret
kubectl create secret tls pi-controller-tls \
  --cert=cert.pem \
  --key=key.pem

# Mount in deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: pi-controller
spec:
  template:
    spec:
      containers:
      - name: pi-controller
        env:
        - name: PI_CONTROLLER_ENVIRONMENT
          value: "production"
        - name: TLS_CERT_FILE
          value: "/etc/tls/tls.crt"
        - name: TLS_KEY_FILE
          value: "/etc/tls/tls.key"
        volumeMounts:
        - name: tls
          mountPath: /etc/tls
          readOnly: true
      volumes:
      - name: tls
        secret:
          secretName: pi-controller-tls
```

---

## Troubleshooting

### Certificate Not Found

**Error:**
```
failed to setup TLS: TLS certificate files not found: cert=/path/to/cert.pem, key=/path/to/key.pem
```

**Solution:**
- Verify file paths are correct
- Check file permissions (certificate must be readable, key should be 600)
- Ensure files exist at specified locations

### Certificate Expired

**Error:**
```
failed to load TLS certificates: x509: certificate has expired
```

**Solution:**
- Renew certificate using your CA's renewal process
- For Let's Encrypt: `certbot renew`
- Restart pi-controller after renewal

### Invalid Certificate

**Error:**
```
failed to load TLS certificates: tls: private key does not match public key
```

**Solution:**
- Ensure certificate and key files match
- Verify you're not mixing certificates from different requests
- Regenerate certificate and key pair together

### Permission Denied

**Error:**
```
failed to read TLS key file: permission denied
```

**Solution:**
```bash
# Set correct ownership (run as root)
chown pi-controller:pi-controller /path/to/key.pem
chmod 600 /path/to/key.pem

# Or run pi-controller with appropriate user
sudo -u pi-controller ./pi-controller
```

### Auto-Generated Cert in Production

**Warning:**
```
Auto-generating self-signed TLS certificate for development
⚠️  DO NOT use auto-generated certificates in production!
```

**Solution:**
- Set `PI_CONTROLLER_ENVIRONMENT=production`
- Provide proper TLS certificate files
- Use Let's Encrypt or corporate CA

### Port 443 Already in Use

**Error:**
```
bind: address already in use
```

**Solution:**
```bash
# Check what's using port 443
sudo lsof -i :443

# Stop conflicting service or use different port
export API_PORT=8443
./pi-controller
```

### Browser Security Warning

**In Development:**
This is expected with self-signed certificates. Add security exception in browser.

**In Production:**
- Ensure certificate is from trusted CA
- Check certificate is valid and not expired
- Verify hostname matches certificate Common Name (CN)
- Check certificate chain is complete

---

## Security Best Practices

1. **Never commit private keys to version control**
   - Add `*.key`, `*.pem` to `.gitignore`
   - Use secrets management (Vault, AWS Secrets Manager, etc.)

2. **Restrict key file permissions**
   ```bash
   chmod 600 /path/to/key.pem
   chown pi-controller:pi-controller /path/to/key.pem
   ```

3. **Use strong keys**
   - Minimum RSA 2048-bit (4096-bit recommended)
   - Or ECDSA P-256 (P-384 for high security)

4. **Monitor certificate expiry**
   - Set up alerts 30 days before expiry
   - Automate renewal where possible

5. **Keep certificates up to date**
   - Renew before expiry
   - Follow CA security advisories
   - Rotate regularly in high-security environments

6. **Use proper certificate chains**
   - Include intermediate certificates
   - Test with SSL Labs (https://www.ssllabs.com/ssltest/)

---

## Additional Resources

- [Let's Encrypt Documentation](https://letsencrypt.org/docs/)
- [Certbot User Guide](https://certbot.eff.org/docs/)
- [Mozilla SSL Configuration Generator](https://ssl-config.mozilla.org/)
- [SSL Labs Server Test](https://www.ssllabs.com/ssltest/)

---

**Last Updated:** 2025-01-30