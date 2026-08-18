package docker

import (
	"fmt"
	"strings"
	"zope/internal/config"
)

// GenerateDockerfile генерирует текст Dockerfile на основе конфигурации цели.
func GenerateDockerfile(projectTitle string, baseImage string, target config.Target) (string, error) {
	// Выбираем логику генерации в зависимости от языка.
	// Каждый case возвращает значение напрямую, завершая функцию.
	switch target.Language {
	case "cpp", "c":
		return generateCppDockerfile(baseImage, target)
	case "python":
		return generatePythonDockerfile(target)
	default:
		return "", fmt.Errorf("language '%s' is not yet supported by docker generator", target.Language)
	}
}

// generateCppDockerfile собирает Multi-stage сценарий для компиляции C/C++ харнессов
func generateCppDockerfile(toolsImage string, target config.Target) (string, error) {
	var sb strings.Builder

	// ЭТАП 1: Получаем инструменты компиляции из base_image
	sb.WriteString("# Stage 1: Extract fuzzing environment and tools\n")
	fmt.Fprintf(&sb, "FROM %s AS fuzzer_tools\n\n", toolsImage)

	// ЭТАП 2: Разворачиваем среду приложения пользователя
	sb.WriteString("# Stage 2: Build application context and compile harness\n")
	if target.Image != "" {
		fmt.Fprintf(&sb, "FROM %s\n", target.Image)
	} else {
		// Если своего образа среды нет, собираем прямо в окружении инструментов
	}

	// Настраиваем компиляторы в зависимости от выбранного движка
	sb.WriteString("\n# Configure environment variables for compilers\n")
	switch target.Engine {
	case "afl++":
		sb.WriteString("ENV CC=afl-clang-fast\n")
		sb.WriteString("ENV CXX=afl-clang-fast++\n")
		// Копируем бинарники компиляторов, если мы собираем в кастомном target.image
		if target.Image != "" {
			sb.WriteString("COPY --from=fuzzer_tools /usr/local/bin/afl-* /usr/local/bin/\n")
			sb.WriteString("COPY --from=fuzzer_tools /usr/local/share/aflplusplus/ /usr/local/share/aflplusplus/\n")
		}
	case "libfuzzer":
		sb.WriteString("ENV CC=clang\n")
		sb.WriteString("ENV CXX=clang++\n")
	}

	// Подготавливаем рабочую директорию и копируем исходный код
	sb.WriteString("\nWORKDIR /src\n")
	sb.WriteString("COPY . .\n")

	// Собираем флаги компиляции и санитайзеры
	var buildFlags []string
	if target.CompileFlags != "" {
		buildFlags = append(buildFlags, target.CompileFlags)
	}

	// Подключаем санитайзеры компилятора (ASan, UBSan и т.д.)
	if len(target.Sanitizers) > 0 {
		sanitizersStr := strings.Join(target.Sanitizers, ",")
		buildFlags = append(buildFlags, fmt.Sprintf("-fsanitize=%s", sanitizersStr))
	}

	// Линковка движка libFuzzer, если выбран он
	if target.Engine == "libfuzzer" {
		buildFlags = append(buildFlags, "-fsanitize=fuzzer")
	}

	// Добавляем флаги линковщика
	if target.LinkFlags != "" {
		buildFlags = append(buildFlags, target.LinkFlags)
	}

	// Определяем команду компиляции. Имя бинарника фиксируем как /target_fuzz
	sb.WriteString("\n# Compile the harness binary\n")
	flagsCombined := strings.Join(buildFlags, " ")

	// Базовая проверка: C или C++ компилятор использовать
	var compiler string
	if target.Language == "c" {
		compiler = "$CC"
	} else {
		compiler = "$CXX"
	}

	fmt.Fprintf(&sb, "RUN %s %s %s -o /target_fuzz\n", compiler, flagsCombined, target.Source)

	// Конечная точка запуска контейнера
	sb.WriteString("\n# Set entrypoint to run the compiled binary\n")
	sb.WriteString("ENTRYPOINT [\"/target_fuzz\"]\n")

	return sb.String(), nil
}

// generatePythonDockerfile собирает сценарий для разворачивания Python (Atheris) харнессов
func generatePythonDockerfile(target config.Target) (string, error) {
	var sb strings.Builder

	sb.WriteString("# Build environment for Python fuzzing (Atheris)\n")
	if target.Image != "" {
		fmt.Fprintf(&sb, "FROM %s\n", target.Image)
	} else {
		// Дефолтный образ с питоном, если пользователь не указал свою среду
		sb.WriteString("FROM docker.io/library/python:3.11-slim\n")
	}

	sb.WriteString("\nWORKDIR /app\n")

	// Устанавливаем Atheris и системные зависимости для сборки нативных расширений питона
	sb.WriteString("# Install essential packages and fuzzing engine\n")
	sb.WriteString("RUN apt-get update && apt-get install -y --no-install-recommends \\\n")
	sb.WriteString("    gcc g++ make libclang-dev && \\\n")
	sb.WriteString("    pip install --no-cache-dir atheris && \\\n")
	sb.WriteString("    apt-get purge -y --auto-remove gcc g++ make && \\\n")
	sb.WriteString("    rm -rf /var/lib/apt/lists/*\n")

	sb.WriteString("\nCOPY . .\n")

	// Точка запуска для питона требует явного вызова интерпретатора
	fmt.Fprintf(&sb, "\nENTRYPOINT [\"python\", \"%s\"]\n", target.Source)

	return sb.String(), nil
}
