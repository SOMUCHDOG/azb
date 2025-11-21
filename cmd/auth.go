package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/casey/azure-boards-cli/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	patFlag string

	authCmd = &cobra.Command{
		Use:   "auth",
		Short: "Manage authentication",
		Long:  `Authenticate with Azure DevOps using a Personal Access Token (PAT).`,
	}

	loginCmd = &cobra.Command{
		Use:   "login",
		Short: "Authenticate with Azure DevOps",
		Long:  `Authenticate with Azure DevOps using a Personal Access Token (PAT).`,
		RunE:  runLogin,
	}

	logoutCmd = &cobra.Command{
		Use:   "logout",
		Short: "Sign out of Azure DevOps",
		Long:  `Remove stored authentication credentials.`,
		RunE:  runLogout,
	}

	statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check authentication status",
		Long:  `Check if you are currently authenticated with Azure DevOps.`,
		RunE:  runStatus,
	}
)

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(loginCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(statusCmd)

	loginCmd.Flags().StringVar(&patFlag, "pat", "", "Personal Access Token")
}

func runLogin(cmd *cobra.Command, args []string) error {
	var token string

	if patFlag != "" {
		token = patFlag
	} else {
		// Prompt for PAT
		fmt.Println("Enter your Personal Access Token (PAT):")
		fmt.Println("You can create a PAT at: https://dev.azure.com/{org}/_usersSettings/tokens")
		fmt.Print("PAT: ")

		// Read password from terminal without echoing
		bytePwd, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("failed to read token: %w", err)
		}
		fmt.Println() // Add newline after password input

		token = strings.TrimSpace(string(bytePwd))
	}

	if token == "" {
		return fmt.Errorf("token cannot be empty")
	}

	// Save token
	if err := auth.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("✓ Authentication successful")
	fmt.Println("✓ Token saved")

	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	if !auth.IsAuthenticated() {
		fmt.Println("Not currently authenticated")
		return nil
	}

	// Confirm logout
	fmt.Print("Are you sure you want to sign out? (y/N): ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response != "y" && response != "yes" {
		fmt.Println("Logout cancelled")
		return nil
	}

	if err := auth.Logout(); err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}

	fmt.Println("✓ Signed out successfully")

	return nil
}

func runStatus(cmd *cobra.Command, args []string) error {
	if auth.IsAuthenticated() {
		fmt.Println("✓ Authenticated")
		tokenPath, _ := auth.GetTokenPath()
		fmt.Printf("Token stored at: %s\n", tokenPath)
	} else {
		fmt.Println("✗ Not authenticated")
		fmt.Println("Run 'azb auth login' to authenticate")
	}

	return nil
}
