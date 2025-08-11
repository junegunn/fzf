//go:build windows

package fzf

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

func startProxyCommand(command string) (*exec.Cmd, error) {
	if command == "" {
		return nil, fmt.Errorf("empty command")
	}

	// Validate command to prevent injection
	if err := validateProxyCommand(command); err != nil {
		return nil, err
	}

	// Get shell path safely
	shellPath := os.Getenv("COMSPEC")
	if shellPath == "" {
		shellPath = "cmd.exe"
	}

	// Split command safely
	args, err := splitCommand(command)
	if err != nil {
		return nil, err
	}

	// Create command with proper argument handling
	cmd := exec.Command(shellPath, append([]string{"/c"}, args...)...)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	
	return cmd, nil
}

func validateProxyCommand(command string) error {
	// Check for dangerous characters that could be used for injection
	dangerousChars := regexp.MustCompile(`[;&|<>$` + "`" + `]`)
	if dangerousChars.MatchString(command) {
		return fmt.Errorf("command contains potentially dangerous characters")
	}

	// Ensure command doesn't start with dangerous prefixes
	trimmed := strings.TrimSpace(command)
	dangerousPrefixes := []string{"cmd", "powershell", "wscript", "cscript", "mshta"}
	for _, prefix := range dangerousPrefixes {
		if strings.HasPrefix(strings.ToLower(trimmed), prefix) {
			return fmt.Errorf("command starts with potentially dangerous prefix: %s", prefix)
		}
	}

	return nil
}

func splitCommand(command string) ([]string, error) {
	// Simple but safe command splitting
	// This avoids shell interpretation while preserving arguments
	args := []string{}
	current := ""
	inQuotes := false
	
	for i, char := range command {
		switch char {
		case '"':
			inQuotes = !inQuotes
			current += string(char)
		case ' ':
			if inQuotes {
				current += string(char)
			} else if current != "" {
				args = append(args, current)
				current = ""
			}
		default:
			current += string(char)
		}
	}
	
	if current != "" {
		args = append(args, current)
	}
	
	if inQuotes {
		return nil, fmt.Errorf("unmatched quotes in command")
	}
	
	return args, nil
}

func isExecutableAllowed(execPath string) bool {
	// Allowlist of common proxy executables
	allowedExecs := []string{
		"ssh.exe",
		"plink.exe", 
		"putty.exe",
		"nc.exe",
		"netcat.exe",
		"socat.exe",
	}
	
	execName := strings.ToLower(filepath.Base(execPath))
	for _, allowed := range allowedExecs {
		if execName == allowed {
			return true
		}
	}
	
	return false
}
