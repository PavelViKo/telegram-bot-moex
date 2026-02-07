package bot

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleConfig(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.isAdmin(userID) {
		return b.sendFormattedMessage(chatID, "❌ Просмотр конфигурации доступен только администраторам")
	}

	// Используем метод PrintConfig из пакета config
	configInfo := b.config.PrintConfig()

	msg := "⚙️ КОНФИГУРАЦИЯ БОТА\n\n"
	msg += configInfo
	msg += "\n💡 Для изменения конфигурации отредактируйте файл config.yaml и перезапустите бота"

	return b.sendFormattedMessage(chatID, msg)
}

func (b *Bot) handleRestart(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.isAdmin(userID) {
		return b.sendFormattedMessage(chatID, "❌ Перезапуск бота доступен только администраторам")
	}

	msg := "🔄 Перезапуск бота...\n\n"
	msg += "Бот будет перезапущен через 3 секунды.\n"
	msg += "Это займет несколько секунд.\n\n"
	msg += "Статус: готов к перезапуску"

	b.sendFormattedMessage(chatID, msg)

	// В реальной реализации здесь был бы вызов перезапуска
	// Пока просто сообщаем, что функция в разработке
	time.Sleep(2 * time.Second)
	b.sendFormattedMessage(chatID, "⏳ Перезапуск в разработке...\n\nДля перезапуска используйте команду:\n`docker-compose restart telegram-bot`")

	return nil
}

func (b *Bot) handleStop(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.isAdmin(userID) {
		return b.sendFormattedMessage(chatID, "❌ Остановка бота доступна только администраторам")
	}

	msg := "🛑 Остановка бота...\n\n"
	msg += "Бот будет остановлен через 3 секунды.\n"
	msg += "Для запуска используйте команду:\n"
	msg += "`docker-compose start telegram-bot`\n\n"
	msg += "Статус: готов к остановке"

	b.sendFormattedMessage(chatID, msg)

	// В реальной реализации здесь была бы остановка
	time.Sleep(2 * time.Second)
	b.sendFormattedMessage(chatID, "⏳ Остановка в разработке...\n\nДля остановки используйте команду:\n`docker-compose stop telegram-bot`")

	return nil
}

func (b *Bot) handleUsers(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.isAdmin(userID) {
		return b.sendFormattedMessage(chatID, "❌ Управление пользователями доступно только администраторам")
	}

	msg := "👥 УПРАВЛЕНИЕ ПОЛЬЗОВАТЕЛЯМИ\n\n"

	// Показываем текущих пользователей
	if len(b.config.Security.AllowedUsers) == 0 {
		msg += "📭 Список пользователей пуст\n"
		msg += "Все пользователи разрешены (аутентификация отключена)\n\n"
	} else {
		msg += "✅ Аутентификация ВКЛЮЧЕНА\n"
		msg += fmt.Sprintf("📋 Разрешенных пользователей: %d\n\n", len(b.config.Security.AllowedUsers))

		for i, userID := range b.config.Security.AllowedUsers {
			isAdmin := b.config.IsAdmin(userID)
			adminText := ""
			if isAdmin {
				adminText = "👑 АДМИН"
			}

			// Проверяем активность пользователя
			isActive := b.stats.IsUserActive(userID)
			activeText := "💤"
			if isActive {
				activeText = "💚"
			}

			msg += fmt.Sprintf("%d. %s ID: %d %s %s\n",
				i+1, activeText, userID, adminText, b.getUserActivityInfo(userID))
		}
	}

	msg += "\n⚙️ КОМАНДЫ УПРАВЛЕНИЯ:\n"
	msg += "• /config - Настройки безопасности\n"
	msg += "• Для изменения списка пользователей отредактируйте config.yaml\n\n"

	msg += "📊 СТАТИСТИКА АКТИВНОСТИ:\n"
	msg += fmt.Sprintf("• Активных пользователей: %d\n", b.stats.GetActiveUsersCount())
	msg += fmt.Sprintf("• Всего команд: %d\n", b.stats.CommandsExecuted)
	msg += fmt.Sprintf("• Ошибок: %d\n", b.stats.Errors)

	return b.sendFormattedMessage(chatID, msg)
}

func (b *Bot) handleBroadcast(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.isAdmin(userID) {
		return b.sendFormattedMessage(chatID, "❌ Рассылка сообщений доступна только администраторам")
	}

	// Начинаем процесс рассылки
	state := &UserState{
		CurrentCommand: "broadcast",
		Step:           1,
		Data:           make(map[string]interface{}),
		LastActivity:   time.Now(),
	}
	b.setUserState(userID, state)

	msg := "📢 РАССЫЛКА СООБЩЕНИЙ\n\n"
	msg += "Эта команда отправит сообщение всем активным пользователям бота.\n\n"
	msg += "Введите текст сообщения для рассылки:"

	return b.sendFormattedMessage(chatID, msg)
}

func (b *Bot) handleBroadcastStep(chatID int64, state *UserState, text string) {

	if text == "" {
		b.sendFormattedMessage(chatID, "❌ Текст сообщения не может быть пустым. Попробуйте снова:")
		return
	}

	state.Data["message"] = text
	state.Step = 2

	msg := "📢 ПОДТВЕРЖДЕНИЕ РАССЫЛКИ\n\n"
	msg += fmt.Sprintf("Текст сообщения:\n«%s»\n\n", text)
	msg += "Получатели: все активные пользователи\n"
	msg += fmt.Sprintf("Количество получателей: %d\n\n", b.stats.GetActiveUsersCount())
	msg += "Отправить рассылку?"

	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Отправить", "broadcast_confirm"),
			tgbotapi.NewInlineKeyboardButtonData("❌ Отменить", "cancel"),
		),
	)

	b.sendMessageWithKeyboard(chatID, msg, keyboard)
}

func (b *Bot) handleDebug(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.isAdmin(userID) {
		return b.sendFormattedMessage(chatID, "❌ Режим отладки доступен только администраторам")
	}

	// Переключаем режим отладки
	b.config.Telegram.Debug = !b.config.Telegram.Debug
	b.botAPI.Debug = b.config.Telegram.Debug

	status := "ВКЛЮЧЕН"
	if !b.config.Telegram.Debug {
		status = "ВЫКЛЮЧЕН"
	}

	msg := fmt.Sprintf("🔧 РЕЖИМ ОТЛАДКИ: %s\n\n", status)

	if b.config.Telegram.Debug {
		msg += "✅ Теперь вы будете получать подробные логи от Telegram API\n"
		msg += "📝 Все запросы и ответы будут логироваться\n\n"
		msg += "💡 Для выключения используйте команду /debug снова"
	} else {
		msg += "✅ Подробное логирование Telegram API отключено\n"
		msg += "📝 Будут логироваться только основные события\n\n"
		msg += "💡 Для включения используйте команду /debug снова"
	}

	return b.sendFormattedMessage(chatID, msg)
}

func (b *Bot) handleSystem(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	if !b.isAdmin(userID) {
		return b.sendFormattedMessage(chatID, "❌ Системная информация доступна только администраторам")
	}

	msg := "🖥 СИСТЕМНАЯ ИНФОРМАЦИЯ\n\n"

	// Информация о боте
	msg += "🤖 БОТ:\n"
	msg += "• Версия: 1.0.0\n"
	msg += fmt.Sprintf("• Запущен: %s\n", b.stats.StartTime.Format("2006-01-02 15:04:05"))
	msg += fmt.Sprintf("• Uptime: %s\n", b.stats.GetUptime().Truncate(time.Second))
	msg += fmt.Sprintf("• ID: %d\n", b.botAPI.Self.ID)
	msg += fmt.Sprintf("• Username: @%s\n", b.botAPI.Self.UserName)
	msg += fmt.Sprintf("• Режим отладки: %v\n\n", b.config.Telegram.Debug)

	// Статистика
	msg += "📊 СТАТИСТИКА:\n"
	msg += fmt.Sprintf("• Сообщений получено: %d\n", b.stats.MessagesReceived)
	msg += fmt.Sprintf("• Сообщений отправлено: %d\n", b.stats.MessagesSent)
	msg += fmt.Sprintf("• Команд выполнено: %d\n", b.stats.CommandsExecuted)
	msg += fmt.Sprintf("• Ошибок: %d\n", b.stats.Errors)
	msg += fmt.Sprintf("• Активных пользователей: %d\n", b.stats.GetActiveUsersCount())
	msg += fmt.Sprintf("• Состояний пользователей: %d\n\n", len(b.userStates))

	// Конфигурация
	msg += "⚙️ КОНФИГУРАЦИЯ:\n"
	msg += fmt.Sprintf("• Стратегия 'Черепах': %v\n", b.config.Strategy.Turtles.Enabled)
	msg += fmt.Sprintf("• Аутентификация: %v\n", b.config.Security.EnableAuth)
	msg += fmt.Sprintf("• Разрешенных пользователей: %d\n", len(b.config.Security.AllowedUsers))
	msg += fmt.Sprintf("• Администраторов: %d\n", len(b.config.Security.AdminUsers))
	msg += fmt.Sprintf("• Таймаут обновлений: %d сек\n", b.config.Telegram.UpdatesTimeout)
	msg += fmt.Sprintf("• Таймаут API: %v\n\n", b.config.API.Timeout)

	// Состояние системы
	msg += "🔄 СОСТОЯНИЕ СИСТЕМЫ:\n"
	msg += "• Бот: 🟢 Работает\n"
	msg += "• Очередь сообщений: 🟢 Активна\n"
	msg += "• Очистка состояний: 🟢 Активна\n"

	if b.config.Strategy.Turtles.Enabled {
		msg += "• Автоанализ стратегии: 🟢 Активен\n"
	} else {
		msg += "• Автоанализ стратегии: 🔴 Отключен\n"
	}

	// Кнопки действий
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔄 Обновить", "system_refresh"),
			tgbotapi.NewInlineKeyboardButtonData("🧹 Очистить кэш", "system_cleanup"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📊 Подробная статистика", "system_stats"),
			tgbotapi.NewInlineKeyboardButtonData("📝 Логи", "system_logs"),
		),
	)

	return b.sendMessageWithKeyboard(chatID, msg, keyboard)
}

func (b *Bot) handleLogs(update tgbotapi.Update) error {
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

	// Здесь можно реализовать чтение и отправку логов
	// Пока просто заглушка
	return b.sendFormattedMessage(chatID, "📝 Функция просмотра логов в разработке")
}

func (b *Bot) handleAdmin(update tgbotapi.Update) error {
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
	msg := "👑 Админ панель\n\n"
	msg += "Доступные команды:\n"
	msg += "• /refresh - Обновить список инструментов\n"
	msg += "• /cleanup - Очистка старых таблиц\n"
	msg += "• /fetch - Принудительная загрузка данных\n"
	msg += "• /log - Показать логи\n"
	msg += "• /restart - Перезапустить бота\n\n"
	msg += "Статистика бота:\n"
	msg += fmt.Sprintf("• Пользователей: %d\n", b.stats.GetActiveUsersCount())
	msg += fmt.Sprintf("• Команд выполнено: %d\n", b.stats.CommandsExecuted)
	msg += fmt.Sprintf("• Ошибок: %d\n", b.stats.Errors)
	msg += fmt.Sprintf("• Uptime: %s\n", b.stats.GetUptime().Truncate(time.Second))

	return b.sendFormattedMessage(chatID, msg)
}

func (b *Bot) getUserActivityInfo(userID int64) string {
	b.stats.mu.RLock()
	defer b.stats.mu.RUnlock()

	if lastActivity, exists := b.stats.ActiveUsers[userID]; exists {
		minutesAgo := int(time.Since(lastActivity).Minutes())
		if minutesAgo < 1 {
			return "(только что)"
		} else if minutesAgo < 60 {
			return fmt.Sprintf("(%d мин. назад)", minutesAgo)
		} else {
			hoursAgo := minutesAgo / 60
			return fmt.Sprintf("(%d ч. назад)", hoursAgo)
		}
	}
	return "(неактивен)"
}
