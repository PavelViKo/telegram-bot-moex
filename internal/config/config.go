package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config основной конфигурационный файл
type Config struct {
	Telegram  TelegramConfig  `yaml:"telegram"`
	API       APIConfig       `yaml:"api"`
	Bot       BotConfig       `yaml:"bot"`
	Strategy  StrategyConfig  `yaml:"strategy"`
	Technical TechnicalConfig `yaml:"technical"`
	Logging   LoggingConfig   `yaml:"logging"`
	Security  SecurityConfig  `yaml:"security"`
}

// TelegramConfig настройки Telegram API
type TelegramConfig struct {
	Token          string `yaml:"token"`
	Debug          bool   `yaml:"debug"`
	UpdatesTimeout int    `yaml:"updates_timeout"`
}

// APIConfig настройки API MOEX Fetcher
type APIConfig struct {
	URL        string        `yaml:"url"`
	Token      string        `yaml:"token"`
	Timeout    time.Duration `yaml:"timeout"`
	MaxRetries int           `yaml:"max_retries"`
	RetryDelay time.Duration `yaml:"retry_delay"`
}

// BotConfig настройки бота
type BotConfig struct {
	Name               string        `yaml:"name"`
	Greeting           string        `yaml:"greeting"`
	HelpMessage        string        `yaml:"help_message"`
	RateLimitInterval  time.Duration `yaml:"rate_limit_interval"`
	MaxMessageLength   int           `yaml:"max_message_length"`
	CommandTimeout     time.Duration `yaml:"command_timeout"`
	NotificationChatID int64         `yaml:"notification_chat_id"`
}

// StrategyConfig настройки торговых стратегий
type StrategyConfig struct {
	Turtles       TurtleStrategyConfig `yaml:"turtles"`
	MACrossover   MAConfig             `yaml:"ma_crossover"`
	Notifications NotificationsConfig  `yaml:"notifications"`
}

// TurtleStrategyConfig настройки стратегии "Черепах"
type TurtleStrategyConfig struct {
	Enabled           bool    `yaml:"enabled"`
	Timeframe         string  `yaml:"timeframe"`
	LookbackPeriod    int     `yaml:"lookback_period"`
	EntryBreakoutDays int     `yaml:"entry_breakout_days"`
	ExitBreakoutDays  int     `yaml:"exit_breakout_days"`
	RiskPerTrade      float64 `yaml:"risk_per_trade"`
	PositionSizing    bool    `yaml:"position_sizing"`
	AtrPeriod         int     `yaml:"atr_period"`
	AtrMultiplier     float64 `yaml:"atr_multiplier"`
}

// MAConfig настройки стратегии Moving Average Crossover
type MAConfig struct {
	Enabled               bool    `yaml:"enabled"`
	Timeframe             string  `yaml:"timeframe"`
	FastPeriod            int     `yaml:"fast_period"`
	SlowPeriod            int     `yaml:"slow_period"`
	SignalPeriod          int     `yaml:"signal_period"`
	UseEMA                bool    `yaml:"use_ema"`
	UseVolumeConfirmation bool    `yaml:"use_volume_confirmation"`
	MinVolumeMultiplier   float64 `yaml:"min_volume_multiplier"`
	RiskPerTrade          float64 `yaml:"risk_per_trade"`
	StopLossATRMultiplier float64 `yaml:"stop_loss_atr_multiplier"`
	TakeProfitRatio       float64 `yaml:"take_profit_ratio"`

	CrossoverTypes struct {
		GoldenCross         bool `yaml:"golden_cross"`
		DeathCross          bool `yaml:"death_cross"`
		RequireConfirmation int  `yaml:"require_confirmation"`
	} `yaml:"crossover_types"`

	Filters struct {
		TrendFilter   string `yaml:"trend_filter"`
		RSIFilter     bool   `yaml:"rsi_filter"`
		RSIOverbought int    `yaml:"rsi_overbought"`
		RSIOversold   int    `yaml:"rsi_oversold"`
	} `yaml:"filters"`
}

// NotificationsConfig настройки уведомлений
type NotificationsConfig struct {
	Enabled         bool  `yaml:"enabled"`
	SignalChatID    int64 `yaml:"signal_chat_id"`
	DailyReport     bool  `yaml:"daily_report"`
	AlertOnBreakout bool  `yaml:"alert_on_breakout"`
}

// TechnicalConfig настройки технического анализа
type TechnicalConfig struct {
	SMA             []int `yaml:"sma"`
	EMA             []int `yaml:"ema"`
	RSIPeriod       int   `yaml:"rsi_period"`
	MACDFast        int   `yaml:"macd_fast"`
	MACDSlow        int   `yaml:"macd_slow"`
	MACDSignal      int   `yaml:"macd_signal"`
	BollingerPeriod int   `yaml:"bollinger_period"`
	BollingerStd    int   `yaml:"bollinger_std"`
}

// LoggingConfig настройки логирования
type LoggingConfig struct {
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	MaxSizeMB  int    `yaml:"max_size_mb"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAgeDays int    `yaml:"max_age_days"`
	JSONFormat bool   `yaml:"json_format"`
}

// SecurityConfig настройки безопасности
type SecurityConfig struct {
	AllowedUsers []int64 `yaml:"allowed_users"`
	AdminUsers   []int64 `yaml:"admin_users"`
	EnableAuth   bool    `yaml:"enable_auth"`
}

// LoadConfig загружает конфигурацию из файла
func LoadConfig(configPath string) (*Config, error) {
	// Проверяем существование файла
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("файл конфигурации не найден: %s", configPath)
	}

	// Читаем файл
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения файла конфигурации: %w", err)
	}

	// Парсим YAML
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга YAML: %w", err)
	}

	// Загружаем переменные окружения (приоритет выше)
	overrideFromEnv(&cfg)

	// Валидация конфигурации
	if err := ValidateConfig(&cfg); err != nil {
		return nil, fmt.Errorf("ошибка валидации конфигурации: %w", err)
	}

	return &cfg, nil
}

// LoadConfigOrDefault загружает конфигурацию или использует значения по умолчанию
func LoadConfigOrDefault(configPath string) (*Config, error) {
	cfg, err := LoadConfig(configPath)
	if err != nil {
		// Используем конфигурацию по умолчанию
		cfg = DefaultConfig()

		// Все равно пытаемся загрузить из env
		overrideFromEnv(cfg)

		// Базовая валидация
		if err := ValidateConfig(cfg); err != nil {
			return nil, fmt.Errorf("ошибка валидации конфигурации по умолчанию: %w", err)
		}
	}

	return cfg, nil
}

// DefaultConfig возвращает конфигурацию по умолчанию
func DefaultConfig() *Config {
	return &Config{
		Telegram: TelegramConfig{
			Debug:          false,
			UpdatesTimeout: 60,
		},
		API: APIConfig{
			URL:        "http://localhost:8080",
			Timeout:    30 * time.Second,
			MaxRetries: 3,
			RetryDelay: 2 * time.Second,
		},
		Bot: BotConfig{
			Name:              "MOEX Data Bot",
			Greeting:          "Добро пожаловать! Я помогу вам работать с данными Московской биржи.",
			HelpMessage:       "Используйте команды для получения данных...",
			RateLimitInterval: time.Second,
			MaxMessageLength:  4096,
			CommandTimeout:    5 * time.Minute,
		},
		Strategy: StrategyConfig{
			Turtles: TurtleStrategyConfig{
				Enabled:           false,
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
				Enabled:         false,
				DailyReport:     false,
				AlertOnBreakout: false,
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
			EnableAuth: false,
		},
	}
}

// overrideFromEnv переопределяет значения из переменных окружения
func overrideFromEnv(cfg *Config) {
	// Telegram
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		cfg.Telegram.Token = token
	}
	if debug := os.Getenv("TELEGRAM_DEBUG"); debug != "" {
		if val, err := strconv.ParseBool(debug); err == nil {
			cfg.Telegram.Debug = val
		}
	}

	// API
	if apiURL := os.Getenv("API_URL"); apiURL != "" {
		cfg.API.URL = apiURL
	}
	if apiToken := os.Getenv("API_TOKEN"); apiToken != "" {
		cfg.API.Token = apiToken
	}
	if timeout := os.Getenv("API_TIMEOUT"); timeout != "" {
		if val, err := time.ParseDuration(timeout); err == nil {
			cfg.API.Timeout = val
		}
	}
	if retries := os.Getenv("API_MAX_RETRIES"); retries != "" {
		if val, err := strconv.Atoi(retries); err == nil {
			cfg.API.MaxRetries = val
		}
	}
	if delay := os.Getenv("API_RETRY_DELAY"); delay != "" {
		if val, err := time.ParseDuration(delay); err == nil {
			cfg.API.RetryDelay = val
		}
	}

	// Security
	if usersStr := os.Getenv("ALLOWED_USERS"); usersStr != "" {
		var users []int64
		for _, userStr := range strings.Split(usersStr, ",") {
			if userID, err := strconv.ParseInt(strings.TrimSpace(userStr), 10, 64); err == nil {
				users = append(users, userID)
			}
		}
		cfg.Security.AllowedUsers = users
	}
	if adminStr := os.Getenv("ADMIN_USERS"); adminStr != "" {
		var admins []int64
		for _, adminStr := range strings.Split(adminStr, ",") {
			if adminID, err := strconv.ParseInt(strings.TrimSpace(adminStr), 10, 64); err == nil {
				admins = append(admins, adminID)
			}
		}
		cfg.Security.AdminUsers = admins
	}
	if auth := os.Getenv("ENABLE_AUTH"); auth != "" {
		if val, err := strconv.ParseBool(auth); err == nil {
			cfg.Security.EnableAuth = val
		}
	}

	// Logging
	if level := os.Getenv("LOG_LEVEL"); level != "" {
		cfg.Logging.Level = level
	}
	if file := os.Getenv("LOG_FILE"); file != "" {
		cfg.Logging.File = file
	}

	// Strategy
	if enabled := os.Getenv("STRATEGY_TURTLES_ENABLED"); enabled != "" {
		if val, err := strconv.ParseBool(enabled); err == nil {
			cfg.Strategy.Turtles.Enabled = val
		}
	}
	if risk := os.Getenv("STRATEGY_RISK_PER_TRADE"); risk != "" {
		if val, err := strconv.ParseFloat(risk, 64); err == nil {
			cfg.Strategy.Turtles.RiskPerTrade = val
		}
	}
}

// OverrideFromEnv переопределяет значения из переменных окружения (публичная версия)
func OverrideFromEnv(cfg *Config) {
	overrideFromEnv(cfg)
}

// SaveConfig сохраняет конфигурацию в файл
func SaveConfig(cfg *Config, configPath string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга конфигурации: %w", err)
	}

	// Создаем директорию если не существует
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("ошибка создания директории: %w", err)
	}

	// Сохраняем файл
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("ошибка записи файла: %w", err)
	}

	return nil
}
