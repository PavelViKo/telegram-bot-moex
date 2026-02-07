package analysis

import (
	"context"
	"fmt"
	"math"
	"time"

	"telegram-bot-moex/internal/api"
)

// MACrossoverStrategy реализует стратегию пересечения скользящих средних
type MACrossoverStrategy struct {
	apiClient  *api.APIClient
	config     MACrossoverConfig
	indicators *TechnicalIndicators
	mathUtils  *MathUtils // ДОБАВЛЯЕМ
}

// MACrossoverConfig конфигурация стратегии
type MACrossoverConfig struct {
	Timeframe             string
	FastPeriod            int
	SlowPeriod            int
	SignalPeriod          int
	UseEMA                bool
	UseVolumeConfirmation bool
	MinVolumeMultiplier   float64
	RiskPerTrade          float64
	StopLossATRMultiplier float64
	TakeProfitRatio       float64

	CrossoverTypes struct {
		GoldenCross         bool
		DeathCross          bool
		RequireConfirmation int
	}

	Filters struct {
		TrendFilter   string
		RSIFilter     bool
		RSIOverbought int
		RSIOversold   int
	}
}

// TechnicalIndicators расчетные индикаторы
type TechnicalIndicators struct {
	FastMA   []float64
	SlowMA   []float64
	SignalMA []float64
	ATR      []float64
	RSI      []float64
	Volumes  []float64
	Prices   []float64
	Dates    []time.Time
}

// NewMACrossoverStrategy создает новую стратегию MA Crossover
func NewMACrossoverStrategy(apiClient *api.APIClient, config MACrossoverConfig) *MACrossoverStrategy {
	return &MACrossoverStrategy{
		apiClient: apiClient,
		config:    config,
		mathUtils: &MathUtils{}, // ДОБАВЛЯЕМ
	}
}

// Analyze анализирует инструмент по стратегии MA Crossover
func (m *MACrossoverStrategy) Analyze(ctx context.Context, instrument string) ([]Signal, error) {
	// Получаем свечи за нужный период
	to := time.Now().Format("2006-01-02")
	from := time.Now().AddDate(0, 0, -m.config.SlowPeriod*3).Format("2006-01-02")

	candles, err := m.apiClient.GetCandles(ctx, instrument, m.config.Timeframe, from, to)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения свечей: %w", err)
	}

	// Преобразуем свечи в массивы
	prices, highs, lows, closes, volumes, dates := m.parseCandles(candles)

	if len(prices) < m.config.SlowPeriod+20 {
		return nil, fmt.Errorf("недостаточно данных для анализа")
	}

	// Рассчитываем индикаторы
	m.calculateIndicators(prices, highs, lows, closes, volumes, dates)

	// Анализируем сигналы
	signals, err := m.analyzeSignals(instrument, prices, closes, volumes, dates)
	if err != nil {
		return nil, err
	}

	return signals, nil
}

// parseCandles преобразует свечи в массивы
func (m *MACrossoverStrategy) parseCandles(candles []map[string]interface{}) (
	prices, highs, lows, closes, volumes []float64,
	dates []time.Time,
) {
	for _, candle := range candles {
		// Цена закрытия
		if closeVal, ok := candle["close"].(float64); ok {
			closes = append(closes, closeVal)
			prices = append(prices, closeVal)
		} else if closeVal, ok := candle["close"].(int); ok {
			closes = append(closes, float64(closeVal))
			prices = append(prices, float64(closeVal))
		}

		// Максимум
		if highVal, ok := candle["high"].(float64); ok {
			highs = append(highs, highVal)
		} else if highVal, ok := candle["high"].(int); ok {
			highs = append(highs, float64(highVal))
		}

		// Минимум
		if lowVal, ok := candle["low"].(float64); ok {
			lows = append(lows, lowVal)
		} else if lowVal, ok := candle["low"].(int); ok {
			lows = append(lows, float64(lowVal))
		}

		// Объем
		if volumeVal, ok := candle["volume"].(float64); ok {
			volumes = append(volumes, volumeVal)
		} else if volumeVal, ok := candle["volume"].(int); ok {
			volumes = append(volumes, float64(volumeVal))
		} else {
			volumes = append(volumes, 0)
		}

		// Дата
		if beginVal, ok := candle["begin"].(string); ok {
			if t, err := time.Parse("2006-01-02 15:04:05", beginVal); err == nil {
				dates = append(dates, t)
			} else if t, err := time.Parse("2006-01-02", beginVal); err == nil {
				dates = append(dates, t)
			} else {
				dates = append(dates, time.Now())
			}
		} else {
			dates = append(dates, time.Now())
		}
	}

	return prices, highs, lows, closes, volumes, dates
}

// calculateIndicators рассчитывает все технические индикаторы
func (m *MACrossoverStrategy) calculateIndicators(
	prices, highs, lows, closes, volumes []float64,
	dates []time.Time,
) {
	m.indicators = &TechnicalIndicators{
		Prices:  prices,
		Volumes: volumes,
		Dates:   dates,
	}

	// Рассчитываем скользящие средние
	if m.config.UseEMA {
		m.indicators.FastMA = m.calculateEMA(closes, m.config.FastPeriod)
		m.indicators.SlowMA = m.calculateEMA(closes, m.config.SlowPeriod)
		if m.config.SignalPeriod > 0 {
			m.indicators.SignalMA = m.calculateEMA(closes, m.config.SignalPeriod)
		}
	} else {
		m.indicators.FastMA = m.calculateSMA(closes, m.config.FastPeriod)
		m.indicators.SlowMA = m.calculateSMA(closes, m.config.SlowPeriod)
		if m.config.SignalPeriod > 0 {
			m.indicators.SignalMA = m.calculateSMA(closes, m.config.SignalPeriod)
		}
	}

	// Рассчитываем ATR для стоп-лосса
	m.indicators.ATR = m.calculateATR(highs, lows, closes, 14)

	// Рассчитываем RSI если нужен фильтр
	if m.config.Filters.RSIFilter {
		m.indicators.RSI = m.calculateRSI(closes, 14)
	}
}

// calculateSMA рассчитывает простую скользящую среднюю
func (m *MACrossoverStrategy) calculateSMA(data []float64, period int) []float64 {
	if len(data) < period {
		return make([]float64, len(data))
	}

	sma := make([]float64, len(data))
	for i := period - 1; i < len(data); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += data[j]
		}
		sma[i] = sum / float64(period)
	}

	// Заполняем начало NaN значениями
	for i := 0; i < period-1; i++ {
		sma[i] = data[i]
	}

	return sma
}

// calculateEMA рассчитывает экспоненциальную скользящую среднюю
func (m *MACrossoverStrategy) calculateEMA(data []float64, period int) []float64 {
	if len(data) < period {
		return make([]float64, len(data))
	}

	ema := make([]float64, len(data))
	multiplier := 2.0 / (float64(period) + 1.0)

	// Первое значение EMA = SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += data[i]
	}
	ema[period-1] = sum / float64(period)

	// Расчет остальных EMA
	for i := period; i < len(data); i++ {
		ema[i] = (data[i]-ema[i-1])*multiplier + ema[i-1]
	}

	// Заполняем начало
	for i := 0; i < period-1; i++ {
		ema[i] = data[i]
	}

	return ema
}

// calculateATR рассчитывает Average True Range
func (m *MACrossoverStrategy) calculateATR(highs, lows, closes []float64, period int) []float64 {
	if len(highs) < period+1 {
		return make([]float64, len(highs))
	}

	tr := make([]float64, len(highs))
	atr := make([]float64, len(highs))

	// Расчет True Range
	for i := 1; i < len(highs); i++ {
		hl := highs[i] - lows[i]
		hcp := math.Abs(highs[i] - closes[i-1])
		lcp := math.Abs(lows[i] - closes[i-1])
		tr[i] = math.Max(hl, math.Max(hcp, lcp))
	}

	// Первое значение ATR = среднее первых period TR
	sum := 0.0
	for i := 1; i <= period; i++ {
		sum += tr[i]
	}
	atr[period] = sum / float64(period)

	// Расчет остальных ATR по формуле Уайлдера
	for i := period + 1; i < len(highs); i++ {
		atr[i] = (atr[i-1]*float64(period-1) + tr[i]) / float64(period)
	}

	// Заполняем начало
	for i := 0; i <= period; i++ {
		atr[i] = tr[i]
	}

	return atr
}

// calculateRSI рассчитывает Relative Strength Index
func (m *MACrossoverStrategy) calculateRSI(closes []float64, period int) []float64 {
	if len(closes) < period+1 {
		return make([]float64, len(closes))
	}

	rsi := make([]float64, len(closes))
	gains := make([]float64, len(closes))
	losses := make([]float64, len(closes))

	// Расчет прибылей и убытков
	for i := 1; i < len(closes); i++ {
		change := closes[i] - closes[i-1]
		if change > 0 {
			gains[i] = change
			losses[i] = 0
		} else {
			gains[i] = 0
			losses[i] = -change
		}
	}

	// Первые средние значения
	avgGain := 0.0
	avgLoss := 0.0
	for i := 1; i <= period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// Первое значение RSI
	if avgLoss == 0 {
		rsi[period] = 100
	} else {
		rs := avgGain / avgLoss
		rsi[period] = 100 - (100 / (1 + rs))
	}

	// Расчет остальных RSI
	for i := period + 1; i < len(closes); i++ {
		avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)

		if avgLoss == 0 {
			rsi[i] = 100
		} else {
			rs := avgGain / avgLoss
			rsi[i] = 100 - (100 / (1 + rs))
		}
	}

	// Заполняем начало
	for i := 0; i < period; i++ {
		rsi[i] = 50
	}

	return rsi
}

// analyzeSignals анализирует и формирует торговые сигналы
func (m *MACrossoverStrategy) analyzeSignals(
	instrument string,
	prices, closes, volumes []float64,
	dates []time.Time,
) ([]Signal, error) {
	var signals []Signal

	if m.indicators == nil || len(m.indicators.FastMA) == 0 || len(m.indicators.SlowMA) == 0 {
		return signals, nil
	}

	currentIdx := len(prices) - 1
	currentPrice := closes[currentIdx]
	currentDate := dates[currentIdx]
	currentVolume := volumes[currentIdx]

	// Проверяем фильтры
	if !m.checkFilters(currentIdx) {
		return signals, nil
	}

	// Анализируем пересечения
	crossSignals := m.analyzeCrossovers(instrument, currentIdx, currentPrice, currentDate)
	signals = append(signals, crossSignals...)

	// Проверяем подтверждение объема
	if m.config.UseVolumeConfirmation && len(signals) > 0 {
		signals = m.filterByVolume(signals, currentVolume, volumes, currentIdx)
	}

	// Если есть сигналы, добавляем расчеты стоп-лосса и тейк-профита
	for i := range signals {
		if signals[i].SignalType == "entry_long" || signals[i].SignalType == "entry_short" {
			m.calculateRiskManagement(&signals[i], currentIdx)
		}
	}

	return signals, nil
}

// checkFilters проверяет все фильтры
func (m *MACrossoverStrategy) checkFilters(currentIdx int) bool {
	// Фильтр по тренду
	if !m.checkTrendFilter(currentIdx) {
		return false
	}

	// Фильтр по RSI
	if !m.checkRSIFilter(currentIdx) {
		return false
	}

	return true
}

// checkTrendFilter проверяет фильтр по тренду
func (m *MACrossoverStrategy) checkTrendFilter(currentIdx int) bool {
	if m.config.Filters.TrendFilter == "none" {
		return true
	}

	//price := m.indicators.Prices[currentIdx]

	switch m.config.Filters.TrendFilter {
	case "sma50":
		if len(m.indicators.Prices) >= 50 {
			sma50 := m.calculateSMA(m.indicators.Prices, 50)
			if currentIdx < len(sma50) {
				// Для лонга: цена выше SMA50 (восходящий тренд)
				// Для шорта: цена ниже SMA50 (нисходящий тренд)
				// Пока пропускаем, т.к. не знаем тип сигнала
				return true
			}
		}
	case "sma200":
		if len(m.indicators.Prices) >= 200 {
			sma200 := m.calculateSMA(m.indicators.Prices, 200)
			if currentIdx < len(sma200) {
				return true
			}
		}
	}

	return true
}

// checkRSIFilter проверяет фильтр по RSI
func (m *MACrossoverStrategy) checkRSIFilter(currentIdx int) bool {
	if !m.config.Filters.RSIFilter || len(m.indicators.RSI) == 0 {
		return true
	}

	if currentIdx >= len(m.indicators.RSI) {
		return true
	}

	//rsi := m.indicators.RSI[currentIdx]

	// Для входа в лонг: RSI не должен быть в зоне перекупленности
	// Для входа в шорт: RSI не должен быть в зоне перепроданности
	// Пока пропускаем, конкретная проверка будет в analyzeCrossovers
	return true
}

// analyzeCrossovers анализирует пересечения скользящих средних
func (m *MACrossoverStrategy) analyzeCrossovers(
	instrument string,
	currentIdx int,
	currentPrice float64,
	currentDate time.Time,
) []Signal {
	var signals []Signal

	fastMA := m.indicators.FastMA
	slowMA := m.indicators.SlowMA

	if currentIdx < 1 || len(fastMA) < 2 || len(slowMA) < 2 {
		return signals
	}

	// Текущие значения
	fastNow := fastMA[currentIdx]
	fastPrev := fastMA[currentIdx-1]
	slowNow := slowMA[currentIdx]
	slowPrev := slowMA[currentIdx-1]

	// Проверяем "Золотое пересечение" (быстрая пересекает медленную снизу вверх)
	if m.config.CrossoverTypes.GoldenCross &&
		fastPrev <= slowPrev && // Быстрая была ниже или равна медленной
		fastNow > slowNow { // Быстрая стала выше медленной

		// Проверяем подтверждение
		if m.checkConfirmation(currentIdx, "golden") {
			// Проверяем RSI фильтр для лонга
			if !m.config.Filters.RSIFilter ||
				(len(m.indicators.RSI) > currentIdx &&
					m.indicators.RSI[currentIdx] < float64(m.config.Filters.RSIOverbought)) {

				distance := ((fastNow - slowNow) / slowNow) * 100
				reason := fmt.Sprintf("Золотое пересечение: SMA%d (%.2f) пересекает SMA%d (%.2f) снизу вверх\n"+
					"Текущая цена: %.2f | Расстояние между MA: %.2f%%\n"+
					"Сигнал подтвержден %d свечами",
					m.config.FastPeriod, fastNow,
					m.config.SlowPeriod, slowNow,
					currentPrice, distance,
					m.config.CrossoverTypes.RequireConfirmation)

				signals = append(signals, Signal{
					Instrument: instrument,
					SignalType: "entry_long",
					Price:      currentPrice,
					Reason:     reason,
					Timestamp:  currentDate,
				})
			}
		}
	}

	// Проверяем "Мертвое пересечение" (быстрая пересекает медленную сверху вниз)
	if m.config.CrossoverTypes.DeathCross &&
		fastPrev >= slowPrev && // Быстрая была выше или равна медленной
		fastNow < slowNow { // Быстрая стала ниже медленной

		// Проверяем подтверждение
		if m.checkConfirmation(currentIdx, "death") {
			// Проверяем RSI фильтр для шорта
			if !m.config.Filters.RSIFilter ||
				(len(m.indicators.RSI) > currentIdx &&
					m.indicators.RSI[currentIdx] > float64(m.config.Filters.RSIOversold)) {

				distance := ((slowNow - fastNow) / fastNow) * 100
				reason := fmt.Sprintf("Мертвое пересечение: SMA%d (%.2f) пересекает SMA%d (%.2f) сверху вниз\n"+
					"Текущая цена: %.2f | Расстояние между MA: %.2f%%\n"+
					"Сигнал подтвержден %d свечами",
					m.config.FastPeriod, fastNow,
					m.config.SlowPeriod, slowNow,
					currentPrice, distance,
					m.config.CrossoverTypes.RequireConfirmation)

				signals = append(signals, Signal{
					Instrument: instrument,
					SignalType: "entry_short",
					Price:      currentPrice,
					Reason:     reason,
					Timestamp:  currentDate,
				})
			}
		}
	}

	return signals
}

// checkConfirmation проверяет подтверждение сигнала
func (m *MACrossoverStrategy) checkConfirmation(currentIdx int, crossType string) bool {
	if m.config.CrossoverTypes.RequireConfirmation <= 0 {
		return true
	}

	fastMA := m.indicators.FastMA
	slowMA := m.indicators.SlowMA

	if currentIdx < m.config.CrossoverTypes.RequireConfirmation {
		return false
	}

	// Проверяем последние N свечей
	for i := 0; i < m.config.CrossoverTypes.RequireConfirmation; i++ {
		idx := currentIdx - i
		if idx < 0 || idx >= len(fastMA) || idx >= len(slowMA) {
			return false
		}

		switch crossType {
		case "golden":
			if fastMA[idx] <= slowMA[idx] {
				return false
			}
		case "death":
			if fastMA[idx] >= slowMA[idx] {
				return false
			}
		}
	}

	return true
}

// filterByVolume фильтрует сигналы по объему
func (m *MACrossoverStrategy) filterByVolume(
	signals []Signal,
	currentVolume float64,
	volumes []float64,
	currentIdx int,
) []Signal {
	if len(volumes) < 20 {
		return signals
	}

	// Рассчитываем средний объем за последние 20 периодов
	startIdx := m.mathUtils.MaxInt(0, currentIdx-19) // ИСПОЛЬЗУЕМ mathUtils

	var avgVolume float64
	count := 0

	for i := startIdx; i <= currentIdx; i++ {
		if i < len(volumes) {
			avgVolume += volumes[i]
			count++
		}
	}

	if count == 0 {
		return signals
	}

	avgVolume /= float64(count)

	// Фильтруем сигналы
	var filteredSignals []Signal
	for _, signal := range signals {
		volumeRatio := currentVolume / avgVolume

		if volumeRatio >= m.config.MinVolumeMultiplier {
			// Добавляем информацию об объеме в причину
			signal.Reason += fmt.Sprintf("\n📊 Подтверждение объемом: %.0f (%.1fx от среднего)",
				currentVolume, volumeRatio)
			filteredSignals = append(filteredSignals, signal)
		}
	}

	return filteredSignals
}

// calculateRiskManagement рассчитывает стоп-лосс и тейк-профит
func (m *MACrossoverStrategy) calculateRiskManagement(signal *Signal, currentIdx int) {
	if len(m.indicators.ATR) == 0 || currentIdx >= len(m.indicators.ATR) {
		return
	}

	currentATR := m.indicators.ATR[currentIdx]

	switch signal.SignalType {
	case "entry_long":
		// Стоп-лосс ниже текущей цены на ATR * множитель
		stopLoss := signal.Price - (currentATR * m.config.StopLossATRMultiplier)
		takeProfit := signal.Price + (currentATR * m.config.StopLossATRMultiplier * m.config.TakeProfitRatio)

		signal.StopLoss = stopLoss
		signal.TakeProfit = takeProfit

		// Расчет размера позиции
		riskAmount := 100000.0 * m.config.RiskPerTrade // Предполагаем счет 100k
		riskPerShare := signal.Price - stopLoss
		if riskPerShare > 0 {
			signal.PositionSize = riskAmount / riskPerShare
		}

		signal.Reason += fmt.Sprintf("\n\n🎯 УПРАВЛЕНИЕ РИСКАМИ:\n"+
			"• Стоп-лосс: %.2f (%.1f%%)\n"+
			"• Тейк-профит: %.2f (риск:прибыль = 1:%.1f)\n"+
			"• Размер позиции: %.0f шт.\n"+
			"• ATR: %.2f (текущая волатильность)",
			stopLoss, ((signal.Price-stopLoss)/signal.Price)*100,
			takeProfit, m.config.TakeProfitRatio,
			signal.PositionSize, currentATR)

	case "entry_short":
		// Стоп-лосс выше текущей цены на ATR * множитель
		stopLoss := signal.Price + (currentATR * m.config.StopLossATRMultiplier)
		takeProfit := signal.Price - (currentATR * m.config.StopLossATRMultiplier * m.config.TakeProfitRatio)

		signal.StopLoss = stopLoss
		signal.TakeProfit = takeProfit

		// Расчет размера позиции
		riskAmount := 100000.0 * m.config.RiskPerTrade
		riskPerShare := stopLoss - signal.Price
		if riskPerShare > 0 {
			signal.PositionSize = riskAmount / riskPerShare
		}

		signal.Reason += fmt.Sprintf("\n\n🎯 УПРАВЛЕНИЕ РИСКАМИ:\n"+
			"• Стоп-лосс: %.2f (%.1f%%)\n"+
			"• Тейк-профит: %.2f (риск:прибыль = 1:%.1f)\n"+
			"• Размер позиции: %.0f шт.\n"+
			"• ATR: %.2f (текущая волатильность)",
			stopLoss, ((stopLoss-signal.Price)/signal.Price)*100,
			takeProfit, m.config.TakeProfitRatio,
			signal.PositionSize, currentATR)
	}
}
