package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
)

// ValidateConfig валидирует конфигурацию
func ValidateConfig(cfg *Config) error {
	// Валидация Telegram токена
	if err := validateTelegramToken(cfg.Telegram.Token); err != nil {
		return fmt.Errorf("неверный Telegram токен: %w", err)
	}

	// Валидация URL API
	if err := validateURL(cfg.API.URL); err != nil {
		return fmt.Errorf("неверный URL API: %w", err)
	}

	// Валидация таймаутов
	if err := validateTimeouts(cfg); err != nil {
		return fmt.Errorf("ошибка таймаутов: %w", err)
	}

	// Валидация настроек логирования
	if err := validateLogging(cfg.Logging); err != nil {
		return fmt.Errorf("ошибка настроек логирования: %w", err)
	}

	// Валидация настроек безопасности
	if err := validateSecurity(cfg.Security); err != nil {
		return fmt.Errorf("ошибка настроек безопасности: %w", err)
	}

	// Валидация настроек стратегии
	if err := validateStrategy(cfg.Strategy); err != nil {
		return fmt.Errorf("ошибка настроек стратегии: %w", err)
	}

	// Валидация настроек бота
	if err := validateBotSettings(cfg.Bot); err != nil {
		return fmt.Errorf("ошибка настроек бота: %w", err)
	}

	return nil
}

// validateTelegramToken проверяет формат Telegram токена
func validateTelegramToken(token string) error {
	if token == "" {
		return fmt.Errorf("токен не может быть пустым")
	}

	// Telegram токены имеют формат: цифры:буквы
	pattern := `^\d{9,10}:[a-zA-Z0-9_-]{35}$`
	match, err := regexp.MatchString(pattern, token)
	if err != nil {
		return fmt.Errorf("ошибка проверки токена: %w", err)
	}

	if !match {
		return fmt.Errorf("неверный формат токена. Ожидается: 123456789:ABCdefGHIjklMNOpqrSTUvwxYZ123456789")
	}

	return nil
}

// validateURL проверяет корректность URL
func validateURL(url string) error {
	if url == "" {
		return fmt.Errorf("URL не может быть пустым")
	}

	// Проверяем, что URL начинается с http:// или https://
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("URL должен начинаться с http:// или https://")
	}

	return nil
}

// validateTimeouts проверяет корректность таймаутов
func validateTimeouts(cfg *Config) error {
	// Проверка таймаута API
	if cfg.API.Timeout <= 0 {
		return fmt.Errorf("API timeout должен быть положительным числом")
	}
	if cfg.API.Timeout > 5*time.Minute {
		return fmt.Errorf("API timeout не может превышать 5 минут")
	}

	// Проверка времени ожидания обновлений
	if cfg.Telegram.UpdatesTimeout < 0 || cfg.Telegram.UpdatesTimeout > 100 {
		return fmt.Errorf("updates timeout должен быть между 0 и 100 секундами")
	}

	// Проверка интервала rate limit
	if cfg.Bot.RateLimitInterval <= 0 {
		return fmt.Errorf("rate limit interval должен быть положительным числом")
	}
	if cfg.Bot.RateLimitInterval > 10*time.Second {
		return fmt.Errorf("rate limit interval не может превышать 10 секунд")
	}

	// Проверка таймаута команд
	if cfg.Bot.CommandTimeout <= 0 {
		return fmt.Errorf("command timeout должен быть положительным числом")
	}
	if cfg.Bot.CommandTimeout > 30*time.Minute {
		return fmt.Errorf("command timeout не может превышать 30 минут")
	}

	// Проверка задержки между повторными попытками
	if cfg.API.RetryDelay <= 0 {
		return fmt.Errorf("retry delay должен быть положительным числом")
	}
	if cfg.API.RetryDelay > 30*time.Second {
		return fmt.Errorf("retry delay не может превышать 30 секунд")
	}

	return nil
}

// validateLogging проверяет настройки логирования
func validateLogging(logging LoggingConfig) error {
	// Проверка уровня логирования
	validLevels := map[string]bool{
		"debug": true,
		"info":  true,
		"warn":  true,
		"error": true,
		"panic": true,
		"fatal": true,
	}

	if !validLevels[strings.ToLower(logging.Level)] {
		return fmt.Errorf("неверный уровень логирования: %s. Допустимые значения: debug, info, warn, error, panic, fatal", logging.Level)
	}

	// Проверка размера файла логов
	if logging.MaxSizeMB <= 0 {
		return fmt.Errorf("max size must be positive")
	}
	if logging.MaxSizeMB > 1024 {
		return fmt.Errorf("max size cannot exceed 1GB")
	}

	// Проверка количества бэкапов
	if logging.MaxBackups < 0 {
		return fmt.Errorf("max backups cannot be negative")
	}
	if logging.MaxBackups > 100 {
		return fmt.Errorf("max backups cannot exceed 100")
	}

	// Проверка возраста логов
	if logging.MaxAgeDays < 0 {
		return fmt.Errorf("max age cannot be negative")
	}
	if logging.MaxAgeDays > 365 {
		return fmt.Errorf("max age cannot exceed 365 days")
	}

	return nil
}

// validateSecurity проверяет настройки безопасности
func validateSecurity(security SecurityConfig) error {
	// Проверка списка пользователей
	if security.EnableAuth && len(security.AllowedUsers) == 0 {
		return fmt.Errorf("при включенной авторизации должен быть указан хотя бы один пользователь")
	}

	// Проверка корректности ID пользователей
	for _, userID := range security.AllowedUsers {
		if userID <= 0 {
			return fmt.Errorf("неверный ID пользователя: %d", userID)
		}
	}

	for _, userID := range security.AdminUsers {
		if userID <= 0 {
			return fmt.Errorf("неверный ID администратора: %d", userID)
		}

		// Проверяем, что администратор также есть в списке разрешенных
		found := false
		for _, allowedID := range security.AllowedUsers {
			if allowedID == userID {
				found = true
				break
			}
		}

		if security.EnableAuth && !found {
			return fmt.Errorf("администратор %d должен быть в списке разрешенных пользователей", userID)
		}
	}

	return nil
}

// validateStrategy проверяет настройки стратегии
func validateStrategy(strategy StrategyConfig) error {
	// Проверка параметров стратегии "Черепах"
	if strategy.Turtles.Enabled {
		if strategy.Turtles.LookbackPeriod <= 0 {
			return fmt.Errorf("lookback period должен быть положительным числом")
		}
		if strategy.Turtles.EntryBreakoutDays <= 0 {
			return fmt.Errorf("entry breakout days должен быть положительным числом")
		}
		if strategy.Turtles.ExitBreakoutDays <= 0 {
			return fmt.Errorf("exit breakout days должен быть положительным числом")
		}
		if strategy.Turtles.RiskPerTrade <= 0 || strategy.Turtles.RiskPerTrade > 1 {
			return fmt.Errorf("risk per trade должен быть между 0 и 1")
		}
		if strategy.Turtles.AtrPeriod <= 0 {
			return fmt.Errorf("atr period должен быть положительным числом")
		}
		if strategy.Turtles.AtrMultiplier <= 0 {
			return fmt.Errorf("atr multiplier должен быть положительным числом")
		}

		// Проверка допустимых таймфреймов
		validTimeframes := map[string]bool{
			"1":  true, // 1 минута
			"10": true, // 10 минут
			"60": true, // 1 час
			"24": true, // 1 день
			"7":  true, // 1 неделя
			"31": true, // 1 месяц
		}

		if !validTimeframes[strategy.Turtles.Timeframe] {
			return fmt.Errorf("неверный таймфрейм для стратегии: %s", strategy.Turtles.Timeframe)
		}
	}

	return nil
}

// validateBotSettings проверяет настройки бота
func validateBotSettings(bot BotConfig) error {
	// Проверка имени бота
	if bot.Name == "" {
		return fmt.Errorf("имя бота не может быть пустым")
	}
	if len(bot.Name) > 64 {
		return fmt.Errorf("имя бота не может превышать 64 символа")
	}

	// Проверка приветственного сообщения
	if bot.Greeting == "" {
		return fmt.Errorf("приветственное сообщение не может быть пустым")
	}

	// Проверка сообщения помощи
	if bot.HelpMessage == "" {
		return fmt.Errorf("сообщение помощи не может быть пустым")
	}

	// Проверка максимальной длины сообщения
	if bot.MaxMessageLength <= 0 {
		return fmt.Errorf("максимальная длина сообщения должна быть положительным числом")
	}
	if bot.MaxMessageLength > 4096 {
		return fmt.Errorf("максимальная длина сообщения не может превышать 4096 символов (ограничение Telegram)")
	}

	// Проверка ID чата для уведомлений
	if bot.NotificationChatID < 0 {
		return fmt.Errorf("ID чата для уведомлений не может быть отрицательным")
	}

	return nil
}

// ValidateEnvironment проверяет переменные окружения
func ValidateEnvironment() error {
	requiredEnvVars := []string{
		"TELEGRAM_TOKEN",
	}

	for _, envVar := range requiredEnvVars {
		if os.Getenv(envVar) == "" {
			return fmt.Errorf("требуется переменная окружения: %s", envVar)
		}
	}

	return nil
}
