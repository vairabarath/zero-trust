package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ztna-system/agent/client"
	"github.com/ztna-system/agent/config"
	"github.com/ztna-system/agent/enforcer"
	"github.com/ztna-system/agent/state"
	pb "github.com/ztna-system/proto"
)

const Version = "0.1.0"

func main() {
	// Parse command-line flags
	var (
		configPath  = flag.String("config", "config.yaml", "Path to configuration file")
		showVersion = flag.Bool("version", false, "Show version and exit")
	)
	flag.Parse()

	if *showVersion {
		fmt.Printf("ZTNA Agent v%s\n", Version)
		os.Exit(0)
	}

	log.Printf("========================================")
	log.Printf("ZTNA Agent v%s", Version)
	log.Printf("========================================")
	log.Println()

	// Check if running as root
	if os.Geteuid() != 0 {
		log.Fatal("✗ Error: Agent must run as root (required for iptables and TUN)")
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Printf("⚠️  Failed to load config, using defaults: %v", err)
		cfg = config.DefaultConfig()
	}

	log.Printf("Configuration loaded:")
	log.Printf("  - Controller: %s", cfg.Controller.Address)
	log.Printf("  - Hostname: %s", cfg.Agent.Hostname)
	log.Printf("  - TUN interface: %s", cfg.Agent.TunName)
	log.Println()

	// Initialize state manager
	log.Println("Step 1: Loading agent state...")
	stateManager := state.NewAgentState(cfg.Agent.StateFile)
	if err := stateManager.Load(); err != nil {
		log.Printf("⚠️  Failed to load state: %v", err)
		log.Println("  Starting with empty state")
	}
	log.Println()

	// Initialize TUN interface
	log.Println("Step 2: Creating TUN interface...")
	tunMgr := enforcer.NewTunManager(cfg.Agent.TunName)
	if err := tunMgr.Create(); err != nil {
		log.Fatalf("✗ Failed to create TUN interface: %v", err)
	}
	defer func() {
		log.Println("Cleaning up TUN interface...")
		tunMgr.Destroy()
	}()
	log.Println()

	// Initialize firewall enforcer
	log.Println("Step 3: Initializing firewall enforcer...")
	firewallEnforcer := enforcer.NewFirewallEnforcer(cfg.Agent.TunName)
	defer func() {
		log.Println("Cleaning up firewall rules...")
		firewallEnforcer.CleanupAll()
	}()

	// Restore protected ports from state
	protectedPorts := stateManager.GetProtectedPorts()
	if len(protectedPorts) > 0 {
		log.Printf("  Restoring %d protected ports from previous session...", len(protectedPorts))
		for _, p := range protectedPorts {
			if err := firewallEnforcer.ProtectPort(p.Port, p.Protocol); err != nil {
				log.Printf("⚠️  Failed to restore protection for port %d: %v", p.Port, err)
			}
		}
	}
	log.Println("✓ Firewall enforcer initialized")
	log.Println()

	// Main loop with reconnection
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start agent loop
	go runAgent(ctx, cfg, stateManager, firewallEnforcer)

	// Wait for interrupt signal
	<-sigChan
	log.Println()
	log.Println("========================================")
	log.Println("Shutting down gracefully...")
	log.Println("========================================")

	cancel()
	time.Sleep(2 * time.Second) // Give goroutines time to cleanup

	log.Println("✓ Agent stopped")
}

func runAgent(
	ctx context.Context,
	cfg *config.Config,
	stateManager *state.AgentState,
	firewallEnforcer *enforcer.FirewallEnforcer,
) {
	// Create reconnection manager
	reconnectMgr := client.NewReconnectManager(&client.MTLSConfig{
		CACertPath:     cfg.Controller.MTLSConfig.CACert,
		ClientCertPath: cfg.Controller.MTLSConfig.ClientCert,
		ClientKeyPath:  cfg.Controller.MTLSConfig.ClientKey,
		ServerAddress:  cfg.Controller.Address,
	})

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		// Connect to controller (with automatic retry)
		log.Println("Step 4: Connecting to controller...")
		conn, grpcClient, err := reconnectMgr.Connect(ctx)
		if err != nil {
			log.Printf("✗ Failed to connect: %v", err)
			return
		}
		defer conn.Close()

		// Register agent
		log.Println("Step 5: Registering with controller...")
		agentInfo := &pb.AgentInfo{
			Hostname:     cfg.Agent.Hostname,
			IpAddress:    "127.0.0.1", // TODO: Get actual IP
			Os:           "linux",
			Version:      Version,
			Capabilities: []string{"tcp_firewall", "tun_support"},
		}

		registration, err := grpcClient.RegisterAgent(ctx, agentInfo)
		if err != nil {
			log.Printf("✗ Registration failed: %v", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		if registration.Status != pb.RegistrationStatus_REGISTERED {
			log.Printf("✗ Registration rejected: %s", registration.ErrorMessage)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("✓ Registered successfully: agent_id=%s", registration.AgentId)
		stateManager.SetAgentID(registration.AgentId)
		stateManager.Save()
		log.Println()

		log.Println("========================================")
		log.Println("Agent is running")
		log.Println("========================================")
		log.Println("  - Connected to controller")
		log.Println("  - Waiting for commands...")
		log.Println()

		// Start stream handler
		streamHandler := client.NewStreamHandler(
			grpcClient,
			registration.AgentId,
			firewallEnforcer,
			stateManager,
		)

		if err := streamHandler.Start(ctx); err != nil {
			log.Printf("✗ Stream error: %v", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		// Wait for stream to close
		streamHandler.Wait()

		log.Println("Stream closed, reconnecting...")
		conn.Close()
		time.Sleep(2 * time.Second)
	}
}
