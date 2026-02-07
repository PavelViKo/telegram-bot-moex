package bot

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// registerCommands регистрирует все команды бота
func (b *Bot) registerCommands() {
	// Основные команды
	b.commands["start"] = b.handleStart
	b.commands["help"] = b.handleHelp
	b.commands["status"] = b.handleStatus
	b.commands["ping"] = b.handlePing

	// Команды данных
	b.commands["instruments"] = b.handleInstruments
	b.commands["candles"] = b.handleCandles
	b.commands["stats"] = b.handleStats
	b.commands["tables"] = b.handleTables
	b.commands["timeframes"] = b.handleTimeframes
	b.commands["health"] = b.handleHealth

	// Команды управления данными
	b.commands["fetch"] = b.handleFetch
	b.commands["refresh"] = b.handleRefresh
	b.commands["add_instrument"] = b.handleAddInstrument
	b.commands["remove_instrument"] = b.handleRemoveInstrument
	b.commands["cleanup"] = b.handleCleanup

	// Команды стратегии "Черепах"
	b.commands["turtle"] = b.handleTurtleAnalysis
	b.commands["turtle_analysis"] = b.handleTurtleAnalysis
	b.commands["turtle_signals"] = b.handleTurtleSignals
	b.commands["turtle_scan"] = b.handleScanTurtles
	b.commands["scan_turtles"] = b.handleScanTurtles
	b.commands["turtle_stats"] = b.handleTurtleStats
	b.commands["turtle_config"] = b.handleTurtleConfig
	b.commands["turtle_enable"] = b.handleTurtleEnable
	b.commands["turtle_disable"] = b.handleTurtleDisable
	b.commands["turtle_test"] = b.handleTurtleTest

	// Команды стратегии "MA"
	b.commands["ma"] = b.handleMA
	b.commands["ma_signals"] = b.handleMASignals
	b.commands["scan_ma"] = b.handleScanMA
	b.commands["ma_config"] = b.handleMAConfig
	b.commands["ma_test"] = b.handleMATest

	// Команды управления ботом
	b.commands["cancel"] = b.handleCancel
	b.commands["config"] = b.handleConfig
	b.commands["restart"] = b.handleRestart
	b.commands["stop"] = b.handleStop
	b.commands["log"] = b.handleLogs

	// Админ команды
	b.commands["admin"] = b.handleAdmin
	b.commands["users"] = b.handleUsers
	b.commands["broadcast"] = b.handleBroadcast
	b.commands["debug"] = b.handleDebug
	b.commands["system"] = b.handleSystem
}

// setBotCommands устанавливает команды в меню Telegram
func (b *Bot) setBotCommands() {
	// Основные команды для всех пользователей
	commands := []tgbotapi.BotCommand{
		{Command: "start", Description: "Начало работы с ботом"},
		{Command: "help", Description: "Справка по всем командам"},
		{Command: "status", Description: "Статус системы и бота"},

		// Команды данных
		{Command: "instruments", Description: "Список инструментов"},
		{Command: "candles", Description: "Получить свечи инструмента"},
		{Command: "stats", Description: "Статистика данных"},
		{Command: "tables", Description: "Список таблиц"},
		{Command: "timeframes", Description: "Доступные таймфреймы"},
		{Command: "health", Description: "Проверка здоровья API"},

		// Команды управления данными
		{Command: "fetch", Description: "Запустить загрузку данных"},
		{Command: "cancel", Description: "Отменить текущую операцию"},
	}

	// Команды стратегии "Черепах" если стратегия включена
	if b.config.Strategy.Turtles.Enabled {
		strategiesCommands := []tgbotapi.BotCommand{
			{Command: "turtle", Description: "Анализ по стратегии 'Черепах'"},
			{Command: "turtle_signals", Description: "Текущие сигналы стратегии"},
			{Command: "scan_turtles", Description: "Сканировать все инструменты"},
			{Command: "turtle_stats", Description: "Статистика стратегии"},
			{Command: "turtle_config", Description: "Настройки стратегии"},
		}
		commands = append(commands, strategiesCommands...)
	}

	if b.config.Strategy.MACrossover.Enabled {
		maCommands := []tgbotapi.BotCommand{
			{Command: "ma", Description: "Анализ по стратегии MA Crossover"},
			{Command: "ma_signals", Description: "Сигналы MA Crossover"},
			{Command: "scan_ma", Description: "Сканировать по MA Crossover"},
			{Command: "ma_config", Description: "Настройки MA Crossover"},
			{Command: "ma_test", Description: "Тестирование MA Crossover"},
		}
		commands = append(maCommands, maCommands...)
		// Комментарий: Нужно добавить эти команды в глобальный список
		// В родительском методе setBotCommands() должно быть что-то вроде:
		// allCommands = append(allCommands, maCommands...)
		//_ = maCommands // Используем переменную, чтобы избежать предупреждения

		// Если в структуре Bot есть поле для хранения команд, добавьте их:
		// b.allCommands = append(b.allCommands, maCommands...)
	}

	// Админ команды - проверяем есть ли админы в чате
	adminCommands := []tgbotapi.BotCommand{
		{Command: "admin", Description: "Админ панель"},
		{Command: "refresh", Description: "Обновить инструменты"},
		{Command: "cleanup", Description: "Очистка таблиц"},
		{Command: "add_instrument", Description: "Добавить инструмент"},
		{Command: "remove_instrument", Description: "Удалить инструмент"},
		{Command: "log", Description: "Просмотр логов"},
		{Command: "config", Description: "Показать конфигурацию"},
		{Command: "restart", Description: "Перезапустить бота"},
		{Command: "users", Description: "Управление пользователями"},
	}
	commands = append(commands, adminCommands...)

	config := tgbotapi.NewSetMyCommands(commands...)
	if _, err := b.botAPI.Request(config); err != nil {
		b.logger.Warn("Не удалось установить команды меню", "error", err)
	}
}

// handleUpdate обрабатывает обновление
func (b *Bot) handleUpdate(update tgbotapi.Update) {
	b.stats.UpdateStats("message_received")

	// Проверяем разрешен ли пользователь
	if !b.isUserAllowed(update) {
		b.logger.Warn("Попытка доступа от запрещенного пользователя",
			"user_id", getUserID(update))
		return
	}

	// Обновляем статистику активных пользователей
	userID := getUserID(update)
	if userID > 0 {
		b.stats.AddActiveUser(userID)
	}

	// Обработка разных типов обновлений
	switch {
	case update.Message != nil && update.Message.IsCommand():
		b.handleCommand(update)
	case update.Message != nil:
		b.handleMessage(update)
	case update.CallbackQuery != nil:
		b.handleCallbackQuery(update)
	case update.InlineQuery != nil:
		b.handleInlineQuery(update)
	case update.ChosenInlineResult != nil:
		b.handleChosenInlineResult(update)
	}
}

// handleCommand обрабатывает команду
func (b *Bot) handleCommand(update tgbotapi.Update) {
	command := update.Message.Command()
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID

	b.logger.Info("Получена команда",
		"command", command,
		"user_id", userID,
		"chat_id", chatID)

	// Получаем или создаем состояние пользователя
	state := b.getUserState(userID)
	if state != nil && state.CurrentCommand != "" && command != "cancel" {
		// Пользователь в процессе выполнения команды
		b.handleCommandStep(update, state)
		return
	}

	// Ищем обработчик команды
	handler, exists := b.commands[command]
	if !exists {
		// Показываем справку для неизвестной команды
		b.sendUnknownCommand(chatID, command, update.Message.From.ID)
		return
	}

	// Сбрасываем состояние пользователя
	b.resetUserState(userID)

	// Запускаем обработчик
	if err := handler(update); err != nil {
		b.stats.UpdateStats("error")
		b.logger.Error("Ошибка обработки команды",
			"command", command,
			"error", err)
		b.sendMessage(chatID, fmt.Sprintf("❌ Ошибка: %v", err))
	} else {
		b.stats.UpdateStats("command_executed")
	}
}

// sendUnknownCommand отправляет сообщение о неизвестной команде
func (b *Bot) sendUnknownCommand(chatID int64, command string, userID int64) {
	msg := fmt.Sprintf("❌ Неизвестная команда: /%s\n\n", command)
	msg += "📚 Доступные команды:\n\n"

	// Основные команды
	msg += "🎯 Основные:\n"
	msg += "• /start - Начало работы\n"
	msg += "• /help - Справка по командам\n"
	msg += "• /status - Статус системы\n"
	msg += "• /ping - Проверка связи\n\n"

	// Команды данных
	msg += "📊 Данные:\n"
	msg += "• /instruments - Список инструментов\n"
	msg += "• /candles - Получить свечи\n"
	msg += "• /stats - Статистика данных\n"
	msg += "• /tables - Список таблиц\n"
	msg += "• /timeframes - Таймфреймы\n"
	msg += "• /health - Проверка здоровья\n\n"

	// Команды стратегии если включена
	if b.config.Strategy.Turtles.Enabled {
		msg += "📈 Стратегия 'Черепах':\n"
		msg += "• /turtle - Анализ стратегии\n"
		msg += "• /turtle_signals - Текущие сигналы\n"
		msg += "• /scan_turtles - Сканировать\n"
		msg += "• /turtle_stats - Статистика\n"
		msg += "• /turtle_config - Настройки\n\n"
	}

	// Админ команды (показываем если пользователь админ)
	if b.isAdmin(userID) {
		msg += "⚙️ Администрирование:\n"
		msg += "• /admin - Админ панель\n"
		msg += "• /fetch - Загрузить данные\n"
		msg += "• /refresh - Обновить\n"
		msg += "• /cleanup - Очистить\n"
		msg += "• /config - Конфигурация\n"
		msg += "• /log - Логи\n"
		msg += "• /users - Пользователи\n"
		msg += "• /broadcast - Рассылка\n"
	}

	msg += "\n💡 Используйте /help для подробной справки"

	b.sendFormattedMessage(chatID, msg)
}

// handleMessage обрабатывает текстовое сообщение
func (b *Bot) handleMessage(update tgbotapi.Update) {

	state := b.getUserState(update.Message.From.ID)
	if state != nil && state.CurrentCommand != "" {
		// Пользователь в процессе выполнения команды
		b.handleCommandStep(update, state)
		return
	}

	// Обработка обычного сообщения
	text := update.Message.Text
	chatID := update.Message.Chat.ID

	b.logger.Debug("Получено сообщение",
		"text", text,
		"chat_id", chatID)

	// Проверяем, не является ли это тикером инструмента
	if b.isValidInstrument(text) {
		b.sendInstrumentInfo(chatID, text)
		return
	}

	// Обработка специальных текстовых команд
	switch strings.ToLower(text) {
	case "меню", "menu", "команды":
		b.sendHelpMenu(chatID)
	case "статус", "status":
		b.handleStatus(update)
	case "сигналы", "signals":
		if b.config.Strategy.Turtles.Enabled {
			b.handleTurtleSignals(update)
		} else {
			b.sendMessage(chatID, "❌ Стратегия 'Черепах' отключена")
		}
	case "сканировать", "scan":
		if b.config.Strategy.Turtles.Enabled {
			b.handleScanTurtles(update)
		} else {
			b.sendMessage(chatID, "❌ Стратегия 'Черепах' отключена")
		}
	default:
		// Общий ответ
		b.sendMessage(chatID, fmt.Sprintf("📝 Получено сообщение: %s\n\nИспользуйте /help для списка команд или отправьте тикер инструмента для получения информации.", text))
	}
}

// handleCallbackQuery обрабатывает callback запросы
func (b *Bot) handleCallbackQuery(update tgbotapi.Update) {
	callback := update.CallbackQuery
	chatID := callback.Message.Chat.ID
	data := callback.Data

	b.logger.Debug("Получен callback запрос",
		"data", data,
		"chat_id", chatID)

	// Обработка разных callback данных
	switch {
	case strings.HasPrefix(data, "timeframe_"):
		tf := strings.TrimPrefix(data, "timeframe_")
		b.handleTimeframeSelection(chatID, callback.From.ID, tf)
	case strings.HasPrefix(data, "cleanup_"):
		daysStr := strings.TrimPrefix(data, "cleanup_")
		days, _ := strconv.Atoi(daysStr)
		b.handleCleanupAction(chatID, days)
	case data == "cancel":
		b.handleCancelCallback(chatID, callback.From.ID)
	case strings.HasPrefix(data, "instrument_"):
		b.handleInstrumentCallback(chatID, data)
	case strings.HasPrefix(data, "admin_"):
		b.handleAdminCallback(chatID, data)
	case strings.HasPrefix(data, "turtle_"):
		b.handleTurtleCallback(chatID, callback.From.ID, data)
	case strings.HasPrefix(data, "ma_"):
		b.handleMACallback(chatID, callback.From.ID, data)
	case strings.HasPrefix(data, "strategy_"):
		b.handleStrategyCallback(chatID, callback.From.ID, data)
	case strings.HasPrefix(data, "help_"):
		b.handleHelpCallback(chatID, data)
	}

	// Отвечаем на callback
	callbackConfig := tgbotapi.NewCallback(callback.ID, "")
	if _, err := b.botAPI.Request(callbackConfig); err != nil {
		b.logger.Warn("Ошибка ответа на callback", "error", err)
	}
}

// handleInlineQuery обрабатывает inline запросы
func (b *Bot) handleInlineQuery(update tgbotapi.Update) {
	query := update.InlineQuery
	if query.Query == "" {
		return
	}

	// Создаем inline результаты
	var results []interface{}

	// Пример: поиск инструментов
	if b.isValidInstrument(query.Query) {
		// Создаем результат для инструмента
		result := tgbotapi.NewInlineQueryResultArticle(
			"instrument_"+query.Query,
			"Инструмент: "+query.Query,
			"Информация об инструменте "+query.Query,
		)
		result.Description = "Нажмите для получения информации"
		results = append(results, result)
	}

	// Отправляем результаты
	inlineConfig := tgbotapi.InlineConfig{
		InlineQueryID: query.ID,
		Results:       results,
		CacheTime:     5,
	}

	if _, err := b.botAPI.Request(inlineConfig); err != nil {
		b.logger.Warn("Ошибка отправки inline результатов", "error", err)
	}
}

// handleChosenInlineResult обрабатывает выбранный inline результат
func (b *Bot) handleChosenInlineResult(update tgbotapi.Update) {
	result := update.ChosenInlineResult
	b.logger.Debug("Выбран inline результат",
		"result_id", result.ResultID,
		"user_id", result.From.ID)

	// Здесь можно обработать выбор inline результата
	// Например, отправить дополнительную информацию
}

// handleCommandStep обрабатывает следующий шаг команды
func (b *Bot) handleCommandStep(update tgbotapi.Update, state *UserState) {
	chatID := update.Message.Chat.ID
	userID := update.Message.From.ID
	text := update.Message.Text

	// Обновляем время последней активности
	state.LastActivity = time.Now()

	switch state.CurrentCommand {
	case "candles":
		b.handleCandlesStep(chatID, userID, state, text)
	case "add_instrument":
		b.handleAddInstrumentStep(chatID, userID, state, text)
	case "remove_instrument":
		b.handleRemoveInstrumentStep(chatID, userID, state, text)
	case "turtle_test":
		b.handleTurtleTestStep(chatID, userID, state, text)
	case "broadcast":
		b.handleBroadcastStep(chatID, state, text)
	default:
		b.sendMessage(chatID, "❌ Неизвестное состояние команды")
		b.resetUserState(userID)
	}
}

// Вспомогательные функции

// getUserID возвращает ID пользователя из update
func getUserID(update tgbotapi.Update) int64 {
	switch {
	case update.Message != nil:
		return update.Message.From.ID
	case update.CallbackQuery != nil:
		return update.CallbackQuery.From.ID
	case update.InlineQuery != nil:
		return update.InlineQuery.From.ID
	case update.ChosenInlineResult != nil:
		return update.ChosenInlineResult.From.ID
	default:
		return 0
	}
}

// getUserState возвращает состояние пользователя
func (b *Bot) getUserState(userID int64) *UserState {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.userStates[userID]
}

// setUserState устанавливает состояние пользователя
func (b *Bot) setUserState(userID int64, state *UserState) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.userStates[userID] = state
}

// resetUserState сбрасывает состояние пользователя
func (b *Bot) resetUserState(userID int64) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if state, exists := b.userStates[userID]; exists && state.CancelFunc != nil {
		state.CancelFunc()
	}

	delete(b.userStates, userID)
}

// cleanupInactiveStates очищает неактивные состояния
func (b *Bot) cleanupInactiveStates(timeout time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	for userID, state := range b.userStates {
		if now.Sub(state.LastActivity) > timeout {
			if state.CancelFunc != nil {
				state.CancelFunc()
			}
			delete(b.userStates, userID)
		}
	}
}

// handleTurtleCallback обрабатывает callback для стратегии
func (b *Bot) handleTurtleCallback(chatID, userID int64, data string) {
	// ПРОВЕРКА b НО БЕЗ ОБРАЩЕНИЯ К b.logger
	if b == nil {
		// Не используем b.logger, так как b == nil!
		// Вместо этого используем стандартный вывод
		fmt.Printf("ERROR: Bot is nil in handleTurtleCallback, chatID=%d, userID=%d, data=%s\n",
			chatID, userID, data)
		debug.PrintStack() // Добавьте этот импорт: import "runtime/debug"
		return
	}

	// Теперь можно использовать b.logger, так как мы убедились что b != nil
	b.logger.Info("handleTurtleCallback called",
		"chatID", chatID,
		"userID", userID,
		"data", data)

	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		b.logger.Warn("Invalid callback data format", "data", data)
		return
	}

	action := parts[1]
	b.logger.Info("Processing turtle callback", "action", action)

	switch action {
	case "enable":
		b.logger.Info("Enabling turtle strategy", "chatID", chatID)
		b.enableTurtleStrategy(chatID)

	case "disable":
		b.logger.Info("Disabling turtle strategy", "chatID", chatID)
		b.disableTurtleStrategy(chatID)

	case "scan":
		b.logger.Info("Handling scan action", "chatID", chatID)
		update := tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				From: &tgbotapi.User{ID: userID},
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: chatID},
				},
			},
		}
		b.handleScanTurtles(update)

	case "signals":
		b.logger.Info("Handling signals action", "chatID", chatID)
		update := tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				From: &tgbotapi.User{ID: userID},
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: chatID},
				},
			},
		}
		b.handleTurtleSignals(update)

	case "config":
		b.logger.Info("Handling config action", "chatID", chatID)
		update := tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				From: &tgbotapi.User{ID: userID},
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: chatID},
				},
			},
		}
		b.handleTurtleConfig(update)

	case "test":
		b.logger.Info("Handling test action", "chatID", chatID)
		update := tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				From: &tgbotapi.User{ID: userID},
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: chatID},
				},
			},
		}
		b.handleTurtleTest(update)

	default:
		b.logger.Warn("Unknown turtle action", "action", action)
		b.sendMessage(chatID, fmt.Sprintf("Неизвестное действие: %s", action))
	}
}

// handleStrategyCallback обрабатывает callback для управления стратегией
func (b *Bot) handleStrategyCallback(chatID, userID int64, data string) {
	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		return
	}

	action := parts[1]
	param := ""
	if len(parts) > 2 {
		param = parts[2]
	}

	switch action {
	case "set_risk":
		if risk, err := strconv.ParseFloat(param, 64); err == nil {
			b.setTurtleRisk(chatID, risk)
		}
	case "set_period":
		if period, err := strconv.Atoi(param); err == nil {
			b.setTurtlePeriod(chatID, period)
		}
	}
}

// sendHelpMenu отправляет меню помощи с клавиатурой
func (b *Bot) sendHelpMenu(chatID int64) {
	msg := "🎯 Меню команд бота\n\n"
	msg += "Выберите категорию:"

	// Создаем клавиатуру с категориями команд
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Данные", "help_data"),
			tgbotapi.NewInlineKeyboardButtonData("📈 Стратегия", "help_strategy"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("⚙️ Управление", "help_management"),
			tgbotapi.NewInlineKeyboardButtonData("👑 Админ", "help_admin"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❓ Подробная справка", "help_full"),
		),
	)

	b.sendMessageWithKeyboard(chatID, msg, keyboard)
}

// handleMACallback обработка callback для стратегии MA Crossover
func (b *Bot) handleMACallback(chatID, userID int64, data string) {
	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		return
	}

	action := parts[1]

	switch action {
	case "enable":
		b.enableMAStrategy(chatID)
	case "disable":
		b.disableMAStrategy(chatID)
	case "signals":
		update := tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				From: &tgbotapi.User{ID: userID},
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: chatID},
				},
			},
		}
		b.handleMASignals(update)
	case "scan":
		update := tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				From: &tgbotapi.User{ID: userID},
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: chatID},
				},
			},
		}
		b.handleScanMA(update)
	case "config":
		update := tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				From: &tgbotapi.User{ID: userID},
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: chatID},
				},
			},
		}
		b.handleMAConfig(update)
	case "test":
		update := tgbotapi.Update{
			CallbackQuery: &tgbotapi.CallbackQuery{
				From: &tgbotapi.User{ID: userID},
				Message: &tgbotapi.Message{
					Chat: &tgbotapi.Chat{ID: chatID},
				},
			},
		}
		b.handleMATest(update)
	case "set_fast_9":
		b.setMAFastPeriod(chatID, 9)
	case "set_fast_12":
		b.setMAFastPeriod(chatID, 12)
	case "set_fast_20":
		b.setMAFastPeriod(chatID, 20)
	case "set_slow_21":
		b.setMASlowPeriod(chatID, 21)
	case "set_slow_50":
		b.setMASlowPeriod(chatID, 50)
	case "set_slow_200":
		b.setMASlowPeriod(chatID, 200)
	case "set_risk_0.01":
		b.setMARisk(chatID, 0.01)
	case "set_risk_0.02":
		b.setMARisk(chatID, 0.02)
	case "set_risk_0.05":
		b.setMARisk(chatID, 0.05)
	}
}

// enableMAStrategy включает стратегию MA Crossover
func (b *Bot) enableMAStrategy(chatID int64) {
	b.config.Strategy.MACrossover.Enabled = true
	b.sendFormattedMessage(chatID, "✅ Стратегия MA Crossover включена!\n\nИспользуйте /ma_signals для поиска сигналов.")
	b.setBotCommands()
}

// disableMAStrategy отключает стратегию MA Crossover
func (b *Bot) disableMAStrategy(chatID int64) {
	b.config.Strategy.MACrossover.Enabled = false
	b.sendFormattedMessage(chatID, "✅ Стратегия MA Crossover отключена.")
	b.setBotCommands()
}

// setMAFastPeriod устанавливает период быстрой MA
func (b *Bot) setMAFastPeriod(chatID int64, period int) {
	b.config.Strategy.MACrossover.FastPeriod = period
	b.sendFormattedMessage(chatID, fmt.Sprintf("✅ Период быстрой MA установлен: %d", period))
}

// setMASlowPeriod устанавливает период медленной MA
func (b *Bot) setMASlowPeriod(chatID int64, period int) {
	b.config.Strategy.MACrossover.SlowPeriod = period
	b.sendFormattedMessage(chatID, fmt.Sprintf("✅ Период медленной MA установлен: %d", period))
}

// setMARisk устанавливает риск на сделку
func (b *Bot) setMARisk(chatID int64, risk float64) {
	b.config.Strategy.MACrossover.RiskPerTrade = risk
	b.sendFormattedMessage(chatID, fmt.Sprintf("✅ Риск на сделку установлен: %.1f%%", risk*100))
}
