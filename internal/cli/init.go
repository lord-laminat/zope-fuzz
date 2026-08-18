package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// initCmd represents the 'zope init' command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new zope fuzzing project",
	Long:  `Creates the internal .zope directory structure, generates an empty zope.toml configuration file, and updates .gitignore.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Initializing zope fuzzing environment...")

		// 1. Ensure the internal .zope directory structure exists
		dirsToCreate := []string{
			filepath.Join(".zope", "build", "context"),
			filepath.Join(".zope", "targets"),
			filepath.Join(".zope", "reports"),
		}

		for _, dir := range dirsToCreate {
			if err := os.MkdirAll(dir, 0755); err != nil {
				fmt.Printf("[ERROR] Failed to create directory %s: %v\n", dir, err)
				os.Exit(1)
			}
		}
		fmt.Println("  ├── Ensured internal directory structure under .zope/")

		// 2. Генерируем пустой файл конфигурации zope.toml со ссылкой на документацию
		configPath := "zope.toml"
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			defaultComment := "# For configuration guide, see: https://github.com/lord-laminat/zope-fuzz/blob/main/docs/CONFIGURATION.md\n"

			if err := os.WriteFile(configPath, []byte(defaultComment), 0644); err != nil {
				fmt.Printf("[ERROR] Failed to create %s: %v\n", configPath, err)
				os.Exit(1)
			}
			fmt.Println("  ├── Generated configuration file: zope.toml")
		} else {
			fmt.Println("  ├── Configuration file zope.toml already exists, skipping generation")
		}

		// 3. Настраиваем .gitignore для скрытия папки .zope/
		if err := updateGitignore(); err != nil {
			fmt.Printf("[ERROR] Failed to update .gitignore: %v\n", err)
			os.Exit(1)
		}
	},
}

// updateGitignore добавляет служебную папку в .gitignore проекта
func updateGitignore() error {
	gitignorePath := ".gitignore"
	entry := ".zope/\n"

	// Если .gitignore не существует, создаем его с записью .zope/
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		err := os.WriteFile(gitignorePath, []byte(entry), 0644)
		if err != nil {
			return err
		}
		fmt.Println("  └── Created .gitignore and added .zope/")
		return nil
	}

	// Если файл существует, проверяем его содержимое
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return err
	}

	// Проверяем наличие подстроки стандартными средствами пакета bytes
	if bytes.Contains(content, []byte(".zope/")) {
		fmt.Println("  └── .zope/ is already present in .gitignore")
		return nil
	}

	// Открываем существующий файл на дозапись (Append)
	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	// Добавляем перенос строки перед записью на случай, если файл не заканчивался пустой строкой
	if len(content) > 0 && content[len(content)-1] != '\n' {
		if _, err := f.WriteString("\n"); err != nil {
			return err
		}
	}

	if _, err := f.WriteString(entry); err != nil {
		return err
	}

	fmt.Println("  └── Added .zope/ to .gitignore")
	return nil
}

func init() {
	RootCmd.AddCommand(initCmd)
}
