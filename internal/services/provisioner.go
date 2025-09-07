package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dsyorkd/pi-controller/internal/errors"
	"github.com/dsyorkd/pi-controller/internal/logger"
	"github.com/dsyorkd/pi-controller/internal/models"
	"github.com/dsyorkd/pi-controller/internal/provisioner"
)

// ProvisioningService handles K3s cluster provisioning
type ProvisioningService struct {
	nodeService *NodeService
	logger      logger.Interface
}

// NewProvisioningService creates a new provisioning service
func NewProvisioningService(nodeService *NodeService, logger logger.Interface) *ProvisioningService {
	return &ProvisioningService{
		nodeService: nodeService,
		logger: logger.WithFields(map[string]interface{}{
			"service": "provisioner",
		}),
	}
}

// K3sConfig represents K3s installation configuration
type K3sConfig struct {
	// K3s installation settings
	Version       string            `json:"version,omitempty"`        // K3s version (e.g., "v1.28.2+k3s1")
	InstallScript string            `json:"install_script,omitempty"` // Custom install script URL
	Channel       string            `json:"channel,omitempty"`        // Release channel (stable, latest, testing)
	ExtraArgs     map[string]string `json:"extra_args,omitempty"`     // Additional K3s arguments

	// Cluster settings
	ClusterToken   string `json:"cluster_token,omitempty"`   // Token for joining nodes to cluster
	ServerURL      string `json:"server_url,omitempty"`      // URL of K3s server (for worker nodes)
	DatastoreToken string `json:"datastore_token,omitempty"` // Token for external datastore

	// Network settings
	ClusterCIDR    string   `json:"cluster_cidr,omitempty"`    // Pod CIDR (default: 10.42.0.0/16)
	ServiceCIDR    string   `json:"service_cidr,omitempty"`    // Service CIDR (default: 10.43.0.0/16)
	ClusterDNS     string   `json:"cluster_dns,omitempty"`     // Cluster DNS IP (default: 10.43.0.10)
	ClusterDomain  string   `json:"cluster_domain,omitempty"`  // Cluster domain (default: cluster.local)
	FlannelBackend string   `json:"flannel_backend,omitempty"` // Flannel backend (vxlan, host-gw, wireguard)
	NodeExternalIP string   `json:"node_external_ip,omitempty"` // Node external IP
	NodeIP         string   `json:"node_ip,omitempty"`         // Node IP address
	AdvertiseIP    string   `json:"advertise_ip,omitempty"`    // IP address to advertise for API server

	// TLS settings
	TLSSan []string `json:"tls_san,omitempty"` // Additional TLS SANs for server certificate

	// Component settings
	DisableComponents []string `json:"disable_components,omitempty"` // Components to disable (traefik, servicelb, etc.)
	KubeAPIArgs       []string `json:"kube_api_args,omitempty"`       // Additional kube-apiserver arguments
	KubeControllerArgs []string `json:"kube_controller_args,omitempty"` // Additional kube-controller-manager args
	KubeSchedulerArgs  []string `json:"kube_scheduler_args,omitempty"`  // Additional kube-scheduler arguments
	KubeletArgs        []string `json:"kubelet_args,omitempty"`         // Additional kubelet arguments
	KubeProxyArgs      []string `json:"kube_proxy_args,omitempty"`      // Additional kube-proxy arguments

	// Registry settings
	PrivateRegistry string `json:"private_registry,omitempty"` // Private registry configuration file path

	// Advanced settings
	DataDir            string `json:"data_dir,omitempty"`             // K3s data directory
	SelinuxWarning     bool   `json:"selinux_warning,omitempty"`      // Disable SELinux warning
	DefaultLocalStorage bool   `json:"default_local_storage,omitempty"` // Enable default local-path storage
	WriteKubeconfig    string `json:"write_kubeconfig,omitempty"`     // Kubeconfig file path
	KubeconfigMode     string `json:"kubeconfig_mode,omitempty"`      // Kubeconfig file permissions
}

// DefaultK3sConfig returns a K3s configuration with sensible defaults
func DefaultK3sConfig() K3sConfig {
	return K3sConfig{
		Channel:             "stable",
		InstallScript:       "https://get.k3s.io",
		ClusterCIDR:         "10.42.0.0/16",
		ServiceCIDR:         "10.43.0.0/16",
		ClusterDNS:          "10.43.0.10",
		ClusterDomain:       "cluster.local",
		FlannelBackend:      "vxlan",
		DataDir:             "/var/lib/rancher/k3s",
		DefaultLocalStorage: true,
		WriteKubeconfig:     "/etc/rancher/k3s/k3s.yaml",
		KubeconfigMode:      "644",
		SelinuxWarning:      false,
	}
}

// ProvisionClusterRequest represents a request to provision a K3s cluster
type ProvisionClusterRequest struct {
	ClusterID     uint                      `json:"cluster_id" validate:"required"`
	MasterNodeID  uint                      `json:"master_node_id" validate:"required"`
	WorkerNodeIDs []uint                    `json:"worker_node_ids,omitempty"`
	K3sConfig     K3sConfig                 `json:"k3s_config,omitempty"`
	SSHConfig     provisioner.SSHClientConfig `json:"ssh_config"`
}

// ProvisionNodeRequest represents a request to provision a single node
type ProvisionNodeRequest struct {
	NodeID    uint                      `json:"node_id" validate:"required"`
	ClusterID uint                      `json:"cluster_id" validate:"required"`
	Role      models.NodeRole           `json:"role" validate:"required,oneof=master worker"`
	K3sConfig K3sConfig                 `json:"k3s_config,omitempty"`
	SSHConfig provisioner.SSHClientConfig `json:"ssh_config"`
}

// ProvisioningResult represents the result of a provisioning operation
type ProvisioningResult struct {
	NodeID       uint                          `json:"node_id"`
	NodeName     string                        `json:"node_name"`
	Success      bool                          `json:"success"`
	Error        string                        `json:"error,omitempty"`
	Duration     time.Duration                 `json:"duration"`
	Commands     []*provisioner.CommandResult  `json:"commands,omitempty"`
	ClusterToken string                        `json:"cluster_token,omitempty"` // Generated for master nodes
	KubeConfig   string                        `json:"kube_config,omitempty"`   // Retrieved from master nodes
}

// ProvisionCluster provisions a complete K3s cluster with master and worker nodes
func (s *ProvisioningService) ProvisionCluster(ctx context.Context, req ProvisionClusterRequest) (*ProvisioningResult, error) {
	s.logger.WithFields(map[string]interface{}{
		"cluster_id":      req.ClusterID,
		"master_node_id":  req.MasterNodeID,
		"worker_node_ids": req.WorkerNodeIDs,
	}).Info("Starting cluster provisioning")

	start := time.Now()

	// Validate nodes exist
	// Note: cluster validation is handled by node service

	masterNode, err := s.nodeService.GetByID(req.MasterNodeID, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get master node %d", req.MasterNodeID)
	}

	if masterNode.Role != models.NodeRoleMaster {
		return nil, fmt.Errorf("node %d is not a master node", req.MasterNodeID)
	}

	// Provision master node first
	s.logger.Info("Provisioning master node")
	masterResult, err := s.ProvisionNode(ctx, ProvisionNodeRequest{
		NodeID:    req.MasterNodeID,
		ClusterID: req.ClusterID,
		Role:      models.NodeRoleMaster,
		K3sConfig: req.K3sConfig,
		SSHConfig: req.SSHConfig,
	})
	if err != nil {
		return &ProvisioningResult{
			NodeID:   req.MasterNodeID,
			NodeName: masterNode.Name,
			Success:  false,
			Error:    err.Error(),
			Duration: time.Since(start),
		}, err
	}

	if !masterResult.Success {
		return masterResult, fmt.Errorf("master node provisioning failed: %s", masterResult.Error)
	}

	// Update K3s config with cluster token from master for worker nodes
	workerConfig := req.K3sConfig
	if masterResult.ClusterToken != "" {
		workerConfig.ClusterToken = masterResult.ClusterToken
		workerConfig.ServerURL = fmt.Sprintf("https://%s:6443", masterNode.IPAddress)
	}

	// Provision worker nodes in parallel
	if len(req.WorkerNodeIDs) > 0 {
		s.logger.WithField("worker_count", len(req.WorkerNodeIDs)).Info("Provisioning worker nodes")
		
		// TODO: Implement parallel worker node provisioning
		// For now, provision sequentially
		for _, workerNodeID := range req.WorkerNodeIDs {
			workerResult, err := s.ProvisionNode(ctx, ProvisionNodeRequest{
				NodeID:    workerNodeID,
				ClusterID: req.ClusterID,
				Role:      models.NodeRoleWorker,
				K3sConfig: workerConfig,
				SSHConfig: req.SSHConfig,
			})
			if err != nil {
				s.logger.WithError(err).WithField("worker_node_id", workerNodeID).Error("Worker node provisioning failed")
				// Continue with other workers, don't fail the entire cluster
				continue
			}

			if !workerResult.Success {
				s.logger.WithField("worker_node_id", workerNodeID).WithField("error", workerResult.Error).Error("Worker node provisioning failed")
			}
		}
	}

	duration := time.Since(start)
	s.logger.WithFields(map[string]interface{}{
		"cluster_id": req.ClusterID,
		"duration":   duration,
	}).Info("Cluster provisioning completed")

	return &ProvisioningResult{
		NodeID:       req.MasterNodeID,
		NodeName:     masterNode.Name,
		Success:      true,
		Duration:     duration,
		ClusterToken: masterResult.ClusterToken,
		KubeConfig:   masterResult.KubeConfig,
	}, nil
}

// ProvisionNode provisions K3s on a single node
func (s *ProvisioningService) ProvisionNode(ctx context.Context, req ProvisionNodeRequest) (*ProvisioningResult, error) {
	start := time.Now()

	// Get node details
	node, err := s.nodeService.GetByID(req.NodeID, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get node %d", req.NodeID)
	}

	s.logger.WithFields(map[string]interface{}{
		"node_id":    req.NodeID,
		"node_name":  node.Name,
		"node_ip":    node.IPAddress,
		"node_role":  req.Role,
		"cluster_id": req.ClusterID,
	}).Info("Starting node provisioning")

	// Mark node as provisioning
	if err := s.nodeService.Provision(req.NodeID, req.ClusterID); err != nil {
		return nil, errors.Wrap(err, "failed to mark node as provisioning")
	}

	// Setup SSH connection to the node
	sshConfig := req.SSHConfig
	sshConfig.Host = node.IPAddress
	if sshConfig.Username == "" {
		sshConfig.Username = "pi" // Default username for Raspberry Pi
	}

	sshClient, err := provisioner.NewSSHClient(sshConfig, s.logger)
	if err != nil {
		return &ProvisioningResult{
			NodeID:   req.NodeID,
			NodeName: node.Name,
			Success:  false,
			Error:    fmt.Sprintf("failed to create SSH client: %v", err),
			Duration: time.Since(start),
		}, errors.Wrap(err, "failed to create SSH client")
	}
	defer sshClient.Close()

	// Merge with default config
	k3sConfig := DefaultK3sConfig()
	if req.K3sConfig.Version != "" {
		k3sConfig.Version = req.K3sConfig.Version
	}
	if req.K3sConfig.Channel != "" {
		k3sConfig.Channel = req.K3sConfig.Channel
	}
	if req.K3sConfig.ClusterToken != "" {
		k3sConfig.ClusterToken = req.K3sConfig.ClusterToken
	}
	if req.K3sConfig.ServerURL != "" {
		k3sConfig.ServerURL = req.K3sConfig.ServerURL
	}
	// Override with custom config values
	if len(req.K3sConfig.TLSSan) > 0 {
		k3sConfig.TLSSan = req.K3sConfig.TLSSan
	}
	if len(req.K3sConfig.DisableComponents) > 0 {
		k3sConfig.DisableComponents = req.K3sConfig.DisableComponents
	}

	// Generate installation commands based on node role and config
	var commands []string
	var allResults []*provisioner.CommandResult

	if req.Role == models.NodeRoleMaster {
		commands = s.generateMasterInstallCommands(node, k3sConfig)
	} else {
		commands = s.generateWorkerInstallCommands(node, k3sConfig)
	}

	// Execute installation commands
	s.logger.WithField("command_count", len(commands)).Debug("Executing K3s installation commands")

	results, err := sshClient.ExecuteCommands(ctx, commands)
	allResults = append(allResults, results...)

	if err != nil {
		// Update node status to failed
		s.updateNodeStatus(req.NodeID, models.NodeStatusFailed)
		
		return &ProvisioningResult{
			NodeID:   req.NodeID,
			NodeName: node.Name,
			Success:  false,
			Error:    fmt.Sprintf("command execution failed: %v", err),
			Duration: time.Since(start),
			Commands: allResults,
		}, err
	}

	// Extract cluster token for master nodes
	var clusterToken, kubeConfig string
	if req.Role == models.NodeRoleMaster {
		// Get the cluster token
		tokenResult, tokenErr := sshClient.ExecuteCommand(ctx, "sudo cat /var/lib/rancher/k3s/server/node-token")
		if tokenErr == nil && tokenResult.Success {
			clusterToken = strings.TrimSpace(tokenResult.Stdout)
		}

		// Get the kubeconfig
		configResult, configErr := sshClient.ExecuteCommand(ctx, "sudo cat /etc/rancher/k3s/k3s.yaml")
		if configErr == nil && configResult.Success {
			kubeConfig = configResult.Stdout
		}
	}

	// Verify K3s installation
	verifyCommands := []string{
		"sudo systemctl is-active k3s || sudo systemctl is-active k3s-agent",
		"sudo k3s kubectl get nodes --no-headers 2>/dev/null | wc -l",
	}

	verifyResults, verifyErr := sshClient.ExecuteCommands(ctx, verifyCommands)
	allResults = append(allResults, verifyResults...)

	success := verifyErr == nil
	var errorMsg string
	if !success {
		errorMsg = fmt.Sprintf("K3s installation verification failed: %v", verifyErr)
	}

	// Update node status based on success
	var finalStatus models.NodeStatus
	if success {
		finalStatus = models.NodeStatusReady
		// Update node with Kubernetes information
		s.updateNodeKubeInfo(req.NodeID, node.Name)
	} else {
		finalStatus = models.NodeStatusFailed
	}
	s.updateNodeStatus(req.NodeID, finalStatus)

	duration := time.Since(start)
	s.logger.WithFields(map[string]interface{}{
		"node_id":   req.NodeID,
		"node_name": node.Name,
		"success":   success,
		"duration":  duration,
	}).Info("Node provisioning completed")

	return &ProvisioningResult{
		NodeID:       req.NodeID,
		NodeName:     node.Name,
		Success:      success,
		Error:        errorMsg,
		Duration:     duration,
		Commands:     allResults,
		ClusterToken: clusterToken,
		KubeConfig:   kubeConfig,
	}, nil
}

// generateMasterInstallCommands generates the commands to install K3s server (master node)
func (s *ProvisioningService) generateMasterInstallCommands(node *models.Node, config K3sConfig) []string {
	var commands []string
	var installCmd strings.Builder

	// Basic install command
	installCmd.WriteString("curl -sfL " + config.InstallScript + " | ")

	// Environment variables
	var envVars []string
	if config.Version != "" {
		envVars = append(envVars, "INSTALL_K3S_VERSION="+config.Version)
	}
	if config.Channel != "" {
		envVars = append(envVars, "INSTALL_K3S_CHANNEL="+config.Channel)
	}
	
	if len(envVars) > 0 {
		installCmd.WriteString(strings.Join(envVars, " ") + " ")
	}

	installCmd.WriteString("sh -s - server")

	// Server arguments
	var serverArgs []string
	
	if config.NodeIP != "" {
		serverArgs = append(serverArgs, "--node-ip="+config.NodeIP)
	} else if node.IPAddress != "" {
		serverArgs = append(serverArgs, "--node-ip="+node.IPAddress)
	}

	if config.AdvertiseIP != "" {
		serverArgs = append(serverArgs, "--advertise-address="+config.AdvertiseIP)
	}

	if len(config.TLSSan) > 0 {
		for _, san := range config.TLSSan {
			serverArgs = append(serverArgs, "--tls-san="+san)
		}
	}

	if config.ClusterCIDR != "" {
		serverArgs = append(serverArgs, "--cluster-cidr="+config.ClusterCIDR)
	}

	if config.ServiceCIDR != "" {
		serverArgs = append(serverArgs, "--service-cidr="+config.ServiceCIDR)
	}

	if config.ClusterDNS != "" {
		serverArgs = append(serverArgs, "--cluster-dns="+config.ClusterDNS)
	}

	if config.ClusterDomain != "" {
		serverArgs = append(serverArgs, "--cluster-domain="+config.ClusterDomain)
	}

	if config.FlannelBackend != "" {
		serverArgs = append(serverArgs, "--flannel-backend="+config.FlannelBackend)
	}

	if config.DataDir != "" {
		serverArgs = append(serverArgs, "--data-dir="+config.DataDir)
	}

	if config.WriteKubeconfig != "" {
		serverArgs = append(serverArgs, "--write-kubeconfig="+config.WriteKubeconfig)
	}

	if config.KubeconfigMode != "" {
		serverArgs = append(serverArgs, "--write-kubeconfig-mode="+config.KubeconfigMode)
	}

	// Disable components
	for _, component := range config.DisableComponents {
		serverArgs = append(serverArgs, "--disable="+component)
	}

	// Additional arguments
	for key, value := range config.ExtraArgs {
		if value != "" {
			serverArgs = append(serverArgs, fmt.Sprintf("--%s=%s", key, value))
		} else {
			serverArgs = append(serverArgs, "--"+key)
		}
	}

	if len(serverArgs) > 0 {
		installCmd.WriteString(" " + strings.Join(serverArgs, " "))
	}

	// Pre-installation commands
	commands = append(commands, 
		"sudo apt-get update", 
		"sudo apt-get install -y curl",
	)

	// Main installation command
	commands = append(commands, installCmd.String())

	// Post-installation commands
	commands = append(commands,
		"sudo systemctl enable k3s",
		"sudo systemctl start k3s",
		"sleep 30", // Wait for K3s to start
		"sudo k3s kubectl get nodes", // Verify installation
	)

	return commands
}

// generateWorkerInstallCommands generates the commands to install K3s agent (worker node)
func (s *ProvisioningService) generateWorkerInstallCommands(node *models.Node, config K3sConfig) []string {
	var commands []string
	var installCmd strings.Builder

	// Basic install command
	installCmd.WriteString("curl -sfL " + config.InstallScript + " | ")

	// Environment variables
	var envVars []string
	if config.Version != "" {
		envVars = append(envVars, "INSTALL_K3S_VERSION="+config.Version)
	}
	if config.Channel != "" {
		envVars = append(envVars, "INSTALL_K3S_CHANNEL="+config.Channel)
	}
	if config.ServerURL != "" {
		envVars = append(envVars, "K3S_URL="+config.ServerURL)
	}
	if config.ClusterToken != "" {
		envVars = append(envVars, "K3S_TOKEN="+config.ClusterToken)
	}

	if len(envVars) > 0 {
		installCmd.WriteString(strings.Join(envVars, " ") + " ")
	}

	installCmd.WriteString("sh -s - agent")

	// Agent arguments
	var agentArgs []string
	
	if config.NodeIP != "" {
		agentArgs = append(agentArgs, "--node-ip="+config.NodeIP)
	} else if node.IPAddress != "" {
		agentArgs = append(agentArgs, "--node-ip="+node.IPAddress)
	}

	if config.DataDir != "" {
		agentArgs = append(agentArgs, "--data-dir="+config.DataDir)
	}

	// Additional arguments
	for key, value := range config.ExtraArgs {
		if value != "" {
			agentArgs = append(agentArgs, fmt.Sprintf("--%s=%s", key, value))
		} else {
			agentArgs = append(agentArgs, "--"+key)
		}
	}

	if len(agentArgs) > 0 {
		installCmd.WriteString(" " + strings.Join(agentArgs, " "))
	}

	// Pre-installation commands
	commands = append(commands, 
		"sudo apt-get update", 
		"sudo apt-get install -y curl",
	)

	// Main installation command
	commands = append(commands, installCmd.String())

	// Post-installation commands
	commands = append(commands,
		"sudo systemctl enable k3s-agent",
		"sudo systemctl start k3s-agent",
		"sleep 30", // Wait for K3s agent to start
	)

	return commands
}

// updateNodeStatus updates the node status in the database
func (s *ProvisioningService) updateNodeStatus(nodeID uint, status models.NodeStatus) {
	updateReq := UpdateNodeRequest{
		Status: &status,
	}
	if _, err := s.nodeService.Update(nodeID, updateReq); err != nil {
		s.logger.WithError(err).WithField("node_id", nodeID).Error("Failed to update node status")
	}
}

// updateNodeKubeInfo updates node with Kubernetes-related information
func (s *ProvisioningService) updateNodeKubeInfo(nodeID uint, nodeName string) {
	// TODO: In a real implementation, we might want to query the node for actual K3s version
	kubeVersion := "v1.28.2+k3s1" // Default, should be detected
	
	updateReq := UpdateNodeRequest{
		NodeName:    &nodeName,
		KubeVersion: &kubeVersion,
	}
	if _, err := s.nodeService.Update(nodeID, updateReq); err != nil {
		s.logger.WithError(err).WithField("node_id", nodeID).Error("Failed to update node Kubernetes info")
	}
}

// DeprovisionNode removes K3s from a node and returns it to discovered state
func (s *ProvisioningService) DeprovisionNode(ctx context.Context, nodeID uint, sshConfig provisioner.SSHClientConfig) (*ProvisioningResult, error) {
	start := time.Now()

	// Get node details
	node, err := s.nodeService.GetByID(nodeID, false)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get node %d", nodeID)
	}

	s.logger.WithFields(map[string]interface{}{
		"node_id":   nodeID,
		"node_name": node.Name,
		"node_ip":   node.IPAddress,
	}).Info("Starting node deprovisioning")

	// Setup SSH connection
	sshConfig.Host = node.IPAddress
	if sshConfig.Username == "" {
		sshConfig.Username = "pi"
	}

	sshClient, err := provisioner.NewSSHClient(sshConfig, s.logger)
	if err != nil {
		return &ProvisioningResult{
			NodeID:   nodeID,
			NodeName: node.Name,
			Success:  false,
			Error:    fmt.Sprintf("failed to create SSH client: %v", err),
			Duration: time.Since(start),
		}, errors.Wrap(err, "failed to create SSH client")
	}
	defer sshClient.Close()

	// Generate deprovisioning commands
	commands := []string{
		"sudo systemctl stop k3s || sudo systemctl stop k3s-agent || true",
		"sudo systemctl disable k3s || sudo systemctl disable k3s-agent || true",
		"sudo /usr/local/bin/k3s-uninstall.sh || true", // For server nodes
		"sudo /usr/local/bin/k3s-agent-uninstall.sh || true", // For agent nodes
		"sudo rm -rf /var/lib/rancher/k3s || true",
		"sudo rm -rf /etc/rancher/k3s || true",
		"sudo rm -f /usr/local/bin/k3s* || true",
	}

	// Execute deprovisioning commands
	results, err := sshClient.ExecuteCommands(ctx, commands)
	
	success := err == nil
	var errorMsg string
	if !success {
		errorMsg = fmt.Sprintf("deprovisioning failed: %v", err)
		s.logger.WithError(err).WithField("node_id", nodeID).Error("Node deprovisioning failed")
	}

	// Update node status in database
	if err := s.nodeService.Deprovision(nodeID); err != nil {
		s.logger.WithError(err).WithField("node_id", nodeID).Error("Failed to update node status after deprovisioning")
	}

	duration := time.Since(start)
	s.logger.WithFields(map[string]interface{}{
		"node_id":   nodeID,
		"node_name": node.Name,
		"success":   success,
		"duration":  duration,
	}).Info("Node deprovisioning completed")

	return &ProvisioningResult{
		NodeID:   nodeID,
		NodeName: node.Name,
		Success:  success,
		Error:    errorMsg,
		Duration: duration,
		Commands: results,
	}, nil
}

// GetClusterToken retrieves the cluster token from a master node
func (s *ProvisioningService) GetClusterToken(ctx context.Context, masterNodeID uint, sshConfig provisioner.SSHClientConfig) (string, error) {
	node, err := s.nodeService.GetByID(masterNodeID, false)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get node %d", masterNodeID)
	}

	if node.Role != models.NodeRoleMaster {
		return "", fmt.Errorf("node %d is not a master node", masterNodeID)
	}

	sshConfig.Host = node.IPAddress
	if sshConfig.Username == "" {
		sshConfig.Username = "pi"
	}

	sshClient, err := provisioner.NewSSHClient(sshConfig, s.logger)
	if err != nil {
		return "", errors.Wrap(err, "failed to create SSH client")
	}
	defer sshClient.Close()

	result, err := sshClient.ExecuteCommand(ctx, "sudo cat /var/lib/rancher/k3s/server/node-token")
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve cluster token")
	}

	if !result.Success {
		return "", fmt.Errorf("failed to retrieve cluster token: %s", result.Stderr)
	}

	return strings.TrimSpace(result.Stdout), nil
}

// GetKubeConfig retrieves the kubeconfig from a master node
func (s *ProvisioningService) GetKubeConfig(ctx context.Context, masterNodeID uint, sshConfig provisioner.SSHClientConfig) (string, error) {
	node, err := s.nodeService.GetByID(masterNodeID, false)
	if err != nil {
		return "", errors.Wrapf(err, "failed to get node %d", masterNodeID)
	}

	if node.Role != models.NodeRoleMaster {
		return "", fmt.Errorf("node %d is not a master node", masterNodeID)
	}

	sshConfig.Host = node.IPAddress
	if sshConfig.Username == "" {
		sshConfig.Username = "pi"
	}

	sshClient, err := provisioner.NewSSHClient(sshConfig, s.logger)
	if err != nil {
		return "", errors.Wrap(err, "failed to create SSH client")
	}
	defer sshClient.Close()

	result, err := sshClient.ExecuteCommand(ctx, "sudo cat /etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return "", errors.Wrap(err, "failed to retrieve kubeconfig")
	}

	if !result.Success {
		return "", fmt.Errorf("failed to retrieve kubeconfig: %s", result.Stderr)
	}

	return result.Stdout, nil
}