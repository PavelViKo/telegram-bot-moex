package analysis

import (
	"context"
	"fmt"
	"math"
	"time"

	"telegram-bot-moex/internal/api"
)

// TurtleStrategy реализует стратегию "Черепах"
type TurtleStrategy struct {
	apiClient         *api.APIClient
	lookbackPeriod    int
	entryBreakoutDays int
	exitBreakoutDays  int
	atrPeriod         int
	atrMultiplier     float64
	riskPerTrade      float64
	mathUtils         *MathUtils // ДОБАВЛЯЕМ
}

// Signal представляет торговый сигнал
type Signal struct {
	Instrument   string
	SignalType   string // "entry_long", "entry_short", "exit_long", "exit_short", "no_signal"
	Price        float64
	StopLoss     float64
	TakeProfit   float64
	PositionSize float64
	Reason       string // Детальное описание условий
	Timestamp    time.Time
}

// NewTurtleStrategy создает новую стратегию "Черепах"
func NewTurtleStrategy(apiClient *api.APIClient, lookbackPeriod, entryBreakoutDays, exitBreakoutDays, atrPeriod int, atrMultiplier, riskPerTrade float64) *TurtleStrategy {
	return &TurtleStrategy{
		apiClient:         apiClient,
		lookbackPeriod:    lookbackPeriod,
		entryBreakoutDays: entryBreakoutDays,
		exitBreakoutDays:  exitBreakoutDays,
		atrPeriod:         atrPeriod,
		atrMultiplier:     atrMultiplier,
		riskPerTrade:      riskPerTrade,
		mathUtils:         &MathUtils{}, // ДОБАВЛЯЕМ
	}
}

// AnalyzeInstrument анализирует инструмент и возвращает сигналы с детальной информацией
func (ts *TurtleStrategy) AnalyzeInstrument(ctx context.Context, instrument string) ([]Signal, error) {
	// Получаем свечи за нужный период
	to := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -ts.lookbackPeriod*2).Format("2006-01-02")

	candles, err := ts.apiClient.GetCandles(ctx, instrument, "24", from, to)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения свечей: %w", err)
	}

	if len(candles) < ts.lookbackPeriod {
		return nil, fmt.Errorf("недостаточно данных для анализа")
	}

	// Конвертируем свечи в удобный формат
	prices := make([]float64, len(candles))
	highs := make([]float64, len(candles))
	lows := make([]float64, len(candles))
	closes := make([]float64, len(candles))
	dates := make([]time.Time, len(candles))

	for i, candle := range candles {
		if closeVal, ok := candle["close"].(float64); ok {
			closes[i] = closeVal
			prices[i] = closeVal
		}
		if highVal, ok := candle["high"].(float64); ok {
			highs[i] = highVal
		}
		if lowVal, ok := candle["low"].(float64); ok {
			lows[i] = lowVal
		}
		if beginVal, ok := candle["begin"].(string); ok {
			if t, err := time.Parse("2006-01-02 15:04:05", beginVal); err == nil {
				dates[i] = t
			}
		}
	}

	// Рассчитываем показатели
	entryBreakoutHigh := ts.calculateBreakout(highs, ts.entryBreakoutDays, "high")
	entryBreakoutLow := ts.calculateBreakout(lows, ts.entryBreakoutDays, "low")
	exitBreakoutHigh := ts.calculateBreakout(highs, ts.exitBreakoutDays, "high")
	exitBreakoutLow := ts.calculateBreakout(lows, ts.exitBreakoutDays, "low")
	atr := ts.calculateATR(highs, lows, closes)

	// Находим даты прорывов
	entryBreakoutHighDate := ts.findBreakoutDate(dates, highs, entryBreakoutHigh, ts.entryBreakoutDays)
	entryBreakoutLowDate := ts.findBreakoutDate(dates, lows, entryBreakoutLow, ts.entryBreakoutDays)
	exitBreakoutHighDate := ts.findBreakoutDate(dates, highs, exitBreakoutHigh, ts.exitBreakoutDays)
	exitBreakoutLowDate := ts.findBreakoutDate(dates, lows, exitBreakoutLow, ts.exitBreakoutDays)

	currentPrice := closes[len(closes)-1]
	currentDate := dates[len(dates)-1]

	var signals []Signal

	// Формируем детальную информацию о прорывах
	entryHighInfo := fmt.Sprintf("Прорыв входа для лонга: %.2f (дата установки: %s)",
		entryBreakoutHigh, entryBreakoutHighDate.Format("02.01.2006"))
	entryLowInfo := fmt.Sprintf("Прорыв входа для шорта: %.2f (дата установки: %s)",
		entryBreakoutLow, entryBreakoutLowDate.Format("02.01.2006"))
	exitHighInfo := fmt.Sprintf("Прорыв выхода для шорта: %.2f (дата установки: %s)",
		exitBreakoutHigh, exitBreakoutHighDate.Format("02.01.2006"))
	exitLowInfo := fmt.Sprintf("Прорыв выхода для лонга: %.2f (дата установки: %s)",
		exitBreakoutLow, exitBreakoutLowDate.Format("02.01.2006"))

	// Проверяем сигналы на вход с детальной информацией
	if currentPrice >= entryBreakoutHigh {
		// Сигнал на вход в длинную позицию
		stopLoss := currentPrice - (atr * ts.atrMultiplier)
		takeProfit := currentPrice + (2 * (currentPrice - stopLoss))
		positionSize := ts.calculatePositionSize(currentPrice, stopLoss)

		distance := currentPrice - entryBreakoutHigh
		distancePercent := (distance / entryBreakoutHigh) * 100

		reason := fmt.Sprintf("%s\n"+
			"Текущая цена: %.2f > %.2f (превышение на %.2f₽ / %.2f%%)\n"+
			"ATR: %.2f | Стоп-лосс: %.2f (%.2f%% от цены)\n"+
			"Тейк-профит: %.2f (риск:прибыль = 1:2)",
			entryHighInfo, currentPrice, entryBreakoutHigh, distance, distancePercent,
			atr, stopLoss, ((currentPrice-stopLoss)/currentPrice)*100, takeProfit)

		signals = append(signals, Signal{
			Instrument:   instrument,
			SignalType:   "entry_long",
			Price:        currentPrice,
			StopLoss:     stopLoss,
			TakeProfit:   takeProfit,
			PositionSize: positionSize,
			Reason:       reason,
			Timestamp:    time.Now(),
		})
	}

	if currentPrice <= entryBreakoutLow {
		// Сигнал на вход в короткую позицию
		stopLoss := currentPrice + (atr * ts.atrMultiplier)
		takeProfit := currentPrice - (2 * (stopLoss - currentPrice))
		positionSize := ts.calculatePositionSize(currentPrice, stopLoss)

		distance := entryBreakoutLow - currentPrice
		distancePercent := (distance / entryBreakoutLow) * 100

		reason := fmt.Sprintf("%s\n"+
			"Текущая цена: %.2f < %.2f (недобор на %.2f₽ / %.2f%%)\n"+
			"ATR: %.2f | Стоп-лосс: %.2f (%.2f%% от цены)\n"+
			"Тейк-профит: %.2f (риск:прибыль = 1:2)",
			entryLowInfo, currentPrice, entryBreakoutLow, distance, distancePercent,
			atr, stopLoss, ((stopLoss-currentPrice)/currentPrice)*100, takeProfit)

		signals = append(signals, Signal{
			Instrument:   instrument,
			SignalType:   "entry_short",
			Price:        currentPrice,
			StopLoss:     stopLoss,
			TakeProfit:   takeProfit,
			PositionSize: positionSize,
			Reason:       reason,
			Timestamp:    time.Now(),
		})
	}

	// Проверяем сигналы на выход с детальной информацией
	if currentPrice <= exitBreakoutLow {
		distance := exitBreakoutLow - currentPrice
		distancePercent := (distance / exitBreakoutLow) * 100

		reason := fmt.Sprintf("%s\n"+
			"Текущая цена: %.2f < %.2f (недобор на %.2f₽ / %.2f%%)\n"+
			"Время удержания уровня: %d дней",
			exitLowInfo, currentPrice, exitBreakoutLow, distance, distancePercent,
			int(currentDate.Sub(exitBreakoutLowDate).Hours()/24))

		signals = append(signals, Signal{
			Instrument: instrument,
			SignalType: "exit_long",
			Price:      currentPrice,
			Reason:     reason,
			Timestamp:  time.Now(),
		})
	}

	if currentPrice >= exitBreakoutHigh {
		distance := currentPrice - exitBreakoutHigh
		distancePercent := (distance / exitBreakoutHigh) * 100

		reason := fmt.Sprintf("%s\n"+
			"Текущая цена: %.2f > %.2f (превышение на %.2f₽ / %.2f%%)\n"+
			"Время удержания уровня: %d дней",
			exitHighInfo, currentPrice, exitBreakoutHigh, distance, distancePercent,
			int(currentDate.Sub(exitBreakoutHighDate).Hours()/24))

		signals = append(signals, Signal{
			Instrument: instrument,
			SignalType: "exit_short",
			Price:      currentPrice,
			Reason:     reason,
			Timestamp:  time.Now(),
		})
	}

	// Если сигналов нет, все равно возвращаем информацию о текущих уровнях
	if len(signals) == 0 {
		reason := fmt.Sprintf("Анализ инструмента %s\n\n"+
			"📊 ТЕКУЩИЕ УРОВНИ:\n"+
			"• Цена: %.2f\n"+
			"• ATR: %.2f (волатильность)\n\n"+
			"🎯 УРОВНИ ВХОДА:\n"+
			"• Для покупки: > %.2f (%s)\n"+
			"• Для продажи: < %.2f (%s)\n\n"+
			"🚪 УРОВНИ ВЫХОДА:\n"+
			"• Из покупки: < %.2f (%s)\n"+
			"• Из продажи: > %.2f (%s)\n\n"+
			"📈 РАССТОЯНИЕ ДО УРОВНЕЙ:\n"+
			"• До входа в лонг: %.2f (%.2f%%)\n"+
			"• До входа в шорт: %.2f (%.2f%%)\n"+
			"• До выхода из лонг: %.2f (%.2f%%)\n"+
			"• До выхода из шорт: %.2f (%.2f%%)\n\n"+
			"💡 РЕКОМЕНДАЦИЯ: Ожидание пробоя уровней",
			instrument,
			currentPrice,
			atr,
			entryBreakoutHigh, entryBreakoutHighDate.Format("02.01"),
			entryBreakoutLow, entryBreakoutLowDate.Format("02.01"),
			exitBreakoutLow, exitBreakoutLowDate.Format("02.01"),
			exitBreakoutHigh, exitBreakoutHighDate.Format("02.01"),
			entryBreakoutHigh-currentPrice, ((entryBreakoutHigh-currentPrice)/currentPrice)*100,
			currentPrice-entryBreakoutLow, ((currentPrice-entryBreakoutLow)/currentPrice)*100,
			currentPrice-exitBreakoutLow, ((currentPrice-exitBreakoutLow)/currentPrice)*100,
			exitBreakoutHigh-currentPrice, ((exitBreakoutHigh-currentPrice)/currentPrice)*100)

		signals = append(signals, Signal{
			Instrument: instrument,
			SignalType: "no_signal",
			Price:      currentPrice,
			Reason:     reason,
			Timestamp:  time.Now(),
		})
	}

	return signals, nil
}

// findBreakoutDate находит дату установки уровня прорыва
func (ts *TurtleStrategy) findBreakoutDate(dates []time.Time, prices []float64, breakoutLevel float64, days int) time.Time {
	if len(prices) < days {
		return time.Now()
	}

	// Ищем, когда был установлен этот уровень (когда цена впервые достигла его)
	recentPrices := prices[len(prices)-days:]
	recentDates := dates[len(dates)-days:]

	// Для максимума ищем последнюю дату, когда цена была на этом уровне
	var foundDate time.Time
	for i := len(recentPrices) - 1; i >= 0; i-- {
		if math.Abs(recentPrices[i]-breakoutLevel) < 0.001 { // Учитываем погрешность округления
			foundDate = recentDates[i]
			break
		}
	}

	if foundDate.IsZero() {
		// Если точной даты не нашли, возвращаем дату середины периода
		return recentDates[len(recentDates)/2]
	}

	return foundDate
}

// calculateBreakout рассчитывает уровень прорыва
func (ts *TurtleStrategy) calculateBreakout(prices []float64, days int, priceType string) float64 {
	if len(prices) < days {
		return 0
	}

	recentPrices := prices[len(prices)-days:]

	if priceType == "high" {
		return ts.mathUtils.MaxFloat(recentPrices)
	} else {
		return ts.mathUtils.MinFloat(recentPrices)
	}
}

// calculateATR рассчитывает Average True Range
func (ts *TurtleStrategy) calculateATR(highs, lows, closes []float64) float64 {
	if len(highs) < ts.atrPeriod+1 {
		return 0
	}

	trValues := make([]float64, 0)
	for i := 1; i < len(highs); i++ {
		hl := highs[i] - lows[i]
		hcp := math.Abs(highs[i] - closes[i-1])
		lcp := math.Abs(lows[i] - closes[i-1])

		tr := math.Max(hl, math.Max(hcp, lcp))
		trValues = append(trValues, tr)
	}

	// Простое скользящее среднее для TR
	if len(trValues) < ts.atrPeriod {
		return 0
	}

	var sum float64
	for i := len(trValues) - ts.atrPeriod; i < len(trValues); i++ {
		sum += trValues[i]
	}

	return sum / float64(ts.atrPeriod)
}

// calculatePositionSize рассчитывает размер позиции
func (ts *TurtleStrategy) calculatePositionSize(entryPrice, stopLoss float64) float64 {
	if stopLoss == 0 {
		return 0
	}

	riskPerUnit := math.Abs(entryPrice - stopLoss)
	if riskPerUnit == 0 {
		return 0
	}

	// Предполагаем счет 100000 рублей
	accountSize := 100000.0
	riskAmount := accountSize * ts.riskPerTrade

	return riskAmount / riskPerUnit
}

func max(values ...float64) float64 {
	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func min(values ...float64) float64 {
	minVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}
