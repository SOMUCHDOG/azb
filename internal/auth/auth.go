package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	tokenFileName = "token"
)

// GetTokenPath returns the path to the token file
func GetTokenPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".azure-boards-cli")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %w", err)
	}

	return filepath.Join(configDir, tokenFileName), nil
}

// SaveToken saves the Personal Access Token to a secure file
func SaveToken(token string) error {
	tokenPath, err := GetTokenPath()
	if err != nil {
		return err
	}

	// Write token to file with secure permissions (owner read/write only)
	if err := os.WriteFile(tokenPath, []byte(token), 0600); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	return nil
}

// GetToken retrieves the stored Personal Access Token
func GetToken() (string, error) {
	tokenPath, err := GetTokenPath()
	if err != nil {
		return "", err
	}

	// Check if token file exists
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return "", fmt.Errorf("not authenticated. Run 'azb auth login' to authenticate")
	}

	// Read token from file
	tokenBytes, err := os.ReadFile(tokenPath)
	if err != nil {
		return "", fmt.Errorf("failed to read token: %w", err)
	}

	token := strings.TrimSpace(string(tokenBytes))
	if token == "" {
		return "", fmt.Errorf("token file is empty. Run 'azb auth login' to authenticate")
	}

	return token, nil
}

// IsAuthenticated checks if the user is authenticated
func IsAuthenticated() bool {
	_, err := GetToken()
	return err == nil
}

// Logout removes the stored token
func Logout() error {
	tokenPath, err := GetTokenPath()
	if err != nil {
		return err
	}

	// Check if token file exists
	if _, err := os.Stat(tokenPath); os.IsNotExist(err) {
		return fmt.Errorf("not authenticated")
	}

	// Remove token file
	if err := os.Remove(tokenPath); err != nil {
		return fmt.Errorf("failed to remove token: %w", err)
	}

	return nil
}
