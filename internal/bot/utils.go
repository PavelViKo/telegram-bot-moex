package bot

import (
	"context"
	"fmt"

	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Отсутствующие методы и вспомогательные функции

// sendFormattedMessage отправляет форматированное сообщение
func (b *Bot) sendFormattedMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = "HTML"

	_, err := b.botAPI.Send(msg)
	if err != nil {
		b.logger.Error("Ошибка отправки сообщения",
			"chat_id", chatID,
			"error", err)
		return err
	}

	b.stats.UpdateStats("message_sent")
	return nil
}

// sendMessage отправляет простое сообщение
func (b *Bot) sendMessage(chatID int64, text string) error {
	msg := tgbotapi.NewMessage(chatID, text)

	_, err := b.botAPI.Send(msg)
	if err != nil {
		b.logger.Error("Ошибка отправки сообщения",
			"chat_id", chatID,
			"error", err)
		return err
	}

	b.stats.UpdateStats("message_sent")
	return nil
}

// sendInstrumentInfo отправляет информацию об инструменте
func (b *Bot) sendInstrumentInfo(chatID int64, instrument string) {
	if !b.isValidInstrument(instrument) {
		b.sendMessage(chatID, fmt.Sprintf("❌ Неверный формат инструмента: %s", instrument))
		return
	}

	// Получаем информацию об инструменте
	info, err := b.apiClient.GetInstrumentInfo(context.Background(), instrument)
	if err != nil {
		b.sendMessage(chatID, fmt.Sprintf("❌ Инструмент %s не найден", instrument))
		return
	}

	msg := fmt.Sprintf("📊 Инструмент: %s\n\n", instrument)

	if timeframes, ok := info["timeframes"].([]interface{}); ok {
		msg += "📈 Доступные данные:\n"
		for _, tf := range timeframes {
			if tfMap, ok := tf.(map[string]interface{}); ok {
				msg += fmt.Sprintf("• %s: ", tfMap["display_name"])
				if lastCandle, ok := tfMap["last_candle"].(map[string]interface{}); ok {
					msg += fmt.Sprintf("последняя свеча %s\n", lastCandle["date"])
				} else {
					msg += "нет данных\n"
				}
			}
		}
	}

	// Добавляем кнопки для быстрого доступа
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📈 Свечи", fmt.Sprintf("instrument_candles_%s", instrument)),
			tgbotapi.NewInlineKeyboardButtonData("📊 Статистика", fmt.Sprintf("instrument_stats_%s", instrument)),
		),
	)

	message := tgbotapi.NewMessage(chatID, msg)
	message.ReplyMarkup = keyboard

	if _, err := b.botAPI.Send(message); err != nil {
		b.logger.Error("Ошибка отправки сообщения", "error", err)
	}
}

// handleTimeframeSelection обработка выбора таймфрейма
func (b *Bot) handleTimeframeSelection(chatID, userID int64, timeframe string) {
	state := b.getUserState(userID)
	if state == nil || state.CurrentCommand != "candles" {
		b.sendMessage(chatID, "❌ Сессия истекла. Начните заново с /candles")
		return
	}

	state.Data["timeframe"] = timeframe
	state.Step = 3

	b.sendFormattedMessage(chatID,
		fmt.Sprintf("📈 Инструмент: %s\nТаймфрейм: %s\n\nТеперь укажите период (например: 7d, 30d, 1y или конкретные даты: 2024-01-01:2024-01-31):",
			state.Data["instrument"], timeframe))
}

// handleCleanupAction обработка очистки таблиц
func (b *Bot) handleCleanupAction(chatID int64, days int) {
	b.sendFormattedMessage(chatID, fmt.Sprintf("🧹 Запуск очистки таблиц неактивных более %d дней...", days))

	go func() {
		result, err := b.apiClient.CleanupTables(context.Background(), days)
		if err != nil {
			b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка очистки: %v", err))
			return
		}

		msg := "✅ Очистка запущена\n\n"
		if status, ok := result["status"].(string); ok {
			msg += fmt.Sprintf("Статус: %s\n", status)
		}
		if message, ok := result["message"].(string); ok {
			msg += fmt.Sprintf("Сообщение: %s\n", message)
		}

		b.sendFormattedMessage(chatID, msg)
	}()
}

// handleInstrumentCallback обработка callback для инструментов
func (b *Bot) handleInstrumentCallback(chatID int64, data string) {
	// Пример: instrument_candles_SBER
	parts := strings.Split(data, "_")
	if len(parts) < 3 {
		return
	}

	action := parts[1]
	instrument := parts[2]

	switch action {
	case "candles":
		// Начинаем процесс получения свечей для этого инструмента
		state := &UserState{
			CurrentCommand: "candles",
			Step:           2, // Пропускаем ввод инструмента
			Data: map[string]interface{}{
				"instrument": instrument,
			},
			LastActivity: time.Now(),
		}
		b.setUserState(chatID, state) // Используем chatID как userID для простоты

		// Получаем доступные таймфреймы
		timeframes, err := b.apiClient.GetInstrumentTimeframes(context.Background(), instrument)
		if err != nil {
			b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка получения таймфреймов для %s", instrument))
			return
		}

		// Создаем клавиатуру с таймфреймами
		var rows [][]tgbotapi.InlineKeyboardButton
		for _, tf := range timeframes {
			rows = append(rows, tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData(
					fmt.Sprintf("%s (%s)", tf["display_name"], tf["timeframe"]),
					fmt.Sprintf("timeframe_%s", tf["timeframe"]),
				),
			))
		}
		rows = append(rows, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Отмена", "cancel"),
		))

		markup := tgbotapi.NewInlineKeyboardMarkup(rows...)

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📈 Инструмент: %s\n\nВыберите таймфрейм:", instrument))
		msg.ReplyMarkup = markup

		b.botAPI.Send(msg)

	case "stats":
		b.sendInstrumentInfo(chatID, instrument)
	}
}

// handleAdminCallback обработка admin callback
func (b *Bot) handleAdminCallback(chatID int64, data string) {
	// Пример: admin_restart, admin_logs, etc
	parts := strings.Split(data, "_")
	if len(parts) < 2 {
		return
	}

	action := parts[1]

	switch action {
	case "restart":
		b.sendMessage(chatID, "🔄 Перезапуск бота...")
		// Здесь будет логика перезапуска
	case "logs":
		b.sendMessage(chatID, "📝 Получение логов...")
		// Здесь будет логика получения логов
	}
}

// isAdmin проверяет, является ли пользователь администратором
func (b *Bot) isAdmin(userID int64) bool {
	for _, adminID := range b.config.Security.AdminUsers {
		if adminID == userID {
			return true
		}
	}
	return false
}

// isUserAllowed проверяет, разрешен ли пользователь
func (b *Bot) isUserAllowed(update tgbotapi.Update) bool {
	userID := getUserID(update)

	// Если проверка отключена, все разрешены
	if !b.config.Security.EnableAuth {
		return true
	}

	// Проверяем список разрешенных пользователей
	for _, allowedID := range b.config.Security.AllowedUsers {
		if allowedID == userID {
			return true
		}
	}

	return false
}

// startCleanupRoutine запускает горутину для очистки неактивных состояний
func (b *Bot) startCleanupRoutine(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Очищаем неактивные состояния (30 минут неактивности)
			b.cleanupInactiveStates(30 * time.Minute)

			// Очищаем неактивных пользователей в статистике
			b.stats.CleanupInactiveUsers(1 * time.Hour)

			b.logger.Debug("Очистка неактивных состояний выполнена",
				"states_count", len(b.userStates),
				"active_users", b.stats.GetActiveUsersCount())
		}
	}
}

// sendMessageWithKeyboard отправляет сообщение с клавиатурой
func (b *Bot) sendMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	return b.sendSafeMessageWithKeyboard(chatID, text, keyboard)
}

// editMessage редактирует существующее сообщение
func (b *Bot) editMessage(chatID int64, messageID int, text string, keyboard interface{}) error {
	// Экранируем текст перед отправкой
	safeText := b.escapeForMarkdown(text)

	editMsg := tgbotapi.NewEditMessageText(chatID, messageID, safeText)
	editMsg.ParseMode = "MarkdownV2"

	// Обрабатываем только InlineKeyboardMarkup, так как другие типы не поддерживаются для редактирования
	if keyboard != nil {
		switch k := keyboard.(type) {
		case *tgbotapi.InlineKeyboardMarkup:
			editMsg.ReplyMarkup = k
		case tgbotapi.InlineKeyboardMarkup:
			editMsg.ReplyMarkup = &k
		default:
			// Другие типы клавиатур не поддерживаются при редактировании сообщений
			// Для них нужно отправлять новое сообщение
			b.logger.Debug("Keyboard type not supported for edit, sending new message",
				"type", fmt.Sprintf("%T", keyboard))

			// Отправляем новое сообщение с нужной клавиатурой
			// Нужно проверить тип клавиатуры
			if inlineKeyboard, ok := keyboard.(tgbotapi.InlineKeyboardMarkup); ok {
				return b.sendSafeMessageWithKeyboard(chatID, text, inlineKeyboard)
			} else if inlineKeyboardPtr, ok := keyboard.(*tgbotapi.InlineKeyboardMarkup); ok {
				return b.sendSafeMessageWithKeyboard(chatID, text, *inlineKeyboardPtr)
			} else {
				// Если не InlineKeyboardMarkup, отправляем без клавиатуры
				return b.sendFormattedMessage(chatID, text)
			}
		}
	}

	_, err := b.botAPI.Send(editMsg)
	if err != nil {
		// Пробуем с HTML если Markdown не работает
		b.logger.Debug("MarkdownV2 не сработал при редактировании, пробуем HTML", "error", err)

		safeText = b.escapeSpecialChars(text)
		editMsg.Text = safeText
		editMsg.ParseMode = "HTML"

		_, err = b.botAPI.Send(editMsg)
		if err != nil {
			// Пробуем без форматирования
			b.logger.Debug("HTML не сработал при редактировании, пробуем без форматирования", "error", err)

			safeText = b.removeSpecialChars(text)
			editMsg.Text = safeText
			editMsg.ParseMode = ""

			_, err = b.botAPI.Send(editMsg)
			if err != nil {
				b.logger.Error("Ошибка редактирования сообщения",
					"chat_id", chatID,
					"message_id", messageID,
					"error", err)
				return err
			}
		}
	}

	return nil
}

// deleteMessage удаляет сообщение
func (b *Bot) deleteMessage(chatID int64, messageID int) error {
	deleteMsg := tgbotapi.NewDeleteMessage(chatID, messageID)

	_, err := b.botAPI.Send(deleteMsg)
	if err != nil {
		b.logger.Warn("Ошибка удаления сообщения",
			"chat_id", chatID,
			"message_id", messageID,
			"error", err)
		return err
	}

	return nil
}

// sendTypingAction отправляет индикатор набора текста
func (b *Bot) sendTypingAction(chatID int64) error {
	action := tgbotapi.NewChatAction(chatID, "typing")

	_, err := b.botAPI.Send(action)
	if err != nil {
		b.logger.Warn("Ошибка отправки действия typing",
			"chat_id", chatID,
			"error", err)
		return err
	}

	return nil
}

// getChatMember получает информацию о участнике чата
func (b *Bot) getChatMember(chatID, userID int64) (*tgbotapi.ChatMember, error) {
	config := tgbotapi.GetChatMemberConfig{
		ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
			ChatID: chatID,
			UserID: userID,
		},
	}

	member, err := b.botAPI.GetChatMember(config)
	if err != nil {
		return nil, err
	}

	return &member, nil
}

// sendDocument отправляет документ
func (b *Bot) sendDocument(chatID int64, filePath, caption string) error {
	doc := tgbotapi.NewDocument(chatID, tgbotapi.FilePath(filePath))
	if caption != "" {
		doc.Caption = caption
	}

	_, err := b.botAPI.Send(doc)
	if err != nil {
		b.logger.Error("Ошибка отправки документа",
			"chat_id", chatID,
			"file_path", filePath,
			"error", err)
		return err
	}

	b.stats.UpdateStats("message_sent")
	return nil
}
