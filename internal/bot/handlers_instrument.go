package bot

import (
	"context"
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleAddInstrument(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	state := &UserState{
		CurrentCommand: "add_instrument",
		Step:           1,
		Data:           make(map[string]interface{}),
		LastActivity:   time.Now(),
	}
	b.setUserState(userID, state)

	return b.sendFormattedMessage(chatID, "➕ Добавление инструмента\n\nВведите тикер инструмента для добавления:")
}

func (b *Bot) handleRemoveInstrument(update tgbotapi.Update) error {
	chatID, err := b.getChatID(update)
	if err != nil {
		return err
	}

	userID, err := b.getUserID(update)
	if err != nil {
		return b.sendFormattedMessage(chatID, "❌ Не удалось определить пользователя")
	}

	state := &UserState{
		CurrentCommand: "remove_instrument",
		Step:           1,
		Data:           make(map[string]interface{}),
		LastActivity:   time.Now(),
	}
	b.setUserState(userID, state)

	return b.sendFormattedMessage(chatID, "➖ Удаление инструмента\n\nВведите тикер инструмента для удаления:")
}

func (b *Bot) handleAddInstrumentStep(chatID, userID int64, state *UserState, text string) {
	// Приводим тикер к верхнему регистру
	instrument := b.normalizeInstrument(text)

	if !b.isValidInstrument(instrument) {
		b.sendFormattedMessage(chatID, "❌ Неверный формат тикера. Попробуйте снова:")
		return
	}

	// Добавляем инструмент
	result, err := b.apiClient.AddInstrument(context.Background(), instrument)
	if err != nil {
		b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка добавления инструмента: %v", err))
	} else {
		status := "unknown"
		if s, ok := result["status"].(string); ok {
			status = s
		}

		message := fmt.Sprintf("✅ Инструмент %s добавлен\nСтатус: %s", instrument, status)
		b.sendFormattedMessage(chatID, message)
	}

	// Завершаем диалог
	b.resetUserState(userID)
}

func (b *Bot) handleRemoveInstrumentStep(chatID, userID int64, state *UserState, text string) {
	// Приводим тикер к верхнему регистру
	instrument := b.normalizeInstrument(text)

	if !b.isValidInstrument(instrument) {
		b.sendFormattedMessage(chatID, "❌ Неверный формат тикера. Попробуйте снова:")
		return
	}

	// Удаляем инструмент
	err := b.apiClient.RemoveInstrument(context.Background(), instrument)
	if err != nil {
		b.sendFormattedMessage(chatID, fmt.Sprintf("❌ Ошибка удаления инструмента: %v", err))
	} else {
		b.sendFormattedMessage(chatID, fmt.Sprintf("✅ Инструмент %s удален", instrument))
	}

	// Завершаем диалог
	b.resetUserState(userID)
}

// handleAddInstrumentStep, handleRemoveInstrumentStep
