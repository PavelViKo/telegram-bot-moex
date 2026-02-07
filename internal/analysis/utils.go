package analysis

import "math"

// MathUtils содержит математические утилиты для стратегий
type MathUtils struct{}

// MaxFloat находит максимальное значение в слайсе float64
func (mu *MathUtils) MaxFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

// MinFloat находит минимальное значение в слайсе float64
func (mu *MathUtils) MinFloat(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

// MaxInt находит максимальное из двух int
func (mu *MathUtils) MaxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// CalculateATR рассчитывает Average True Range
func (mu *MathUtils) CalculateATR(highs, lows, closes []float64, period int) []float64 {
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

		// Находим максимум из трех значений
		maxVal := hl
		if hcp > maxVal {
			maxVal = hcp
		}
		if lcp > maxVal {
			maxVal = lcp
		}
		tr[i] = maxVal
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
