package cli

import (
	"fmt"
	"os"
	"strings"
	"zope/internal/config"

	"github.com/spf13/cobra"
)

// configCmd represents the 'zope config' command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Validate and display the current project configuration",
	Long:  `Reads the zope.toml file from the project root, validates data types, checks structure, and prints a summary.`,
	Run: func(cmd *cobra.Command, args []string) {

		cfg, err := config.LoadConfig("zope.toml")
		if err != nil {
			// Ошибки перенаправляем в Stderr
			fmt.Fprintf(os.Stderr, "[ERROR] Load error: %v\n", err)
			os.Exit(1)
		}

		if err := cfg.Validate(); err != nil {
			// Ошибки перенаправляем в Stderr
			fmt.Fprintf(os.Stderr, "[ERROR] Configuration is invalid: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Project Name : %s\n", cfg.Project.Name)
		fmt.Printf("Runtime      : %s (Base Image: %s)\n", cfg.Runtime.Engine, cfg.Runtime.BaseImage)

		// -----------------------------------------------------------------
		// Отрисовка секции Targets
		// -----------------------------------------------------------------
		targetsCount := len(cfg.Targets)
		fmt.Printf("Fuzzing Targets (%d):\n", targetsCount)

		for i, target := range cfg.Targets {
			var targetPrefix string
			var childPadding string

			if i == targetsCount-1 {
				targetPrefix = "  └── "
				childPadding = "      "
			} else {
				targetPrefix = "  ├── "
				childPadding = "  │   "
			}

			fmt.Printf("%s%d) %s\n", targetPrefix, i+1, target.Name)

			// Проверки на наследование базовых полей
			langLabel := getInheritLabel(target.Language == cfg.Defaults.Language && cfg.Defaults.Language != "")
			engLabel := getInheritLabel(target.Engine == cfg.Defaults.Engine && cfg.Defaults.Engine != "")

			fmt.Printf("%s├── Language: %s%s\n", childPadding, target.Language, langLabel)
			fmt.Printf("%s├── Engine:   %s%s\n", childPadding, target.Engine, engLabel)

			// Опциональные поля сборки
			if target.CompileFlags != "" {
				compLabel := getInheritLabel(target.CompileFlags == cfg.Defaults.CompileFlags && cfg.Defaults.CompileFlags != "")
				fmt.Printf("%s├── Compile Flags: %s%s\n", childPadding, target.CompileFlags, compLabel)
			}
			if target.LinkFlags != "" {
				linkLabel := getInheritLabel(target.LinkFlags == cfg.Defaults.LinkFlags && cfg.Defaults.LinkFlags != "")
				fmt.Printf("%s├── Link Flags:    %s%s\n", childPadding, target.LinkFlags, linkLabel)
			}
			if len(target.Sanitizers) > 0 {
				targetSanStr := strings.Join(target.Sanitizers, ", ")
				defSanStr := strings.Join(cfg.Defaults.Sanitizers, ", ")
				sanLabel := getInheritLabel(targetSanStr == defSanStr && defSanStr != "")
				fmt.Printf("%s├── Sanitizers:    %s%s\n", childPadding, targetSanStr, sanLabel)
			}

			stopConditions := target.StopConditions
			hasStopConditions := stopConditions.MaxTimeSinceLastPath != "" || stopConditions.MinCoveragePct > 0 || stopConditions.MaxTotalTime != ""

			if hasStopConditions {
				fmt.Printf("%s├── Source:   %s\n", childPadding, target.Source)
				fmt.Printf("%s└── Smart Stop Conditions:\n", childPadding)
				printStopConditions(childPadding, stopConditions, cfg.Defaults.StopConditions)
			} else {
				fmt.Printf("%s└── Source:   %s\n", childPadding, target.Source)
			}
		}
	},
}

// printStopConditions выводит блок Smart Stop с правильными отступами, псевдографикой и метками наследования
func printStopConditions(padding string, sc config.StopConditions, defSc config.StopConditions) {
	var conditions []string

	if sc.MaxTimeSinceLastPath != "" {
		label := getInheritLabel(sc.MaxTimeSinceLastPath == defSc.MaxTimeSinceLastPath && defSc.MaxTimeSinceLastPath != "")
		conditions = append(conditions, fmt.Sprintf("Path Timeout: %s%s", sc.MaxTimeSinceLastPath, label))
	}
	if sc.MinCoveragePct > 0 {
		label := getInheritLabel(sc.MinCoveragePct == defSc.MinCoveragePct && defSc.MinCoveragePct > 0)
		conditions = append(conditions, fmt.Sprintf("Min Coverage: %s%%%s", fmt.Sprintf("%.1f", sc.MinCoveragePct), label))
	}
	if sc.MaxTotalTime != "" {
		label := getInheritLabel(sc.MaxTotalTime == defSc.MaxTotalTime && defSc.MaxTotalTime != "")
		conditions = append(conditions, fmt.Sprintf("Max Run Time: %s%s", sc.MaxTotalTime, label))
	}

	condCount := len(conditions)
	for j, cond := range conditions {
		isLastCond := (j == condCount-1)
		condPrefix := "├── "
		if isLastCond {
			condPrefix = "└── "
		}
		fmt.Printf("%s    %s%s\n", padding, condPrefix, cond)
	}
}

// getInheritLabel возвращает строковый маркер, если значение унаследовано
func getInheritLabel(inherited bool) string {
	if inherited {
		return " (default)"
	}
	return ""
}

func init() {
	RootCmd.AddCommand(configCmd)
}
