package bot

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Основные обработчики команд
func (b *Bot) handleStart(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	user, err := b.getUser(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "👋 Привет! Я бот для работы с данными Московской биржи.")
	}

	greeting := fmt.Sprintf("👋 Привет, %s!\n\n", user.FirstName)
	greeting += b.config.Bot.Greeting + "\n\n"
	greeting += "🤖 Я бот для работы с данными Московской биржи и анализа по стратегии 'Черепах'.\n\n"
	greeting += "🎯 Основные команды:\n"
	greeting += "• /help - Полная справка\n"
	greeting += "• /status - Статус системы\n"
	greeting += "• /instruments - Список инструментов\n"
	greeting += "• /candles - Получить свечи\n\n"

	if b.config.Strategy.Turtles.Enabled {
		greeting += "📈 Стратегия 'Черепах' ВКЛЮЧЕНА:\n"
		greeting += "• /turtle - Анализ стратегии\n"
		greeting += "• /turtle_signals - Текущие сигналы\n"
		greeting += "• /scan_turtles - Сканировать все\n\n"
	} else {
		greeting += "📈 Стратегия 'Черепах' отключена\nИспользуйте /turtle_config для включения\n\n"
	}

	if b.isAdmin(user.ID) {
		greeting += "👑 Админ команды:\n"
		greeting += "• /admin - Админ панель\n"
		greeting += "• /fetch - Загрузить данные\n"
		greeting += "• /config - Конфигурация\n"
	}
	greeting += "\n💡 Отправьте тикер инструмента (например: SBER) для получения информации."

	// Добавляем кнопки быстрого доступа
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Инструменты", "quick_instruments"),
			tgbotapi.NewInlineKeyboardButtonData("📈 Сигналы", "quick_signals"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "quick_config"),
			tgbotapi.NewInlineKeyboardButtonData("❓ Помощь", "quick_help"),
		),
	)

	return b.sendMessageWithKeyboard(chatID, greeting, keyboard)
}

func (b *Bot) handleHelp(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "📚 ПОЛНАЯ СПРАВКА ПО КОМАНДАМ (для пользователей)")
	}

	// Получаем пользователя для имени
	user, err := b.getUser(update)
	userName := "Пользователь"
	if err == nil && user.FirstName != "" {
		userName = user.FirstName
	}

	msg := fmt.Sprintf("📚 ПОЛНАЯ СПРАВКА ПО КОМАНДАМ\n\nПривет, %s!\n\n", userName)

	// Остальной код остается тот же, но используем b.isAdmin(userID)
	if b.isAdmin(userID) {
		msg += "👑 АДМИН КОМАНДЫ:\n"
		msg += "• /admin - Админ панель\n"
		msg += "• /config - Показать конфигурацию\n"
		msg += "• /log - Просмотр логов\n"
		msg += "• /restart - Перезапустить бота\n"
		msg += "• /users - Управление пользователями\n"
		msg += "• /broadcast - Рассылка сообщений\n"
		msg += "• /debug - Режим отладки\n"
		msg += "• /system - Системная информация\n"
	}

	msg += "\n💡 КАК ИСПОЛЬЗОВАТЬ:\n"
	msg += "1. Получить список инструментов: /instruments\n"
	msg += "2. Получить свечи: /candles → выберите инструмент → таймфрейм → период\n"
	msg += "3. Проверить сигналы: /turtle_signals\n"
	msg += "4. Загрузить данные: /fetch\n\n"

	msg += "📱 ТЕКСТОВЫЕ КОМАНДЫ:\n"
	msg += "• 'меню' или 'команды' - показать меню\n"
	msg += "• 'статус' - показать статус\n"
	msg += "• 'сигналы' - показать сигналы (если стратегия включена)\n"
	msg += "• 'сканировать' - запустить сканирование\n\n"

	msg += "💡 Просто отправьте тикер инструмента (например: SBER) для получения информации о нем."

	return b.sendFormattedMessage(chatID, msg)
}

func (b *Bot) handleStatus(update tgbotapi.Update) error {

	chatID := update.Message.Chat.ID

	// Создаем сообщение со статусом
	msg := "📊 СТАТУС СИСТЕМЫ\n\n"

	// Статус бота
	msg += "🤖 БОТ:\n"
	msg += fmt.Sprintf("• Имя: %s\n", b.botAPI.Self.UserName)
	msg += fmt.Sprintf("• Запущен: %s\n", b.stats.StartTime.Format("2006-01-02 15:04"))
	msg += fmt.Sprintf("• Uptime: %s\n", b.stats.GetUptime().Truncate(time.Second))
	msg += fmt.Sprintf("• Команд выполнено: %d\n", b.stats.CommandsExecuted)
	msg += fmt.Sprintf("• Активных пользователей: %d\n\n", b.stats.GetActiveUsersCount())

	// Статус стратегии
	msg += "📈 СТРАТЕГИЯ 'ЧЕРЕПАХ':\n"
	if b.config.Strategy.Turtles.Enabled {
		msg += "• Статус: 🟢 ВКЛЮЧЕНА\n"
		msg += fmt.Sprintf("• Автоанализ: каждый %s\n", "час")
		msg += fmt.Sprintf("• Таймфрейм: %s\n", b.config.Strategy.Turtles.Timeframe)
		msg += fmt.Sprintf("• Период анализа: %d дней\n", b.config.Strategy.Turtles.LookbackPeriod)
		msg += fmt.Sprintf("• Риск на сделку: %.1f%%\n", b.config.Strategy.Turtles.RiskPerTrade*100)
		if b.config.Strategy.Notifications.Enabled {
			msg += "• Уведомления: 🟢 ВКЛЮЧЕНЫ\n"
		} else {
			msg += "• Уведомления: 🔴 ВЫКЛЮЧЕНЫ\n"
		}
	} else {
		msg += "• Статус: 🔴 ВЫКЛЮЧЕНА\n"
		msg += "• Используйте /turtle_config для включения\n"
	}
	msg += "\n"

	// Проверяем доступность API
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	health, err := b.apiClient.HealthCheck(ctx)
	if err != nil {
		msg += "🔌 API СЕРВЕР: 🔴 НЕДОСТУПЕН\n"
		msg += fmt.Sprintf("• Ошибка: %v\n", err)
	} else {
		msg += "🔌 API СЕРВЕР: 🟢 ДОСТУПЕН\n"
		if status, ok := health["status"].(string); ok {
			msg += fmt.Sprintf("• Статус: %s\n", status)
		}
		if uptime, ok := health["uptime"].(string); ok {
			msg += fmt.Sprintf("• Uptime: %s\n", uptime)
		}

		// Получаем статистику если API доступен
		stats, err := b.apiClient.GetStats(ctx)
		if err == nil {
			if dbStats, ok := stats["database"].(map[string]interface{}); ok {
				msg += fmt.Sprintf("• Таблиц: %v\n", dbStats["tables_count"])
				msg += fmt.Sprintf("• Свечей: %v\n", dbStats["total_candles"])
			}
		}
	}

	// Добавляем кнопки быстрых действий
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "status_refresh"),
			tgbotapi.NewInlineKeyboardButtonData("📈 Сигналы", "quick_signals"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Настройки", "quick_config"),
			tgbotapi.NewInlineKeyboardButtonData("🔍 Проверить API", "status_check_api"),
		),
	)

	return b.sendMessageWithKeyboard(chatID, msg, keyboard)
}

func (b *Bot) handlePing(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	start := time.Now()
	msg := tgbotapi.NewMessage(chatID, "🏓 Pong!")
	_, sendErr := b.botAPI.Send(msg) // Изменили имя переменной
	responseTime := time.Since(start)

	if sendErr != nil {
		return b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка: %v", sendErr))
	}

	return b.sendFormattedMessage(chatID, fmt.Sprintf("✅ Pong! Время ответа: %v", responseTime))
}

func (b *Bot) handleInstruments(update tgbotapi.Update) error {
	//chatID := update.Message.Chat.ID

	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	instruments, err := b.apiClient.GetInstruments(context.Background())
	if err != nil {
		return b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка получения инструментов: %v", err))
	}

	if len(instruments) == 0 {
		return b.sendFormattedMessage(chatID, "📭 Список инструментов пуст")
	}

	// Разбиваем на части для отправки
	const maxInstrumentsPerMessage = 30
	messages := []string{}
	currentMsg := "📋 Список инструментов:\n\n"

	for i, instrument := range instruments {
		currentMsg += fmt.Sprintf("%d. %s\n", i+1, instrument)

		if (i+1)%maxInstrumentsPerMessage == 0 {
			messages = append(messages, currentMsg)
			currentMsg = ""
		}
	}

	if currentMsg != "" {
		messages = append(messages, currentMsg)
	}

	// Отправляем все части
	for _, msg := range messages {
		if err := b.sendFormattedMessage(chatID, msg); err != nil {
			return err
		}
	}

	return nil
}

func (b *Bot) handleCandles(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	// Начинаем диалог
	state := &UserState{
		CurrentCommand: "candles",
		Step:           1,
		Data:           make(map[string]interface{}),
		LastActivity:   time.Now(),
	}
	b.setUserState(userID, state)

	// Запрашиваем инструмент
	return b.sendFormattedMessage(chatID, "📈 Получение свечей\n\nПожалуйста, введите тикер инструмента (например: SBER, GAZP):")
}

func (b *Bot) handleStats(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	stats, err := b.apiClient.GetStats(context.Background())
	if err != nil {
		return b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка получения статистики: %v", err))
	}

	msg := "📊 Статистика данных:\n\n"

	// Форматируем статистику
	if dbStats, ok := stats["database"].(map[string]interface{}); ok {
		msg += "🗄 База данных:\n"
		msg += fmt.Sprintf("• Таблиц: %v\n", dbStats["tables_count"])
		msg += fmt.Sprintf("• Всего свечей: %v\n", dbStats["total_candles"])
		msg += fmt.Sprintf("• Последнее обновление: %v\n", dbStats["last_update"])
	}

	if fetcherStats, ok := stats["fetcher"].(map[string]interface{}); ok {
		msg += "\n📈 Фетчер:\n"
		msg += fmt.Sprintf("• Инструментов: %v\n", fetcherStats["instruments_count"])
		msg += fmt.Sprintf("• Таймфреймы: %v\n", fetcherStats["timeframes"])
	}

	return b.sendFormattedMessage(chatID, msg)
}

func (b *Bot) handleFetch(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	// Отправляем подтверждение
	b.sendFormattedMessage(chatID, "🔄 Запуск загрузки данных...")

	// Запускаем загрузку в фоне
	go func() {
		result, err := b.apiClient.TriggerFetch(context.Background())
		if err != nil {
			b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка загрузки: %v", err))
			return
		}

		msg := "✅ Загрузка запущена\n\n"
		if status, ok := result["status"].(string); ok {
			msg += fmt.Sprintf("Статус: %s\n", status)
		}
		if message, ok := result["message"].(string); ok {
			msg += fmt.Sprintf("Сообщение: %s\n", message)
		}

		b.sendFormattedMessage(chatID, msg)
	}()

	return nil
}

func (b *Bot) handleHealth(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	health, err := b.apiClient.HealthCheck(context.Background())
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ API сервер недоступен")
	}

	msg := "🏥 Проверка здоровья:\n\n"
	msg += fmt.Sprintf("✅ Статус: %s\n", health["status"])
	msg += fmt.Sprintf("⏱ Uptime: %s\n", health["uptime"])
	msg += fmt.Sprintf("🔋 Версия: %s\n", health["version"])
	msg += fmt.Sprintf("🕒 Время сервера: %s\n", health["timestamp"])

	if stats, ok := health["stats"].(map[string]interface{}); ok {
		msg += "\n📊 Статистика:\n"
		if tables, ok := stats["tables_count"].(float64); ok {
			msg += fmt.Sprintf("• Таблиц: %.0f\n", tables)
		}
		if candles, ok := stats["total_candles"].(float64); ok {
			msg += fmt.Sprintf("• Свечей: %.0f\n", candles)
		}
	}

	return b.sendFormattedMessage(chatID, msg)
}

func (b *Bot) handleTimeframes(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	timeframes, err := b.apiClient.GetTimeframes(context.Background())
	if err != nil {
		return b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка получения таймфреймов: %v", err))
	}

	msg := "⏱ Доступные таймфреймы:\n\n"
	for _, tf := range timeframes {
		msg += fmt.Sprintf("• %s (%s)\n", tf["code"], tf["display_name"])
		msg += fmt.Sprintf("  %s\n\n", tf["description"])
	}

	return b.sendFormattedMessage(chatID, msg)
}

func (b *Bot) handleTables(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	tables, err := b.apiClient.GetTables(context.Background())
	if err != nil {
		return b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка получения таблиц: %v", err))
	}

	msg := "🗄 Список таблиц:\n\n"
	for i, table := range tables {
		msg += fmt.Sprintf("%d. %s\n", i+1, table["table_name"])
		msg += fmt.Sprintf("   Инструмент: %s | Таймфрейм: %s\n", table["instrument"], table["timeframe"])
		msg += fmt.Sprintf("   Записей: %v | Последнее обновление: %v\n\n", table["row_count"], table["last_update"])
	}

	return b.sendFormattedMessage(chatID, msg)
}

func (b *Bot) handleRefresh(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.isAdmin(userID) {
		return b.sendFormattedMessage(chatID, "❌ Эта команда доступна только администраторам")
	}

	b.sendFormattedMessage(chatID, "🔄 Обновление списка инструментов...")

	go func() {
		result, err := b.apiClient.RefreshInstruments(context.Background())
		if err != nil {
			b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка обновления: %v", err))
			return
		}

		msg := "✅ Инструменты обновлены\n\n"
		if status, ok := result["status"].(string); ok {
			msg += fmt.Sprintf("Статус: %s\n", status)
		}
		if count, ok := result["instruments_count"].(float64); ok {
			msg += fmt.Sprintf("Инструментов: %.0f\n", count)
		}

		b.sendFormattedMessage(chatID, msg)
	}()

	return nil
}

func (b *Bot) handleCleanup(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.isAdmin(userID) {
		return b.sendFormattedMessage(chatID, "❌ Эта команда доступна только администраторам")
	}

	// Запрашиваем подтверждение
	msg := tgbotapi.NewMessage(chatID, "🧹 Очистка старых таблиц\n\nВведите количество дней неактивности (по умолчанию 90):")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("90 дней", "cleanup_90"),
			tgbotapi.NewInlineKeyboardButtonData("180 дней", "cleanup_180"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("Отмена", "cancel"),
		),
	)

	_, err = b.botAPI.Send(msg)
	return err
}

func (b *Bot) handleCancel(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	// Удаляем состояние пользователя
	b.resetUserState(userID)

	return b.sendFormattedMessage(chatID, "❌ Операция отменена")
}
