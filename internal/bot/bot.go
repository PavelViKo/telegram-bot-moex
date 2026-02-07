package bot

import (
	"context"
	"fmt"
	"runtime/debug"
	"sync"
	"time"

	"telegram-bot-moex/internal/analysis"
	"telegram-bot-moex/internal/api"
	"telegram-bot-moex/internal/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Logger интерфейс для логирования
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
}

type Bot struct {
	config     *config.Config
	botAPI     *tgbotapi.BotAPI
	apiClient  *api.APIClient
	logger     Logger
	commands   map[string]CommandHandler
	userStates map[int64]*UserState
	stats      *BotStats
	mu         sync.RWMutex
	stopChan   chan struct{}

	// Для фонового анализа
	analysisTicker   *time.Ticker
	analysisStopChan chan struct{}
	analysisWg       sync.WaitGroup
}

// NewBot создает новый экземпляр бота
func NewBot(cfg *config.Config, logger Logger) (*Bot, error) {
	// Инициализация Telegram API
	botAPI, err := tgbotapi.NewBotAPI(cfg.Telegram.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	botAPI.Debug = cfg.Telegram.Debug

	// Создание API клиента
	apiClient := api.NewAPIClient(
		cfg.API.URL,
		cfg.API.Token,
		cfg.API.Timeout,
	)

	bot := &Bot{
		config:           cfg,
		botAPI:           botAPI,
		apiClient:        apiClient,
		logger:           logger,
		commands:         make(map[string]CommandHandler),
		userStates:       make(map[int64]*UserState),
		stats:            NewBotStats(),
		stopChan:         make(chan struct{}),
		analysisStopChan: make(chan struct{}),
	}

	// Регистрация команд
	bot.registerCommands()

	// Установка команд меню
	bot.setBotCommands()

	return bot, nil
}

// Start запускает бота
func (b *Bot) Start(ctx context.Context) error {
	b.logger.Info("Starting bot",
		"username", b.botAPI.Self.UserName,
		"id", b.botAPI.Self.ID,
	)

	// Запускаем горутину для очистки неактивных состояний
	go b.startCleanupRoutine(ctx)

	// Запускаем фоновый анализ стратегий
	go b.startBackgroundAnalysis(ctx)

	// Используем только polling режим
	return b.startPolling(ctx)
}

// startPolling запускает polling режим
func (b *Bot) startPolling(ctx context.Context) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = b.config.Telegram.UpdatesTimeout

	updates := b.botAPI.GetUpdatesChan(u)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case update := <-updates:
			go b.handleUpdate(update)
		}
	}
}

// Shutdown корректно останавливает бота
func (b *Bot) Shutdown(ctx context.Context) error {
	b.logger.Info("Shutting down bot")

	// Останавливаем фоновый анализ
	close(b.analysisStopChan)

	// Ждем завершения анализа
	analysisCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	analysisDone := make(chan struct{})
	go func() {
		b.analysisWg.Wait()
		close(analysisDone)
	}()

	select {
	case <-analysisDone:
		b.logger.Info("Фоновый анализ остановлен")
	case <-analysisCtx.Done():
		b.logger.Warn("Таймаут остановки фонового анализа")
	}

	// Закрываем канал остановки
	close(b.stopChan)

	// Очищаем состояния пользователей
	b.mu.Lock()
	for userID, state := range b.userStates {
		if state.CancelFunc != nil {
			state.CancelFunc()
		}
		delete(b.userStates, userID)
	}
	b.mu.Unlock()

	b.logger.Info("Bot shutdown completed")
	return nil
}

// GetUsername возвращает username бота
func (b *Bot) GetUsername() string {
	return b.botAPI.Self.UserName
}

// GetStats возвращает статистику бота
func (b *Bot) GetStats() *BotStats {
	return b.stats
}

// GetConfig возвращает конфигурацию бота
func (b *Bot) GetConfig() *config.Config {
	return b.config
}

// startTurtleAnalysisRoutine запускает горутину для анализа по стратегии "Черепах"
func (b *Bot) startTurtleAnalysisRoutine(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour) // Анализируем каждый час
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.analyzeTurtleStrategy()
		}
	}
}

func (b *Bot) startMAAnalysisRoutine(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Hour) // Анализируем каждые 2 часа
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			b.analyzeMAStrategy()
		}
	}
}

// startBackgroundAnalysis запускает фоновый анализ всех стратегий
func (b *Bot) startBackgroundAnalysis(ctx context.Context) {
	b.logger.Info("Запуск фонового анализа стратегий")

	// Создаем тикеры для каждой стратегии
	var tickers []*time.Ticker

	// Анализ по стратегии "Черепах"
	if b.config.Strategy.Turtles.Enabled && b.config.Strategy.Notifications.Enabled {
		turtleTicker := time.NewTicker(1 * time.Hour)
		tickers = append(tickers, turtleTicker)
		b.startStrategyAnalysis(ctx, "turtle", turtleTicker, b.analyzeTurtleStrategy)
	}

	// Анализ по стратегии MA Crossover
	if b.config.Strategy.MACrossover.Enabled && b.config.Strategy.Notifications.Enabled {
		maTicker := time.NewTicker(2 * time.Hour)
		tickers = append(tickers, maTicker)
		b.startStrategyAnalysis(ctx, "ma_crossover", maTicker, b.analyzeMAStrategy)
	}

	// Ожидаем остановки
	<-ctx.Done()

	// Останавливаем все тикеры
	for _, ticker := range tickers {
		ticker.Stop()
	}

	// Ожидаем завершения всех горутин анализа
	b.analysisWg.Wait()
	b.logger.Info("Фоновый анализ остановлен")
}

// startStrategyAnalysis запускает анализ для конкретной стратегии
func (b *Bot) startStrategyAnalysis(ctx context.Context, strategyName string, ticker *time.Ticker, analysisFunc func()) {
	b.analysisWg.Add(1)

	go func() {
		defer b.analysisWg.Done()
		defer ticker.Stop()

		b.logger.Info("Запуск фонового анализа", "strategy", strategyName)

		// Первый запуск сразу
		b.safeAnalysisRun(strategyName, analysisFunc)

		for {
			select {
			case <-ctx.Done():
				b.logger.Info("Остановка анализа", "strategy", strategyName)
				return
			case <-ticker.C:
				b.safeAnalysisRun(strategyName, analysisFunc)
			}
		}
	}()
}

// safeAnalysisRun безопасно запускает функцию анализа
func (b *Bot) safeAnalysisRun(strategyName string, analysisFunc func()) {
	defer func() {
		if r := recover(); r != nil {
			b.logger.Error("Паника при анализе стратегии",
				"strategy", strategyName,
				"panic", r,
				"stack", string(debug.Stack()))
		}
	}()

	startTime := time.Now()
	b.logger.Debug("Запуск анализа", "strategy", strategyName)

	analysisFunc()

	duration := time.Since(startTime)
	b.logger.Debug("Анализ завершен",
		"strategy", strategyName,
		"duration", duration)
}

// analyzeTurtleStrategy анализирует инструменты по стратегии "Черепах"
func (b *Bot) analyzeTurtleStrategy() {
	b.logger.Debug("Запуск анализа по стратегии 'Черепах'")

	// Проверяем, нужно ли отправлять уведомления
	if !b.config.Strategy.Notifications.Enabled ||
		b.config.Strategy.Notifications.SignalChatID == 0 {
		b.logger.Debug("Уведомления отключены для стратегии 'Черепах'")
		return
	}

	// Запускаем сканирование в фоне
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// Создаем стратегию
		strategy := analysis.NewTurtleStrategy(
			b.apiClient,
			b.config.Strategy.Turtles.LookbackPeriod,
			b.config.Strategy.Turtles.EntryBreakoutDays,
			b.config.Strategy.Turtles.ExitBreakoutDays,
			b.config.Strategy.Turtles.AtrPeriod,
			b.config.Strategy.Turtles.AtrMultiplier,
			b.config.Strategy.Turtles.RiskPerTrade,
		)

		// Получаем инструменты
		instruments, err := b.apiClient.GetInstruments(ctx)
		if err != nil {
			b.logger.Error("Ошибка получения инструментов для анализа",
				"strategy", "turtle",
				"error", err)
			return
		}

		if len(instruments) == 0 {
			b.logger.Debug("Нет инструментов для анализа", "strategy", "turtle")
			return
		}

		// Анализируем каждый инструмент
		var signals []analysis.Signal
		analyzedCount := 0

		for _, instrument := range instruments {
			// Пропускаем некорректные инструменты
			if !b.isValidInstrument(instrument) {
				continue
			}

			instrumentSignals, err := strategy.AnalyzeInstrument(ctx, instrument)
			if err != nil {
				b.logger.Debug("Ошибка анализа инструмента",
					"strategy", "turtle",
					"instrument", instrument,
					"error", err)
				continue
			}

			analyzedCount++

			// Добавляем только сигналы на вход
			for _, signal := range instrumentSignals {
				if signal.SignalType == "entry_long" || signal.SignalType == "entry_short" {
					signals = append(signals, signal)
				}
			}

			// Делаем паузу между запросами
			time.Sleep(100 * time.Millisecond)
		}

		b.logger.Info("Анализ стратегии 'Черепах' завершен",
			"instruments", analyzedCount,
			"signals", len(signals))

		// Если есть сигналы, отправляем уведомление
		if len(signals) > 0 {
			b.sendStrategyNotification("turtle", signals, b.config.Strategy.Notifications.SignalChatID)
		}
	}()
}

// analyzeMAStrategy анализирует инструменты по стратегии MA Crossover
func (b *Bot) analyzeMAStrategy() {
	b.logger.Debug("Запуск анализа по стратегии MA Crossover")

	// Проверяем, нужно ли отправлять уведомления
	if !b.config.Strategy.Notifications.Enabled ||
		b.config.Strategy.Notifications.SignalChatID == 0 {
		b.logger.Debug("Уведомления отключены для стратегии MA Crossover")
		return
	}

	// Запускаем сканирование в фоне
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		// Создаем стратегию
		strategy := b.createMACrossoverStrategy()

		// Получаем инструменты
		instruments, err := b.apiClient.GetInstruments(ctx)
		if err != nil {
			b.logger.Error("Ошибка получения инструментов для анализа",
				"strategy", "ma_crossover",
				"error", err)
			return
		}

		if len(instruments) == 0 {
			b.logger.Debug("Нет инструментов для анализа", "strategy", "ma_crossover")
			return
		}

		// Анализируем каждый инструмент
		var signals []analysis.Signal
		analyzedCount := 0

		for _, instrument := range instruments {
			// Пропускаем некорректные инструменты
			if !b.isValidInstrument(instrument) {
				continue
			}

			instrumentSignals, err := strategy.Analyze(ctx, instrument)
			if err != nil {
				b.logger.Debug("Ошибка анализа инструмента",
					"strategy", "ma_crossover",
					"instrument", instrument,
					"error", err)
				continue
			}

			analyzedCount++

			// Добавляем сигналы на вход
			for _, signal := range instrumentSignals {
				if signal.SignalType == "entry_long" || signal.SignalType == "entry_short" {
					signals = append(signals, signal)
				}
			}

			// Делаем паузу между запросами
			time.Sleep(100 * time.Millisecond)
		}

		b.logger.Info("Анализ стратегии MA Crossover завершен",
			"instruments", analyzedCount,
			"signals", len(signals))

		// Если есть сигналы, отправляем уведомление
		if len(signals) > 0 {
			b.sendStrategyNotification("ma_crossover", signals, b.config.Strategy.Notifications.SignalChatID)
		}
	}()
}

// sendStrategyNotification отправляет уведомление о сигналах
func (b *Bot) sendStrategyNotification(strategyName string, signals []analysis.Signal, chatID int64) {
	if len(signals) == 0 || chatID == 0 {
		return
	}

	// Группируем сигналы по типу
	entryLong := []analysis.Signal{}
	entryShort := []analysis.Signal{}

	for _, signal := range signals {
		switch signal.SignalType {
		case "entry_long":
			entryLong = append(entryLong, signal)
		case "entry_short":
			entryShort = append(entryShort, signal)
		}
	}

	// Формируем сообщение
	var msg string

	switch strategyName {
	case "turtle":
		msg = "🐢 СИГНАЛЫ СТРАТЕГИИ 'ЧЕРЕПАХ'\n\n"
	case "ma_crossover":
		msg = "📊 СИГНАЛЫ MA CROSSOVER\n\n"
	default:
		msg = "📈 ТОРГОВЫЕ СИГНАЛЫ\n\n"
	}

	msg += fmt.Sprintf("📅 Время анализа: %s\n", time.Now().Format("02.01.2006 15:04"))
	msg += fmt.Sprintf("📊 Всего сигналов: %d\n\n", len(signals))

	// Добавляем сигналы на покупку
	if len(entryLong) > 0 {
		msg += "🟢 СИГНАЛЫ НА ПОКУПКУ:\n"
		for i, signal := range entryLong {
			if i >= 10 { // Ограничиваем количество сигналов
				msg += fmt.Sprintf("... и ещё %d сигналов\n", len(entryLong)-10)
				break
			}
			msg += fmt.Sprintf("• %s - %.2f₽", signal.Instrument, signal.Price)
			if signal.StopLoss > 0 && signal.TakeProfit > 0 {
				stopPercent := ((signal.Price - signal.StopLoss) / signal.Price) * 100
				profitPercent := ((signal.TakeProfit - signal.Price) / signal.Price) * 100
				msg += fmt.Sprintf(" (SL: -%.1f%%, TP: +%.1f%%)", stopPercent, profitPercent)
			}
			msg += "\n"
		}
		msg += "\n"
	}

	// Добавляем сигналы на продажу
	if len(entryShort) > 0 {
		msg += "🔴 СИГНАЛЫ НА ПРОДАЖУ:\n"
		for i, signal := range entryShort {
			if i >= 10 {
				msg += fmt.Sprintf("... и ещё %d сигналов\n", len(entryShort)-10)
				break
			}
			msg += fmt.Sprintf("• %s - %.2f₽", signal.Instrument, signal.Price)
			if signal.StopLoss > 0 && signal.TakeProfit > 0 {
				stopPercent := ((signal.StopLoss - signal.Price) / signal.Price) * 100
				profitPercent := ((signal.Price - signal.TakeProfit) / signal.Price) * 100
				msg += fmt.Sprintf(" (SL: +%.1f%%, TP: -%.1f%%)", stopPercent, profitPercent)
			}
			msg += "\n"
		}
		msg += "\n"
	}

	msg += "💡 Для получения детальной информации используйте команды:\n"
	if strategyName == "turtle" {
		msg += "• /turtle_signals - детальные сигналы 'Черепах'\n"
		msg += "• /scan_turtles - полное сканирование\n"
	} else if strategyName == "ma_crossover" {
		msg += "• /ma_signals - детальные сигналы MA Crossover\n"
		msg += "• /scan_ma - полное сканирование\n"
	}

	// Отправляем сообщение
	if err := b.sendFormattedMessage(chatID, msg); err != nil {
		b.logger.Error("Ошибка отправки уведомления",
			"strategy", strategyName,
			"chat_id", chatID,
			"error", err)
	}
}
