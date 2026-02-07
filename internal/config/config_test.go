package config

import (
	"os"
	"testing"
	"time"
)

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		wantErr     bool
		errorPrefix string
	}{
		{
			name: "Valid config",
			config: &Config{
				Telegram: TelegramConfig{
					Token:          "123456789:ABCdefGHIjklMNOpqrSTUvwxYZ123456789",
					Debug:          false,
					UpdatesTimeout: 60,
				},
				API: APIConfig{
					URL:        "http://localhost:8080",
					Token:      "test-token",
					Timeout:    30 * time.Second,
					MaxRetries: 3,
					RetryDelay: 2 * time.Second,
				},
				Bot: BotConfig{
					Name:              "Test Bot",
					Greeting:          "Hello",
					HelpMessage:       "Help",
					RateLimitInterval: time.Second,
					MaxMessageLength:  4096,
					CommandTimeout:    5 * time.Minute,
				},
				Strategy: StrategyConfig{
					Turtles: TurtleStrategyConfig{
						Enabled:           true,
						Timeframe:         "24",
						LookbackPeriod:    20,
						EntryBreakoutDays: 20,
						ExitBreakoutDays:  10,
						RiskPerTrade:      0.02,
						PositionSizing:    true,
						AtrPeriod:         20,
						AtrMultiplier:     2.0,
					},
					Notifications: NotificationsConfig{
						Enabled:         true,
						DailyReport:     true,
						AlertOnBreakout: true,
					},
				},
				Technical: TechnicalConfig{
					SMA:             []int{20, 50, 200},
					EMA:             []int{12, 26},
					RSIPeriod:       14,
					MACDFast:        12,
					MACDSlow:        26,
					MACDSignal:      9,
					BollingerPeriod: 20,
					BollingerStd:    2,
				},
				Logging: LoggingConfig{
					Level:      "info",
					File:       "logs/bot.log",
					MaxSizeMB:  100,
					MaxBackups: 3,
					MaxAgeDays: 30,
					JSONFormat: false,
				},
				Security: SecurityConfig{
					AllowedUsers: []int64{123456789},
					AdminUsers:   []int64{123456789},
					EnableAuth:   true,
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid Telegram token",
			config: &Config{
				Telegram: TelegramConfig{
					Token: "invalid-token",
				},
				API: APIConfig{
					URL: "http://localhost:8080",
				},
				Bot: BotConfig{
					Name:        "Test",
					Greeting:    "Hello",
					HelpMessage: "Help",
				},
				Strategy: StrategyConfig{
					Turtles: TurtleStrategyConfig{
						Enabled: false,
					},
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Security: SecurityConfig{
					EnableAuth: false,
				},
			},
			wantErr:     true,
			errorPrefix: "неверный Telegram токен",
		},
		{
			name: "Invalid API URL",
			config: &Config{
				Telegram: TelegramConfig{
					Token: "123456789:ABCdefGHIjklMNOpqrSTUvwxYZ123456789",
				},
				API: APIConfig{
					URL: "invalid-url",
				},
				Bot: BotConfig{
					Name:        "Test",
					Greeting:    "Hello",
					HelpMessage: "Help",
				},
				Strategy: StrategyConfig{
					Turtles: TurtleStrategyConfig{
						Enabled: false,
					},
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Security: SecurityConfig{
					EnableAuth: false,
				},
			},
			wantErr:     true,
			errorPrefix: "неверный URL API",
		},
		{
			name: "Invalid strategy timeframe",
			config: &Config{
				Telegram: TelegramConfig{
					Token: "123456789:ABCdefGHIjklMNOpqrSTUvwxYZ123456789",
				},
				API: APIConfig{
					URL: "http://localhost:8080",
				},
				Bot: BotConfig{
					Name:        "Test",
					Greeting:    "Hello",
					HelpMessage: "Help",
				},
				Strategy: StrategyConfig{
					Turtles: TurtleStrategyConfig{
						Enabled:           true,
						Timeframe:         "invalid",
						LookbackPeriod:    20,
						EntryBreakoutDays: 20,
						ExitBreakoutDays:  10,
						RiskPerTrade:      0.02,
					},
				},
				Logging: LoggingConfig{
					Level: "info",
				},
				Security: SecurityConfig{
					EnableAuth: false,
				},
			},
			wantErr:     true,
			errorPrefix: "неверный таймфрейм для стратегии",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.config)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateConfig() expected error, got nil")
				} else if tt.errorPrefix != "" && !contains(err.Error(), tt.errorPrefix) {
					t.Errorf("ValidateConfig() error = %v, want prefix %v", err, tt.errorPrefix)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateConfig() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestValidateEnvironment(t *testing.T) {
	// Сохраняем оригинальные значения
	origToken := os.Getenv("TELEGRAM_TOKEN")
	defer os.Setenv("TELEGRAM_TOKEN", origToken)

	// Тест 1: Переменная установлена
	os.Setenv("TELEGRAM_TOKEN", "test-token")
	if err := ValidateEnvironment(); err != nil {
		t.Errorf("ValidateEnvironment() с установленной переменной вернула ошибку: %v", err)
	}

	// Тест 2: Переменная не установлена
	os.Setenv("TELEGRAM_TOKEN", "")
	if err := ValidateEnvironment(); err == nil {
		t.Errorf("ValidateEnvironment() с пустой переменной ожидалась ошибка")
	} else if !contains(err.Error(), "TELEGRAM_TOKEN") {
		t.Errorf("ValidateEnvironment() ошибка не содержит имя переменной: %v", err)
	}
}

// Вспомогательная функция для проверки подстроки
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && contains(s[1:], substr))
}
