package client

import (
	"context"
	"fmt"
	"io"
	"log"
	"time"

	"github.com/ztna-system/agent/enforcer"
	"github.com/ztna-system/agent/state"
	pb "github.com/ztna-system/proto"
)

// StreamHandler handles the bidirectional stream with the controller
type StreamHandler struct {
	client     pb.ZTNAServiceClient
	agentID    string
	enforcer   *enforcer.FirewallEnforcer
	state      *state.AgentState
	stream     pb.ZTNAService_StreamCommandsClient
	stopChan   chan struct{}
	doneChan   chan struct{}
}

// NewStreamHandler creates a new stream handler
func NewStreamHandler(
	client pb.ZTNAServiceClient,
	agentID string,
	enf *enforcer.FirewallEnforcer,
	st *state.AgentState,
) *StreamHandler {
	return &StreamHandler{
		client:   client,
		agentID:  agentID,
		enforcer: enf,
		state:    st,
		stopChan: make(chan struct{}),
		doneChan: make(chan struct{}),
	}
}

// Start starts the stream and begins listening for commands
func (sh *StreamHandler) Start(ctx context.Context) error {
	log.Println("Opening bidirectional stream...")

	stream, err := sh.client.StreamCommands(ctx)
	if err != nil {
		return fmt.Errorf("failed to open stream: %v", err)
	}

	sh.stream = stream
	log.Println("✓ Stream opened successfully")

	// Start listening for commands
	go sh.receiveCommands()

	// Start heartbeat
	go sh.sendHeartbeats()

	return nil
}

// receiveCommands listens for commands from the controller
func (sh *StreamHandler) receiveCommands() {
	defer close(sh.doneChan)

	for {
		select {
		case <-sh.stopChan:
			log.Println("Stop signal received, closing stream...")
			return
		default:
		}

		// Receive command from controller
		command, err := sh.stream.Recv()
		if err == io.EOF {
			log.Println("Stream closed by controller")
			return
		}
		if err != nil {
			log.Printf("✗ Stream error: %v", err)
			return
		}

		// Process the command
		sh.handleCommand(command)
	}
}

// handleCommand processes a command from the controller
func (sh *StreamHandler) handleCommand(cmd *pb.Command) {
	log.Printf("← Received command: type=%s, id=%s", cmd.Type, cmd.CommandId)

	var response *pb.CommandResponse

	switch cmd.Type {
	case pb.CommandType_PROTECT:
		response = sh.handleProtectCommand(cmd)
	case pb.CommandType_UNPROTECT:
		response = sh.handleUnprotectCommand(cmd)
	case pb.CommandType_PING:
		response = sh.handlePingCommand(cmd)
	case pb.CommandType_SHUTDOWN:
		response = sh.handleShutdownCommand(cmd)
	default:
		response = &pb.CommandResponse{
			CommandId:    cmd.CommandId,
			Status:       pb.CommandStatus_FAILED,
			ErrorMessage: fmt.Sprintf("Unknown command type: %s", cmd.Type),
			Timestamp:    time.Now().Unix(),
		}
	}

	// Send response back to controller
	if err := sh.stream.Send(response); err != nil {
		log.Printf("✗ Failed to send response: %v", err)
	} else {
		log.Printf("→ Response sent: command=%s, status=%s", response.CommandId, response.Status)
	}
}

// handleProtectCommand handles a PROTECT command
func (sh *StreamHandler) handleProtectCommand(cmd *pb.Command) *pb.CommandResponse {
	if cmd.Protect == nil {
		return &pb.CommandResponse{
			CommandId:    cmd.CommandId,
			Status:       pb.CommandStatus_FAILED,
			ErrorMessage: "Missing protect request data",
			Timestamp:    time.Now().Unix(),
		}
	}

	port := int(cmd.Protect.Port)
	protocol := cmd.Protect.Protocol.String()

	log.Printf("Protecting port %d (%s)...", port, protocol)

	// Apply firewall rules
	if err := sh.enforcer.ProtectPort(port, protocol); err != nil {
		log.Printf("✗ Failed to protect port %d: %v", port, err)
		return &pb.CommandResponse{
			CommandId:    cmd.CommandId,
			Status:       pb.CommandStatus_FAILED,
			ErrorMessage: err.Error(),
			Timestamp:    time.Now().Unix(),
		}
	}

	// Update state
	sh.state.AddProtectedPort(port, protocol, cmd.CommandId)

	log.Printf("✓ Port %d protected successfully", port)

	return &pb.CommandResponse{
		CommandId: cmd.CommandId,
		Status:    pb.CommandStatus_SUCCESS,
		Timestamp: time.Now().Unix(),
		Details: map[string]string{
			"port":     fmt.Sprintf("%d", port),
			"protocol": protocol,
		},
	}
}

// handleUnprotectCommand handles an UNPROTECT command
func (sh *StreamHandler) handleUnprotectCommand(cmd *pb.Command) *pb.CommandResponse {
	if cmd.Unprotect == nil {
		return &pb.CommandResponse{
			CommandId:    cmd.CommandId,
			Status:       pb.CommandStatus_FAILED,
			ErrorMessage: "Missing unprotect request data",
			Timestamp:    time.Now().Unix(),
		}
	}

	port := int(cmd.Unprotect.Port)

	log.Printf("Unprotecting port %d...", port)

	// Remove firewall rules
	if err := sh.enforcer.UnprotectPort(port); err != nil {
		log.Printf("✗ Failed to unprotect port %d: %v", port, err)
		return &pb.CommandResponse{
			CommandId:    cmd.CommandId,
			Status:       pb.CommandStatus_FAILED,
			ErrorMessage: err.Error(),
			Timestamp:    time.Now().Unix(),
		}
	}

	// Update state
	sh.state.RemoveProtectedPort(port)

	log.Printf("✓ Port %d unprotected successfully", port)

	return &pb.CommandResponse{
		CommandId: cmd.CommandId,
		Status:    pb.CommandStatus_SUCCESS,
		Timestamp: time.Now().Unix(),
		Details: map[string]string{
			"port": fmt.Sprintf("%d", port),
		},
	}
}

// handlePingCommand handles a PING command
func (sh *StreamHandler) handlePingCommand(cmd *pb.Command) *pb.CommandResponse {
	log.Println("PING received")

	return &pb.CommandResponse{
		CommandId: cmd.CommandId,
		Status:    pb.CommandStatus_SUCCESS,
		Timestamp: time.Now().Unix(),
		Details: map[string]string{
			"message": "pong",
		},
	}
}

// handleShutdownCommand handles a SHUTDOWN command
func (sh *StreamHandler) handleShutdownCommand(cmd *pb.Command) *pb.CommandResponse {
	log.Println("SHUTDOWN command received")

	// Send success response first
	response := &pb.CommandResponse{
		CommandId: cmd.CommandId,
		Status:    pb.CommandStatus_SUCCESS,
		Timestamp: time.Now().Unix(),
		Details: map[string]string{
			"message": "shutting down",
		},
	}

	// Trigger shutdown after a delay
	go func() {
		time.Sleep(1 * time.Second)
		close(sh.stopChan)
	}()

	return response
}

// sendHeartbeats sends periodic heartbeat messages
func (sh *StreamHandler) sendHeartbeats() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-sh.stopChan:
			return
		case <-ticker.C:
			// Send heartbeat as a status report
			protectedPorts := sh.enforcer.GetProtectedPorts()
			ports32 := make([]int32, len(protectedPorts))
			for i, p := range protectedPorts {
				ports32[i] = int32(p)
			}

			// Send status report to update heartbeat
			status := &pb.AgentStatus{
				AgentId:        sh.agentID,
				Status:         pb.AgentHealthStatus_ONLINE,
				ProtectedPorts: ports32,
				Timestamp:      time.Now().Unix(),
			}

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := sh.client.ReportStatus(ctx, status)
			cancel()

			if err != nil {
				log.Printf("⚠️  Failed to send heartbeat: %v", err)
			} else {
				log.Printf("💓 Heartbeat sent (protected ports: %v)", protectedPorts)
			}
		}
	}
}

// Stop stops the stream handler
func (sh *StreamHandler) Stop() {
	close(sh.stopChan)
	<-sh.doneChan
	log.Println("✓ Stream handler stopped")
}

// Wait waits for the stream to close
func (sh *StreamHandler) Wait() {
	<-sh.doneChan
}
