package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// MTLSConfig holds mTLS configuration for the client
type MTLSConfig struct {
	CACertPath     string
	ClientCertPath string
	ClientKeyPath  string
	ServerAddress  string
}

// LoadClientCredentials creates TLS credentials for the gRPC client with mTLS
func LoadClientCredentials(config *MTLSConfig) (credentials.TransportCredentials, error) {
	// Load client certificate and key
	clientCert, err := tls.LoadX509KeyPair(config.ClientCertPath, config.ClientKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %v", err)
	}

	// Load CA certificate
	caCert, err := os.ReadFile(config.CACertPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %v", err)
	}

	// Create certificate pool with our CA
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to add CA certificate to pool")
	}

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      certPool,
		MinVersion:   tls.VersionTLS12,
	}

	log.Println("✓ mTLS client credentials loaded")
	log.Printf("  - Client cert: %s", config.ClientCertPath)
	log.Printf("  - CA cert: %s", config.CACertPath)

	return credentials.NewTLS(tlsConfig), nil
}

// CreateConnection creates a gRPC connection to the controller
func CreateConnection(config *MTLSConfig) (*grpc.ClientConn, error) {
	// Load mTLS credentials
	creds, err := LoadClientCredentials(config)
	if err != nil {
		return nil, err
	}

	// Create connection with mTLS and keepalive
	conn, err := grpc.NewClient(
		config.ServerAddress,
		grpc.WithTransportCredentials(creds),
		grpc.WithBlock(), // Wait for connection
	)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to controller: %v", err)
	}

	log.Printf("✓ Connected to controller: %s", config.ServerAddress)
	return conn, nil
}
