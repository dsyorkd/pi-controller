# REST API Reference

The Pi-Controller provides a comprehensive RESTful API for managing clusters, nodes, and GPIO resources. The API is served by the main `pi-controller` control plane binary.

## Authentication

API endpoints are secured and require proper authentication. Depending on the configuration, this can be certificate-based (mTLS) or token-based (JWT/OIDC). Refer to the Security Documentation for more details.

## API Versioning

The current stable API version is `v1`, prefixed under `/api/v1/`.

---

## Cluster Management

Endpoints for creating, managing, and monitoring K3s clusters.

| Method | Endpoint                               | Description                  |
|--------|----------------------------------------|------------------------------|
| `GET`  | `/api/v1/clusters`                     | List all managed clusters.   |
| `POST` | `/api/v1/clusters`                     | Create a new cluster.        |
| `GET`  | `/api/v1/clusters/{id}`                | Get details for a specific cluster. |
| `PUT`  | `/api/v1/clusters/{id}`                | Update a cluster's configuration. |
| `DELETE`| `/api/v1/clusters/{id}`                | Delete a cluster.            |

### Cluster Nodes

| Method | Endpoint                               | Description                  |
|--------|----------------------------------------|------------------------------|
| `GET`  | `/api/v1/clusters/{id}/nodes`          | List all nodes in a cluster. |
| `POST` | `/api/v1/clusters/{id}/nodes`          | Add a node to a cluster.     |
| `DELETE`| `/api/v1/clusters/{id}/nodes/{node}`   | Remove a node from a cluster.|

---

## Node Management

Endpoints for managing individual Raspberry Pi nodes discovered on the network.

| Method | Endpoint                         | Description                  |
|--------|----------------------------------|------------------------------|
| `GET`  | `/api/v1/nodes`                  | List all discovered nodes.   |
| `GET`  | `/api/v1/nodes/{id}`             | Get details for a specific node. |
| `PUT`  | `/api/v1/nodes/{id}`             | Update a node's configuration. |
| `POST` | `/api/v1/nodes/{id}/provision`   | Provision K3s on a node.     |
| `POST` | `/api/v1/nodes/{id}/deprovision` | Deprovision a node.          |

---

## GPIO Resources

Endpoints for interacting with GPIO resources. These are abstractions over the `GPIOPin` Kubernetes CRDs.

| Method | Endpoint                         | Description                  |
|--------|----------------------------------|------------------------------|
| `GET`  | `/api/v1/gpio`                   | List all GPIO resources.     |
| `POST` | `/api/v1/gpio`                   | Create a new GPIO resource.  |
| `GET`  | `/api/v1/gpio/{id}`              | Get the state of a GPIO resource. |
| `PUT`  | `/api/v1/gpio/{id}`              | Update the state of a GPIO resource. |
| `DELETE`| `/api/v1/gpio/{id}`              | Delete a GPIO resource.      |