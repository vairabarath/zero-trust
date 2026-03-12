package enforcer

import (
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"sync"
)

// FirewallEnforcer manages nftables rules for port protection
type FirewallEnforcer struct {
	tunName        string
	tableName      string // nftables table name
	chainName      string // nftables chain name
	protectedPorts map[int]*ProtectionRule
	mu             sync.RWMutex
}

// ProtectionRule tracks a protected port
type ProtectionRule struct {
	Port     int
	Protocol string
	Active   bool
	RuleIDs  []string // Track which nftables rules were added
}

// NewFirewallEnforcer creates a new firewall enforcer
func NewFirewallEnforcer(tunName string) *FirewallEnforcer {
	fe := &FirewallEnforcer{
		tunName:        tunName,
		tableName:      "ztna",
		chainName:      "input_filter",
		protectedPorts: make(map[int]*ProtectionRule),
	}

	// Initialize nftables table and chain
	if err := fe.initializeNftables(); err != nil {
		log.Printf("⚠️  Failed to initialize nftables: %v", err)
	}

	return fe
}

// initializeNftables sets up the nftables table and chain
func (f *FirewallEnforcer) initializeNftables() error {
	log.Println("Initializing nftables table and chain...")

	// Check if nftables is available
	if err := exec.Command("nft", "--version").Run(); err != nil {
		return fmt.Errorf("nftables not available: %v (install with: sudo apt install nftables)", err)
	}

	// Create table if it doesn't exist
	// Using 'inet' family for both IPv4 and IPv6
	cmd := exec.Command("nft", "add", "table", "inet", f.tableName)
	if err := cmd.Run(); err != nil {
		// Table might already exist, that's OK
		log.Printf("Table creation: %v (may already exist)", err)
	}

	// Create chain with high priority to hook into input
	// Priority -150 is before conntrack (priority -200) but after other filters
	cmd = exec.Command("nft", "add", "chain", "inet", f.tableName, f.chainName,
		"{ type filter hook input priority -150 ; policy accept ; }")
	if err := cmd.Run(); err != nil {
		log.Printf("Chain creation: %v (may already exist)", err)
	}

	log.Printf("✓ nftables initialized: table=%s, chain=%s", f.tableName, f.chainName)
	return nil
}

// ProtectPort applies nftables rules to protect a port
func (f *FirewallEnforcer) ProtectPort(port int, protocol string) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Check if already protected
	if rule, exists := f.protectedPorts[port]; exists && rule.Active {
		return fmt.Errorf("port %d is already protected", port)
	}

	portStr := strconv.Itoa(port)

	log.Printf("Applying nftables rules for port %d (%s)...", port, protocol)

	// nftables handles all rules in a single chain with better performance
	// We use a named handle system for easier rule management

	// Rule 1: ACCEPT from localhost (lo interface)
	handle1, err := f.addRule(fmt.Sprintf(
		"iifname \"lo\" tcp dport %s accept",
		portStr,
	))
	if err != nil {
		return fmt.Errorf("failed to add localhost rule: %v", err)
	}

	// Rule 2: ACCEPT from TUN interface
	handle2, err := f.addRule(fmt.Sprintf(
		"iifname \"%s\" tcp dport %s accept",
		f.tunName, portStr,
	))
	if err != nil {
		// Rollback rule 1
		f.deleteRuleByHandle(handle1)
		return fmt.Errorf("failed to add TUN rule: %v", err)
	}

	// Rule 3: DROP everything else to this port
	handle3, err := f.addRule(fmt.Sprintf(
		"tcp dport %s drop",
		portStr,
	))
	if err != nil {
		// Rollback
		f.deleteRuleByHandle(handle1)
		f.deleteRuleByHandle(handle2)
		return fmt.Errorf("failed to add DROP rule: %v", err)
	}

	// Store the rule with handles
	f.protectedPorts[port] = &ProtectionRule{
		Port:     port,
		Protocol: protocol,
		Active:   true,
		RuleIDs:  []string{handle1, handle2, handle3},
	}

	log.Printf("✓ Port %d is now protected (handles: %s, %s, %s)", port, handle1, handle2, handle3)
	return nil
}

// UnprotectPort removes nftables rules for a port
func (f *FirewallEnforcer) UnprotectPort(port int) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	rule, exists := f.protectedPorts[port]
	if !exists || !rule.Active {
		return fmt.Errorf("port %d is not protected", port)
	}

	log.Printf("Removing nftables rules for port %d...", port)

	errors := []error{}

	// Delete rules by their handles
	for _, handle := range rule.RuleIDs {
		if err := f.deleteRuleByHandle(handle); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		log.Printf("⚠️  Some rules failed to remove: %v", errors)
	}

	// Mark as inactive
	rule.Active = false
	delete(f.protectedPorts, port)

	log.Printf("✓ Port %d is now unprotected", port)
	return nil
}

// GetProtectedPorts returns list of currently protected ports
func (f *FirewallEnforcer) GetProtectedPorts() []int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	ports := make([]int, 0, len(f.protectedPorts))
	for port, rule := range f.protectedPorts {
		if rule.Active {
			ports = append(ports, port)
		}
	}
	return ports
}

// IsPortProtected checks if a port is protected
func (f *FirewallEnforcer) IsPortProtected(port int) bool {
	f.mu.RLock()
	defer f.mu.RUnlock()

	rule, exists := f.protectedPorts[port]
	return exists && rule.Active
}

// addRule adds an nftables rule and returns its handle
func (f *FirewallEnforcer) addRule(ruleSpec string) (string, error) {
	// Add rule with echo option to get handle
	cmd := exec.Command("nft", "-ae", "add", "rule", "inet", f.tableName, f.chainName, ruleSpec)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("nft error: %v, output: %s", err, string(output))
	}

	// Output format: "add rule inet ztna input_filter tcp dport 5173 accept # handle 5"
	// Parse to extract handle number
	outputStr := string(output)
	if idx := lastIndex(outputStr, "handle "); idx != -1 {
		handleStr := outputStr[idx+7:] // Skip "handle "
		// Trim any whitespace/newlines
		handleStr = trimSpace(handleStr)
		return handleStr, nil
	}

	// If we can't parse handle, return empty (will use alternative deletion method)
	return "", nil
}

// deleteRuleByHandle deletes an nftables rule by handle
func (f *FirewallEnforcer) deleteRuleByHandle(handle string) error {
	if handle == "" {
		// No handle available, skip
		return nil
	}

	cmd := exec.Command("nft", "delete", "rule", "inet", f.tableName, f.chainName, "handle", handle)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("nft error: %v, output: %s", err, string(output))
	}
	return nil
}

// Helper functions
func lastIndex(s, substr string) int {
	for i := len(s) - len(substr); i >= 0; i-- {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

func trimSpace(s string) string {
	// Simple whitespace trimmer
	start := 0
	end := len(s)

	for start < end && (s[start] == ' ' || s[start] == '\n' || s[start] == '\t' || s[start] == '\r') {
		start++
	}

	for end > start && (s[end-1] == ' ' || s[end-1] == '\n' || s[end-1] == '\t' || s[end-1] == '\r') {
		end--
	}

	return s[start:end]
}

func contains(haystack, needle string) bool {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}

func (f *FirewallEnforcer) CleanupAll() {
	f.mu.Lock()
	defer f.mu.Unlock()

	log.Printf("🧹 Cleaning up all firewall rules in table %s...", f.tableName)

	// Deleting the table removes all chains and rules within it
	cmd := exec.Command("nft", "delete", "table", "inet", f.tableName)
	if err := cmd.Run(); err != nil {
		log.Printf("⚠️  Note: Could not delete table (it may have been already removed): %v", err)
	} else {
		log.Printf("✓ nftables table %s removed", f.tableName)
	}

	// Reset the local map
	f.protectedPorts = make(map[int]*ProtectionRule)
}
