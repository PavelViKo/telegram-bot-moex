package bot

import (
	"fmt"
	"regexp"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) getChatID(update tgbotapi.Update) (int64, error) {
	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		return update.CallbackQuery.Message.Chat.ID, nil
	}

	if update.Message != nil {
		return update.Message.Chat.ID, nil
	}

	return 0, fmt.Errorf("не удалось получить ChatID из Update")
}

// getUserID безопасно извлекает UserID из Update
func (b *Bot) getUserID(update tgbotapi.Update) (int64, error) {
	if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		return update.CallbackQuery.From.ID, nil
	}

	if update.Message != nil && update.Message.From != nil {
		return update.Message.From.ID, nil
	}

	return 0, fmt.Errorf("не удалось получить UserID из Update")
}

// getUser безопасно извлекает информацию о пользователе
func (b *Bot) getUser(update tgbotapi.Update) (*tgbotapi.User, error) {
	if update.CallbackQuery != nil && update.CallbackQuery.From != nil {
		return update.CallbackQuery.From, nil
	}

	if update.Message != nil && update.Message.From != nil {
		return update.Message.From, nil
	}

	return nil, fmt.Errorf("не удалось получить информацию о пользователе")
}

func (b *Bot) getSignalTypeText(signalType string) string {
	switch signalType {
	case "entry_long":
		return "🟢 ПОКУПКА"
	case "entry_short":
		return "🔴 ПРОДАЖА"
	case "exit_long":
		return "📤 ВЫХОД ИЗ ПОКУПКИ"
	case "exit_short":
		return "📤 ВЫХОД ИЗ ПРОДАЖИ"
	case "no_signal":
		return "📊 ИНФОРМАЦИЯ"
	default:
		return signalType
	}
}

// isValidInstrument проверяет валидность инструмента (должен быть в utils.go)
func (b *Bot) isValidInstrument(instrument string) bool {
	if instrument == "" || len(instrument) > 20 {
		return false
	}

	// Проверяем, что тикер состоит из букв и цифр
	pattern := `^[A-Za-z0-9]+$`
	match, err := regexp.MatchString(pattern, instrument)
	if err != nil || !match {
		return false
	}

	return true
}

// handleCancelCallback обработка отмены через callback
func (b *Bot) handleCancelCallback(chatID, userID int64) {
	b.resetUserState(userID)
	b.sendMessage(chatID, "❌ Операция отменена")
}

func (b *Bot) normalizeInstrument(ticker string) string {
	return strings.ToUpper(strings.TrimSpace(ticker))
}

// escapeForMarkdown экранирует специальные символы для MarkdownV2
func (b *Bot) escapeForMarkdown(text string) string {
	// Специальные символы для экранирования в MarkdownV2
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}

	result := text
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "\\"+char)
	}
	return result
}

// escapeSpecialChars экранирует специальные символы для HTML
func (b *Bot) escapeSpecialChars(text string) string {
	// Экранируем символы, которые могут быть интерпретированы как HTML-теги
	replacements := map[string]string{
		"<":  "&lt;",
		">":  "&gt;",
		"&":  "&amp;",
		"\"": "&quot;",
		"'":  "&#39;",
		"`":  "&#96;",
	}

	result := text
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}
	return result
}

// sendSafeMessageWithKeyboard отправляет сообщение с экранированными символами
func (b *Bot) sendSafeMessageWithKeyboard(chatID int64, text string, keyboard tgbotapi.InlineKeyboardMarkup) error {
	// Пробуем отправить с MarkdownV2
	safeText := b.escapeForMarkdown(text)

	msg := tgbotapi.NewMessage(chatID, safeText)
	msg.ParseMode = "MarkdownV2"
	msg.ReplyMarkup = keyboard

	_, err := b.botAPI.Send(msg)
	if err != nil {
		// Если Markdown не работает, пробуем HTML
		b.logger.Debug("MarkdownV2 не сработал, пробуем HTML", "error", err)

		safeText = b.escapeSpecialChars(text)
		msg.Text = safeText
		msg.ParseMode = "HTML"

		_, err = b.botAPI.Send(msg)
		if err != nil {
			// Если и HTML не работает, отправляем без форматирования
			b.logger.Debug("HTML не сработал, отправляем без форматирования", "error", err)

			// Удаляем все специальные символы
			msg.Text = b.removeSpecialChars(text)
			msg.ParseMode = ""

			_, err = b.botAPI.Send(msg)
			if err != nil {
				b.logger.Error("Ошибка отправки сообщения",
					"chat_id", chatID, "error", err)
				return err
			}
		}
	}

	b.stats.MessagesSent++
	return nil
}

// removeSpecialChars удаляет все специальные символы
func (b *Bot) removeSpecialChars(text string) string {
	specialChars := []string{"<", ">", "&", "\"", "'", "`", "_", "*", "[", "]", "(", ")", "~", "#", "+", "-", "=", "|", "{", "}", ".", "!"}

	result := text
	for _, char := range specialChars {
		result = strings.ReplaceAll(result, char, "")
	}
	return result
}
