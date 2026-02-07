package bot

import (
	"context"
	"fmt"
	"time"

	"telegram-bot-moex/internal/analysis"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Регистрируем MA команды в registerCommands() в commands.go
func (b *Bot) registerMACommands() {
	// Убираем b.handleMAAnalysis, так как этого метода нет
	b.commands["ma"] = b.handleMA
	b.commands["ma_signals"] = b.handleMASignals
	b.commands["scan_ma"] = b.handleScanMA
	b.commands["ma_config"] = b.handleMAConfig
	b.commands["ma_test"] = b.handleMATest
}

// handleMA обработчик команды /ma
func (b *Bot) handleMA(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	msg := "📊 АНАЛИЗ ПО СТРАТЕГИИ MOVING AVERAGE CROSSOVER\n\n"

	if !b.config.Strategy.MACrossover.Enabled {
		msg += "❌ Стратегия отключена\n\n"
		msg += "Используйте /ma_config для включения и настройки"
		return b.sendFormattedMessage(chatID, msg)
	}

	msg += "📖 ОПИСАНИЕ СТРАТЕГИИ:\n"
	msg += "Классическая стратегия следования за трендом на основе пересечения скользящих средних.\n\n"

	msg += "⚙️ ТЕКУЩИЕ НАСТРОЙКИ:\n"
	cfg := b.config.Strategy.MACrossover
	maType := "SMA"
	if cfg.UseEMA {
		maType = "EMA"
	}

	msg += fmt.Sprintf("• Статус: %s\n", b.getMAStatus())
	msg += fmt.Sprintf("• Тип MA: %s\n", maType)
	msg += fmt.Sprintf("• Быстрая MA: %d периодов\n", cfg.FastPeriod)
	msg += fmt.Sprintf("• Медленная MA: %d периодов\n", cfg.SlowPeriod)
	if cfg.SignalPeriod > 0 {
		msg += fmt.Sprintf("• Сигнальная MA: %d периодов\n", cfg.SignalPeriod)
	}
	msg += fmt.Sprintf("• Таймфрейм: %s\n", cfg.Timeframe)
	msg += fmt.Sprintf("• Риск на сделку: %.1f%%\n", cfg.RiskPerTrade*100)
	msg += fmt.Sprintf("• Стоп-лосс: %.1fxATR\n", cfg.StopLossATRMultiplier)
	msg += fmt.Sprintf("• Тейк-профит: 1:%.1f\n", cfg.TakeProfitRatio)
	msg += fmt.Sprintf("• Подтверждение объема: %v\n", cfg.UseVolumeConfirmation)
	msg += fmt.Sprintf("• RSI фильтр: %v\n\n", cfg.Filters.RSIFilter)

	msg += "🎯 ТИПЫ СИГНАЛОВ:\n"
	if cfg.CrossoverTypes.GoldenCross {
		msg += "• 🟢 Золотое пересечение (покупка)\n"
	}
	if cfg.CrossoverTypes.DeathCross {
		msg += "• 🔴 Мертвое пересечение (продажа)\n"
	}
	msg += fmt.Sprintf("• Подтверждение: %d свечей\n\n", cfg.CrossoverTypes.RequireConfirmation)

	msg += "📈 КОМАНДЫ УПРАВЛЕНИЯ:\n"
	msg += "• /ma_signals - Текущие сигналы\n"
	msg += "• /scan_ma - Сканировать все инструменты\n"
	msg += "• /ma_config - Настройки стратегии\n"
	msg += "• /ma_test - Тестирование на инструменте\n"

	// Добавляем кнопки управления
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📈 Сигналы", "ma_signals"),
			tgbotapi.NewInlineKeyboardButtonData("🔍 Сканировать", "ma_scan"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "ma_config"),
			tgbotapi.NewInlineKeyboardButtonData("🧪 Тестировать", "ma_test"),
		),
	)

	return b.sendMessageWithKeyboard(chatID, msg, keyboard)
}

// handleMASignals обработчик команды /ma_signals
func (b *Bot) handleMASignals(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	if !b.config.Strategy.MACrossover.Enabled {
		return b.sendFormattedMessage(chatID, "❌ Стратегия MA Crossover отключена.\nИспользуйте /ma_config для включения.")
	}

	b.sendFormattedMessage(chatID, "🔍 Поиск сигналов по стратегии MA Crossover...")

	// Запускаем анализ в фоне
	go b.scanAndShowMASignals(chatID)

	return nil
}

// handleScanMA обработчик команды /scan_ma
func (b *Bot) handleScanMA(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	if !b.config.Strategy.MACrossover.Enabled {
		return b.sendFormattedMessage(chatID, "❌ Стратегия MA Crossover отключена.\nИспользуйте /ma_config для включения.")
	}

	b.sendFormattedMessage(chatID, "🔍 Сканирование всех инструментов по стратегии MA Crossover...\n\n⏳ Это может занять несколько минут.")

	// Запускаем сканирование в фоне
	go b.scanAndShowMASignals(chatID)

	return nil
}

// handleMAConfig обработчик команды /ma_config
func (b *Bot) handleMAConfig(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.isAdmin(userID) {
		return b.sendFormattedMessage(chatID, "❌ Настройки стратегии доступны только администраторам")
	}

	msg := "⚙️ НАСТРОЙКИ СТРАТЕГИИ MA CROSSOVER\n\n"

	cfg := b.config.Strategy.MACrossover
	maType := "SMA"
	if cfg.UseEMA {
		maType = "EMA"
	}

	msg += "📊 ТЕКУЩИЕ НАСТРОЙКИ:\n"
	msg += fmt.Sprintf("• Статус: %s\n", b.getMAStatus())
	msg += fmt.Sprintf("• Тип MA: %s\n", maType)
	msg += fmt.Sprintf("• Быстрая MA: %d периодов\n", cfg.FastPeriod)
	msg += fmt.Sprintf("• Медленная MA: %d периодов\n", cfg.SlowPeriod)
	msg += fmt.Sprintf("• Таймфрейм: %s\n", cfg.Timeframe)
	msg += fmt.Sprintf("• Риск на сделку: %.1f%%\n", cfg.RiskPerTrade*100)
	msg += fmt.Sprintf("• Стоп-лосс: %.1fxATR\n", cfg.StopLossATRMultiplier)
	msg += fmt.Sprintf("• Тейк-профит: 1:%.1f\n", cfg.TakeProfitRatio)
	msg += fmt.Sprintf("• Подтверждение объема: %v\n", cfg.UseVolumeConfirmation)
	if cfg.UseVolumeConfirmation {
		msg += fmt.Sprintf("• Мин. множитель объема: %.1fx\n", cfg.MinVolumeMultiplier)
	}
	msg += fmt.Sprintf("• RSI фильтр: %v\n", cfg.Filters.RSIFilter)
	if cfg.Filters.RSIFilter {
		msg += fmt.Sprintf("• RSI перекупленность: %d\n", cfg.Filters.RSIOverbought)
		msg += fmt.Sprintf("• RSI перепроданность: %d\n", cfg.Filters.RSIOversold)
	}
	msg += fmt.Sprintf("• Фильтр тренда: %s\n\n", cfg.Filters.TrendFilter)

	msg += "⚡ БЫСТРЫЕ ДЕЙСТВИЯ:\n"

	// Добавляем кнопки управления
	var rows [][]tgbotapi.InlineKeyboardButton

	if cfg.Enabled {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔴 Выключить", "ma_disable"),
			tgbotapi.NewInlineKeyboardButtonData("🔄 Тип MA", "ma_toggle_type"),
		))
	} else {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟢 Включить", "ma_enable"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Параметры", "ma_params"),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📊 Быстрая 9", "ma_set_fast_9"),
		tgbotapi.NewInlineKeyboardButtonData("📊 Быстрая 12", "ma_set_fast_12"),
		tgbotapi.NewInlineKeyboardButtonData("📊 Быстрая 20", "ma_set_fast_20"),
	))

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📈 Медленная 21", "ma_set_slow_21"),
		tgbotapi.NewInlineKeyboardButtonData("📈 Медленная 50", "ma_set_slow_50"),
		tgbotapi.NewInlineKeyboardButtonData("📈 Медленная 200", "ma_set_slow_200"),
	))

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🎯 Риск 1%", "ma_set_risk_0.01"),
		tgbotapi.NewInlineKeyboardButtonData("🎯 Риск 2%", "ma_set_risk_0.02"),
		tgbotapi.NewInlineKeyboardButtonData("🎯 Риск 5%", "ma_set_risk_0.05"),
	))

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🧪 Тестировать", "ma_test"),
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "help_strategy"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	return b.sendMessageWithKeyboard(chatID, msg, keyboard)
}

// handleMATest обработчик команды /ma_test
func (b *Bot) handleMATest(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.config.Strategy.MACrossover.Enabled {
		return b.sendFormattedMessage(chatID, "❌ Стратегия MA Crossover отключена.\nИспользуйте /ma_config для включения.")
	}

	// Начинаем процесс тестирования
	state := &UserState{
		CurrentCommand: "ma_test",
		Step:           1,
		Data:           make(map[string]interface{}),
		LastActivity:   time.Now(),
	}
	b.setUserState(userID, state)

	msg := "🧪 ТЕСТИРОВАНИЕ СТРАТЕГИИ MA CROSSOVER\n\n"
	msg += "Этот тест покажет, как работает стратегия на конкретном инструменте.\n\n"
	msg += "Введите тикер инструмента для тестирования (например: SBER):"

	return b.sendFormattedMessage(chatID, msg)
}

// scanAndShowMASignals сканирует и показывает сигналы MA Crossover
func (b *Bot) scanAndShowMASignals(chatID int64) {
	// Создаем стратегию
	strategy := b.createMACrossoverStrategy()

	// Получаем инструменты
	instruments, err := b.apiClient.GetInstruments(context.Background())
	if err != nil {
		b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка получения инструментов: %v", err))
		return
	}

	if len(instruments) == 0 {
		b.sendFormattedMessage(chatID, "📭 Нет доступных инструментов для анализа")
		return
	}

	// Анализируем каждый инструмент
	var allSignals []analysis.Signal
	totalInstruments := 0
	analyzedInstruments := 0

	for _, instrument := range instruments {
		totalInstruments++

		// Пропускаем некорректные инструменты
		if !b.isValidInstrument(instrument) {
			continue
		}

		signals, err := strategy.Analyze(context.Background(), instrument)
		if err != nil {
			b.logger.Debug("Ошибка анализа инструмента", "instrument", instrument, "error", err)
			continue
		}

		analyzedInstruments++

		if len(signals) > 0 {
			allSignals = append(allSignals, signals...)
		}

		// Делаем небольшую паузу между запросами
		time.Sleep(100 * time.Millisecond)
	}

	// Формируем сообщение с результатами
	msg := "📈 РЕЗУЛЬТАТЫ СКАНИРОВАНИЯ MA CROSSOVER\n\n"
	msg += fmt.Sprintf("📊 Всего инструментов: %d\n", totalInstruments)
	msg += fmt.Sprintf("📊 Проанализировано: %d\n", analyzedInstruments)
	msg += fmt.Sprintf("🚨 Найдено сигналов: %d\n\n", len(allSignals))

	if len(allSignals) == 0 {
		msg += "📭 Торговых сигналов не найдено\n"
		msg += "💡 Возможные причины:\n"
		msg += "• Нет пересечений скользящих средних\n"
		msg += "• Сигналы не прошли фильтры (объем, RSI)\n"
		msg += "• Недостаточно данных для анализа\n\n"
		msg += "Попробуйте изменить параметры стратегии через /ma_config"
		b.sendFormattedMessage(chatID, msg)
		return
	}

	// Группируем сигналы
	goldenCross := []analysis.Signal{}
	deathCross := []analysis.Signal{}

	for _, signal := range allSignals {
		switch signal.SignalType {
		case "entry_long":
			goldenCross = append(goldenCross, signal)
		case "entry_short":
			deathCross = append(deathCross, signal)
		}
	}

	// Добавляем сигналы в сообщение
	if len(goldenCross) > 0 {
		msg += "🟢 ЗОЛОТЫЕ ПЕРЕСЕЧЕНИЯ (ПОКУПКА):\n"
		for _, signal := range goldenCross {
			msg += fmt.Sprintf("• %s - %.2f₽\n", signal.Instrument, signal.Price)
			if signal.StopLoss > 0 {
				msg += fmt.Sprintf("  Стоп: %.2f | Тейк: %.2f\n", signal.StopLoss, signal.TakeProfit)
			}
			if signal.PositionSize > 0 {
				msg += fmt.Sprintf("  Размер: %.0f шт.\n", signal.PositionSize)
			}
			msg += fmt.Sprintf("  📅 %s\n\n", signal.Timestamp.Format("02.01 15:04"))
		}
	}

	if len(deathCross) > 0 {
		msg += "🔴 МЕРТВЫЕ ПЕРЕСЕЧЕНИЯ (ПРОДАЖА):\n"
		for _, signal := range deathCross {
			msg += fmt.Sprintf("• %s - %.2f₽\n", signal.Instrument, signal.Price)
			if signal.StopLoss > 0 {
				msg += fmt.Sprintf("  Стоп: %.2f | Тейк: %.2f\n", signal.StopLoss, signal.TakeProfit)
			}
			if signal.PositionSize > 0 {
				msg += fmt.Sprintf("  Размер: %.0f шт.\n", signal.PositionSize)
			}
			msg += fmt.Sprintf("  📅 %s\n\n", signal.Timestamp.Format("02.01 15:04"))
		}
	}

	msg += "\n💡 РЕКОМЕНДАЦИИ:\n"
	msg += "• Проверьте объемы по инструментам\n"
	msg += "• Учитывайте общий рыночный тренд\n"
	msg += "• Используйте стоп-лоссы для управления рисками\n"

	// Добавляем кнопки действий
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "ma_signals"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "ma_config"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Подробнее о стратегии", "help_strategy"),
			tgbotapi.NewInlineKeyboardButtonData("🧪 Тестировать", "ma_test"),
		),
	)

	b.sendSafeMessageWithKeyboard(chatID, msg, keyboard)
}

// createMACrossoverStrategy создает стратегию MA Crossover из конфига
func (b *Bot) createMACrossoverStrategy() *analysis.MACrossoverStrategy {
	cfg := b.config.Strategy.MACrossover

	maConfig := analysis.MACrossoverConfig{
		Timeframe:             cfg.Timeframe,
		FastPeriod:            cfg.FastPeriod,
		SlowPeriod:            cfg.SlowPeriod,
		SignalPeriod:          cfg.SignalPeriod,
		UseEMA:                cfg.UseEMA,
		UseVolumeConfirmation: cfg.UseVolumeConfirmation,
		MinVolumeMultiplier:   cfg.MinVolumeMultiplier,
		RiskPerTrade:          cfg.RiskPerTrade,
		StopLossATRMultiplier: cfg.StopLossATRMultiplier,
		TakeProfitRatio:       cfg.TakeProfitRatio,
	}

	maConfig.CrossoverTypes.GoldenCross = cfg.CrossoverTypes.GoldenCross
	maConfig.CrossoverTypes.DeathCross = cfg.CrossoverTypes.DeathCross
	maConfig.CrossoverTypes.RequireConfirmation = cfg.CrossoverTypes.RequireConfirmation

	maConfig.Filters.TrendFilter = cfg.Filters.TrendFilter
	maConfig.Filters.RSIFilter = cfg.Filters.RSIFilter
	maConfig.Filters.RSIOverbought = cfg.Filters.RSIOverbought
	maConfig.Filters.RSIOversold = cfg.Filters.RSIOversold

	return analysis.NewMACrossoverStrategy(b.apiClient, maConfig)
}

// getMAStatus возвращает статус стратегии MA Crossover
func (b *Bot) getMAStatus() string {
	if b.config.Strategy.MACrossover.Enabled {
		return "🟢 ВКЛЮЧЕНА"
	}
	return "🔴 ВЫКЛЮЧЕНА"
}
