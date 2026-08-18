package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// RootCmd - это базовая команда 'zope'
var RootCmd = &cobra.Command{
	Use:   "zope",
	Short: "zope - мульти-таргет оркестратор фаззинга",
	Long:  `Автоматизированный инструмент для управления сессиями фаззинга.`,
}

// Execute добавляет дочерние команды к корневой и запускает парсинг флагов CLI
func Execute() {
	if err := RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Здесь в будущем будут настраиваться глобальные флаги (например, --verbose)
}
