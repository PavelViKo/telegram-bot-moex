package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GetConfigPath возвращает путь к конфигурационному файлу
// Ищет в следующем порядке:
// 1. Переданный путь
// 2. Переменная окружения CONFIG_PATH
// 3. Стандартные расположения
func GetConfigPath(userPath string) (string, error) {
	// 1. Используем переданный путь если он существует
	if userPath != "" {
		if _, err := os.Stat(userPath); err == nil {
			return userPath, nil
		}
	}

	// 2. Проверяем переменную окружения
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			return envPath, nil
		}
	}

	// 3. Проверяем стандартные расположения
	possiblePaths := []string{
		"./configs/config.yaml",
		"./config.yaml",
		"/etc/moex-bot/config.yaml",
		"$HOME/.config/moex-bot/config.yaml",
		"$HOME/.moex-bot.yaml",
	}

	for _, path := range possiblePaths {
		// Разворачиваем переменные окружения
		expandedPath := os.ExpandEnv(path)
		if _, err := os.Stat(expandedPath); err == nil {
			return expandedPath, nil
		}
	}

	return "", fmt.Errorf("конфигурационный файл не найден")
}

// PrintConfig выводит конфигурацию (без секретов)
func (c *Config) PrintConfig() string {
	var sb strings.Builder

	sb.WriteString("📋 Конфигурация бота:\n\n")

	// Telegram
	sb.WriteString("🤖 Telegram:\n")
	sb.WriteString(fmt.Sprintf("  • Debug: %v\n", c.Telegram.Debug))
	sb.WriteString(fmt.Sprintf("  • Updates Timeout: %d\n", c.Telegram.UpdatesTimeout))
	sb.WriteString("\n")

	// API
	sb.WriteString("🔌 API:\n")
	sb.WriteString(fmt.Sprintf("  • URL: %s\n", c.API.URL))
	sb.WriteString(fmt.Sprintf("  • Timeout: %v\n", c.API.Timeout))
	sb.WriteString(fmt.Sprintf("  • Max Retries: %d\n", c.API.MaxRetries))
	sb.WriteString("\n")

	// Bot
	sb.WriteString("⚙️ Bot:\n")
	sb.WriteString(fmt.Sprintf("  • Name: %s\n", c.Bot.Name))
	sb.WriteString(fmt.Sprintf("  • Rate Limit: %v\n", c.Bot.RateLimitInterval))
	sb.WriteString(fmt.Sprintf("  • Command Timeout: %v\n", c.Bot.CommandTimeout))
	sb.WriteString("\n")

	// Strategy
	sb.WriteString("📈 Strategy:\n")
	sb.WriteString(fmt.Sprintf("  • Turtles Enabled: %v\n", c.Strategy.Turtles.Enabled))
	if c.Strategy.Turtles.Enabled {
		sb.WriteString(fmt.Sprintf("  • Timeframe: %s\n", c.Strategy.Turtles.Timeframe))
		sb.WriteString(fmt.Sprintf("  • Lookback Period: %d дней\n", c.Strategy.Turtles.LookbackPeriod))
		sb.WriteString(fmt.Sprintf("  • Risk per Trade: %.1f%%\n", c.Strategy.Turtles.RiskPerTrade*100))
	}
	sb.WriteString(fmt.Sprintf("  • Notifications: %v\n", c.Strategy.Notifications.Enabled))
	sb.WriteString("\n")

	// Security
	sb.WriteString("🔒 Security:\n")
	sb.WriteString(fmt.Sprintf("  • Auth Enabled: %v\n", c.Security.EnableAuth))
	if c.Security.EnableAuth {
		sb.WriteString(fmt.Sprintf("  • Allowed Users: %d\n", len(c.Security.AllowedUsers)))
		sb.WriteString(fmt.Sprintf("  • Admin Users: %d\n", len(c.Security.AdminUsers)))
	}
	sb.WriteString("\n")

	// Logging
	sb.WriteString("📝 Logging:\n")
	sb.WriteString(fmt.Sprintf("  • Level: %s\n", c.Logging.Level))
	sb.WriteString(fmt.Sprintf("  • File: %s\n", c.Logging.File))
	sb.WriteString(fmt.Sprintf("  • Max Size: %dMB\n", c.Logging.MaxSizeMB))

	return sb.String()
}

// IsAdmin проверяет, является ли пользователь администратором
func (c *Config) IsAdmin(userID int64) bool {
	for _, adminID := range c.Security.AdminUsers {
		if adminID == userID {
			return true
		}
	}
	return false
}

// IsUserAllowed проверяет, разрешен ли пользователь
func (c *Config) IsUserAllowed(userID int64) bool {
	// Если проверка отключена, все разрешены
	if !c.Security.EnableAuth {
		return true
	}

	// Проверяем список разрешенных пользователей
	for _, allowedID := range c.Security.AllowedUsers {
		if allowedID == userID {
			return true
		}
	}

	return false
}

// GetLogFile возвращает полный путь к файлу логов
func (c *Config) GetLogFile() string {
	// Если путь абсолютный, используем как есть
	if filepath.IsAbs(c.Logging.File) {
		return c.Logging.File
	}

	// Если путь относительный, делаем абсолютным относительно рабочей директории
	wd, err := os.Getwd()
	if err != nil {
		return c.Logging.File
	}

	return filepath.Join(wd, c.Logging.File)
}

// EnsureDirectories создает необходимые директории
func (c *Config) EnsureDirectories() error {
	// Создаем директорию для логов
	logDir := filepath.Dir(c.GetLogFile())
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории логов: %w", err)
	}

	// Создаем директорию для данных если нужно
	dataDir := "./data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории данных: %w", err)
	}

	return nil
}
