package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telegram-bot-moex/internal/bot"
	"telegram-bot-moex/internal/config"
	"telegram-bot-moex/internal/utils"
)

func main() {
	// Загрузка конфигурации
	cfgPath, err := config.GetConfigPath("")
	if err != nil {
		log.Printf("⚠️ Конфигурационный файл не найден, используем значения по умолчанию: %v", err)
	}

	var cfg *config.Config
	if cfgPath != "" {
		cfg, err = config.LoadConfig(cfgPath)
		if err != nil {
			log.Fatalf("❌ Ошибка загрузки конфигурации: %v", err)
		}
	} else {
		cfg = config.DefaultConfig()
		// Загружаем из переменных окружения
		config.OverrideFromEnv(cfg)
	}

	// Выводим конфигурацию (без секретов)
	log.Println(cfg.PrintConfig())

	// Выводим информацию о стратегиях
	log.Printf("📊 Конфигурация стратегий:")
	log.Printf("   • 'Черепах': %v (уведомления: %v)",
		cfg.Strategy.Turtles.Enabled,
		cfg.Strategy.Notifications.Enabled)
	log.Printf("   • MA Crossover: %v (уведомления: %v)",
		cfg.Strategy.MACrossover.Enabled,
		cfg.Strategy.Notifications.Enabled)
	log.Printf("   • ID чата для уведомлений: %d",
		cfg.Strategy.Notifications.SignalChatID)

	// Создаем необходимые директории
	if err := cfg.EnsureDirectories(); err != nil {
		log.Printf("⚠️ Ошибка создания директорий: %v", err)
	}

	// Инициализация логгера
	logger := utils.NewLogger(cfg.GetLogFile(), cfg.Logging.Level, cfg.Logging.JSONFormat)
	defer logger.Sync()

	// Создание бота
	telegramBot, err := bot.NewBot(cfg, logger)
	if err != nil {
		logger.Fatal("Ошибка создания бота", "error", err)
	}

	// Контекст для graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Канал для сигналов ОС
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	// Запуск бота в горутине
	go func() {
		logger.Info("Запуск бота...")
		if err := telegramBot.Start(ctx); err != nil {
			logger.Error("Ошибка запуска бота", "error", err)
			cancel()
		}
	}()

	logger.Info("🤖 Бот запущен",
		"username", telegramBot.GetUsername(),
		"start_time", time.Now().Format(time.RFC3339),
	)

	// Логируем информацию о стратегиях
	if cfg.Strategy.Turtles.Enabled {
		logger.Info("Стратегия 'Черепах' включена",
			"таймфрейм", cfg.Strategy.Turtles.Timeframe,
			"период", cfg.Strategy.Turtles.LookbackPeriod,
			"автоанализ", "каждый час")
	}

	if cfg.Strategy.MACrossover.Enabled {
		logger.Info("Стратегия MA Crossover включена",
			"быстрая MA", cfg.Strategy.MACrossover.FastPeriod,
			"медленная MA", cfg.Strategy.MACrossover.SlowPeriod,
			"автоанализ", "каждые 2 часа")
	}

	// Ожидание сигнала завершения
	select {
	case sig := <-sigChan:
		logger.Info("Получен сигнал завершения", "signal", sig)
	case <-ctx.Done():
		logger.Info("Контекст завершен")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	logger.Info("Остановка бота...")
	if err := telegramBot.Shutdown(shutdownCtx); err != nil {
		logger.Error("Ошибка graceful shutdown", "error", err)
	}

	logger.Info("👋 Бот остановлен")
}
