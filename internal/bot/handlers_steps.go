package bot

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func (b *Bot) handleCandlesStep(chatID, userID int64, state *UserState, text string) {
	switch state.Step {
	case 1: // Ввод инструмента
		instrument := b.normalizeInstrument(text)

		if !b.isValidInstrument(instrument) {
			b.sendFormattedMessage(chatID, "❌ Неверный тикер инструмента. Попробуйте снова:")
			return
		}

		state.Data["instrument"] = instrument
		state.Step = 2

		// Получаем доступные таймфреймы
		timeframes, err := b.apiClient.GetInstrumentTimeframes(context.Background(), text)
		if err != nil {
			b.sendFormattedMessage(chatID, "❌ Инструмент не найден или нет данных")
			b.resetUserState(userID)
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

		msg := tgbotapi.NewMessage(chatID, fmt.Sprintf("📈 Инструмент: %s\n\nВыберите таймфрейм:", text))
		msg.ReplyMarkup = markup

		if _, err := b.botAPI.Send(msg); err != nil {
			b.logger.Error("Ошибка отправки сообщения", "error", err)
		}
	}
}

func (b *Bot) handleTurtleTestStep(chatID, userID int64, state *UserState, text string) {
	instrument := b.normalizeInstrument(text)

	if !b.isValidInstrument(instrument) {
		b.sendFormattedMessage(chatID, "❌ Неверный тикер инструмента. Попробуйте снова (например: SBER):")
		return
	}

	state.Data["instrument"] = instrument

	// Запускаем тест в фоне
	go b.runTurtleTest(chatID, instrument)

	// Завершаем состояние
	b.resetUserState(userID)
}
