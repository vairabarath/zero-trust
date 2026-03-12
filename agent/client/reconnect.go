package client

import (
	"context"
	"log"
	"math"
	"time"

	pb "github.com/ztna-system/proto"
	"google.golang.org/grpc"
)

// ReconnectManager handles automatic reconnection with exponential backoff
type ReconnectManager struct {
	config         *MTLSConfig
	maxRetries     int
	baseDelay      time.Duration
	maxDelay       time.Duration
	currentAttempt int
}

// NewReconnectManager creates a new reconnection manager
func NewReconnectManager(config *MTLSConfig) *ReconnectManager {
	return &ReconnectManager{
		config:     config,
		maxRetries: 0, // 0 means infinite retries
		baseDelay:  1 * time.Second,
		maxDelay:   60 * time.Second,
	}
}

// Connect attempts to connect with automatic retry
func (rm *ReconnectManager) Connect(ctx context.Context) (*grpc.ClientConn, pb.ZTNAServiceClient, error) {
	for {
		// Try to connect
		conn, err := CreateConnection(rm.config)
		if err == nil {
			// Success!
			rm.currentAttempt = 0
			client := pb.NewZTNAServiceClient(conn)
			return conn, client, nil
		}

		// Connection failed
		rm.currentAttempt++
		delay := rm.calculateBackoff()

		log.Printf("✗ Connection failed (attempt %d): %v", rm.currentAttempt, err)
		log.Printf("  Retrying in %v...", delay)

		// Wait before retry (respecting context cancellation)
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}
}

// calculateBackoff calculates the backoff delay using exponential backoff
func (rm *ReconnectManager) calculateBackoff() time.Duration {
	// Exponential backoff: delay = baseDelay * 2^attempt
	delay := time.Duration(float64(rm.baseDelay) * math.Pow(2, float64(rm.currentAttempt)))

	// Cap at maxDelay
	if delay > rm.maxDelay {
		delay = rm.maxDelay
	}

	return delay
}

// ResetAttempts resets the retry counter (call after successful connection)
func (rm *ReconnectManager) ResetAttempts() {
	rm.currentAttempt = 0
}
