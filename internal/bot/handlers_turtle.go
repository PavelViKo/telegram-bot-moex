package bot

import (
	"context"
	"fmt"
	"math"
	"runtime/debug"
	"telegram-bot-moex/internal/analysis"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleTurtleAnalysis(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	msg := "📊 АНАЛИЗ ПО СТРАТЕГИИ 'ЧЕРЕПАХ'\n\n"

	msg += "📖 ОПИСАНИЕ СТРАТЕГИИ:\n"
	msg += "Стратегия следования за трендом, основанная на прорывах ценовых уровней.\n\n"

	msg += "⚙️ ТЕКУЩИЕ НАСТРОЙКИ:\n"
	msg += fmt.Sprintf("• Статус: %s\n", b.getTurtleStatus())
	msg += fmt.Sprintf("• Таймфрейм: %s (дневной)\n", b.config.Strategy.Turtles.Timeframe)
	msg += fmt.Sprintf("• Период анализа: %d дней\n", b.config.Strategy.Turtles.LookbackPeriod)
	msg += fmt.Sprintf("• Прорыв для входа: %d дней\n", b.config.Strategy.Turtles.EntryBreakoutDays)
	msg += fmt.Sprintf("• Прорыв для выхода: %d дней\n", b.config.Strategy.Turtles.ExitBreakoutDays)
	msg += fmt.Sprintf("• Риск на сделку: %.1f%%\n", b.config.Strategy.Turtles.RiskPerTrade*100)
	msg += fmt.Sprintf("• ATR период: %d\n", b.config.Strategy.Turtles.AtrPeriod)
	msg += fmt.Sprintf("• ATR множитель: %.1f\n\n", b.config.Strategy.Turtles.AtrMultiplier)

	msg += "🎯 КАК РАБОТАЕТ:\n"
	msg += "1. Ищет прорыв максимума/минимума за N дней\n"
	msg += "2. Рассчитывает стоп-лосс на основе волатильности (ATR)\n"
	msg += "3. Определяет размер позиции\n"
	msg += "4. Выходит при обратном прорыве\n\n"

	msg += "📈 КОМАНДЫ УПРАВЛЕНИЯ:\n"
	msg += "• /turtle_signals - Текущие сигналы\n"
	msg += "• /scan_turtles - Сканировать все инструменты\n"
	msg += "• /turtle_stats - Статистика\n"
	msg += "• /turtle_config - Настройки\n"
	msg += "• /turtle_test - Тестирование\n"

	// Добавляем кнопки управления
	var rows [][]tgbotapi.InlineKeyboardButton

	if b.config.Strategy.Turtles.Enabled {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔴 Выключить", "turtle_disable"),
			tgbotapi.NewInlineKeyboardButtonData("📈 Сигналы", "turtle_signals"),
		))
	} else {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟢 Включить", "turtle_enable"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настроить", "turtle_config"),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🔍 Сканировать", "turtle_scan"),
		tgbotapi.NewInlineKeyboardButtonData("📊 Тестировать", "turtle_test"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	return b.sendMessageWithKeyboard(chatID, msg, keyboard)
}

func (b *Bot) handleTurtleSignals(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	if !b.config.Strategy.Turtles.Enabled {
		return b.sendFormattedMessage(chatID, "❌ Стратегия 'Черепах' отключена.\nИспользуйте /turtle_config для включения.")
	}

	b.sendFormattedMessage(chatID, "🔍 Поиск сигналов по стратегии 'Черепах'...")

	// Запускаем анализ в фоне
	go b.scanAndShowTurtleSignals(chatID)

	return nil
}

func (b *Bot) handleScanTurtles(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	if !b.config.Strategy.Turtles.Enabled {
		return b.sendFormattedMessage(chatID, "❌ Стратегия 'Черепах' отключена.\nИспользуйте /turtle_config для включения.")
	}

	// Отправляем сообщение о начале сканирования
	b.sendFormattedMessage(chatID, "🔍 Сканирование всех инструментов по стратегии 'Черепах'...\n\n⏳ Это может занять несколько минут.")

	// Запускаем сканирование в фоне
	go b.scanAndShowTurtleSignals(chatID)

	return nil
}

func (b *Bot) handleTurtleStats(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	msg := "📊 СТАТИСТИКА СТРАТЕГИИ 'ЧЕРЕПАХ'\n\n"

	if !b.config.Strategy.Turtles.Enabled {
		msg += "❌ Стратегия отключена\n\n"
		msg += "Используйте /turtle_config для включения и настройки"
		return b.sendFormattedMessage(chatID, msg)
	}

	msg += "📈 ОБЩАЯ ИНФОРМАЦИЯ:\n"
	msg += fmt.Sprintf("• Включена: %s\n", b.getTurtleStatus())
	msg += fmt.Sprintf("• Таймфрейм анализа: %s\n", b.config.Strategy.Turtles.Timeframe)
	msg += "• Последнее сканирование: сегодня\n"
	msg += "• Автосканирование: каждый час\n\n"

	msg += "⚙️ ПАРАМЕТРЫ РИСК-МЕНЕДЖМЕНТА:\n"
	msg += fmt.Sprintf("• Риск на сделку: %.1f%%\n", b.config.Strategy.Turtles.RiskPerTrade*100)
	msg += fmt.Sprintf("• Размер позиции: %s\n", b.getPositionSizingStatus())
	msg += fmt.Sprintf("• Стоп-лосс: %.1fxATR\n", b.config.Strategy.Turtles.AtrMultiplier)
	msg += "• Тейк-профит: 2xриск\n\n"

	msg += "📊 ИСТОРИЧЕСКАЯ ЭФФЕКТИВНОСТЬ:\n"
	msg += "• В разработке...\n\n"

	msg += "📈 ПЛАНИРУЕМЫЕ УЛУЧШЕНИЯ:\n"
	msg += "• Сбор статистики по сделкам\n"
	msg += "• Анализ эффективности\n"
	msg += "• Оптимизация параметров\n"
	msg += "• Бэктестинг\n"

	// Добавляем кнопки
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "turtle_stats"),
			tgbotapi.NewInlineKeyboardButtonData("📈 Сигналы", "turtle_signals"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "turtle_config"),
			tgbotapi.NewInlineKeyboardButtonData("🔍 Сканировать", "turtle_scan"),
		),
	)

	return b.sendMessageWithKeyboard(chatID, msg, keyboard)
}

func (b *Bot) handleTurtleConfig(update tgbotapi.Update) error {
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

	msg := "⚙️ НАСТРОЙКИ СТРАТЕГИИ 'ЧЕРЕПАХ'\n\n"

	msg += "📊 ТЕКУЩИЕ НАСТРОЙКИ:\n"
	msg += fmt.Sprintf("• Статус: %s\n", b.getTurtleStatus())
	msg += fmt.Sprintf("• Таймфрейм: %s\n", b.config.Strategy.Turtles.Timeframe)
	msg += fmt.Sprintf("• Период анализа: %d дней\n", b.config.Strategy.Turtles.LookbackPeriod)
	msg += fmt.Sprintf("• Прорыв входа: %d дней\n", b.config.Strategy.Turtles.EntryBreakoutDays)
	msg += fmt.Sprintf("• Прорыв выхода: %d дней\n", b.config.Strategy.Turtles.ExitBreakoutDays)
	msg += fmt.Sprintf("• Риск на сделку: %.1f%%\n", b.config.Strategy.Turtles.RiskPerTrade*100)
	msg += fmt.Sprintf("• ATR период: %d\n", b.config.Strategy.Turtles.AtrPeriod)
	msg += fmt.Sprintf("• ATR множитель: %.1f\n", b.config.Strategy.Turtles.AtrMultiplier)
	msg += fmt.Sprintf("• Расчет позиции: %s\n", b.getPositionSizingStatus())
	msg += fmt.Sprintf("• Уведомления: %s\n\n", b.getNotificationsStatus())

	msg += "⚡ БЫСТРЫЕ ДЕЙСТВИЯ:\n"

	// Добавляем кнопки управления
	var rows [][]tgbotapi.InlineKeyboardButton

	if b.config.Strategy.Turtles.Enabled {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔴 Выключить", "turtle_disable"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Уведомления", "config_notifications"),
		))
	} else {
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🟢 Включить", "turtle_enable"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Уведомления", "config_notifications"),
		))
	}

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📊 Риск 1%", "strategy_set_risk_0.01"),
		tgbotapi.NewInlineKeyboardButtonData("📊 Риск 2%", "strategy_set_risk_0.02"),
		tgbotapi.NewInlineKeyboardButtonData("📊 Риск 5%", "strategy_set_risk_0.05"),
	))

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📅 Период 10д", "strategy_set_period_10"),
		tgbotapi.NewInlineKeyboardButtonData("📅 Период 20д", "strategy_set_period_20"),
		tgbotapi.NewInlineKeyboardButtonData("📅 Период 50д", "strategy_set_period_50"),
	))

	rows = append(rows, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("📈 Тестировать", "turtle_test"),
		tgbotapi.NewInlineKeyboardButtonData("🔙 Назад", "help_strategy"),
	))

	keyboard := tgbotapi.NewInlineKeyboardMarkup(rows...)

	return b.sendMessageWithKeyboard(chatID, msg, keyboard)
}

func (b *Bot) handleTurtleTest(update tgbotapi.Update) error {
	if b == nil {
		fmt.Printf("CRITICAL ERROR: Bot is nil in handleTurtleTest\n")
		debug.PrintStack()
		return fmt.Errorf("bot is nil")
	}

	b.logger.Info("handleTurtleTest called")

	chatID, err := b.getChatID(update)
	if err != nil {
		b.logger.Error("Failed to get chat ID", "error", err)
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		b.logger.Error("Failed to get user ID", "error", err)
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	// Проверяем chatID и userID
	if chatID == 0 {
		b.logger.Error("chatID is 0")
		return fmt.Errorf("invalid chat ID")
	}

	if userID == 0 {
		b.logger.Error("userID is 0")
		return fmt.Errorf("invalid user ID")
	}

	if !b.config.Strategy.Turtles.Enabled {
		return b.sendFormattedMessage(chatID, "❌ Стратегия 'Черепах' отключена.\nИспользуйте /turtle_config для включения.")
	}

	// Начинаем процесс тестирования
	state := &UserState{
		CurrentCommand: "turtle_test",
		Step:           1,
		Data:           make(map[string]interface{}),
		LastActivity:   time.Now(),
	}
	b.setUserState(userID, state)

	msg := "🧪 ТЕСТИРОВАНИЕ СТРАТЕГИИ 'ЧЕРЕПАХ'\n\n"
	msg += "Этот тест покажет, как работает стратегия на конкретном инструменте.\n\n"
	msg += "Введите тикер инструмента для тестирования (например: SBER):"

	return b.sendFormattedMessage(chatID, msg)
}

func (b *Bot) handleTurtleEnable(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.isAdmin(userID) {
		return b.sendFormattedMessage(chatID, "❌ Включение стратегии доступно только администраторам")
	}

	b.config.Strategy.Turtles.Enabled = true
	b.sendFormattedMessage(chatID, "✅ Стратегия 'Черепах' включена!\n\nИспользуйте /turtle_signals для поиска сигналов или /scan_turtles для сканирования всех инструментов.")

	// Обновляем команды меню
	b.setBotCommands()

	return nil
}

func (b *Bot) handleTurtleDisable(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.isAdmin(userID) {
		return b.sendFormattedMessage(chatID, "❌ Отключение стратегии доступно только администраторам")
	}

	b.config.Strategy.Turtles.Enabled = false
	b.sendFormattedMessage(chatID, "✅ Стратегия 'Черепах' отключена.\n\nДля включения используйте /turtle_config")

	// Обновляем команды меню
	b.setBotCommands()

	return nil
}

func (b *Bot) scanAndShowTurtleSignals(chatID int64) {
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

	for _, instrument := range instruments {
		totalInstruments++
		signals, err := strategy.AnalyzeInstrument(context.Background(), instrument)
		if err != nil {
			b.logger.Debug("Ошибка анализа инструмента", "instrument", instrument, "error", err)
			continue
		}

		if len(signals) > 0 {
			allSignals = append(allSignals, signals...)
		}
	}

	// Формируем сообщение с результатами
	//nolint:gocritic
	msg := "📈 РЕЗУЛЬТАТЫ СКАНИРОВАНИЯ\n\n"
	msg += fmt.Sprintf("📊 Проанализировано инструментов: %d\n", totalInstruments)
	msg += fmt.Sprintf("🚨 Найдено сигналов: %d\n\n", len(allSignals))

	if len(allSignals) == 0 {
		msg += "📭 Торговых сигналов не найдено\n"
		msg += "Попробуйте позже или измените параметры стратегии"
		b.sendFormattedMessage(chatID, msg)
		return
	}

	// Группируем сигналы
	entryLong := []analysis.Signal{}
	entryShort := []analysis.Signal{}
	exitLong := []analysis.Signal{}
	exitShort := []analysis.Signal{}

	for _, signal := range allSignals {
		switch signal.SignalType {
		case "entry_long":
			entryLong = append(entryLong, signal)
		case "entry_short":
			entryShort = append(entryShort, signal)
		case "exit_long":
			exitLong = append(exitLong, signal)
		case "exit_short":
			exitShort = append(exitShort, signal)
		}
	}

	// Добавляем сигналы в сообщение
	if len(entryLong) > 0 {
		msg += "🟢 СИГНАЛЫ НА ПОКУПКУ:\n"
		for _, signal := range entryLong {
			msg += fmt.Sprintf("• %s - %.2f₽\n", signal.Instrument, signal.Price)
			msg += fmt.Sprintf("  Стоп: %.2f | Тейк: %.2f\n", signal.StopLoss, signal.TakeProfit)
			msg += fmt.Sprintf("  Размер: %.0f шт. | Риск: %.1f%%\n", signal.PositionSize, b.config.Strategy.Turtles.RiskPerTrade*100)
			msg += fmt.Sprintf("  📅 %s\n\n", signal.Timestamp.Format("02.01 15:04"))
		}
	}

	if len(entryShort) > 0 {
		msg += "🔴 СИГНАЛЫ НА ПРОДАЖУ:\n"
		for _, signal := range entryShort {
			msg += fmt.Sprintf("• %s - %.2f₽\n", signal.Instrument, signal.Price)
			msg += fmt.Sprintf("  Стоп: %.2f | Тейк: %.2f\n", signal.StopLoss, signal.TakeProfit)
			msg += fmt.Sprintf("  Размер: %.0f шт. | Риск: %.1f%%\n", signal.PositionSize, b.config.Strategy.Turtles.RiskPerTrade*100)
			msg += fmt.Sprintf("  📅 %s\n\n", signal.Timestamp.Format("02.01 15:04"))
		}
	}

	if len(exitLong) > 0 {
		msg += "📤 СИГНАЛЫ НА ВЫХОД ИЗ ПОКУПОК:\n"
		for _, signal := range exitLong {
			msg += fmt.Sprintf("• %s - %.2f₽\n", signal.Instrument, signal.Price)
			msg += fmt.Sprintf("  Причина: %s\n", signal.Reason)
		}
		msg += "\n"
	}

	if len(exitShort) > 0 {
		msg += "📤 СИГНАЛЫ НА ВЫХОД ИЗ ПРОДАЖ:\n"
		for _, signal := range exitShort {
			msg += fmt.Sprintf("• %s - %.2f₽\n", signal.Instrument, signal.Price)
			msg += fmt.Sprintf("  Причина: %s\n", signal.Reason)
		}
	}

	msg += "\n💡 РЕКОМЕНДАЦИИ:\n"
	msg += "• Убедитесь в достаточном объеме\n"
	msg += "• Проверьте новости по инструменту\n"
	msg += "• Используйте стоп-лоссы\n"

	// Добавляем кнопки действий
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "turtle_signals"),
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "turtle_config"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Подробнее о стратегии", "help_strategy"),
			tgbotapi.NewInlineKeyboardButtonData("📈 Тестировать", "turtle_test"),
		),
	)

	b.sendSafeMessageWithKeyboard(chatID, msg, keyboard)
}

func (b *Bot) runTurtleTest(chatID int64, instrument string) {
	b.sendFormattedMessage(chatID, fmt.Sprintf("🧪 Тестирование стратегии для %s...", instrument))

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

	// Анализируем инструмент
	signals, err := strategy.AnalyzeInstrument(context.Background(), instrument)
	if err != nil {
		b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка тестирования: %v", err))
		return
	}

	// Формируем отчет
	msg := fmt.Sprintf("📊 ОТЧЕТ ПО ТЕСТИРОВАНИЮ: %s\n\n", instrument)

	if len(signals) == 0 {
		msg += "📭 Сигналов не найдено\n\n"
		msg += "ПАРАМЕТРЫ ТЕСТА:\n"
		msg += fmt.Sprintf("• Таймфрейм: %s\n", b.config.Strategy.Turtles.Timeframe)
		msg += fmt.Sprintf("• Период анализа: %d дней\n", b.config.Strategy.Turtles.LookbackPeriod)
		msg += fmt.Sprintf("• Прорыв входа: %d дней\n", b.config.Strategy.Turtles.EntryBreakoutDays)
		msg += fmt.Sprintf("• Прорыв выхода: %d дней\n", b.config.Strategy.Turtles.ExitBreakoutDays)
		msg += "• Последняя цена: получение...\n\n"
		msg += "💡 РЕКОМЕНДАЦИИ:\n"
		msg += "• Проверьте наличие данных по инструменту\n"
		msg += "• Попробуйте другой инструмент\n"
		msg += "• Измените параметры стратегии\n"
	} else {
		msg += fmt.Sprintf("🚨 НАЙДЕНО СИГНАЛОВ: %d\n\n", len(signals))

		for i, signal := range signals {
			msg += fmt.Sprintf("📈 СИГНАЛ #%d:\n", i+1)
			msg += fmt.Sprintf("• Тип: %s\n", b.getSignalTypeText(signal.SignalType))
			msg += fmt.Sprintf("• Цена: %.2f₽\n", signal.Price)

			if signal.StopLoss > 0 {
				msg += fmt.Sprintf("• Стоп-лосс: %.2f₽\n", signal.StopLoss)
				msg += fmt.Sprintf("• Риск: %.2f₽ (%.1f%%)\n",
					math.Abs(signal.Price-signal.StopLoss),
					(math.Abs(signal.Price-signal.StopLoss)/signal.Price)*100)
			}

			if signal.TakeProfit > 0 {
				msg += fmt.Sprintf("• Тейк-профит: %.2f₽\n", signal.TakeProfit)
				msg += fmt.Sprintf("• Прибыль: %.2f₽ (%.1f%%)\n",
					math.Abs(signal.TakeProfit-signal.Price),
					(math.Abs(signal.TakeProfit-signal.Price)/signal.Price)*100)
			}

			if signal.PositionSize > 0 {
				msg += fmt.Sprintf("• Размер позиции: %.0f шт.\n", signal.PositionSize)
			}

			msg += fmt.Sprintf("• Причина: %s\n", signal.Reason)
			msg += fmt.Sprintf("• Время: %s\n\n", signal.Timestamp.Format("02.01.2006 15:04"))
		}
	}

	// Добавляем кнопки действий
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Новый тест", "turtle_test"),
			tgbotapi.NewInlineKeyboardButtonData("📈 Все сигналы", "turtle_signals"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "turtle_config"),
			tgbotapi.NewInlineKeyboardButtonData("🔍 Сканировать", "turtle_scan"),
		),
	)

	b.sendSafeMessageWithKeyboard(chatID, msg, keyboard)
}

func (b *Bot) getTurtleStatus() string {
	if b.config.Strategy.Turtles.Enabled {
		return "🟢 ВКЛЮЧЕНА"
	}
	return "🔴 ВЫКЛЮЧЕНА"
}

func (b *Bot) getPositionSizingStatus() string {
	if b.config.Strategy.Turtles.PositionSizing {
		return "🟢 ВКЛЮЧЕН"
	}
	return "🔴 ВЫКЛЮЧЕН"
}

func (b *Bot) getNotificationsStatus() string {
	if b.config.Strategy.Notifications.Enabled {
		return "🟢 ВКЛЮЧЕНЫ"
	}
	return "🔴 ВЫКЛЮЧЕНЫ"
}

func (b *Bot) enableTurtleStrategy(chatID int64) {
	b.config.Strategy.Turtles.Enabled = true
	b.sendFormattedMessage(chatID, "✅ Стратегия 'Черепах' включена!")
	b.setBotCommands()
}

func (b *Bot) disableTurtleStrategy(chatID int64) {
	b.config.Strategy.Turtles.Enabled = false
	b.sendFormattedMessage(chatID, "✅ Стратегия 'Черепах' отключена.")
	b.setBotCommands()
}

func (b *Bot) setTurtleRisk(chatID int64, risk float64) {
	b.config.Strategy.Turtles.RiskPerTrade = risk
	b.sendFormattedMessage(chatID, fmt.Sprintf("✅ Риск на сделку установлен: %.1f%%", risk*100))
}

func (b *Bot) setTurtlePeriod(chatID int64, period int) {
	b.config.Strategy.Turtles.LookbackPeriod = period
	b.sendFormattedMessage(chatID, fmt.Sprintf("✅ Период анализа установлен: %d дней", period))
}
