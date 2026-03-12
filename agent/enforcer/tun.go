package enforcer

import (
	"fmt"
	"log"
	"os/exec"
)

// TunManager manages the TUN interface
type TunManager struct {
	Name string
}

// NewTunManager creates a new TUN manager
func NewTunManager(name string) *TunManager {
	return &TunManager{
		Name: name,
	}
}

// Create creates and brings up the TUN interface
func (t *TunManager) Create() error {
	log.Printf("Creating TUN interface: %s", t.Name)

	// Clean up any existing interface with this name
	exec.Command("ip", "link", "delete", t.Name).Run() // Best effort

	// Create TUN device
	cmd := exec.Command("ip", "tuntap", "add", "dev", t.Name, "mode", "tun")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create TUN device: %v", err)
	}

	// Assign IP address
	cmd = exec.Command("ip", "addr", "add", "10.99.99.2/24", "dev", t.Name)
	if err := cmd.Run(); err != nil {
		// Cleanup on failure
		exec.Command("ip", "link", "delete", t.Name).Run()
		return fmt.Errorf("failed to assign IP to TUN device: %v", err)
	}

	// Bring the interface up
	cmd = exec.Command("ip", "link", "set", t.Name, "up")
	if err := cmd.Run(); err != nil {
		// Cleanup on failure
		exec.Command("ip", "link", "delete", t.Name).Run()
		return fmt.Errorf("failed to bring up TUN device: %v", err)
	}

	log.Printf("✓ TUN interface %s created (10.99.99.2/24)", t.Name)
	return nil
}

// Destroy removes the TUN interface
func (t *TunManager) Destroy() error {
	log.Printf("Destroying TUN interface: %s", t.Name)

	// Bring down the interface
	cmd := exec.Command("ip", "link", "set", t.Name, "down")
	cmd.Run() // Best effort

	// Delete the TUN device
	cmd = exec.Command("ip", "link", "delete", t.Name)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete TUN device: %v", err)
	}

	log.Printf("✓ TUN interface %s destroyed", t.Name)
	return nil
}

// Exists checks if the TUN interface exists
func (t *TunManager) Exists() bool {
	cmd := exec.Command("ip", "link", "show", t.Name)
	return cmd.Run() == nil
}

// IsUp checks if the TUN interface is up
func (t *TunManager) IsUp() bool {
	cmd := exec.Command("ip", "link", "show", "up", t.Name)
	return cmd.Run() == nil
}
