package config

import (
	"fmt"
	"os"
	"slices"
	"time"

	"github.com/BurntSushi/toml"
)

// Project представляет секцию [project]
type Project struct {
	Name string `toml:"name"`
}

// Runtime представляет секцию [runtime]
type Runtime struct {
	Engine    string `toml:"engine"`
	BaseImage string `toml:"base_image"`
}

// Defaults содержит те же параметры, что и Target, для переиспользования
type Defaults struct {
	Language       string         `toml:"language"`
	Engine         string         `toml:"engine"`
	CompileFlags   string         `toml:"compile_flags"`
	LinkFlags      string         `toml:"link_flags"`
	Sanitizers     []string       `toml:"sanitizers"`
	StopConditions StopConditions `toml:"stop_conditions"`
}

// StopConditions представляет настройки автоматической остановки
type StopConditions struct {
	MaxTimeSinceLastPath string  `toml:"max_time_since_last_path"`
	MinCoveragePct       float64 `toml:"min_coverage_pct"`
	MaxTotalTime         string  `toml:"max_total_time"`
}

// Target представляет элемент массива [[target]]
type Target struct {
	Name           string         `toml:"name"`
	Language       string         `toml:"language"`
	Engine         string         `toml:"engine"`
	Source         string         `toml:"source"`
	Image          string         `toml:"image"`
	CompileFlags   string         `toml:"compile_flags"`
	LinkFlags      string         `toml:"link_flags"`
	Sanitizers     []string       `toml:"sanitizers"`
	StopConditions StopConditions `toml:"stop_conditions"`
}

// Config — это мастер-структура, объединяющая весь файл zope.toml
type Config struct {
	Project  Project  `toml:"project"`
	Runtime  Runtime  `toml:"runtime"`
	Defaults Defaults `toml:"defaults"`
	Targets  []Target `toml:"target"`
}

// LoadConfig читает файл с диска и парсит его в структуру Config
func LoadConfig(filePath string) (*Config, error) {
	var cfg Config

	// Проверяем, существует ли файл
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("конфигурационный файл %s не найден", filePath)
	}

	// Декодируем TOML в нашу структуру
	if _, err := toml.DecodeFile(filePath, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка синтаксиса TOML: %w", err)
	}

	cfg.applyDefaults()

	return &cfg, nil
}

// applyDefaults автоматически заполняет пустые поля таргетов значениями из [defaults]
// с учетом специфики языков программирования.
func (c *Config) applyDefaults() {
	for i := range c.Targets {
		t := &c.Targets[i]

		// 1. Сначала наследуем базовые метаданные рантайма
		if t.Language == "" {
			t.Language = c.Defaults.Language
		}
		if t.Engine == "" {
			t.Engine = c.Defaults.Engine
		}

		// 2. Наследуем флаги сборки ТОЛЬКО для C/C++ целей
		if t.Language == "cpp" || t.Language == "c" {
			if t.CompileFlags == "" {
				t.CompileFlags = c.Defaults.CompileFlags
			}
			if t.LinkFlags == "" {
				t.LinkFlags = c.Defaults.LinkFlags
			}
			if len(t.Sanitizers) == 0 {
				t.Sanitizers = c.Defaults.Sanitizers
			}
		}

		// 3. Слияние Stop Conditions (актуально для всех языков)
		if t.StopConditions.MaxTimeSinceLastPath == "" {
			t.StopConditions.MaxTimeSinceLastPath = c.Defaults.StopConditions.MaxTimeSinceLastPath
		}
		if t.StopConditions.MinCoveragePct == 0 {
			t.StopConditions.MinCoveragePct = c.Defaults.StopConditions.MinCoveragePct
		}
		if t.StopConditions.MaxTotalTime == "" {
			t.StopConditions.MaxTotalTime = c.Defaults.StopConditions.MaxTotalTime
		}
	}
}

// Validate проверяет корректность заполнения конфига
func (c *Config) Validate() error {
	if c.Project.Name == "" {
		return fmt.Errorf("the [project] section must contain a 'name'")
	}

	// Валидация дефолтных временных интервалов
	if c.Defaults.StopConditions.MaxTimeSinceLastPath != "" {
		if _, err := time.ParseDuration(c.Defaults.StopConditions.MaxTimeSinceLastPath); err != nil {
			return fmt.Errorf("defaults: invalid max_time_since_last_path format")
		}
	}
	if c.Defaults.StopConditions.MaxTotalTime != "" {
		if _, err := time.ParseDuration(c.Defaults.StopConditions.MaxTotalTime); err != nil {
			return fmt.Errorf("defaults: invalid max_total_time format")
		}
	}

	seenNames := make(map[string]bool)
	for _, target := range c.Targets {
		if seenNames[target.Name] {
			return fmt.Errorf("duplicate target name discovered: %s", target.Name)
		}
		seenNames[target.Name] = true

		if target.Language == "" {
			return fmt.Errorf("target %s: language is required (or must be set in [defaults])", target.Name)
		}
		if target.Engine == "" {
			return fmt.Errorf("target %s: engine is required (or must be set in [defaults])", target.Name)
		}

		// Строгая валидация связки Язык-Движок
		switch target.Language {
		case "cpp", "c":
			if !slices.Contains([]string{"afl", "afl++", "libfuzzer"}, target.Engine) {
				return fmt.Errorf("target %s: C/C++ language is incompatible with engine '%s' (allowed: afl++, libfuzzer)", target.Name, target.Engine)
			}
		case "python":
			if !slices.Contains([]string{"atheris"}, target.Engine) {
				return fmt.Errorf("target %s: Python language is incompatible with engine '%s' (allowed: atheris)", target.Name, target.Engine)
			}
			// Защита: если пользователь вручную прописал флаги компиляции для питона в самом таргете
			if target.CompileFlags != "" || target.LinkFlags != "" || len(target.Sanitizers) > 0 {
				return fmt.Errorf("target %s: compilation flags and sanitizers cannot be used with Python", target.Name)
			}
		case "java":
			if !slices.Contains([]string{"jazzer"}, target.Engine) {
				return fmt.Errorf("target %s: Java language is incompatible with engine '%s' (allowed: jazzer)", target.Name, target.Engine)
			}
		}

		// Валидация времени условий останова
		if target.StopConditions.MaxTimeSinceLastPath != "" {
			if _, err := time.ParseDuration(target.StopConditions.MaxTimeSinceLastPath); err != nil {
				return fmt.Errorf("target %s: invalid max_time_since_last_path format", target.Name)
			}
		}
		if target.StopConditions.MaxTotalTime != "" {
			if _, err := time.ParseDuration(target.StopConditions.MaxTotalTime); err != nil {
				return fmt.Errorf("target %s: invalid max_total_time format", target.Name)
			}
		}
	}

	return nil
}
