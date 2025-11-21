package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/casey/azure-boards-cli/internal/templates"
	"github.com/spf13/cobra"
)

var (
	templateEditCmd = &cobra.Command{
		Use:   "edit <template-name>",
		Short: "Edit a template file",
		Long:  `Open a template file in your default editor ($EDITOR or $VISUAL).`,
		Args:  cobra.ExactArgs(1),
		RunE:  runTemplateEdit,
	}

	templatePathCmd = &cobra.Command{
		Use:   "path [template-name]",
		Short: "Show template file path",
		Long:  `Show the path to a template file or the templates directory.`,
		Args:  cobra.MaximumNArgs(1),
		RunE:  runTemplatePath,
	}
)

func init() {
	templateCmd.AddCommand(templateEditCmd)
	templateCmd.AddCommand(templatePathCmd)
}

func runTemplateEdit(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Check if template exists
	exists, err := templates.Exists(name)
	if err != nil {
		return err
	}

	if !exists {
		return fmt.Errorf("template '%s' not found. Use 'azb template init %s <type>' to create it", name, name)
	}

	// Get template path
	path, err := templates.GetTemplatePath(name)
	if err != nil {
		return err
	}

	// Get editor from environment
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = os.Getenv("VISUAL")
	}
	if editor == "" {
		editor = "vi" // fallback to vi
	}

	// Open editor
	editorCmd := exec.Command(editor, path)
	editorCmd.Stdin = os.Stdin
	editorCmd.Stdout = os.Stdout
	editorCmd.Stderr = os.Stderr

	if err := editorCmd.Run(); err != nil {
		return fmt.Errorf("failed to open editor: %w", err)
	}

	fmt.Printf("\n✓ Template '%s' updated\n", name)
	fmt.Println("\nUse 'azb template show %s' to view the changes", name)

	return nil
}

func runTemplatePath(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Show templates directory
		dir, err := templates.GetTemplatesDir()
		if err != nil {
			return err
		}
		fmt.Println(dir)
		return nil
	}

	// Show specific template path
	name := args[0]
	path, err := templates.GetTemplatePath(name)
	if err != nil {
		return err
	}

	fmt.Println(path)
	return nil
}
