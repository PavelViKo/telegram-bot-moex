package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// APIClient клиент для работы с API MOEX Fetcher
type APIClient struct {
	baseURL    string
	token      string
	httpClient *http.Client
}

// NewAPIClient создает новый API клиент
func NewAPIClient(baseURL, token string, timeout time.Duration) *APIClient {
	return &APIClient{
		baseURL: baseURL,
		token:   token,
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  true,
				MaxIdleConnsPerHost: 10,
			},
		},
	}
}

// HealthCheck проверяет доступность API
func (c *APIClient) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	return c.doRequest(ctx, "GET", "/health", nil)
}

// GetStats получает статистику
func (c *APIClient) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return c.doRequest(ctx, "GET", "/api/stats", nil)
}

// GetInstruments получает список инструментов
func (c *APIClient) GetInstruments(ctx context.Context) ([]string, error) {
	data, err := c.doRequest(ctx, "GET", "/api/instruments", nil)
	if err != nil {
		return nil, err
	}

	if instruments, ok := data["instruments"].([]interface{}); ok {
		var result []string
		for _, inst := range instruments {
			if str, ok := inst.(string); ok {
				result = append(result, str)
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("неверный формат ответа")
}

// GetCandles получает свечи
func (c *APIClient) GetCandles(ctx context.Context, instrument, timeframe, from, to string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("/api/candles?instrument=%s&timeframe=%s", instrument, timeframe)
	if from != "" {
		url += fmt.Sprintf("&from=%s", from)
	}
	if to != "" {
		url += fmt.Sprintf("&to=%s", to)
	}

	data, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if candles, ok := data["candles"].([]interface{}); ok {
		var result []map[string]interface{}
		for _, candle := range candles {
			if cmap, ok := candle.(map[string]interface{}); ok {
				result = append(result, cmap)
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("неверный формат свечей")
}

// GetInstrumentInfo получает информацию об инструменте
func (c *APIClient) GetInstrumentInfo(ctx context.Context, instrument string) (map[string]interface{}, error) {
	url := fmt.Sprintf("/api/instruments/%s", instrument)
	return c.doRequest(ctx, "GET", url, nil)
}

// GetInstrumentTimeframes получает таймфреймы для инструмента
func (c *APIClient) GetInstrumentTimeframes(ctx context.Context, instrument string) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("/api/instruments/%s/timeframes", instrument)
	data, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if timeframes, ok := data["timeframes"].([]interface{}); ok {
		var result []map[string]interface{}
		for _, tf := range timeframes {
			if tmap, ok := tf.(map[string]interface{}); ok {
				result = append(result, tmap)
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("неверный формат таймфреймов")
}

// GetTables получает список таблиц
func (c *APIClient) GetTables(ctx context.Context) ([]map[string]interface{}, error) {
	data, err := c.doRequest(ctx, "GET", "/api/tables", nil)
	if err != nil {
		return nil, err
	}

	if tables, ok := data["tables"].([]interface{}); ok {
		var result []map[string]interface{}
		for _, table := range tables {
			if tmap, ok := table.(map[string]interface{}); ok {
				result = append(result, tmap)
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("неверный формат таблиц")
}

// GetTimeframes получает доступные таймфреймы
func (c *APIClient) GetTimeframes(ctx context.Context) ([]map[string]interface{}, error) {
	data, err := c.doRequest(ctx, "GET", "/api/timeframes", nil)
	if err != nil {
		return nil, err
	}

	if timeframes, ok := data["timeframes"].([]interface{}); ok {
		var result []map[string]interface{}
		for _, tf := range timeframes {
			if tmap, ok := tf.(map[string]interface{}); ok {
				result = append(result, tmap)
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("неверный формат таймфреймов")
}

// TriggerFetch запускает загрузку данных
func (c *APIClient) TriggerFetch(ctx context.Context) (map[string]interface{}, error) {
	body := map[string]interface{}{}                    //добавил тело запроса вместо nil
	return c.doRequest(ctx, "POST", "/api/fetch", body) //
}

// RefreshInstruments обновляет список инструментов
func (c *APIClient) RefreshInstruments(ctx context.Context) (map[string]interface{}, error) {
	return c.doRequest(ctx, "POST", "/api/refresh-instruments", nil)
}

// AddInstrument добавляет инструмент
func (c *APIClient) AddInstrument(ctx context.Context, instrument string) (map[string]interface{}, error) {
	body := map[string]interface{}{
		"instrument": instrument,
		"source":     "telegram_bot",
	}

	return c.doRequest(ctx, "POST", "/api/instruments/add", body)
}

// RemoveInstrument удаляет инструмент
func (c *APIClient) RemoveInstrument(ctx context.Context, instrument string) error {
	url := fmt.Sprintf("/api/instruments/%s", instrument)
	_, err := c.doRequest(ctx, "DELETE", url, nil)
	return err
}

// CleanupTables очищает старые таблицы
func (c *APIClient) CleanupTables(ctx context.Context, days int) (map[string]interface{}, error) {
	url := fmt.Sprintf("/api/tables/cleanup?inactive_days=%d", days)
	return c.doRequest(ctx, "POST", url, nil)
}

// doRequest выполняет HTTP запрос
func (c *APIClient) doRequest(ctx context.Context, method, path string, body interface{}) (map[string]interface{}, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("ошибка маршалинга JSON: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := c.baseURL + path
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Добавляем заголовки
	req.Header.Set("Content-Type", "application/json")
	if c.token != "" {
		req.Header.Set("X-API-Key", c.token)
	}

	// Выполняем запрос
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	// Читаем ответ
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения ответа: %w", err)
	}

	// Проверяем статус код
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("ошибка API: %s - %s", resp.Status, string(respBody))
	}

	// Парсим JSON ответ
	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		// Если не JSON, возвращаем текстовый ответ
		return map[string]interface{}{
			"response": string(respBody),
			"status":   resp.StatusCode,
		}, nil
	}

	return result, nil
}

// GetTableData получает данные из таблицы
func (c *APIClient) GetTableData(ctx context.Context, instrument, timeframe, from, to string, limit int) ([]map[string]interface{}, error) {
	url := fmt.Sprintf("/api/tables/%s/%s", instrument, timeframe)

	// Добавляем параметры запроса
	params := []string{}
	if from != "" {
		params = append(params, "from="+from)
	}
	if to != "" {
		params = append(params, "to="+to)
	}
	if limit > 0 {
		params = append(params, "limit="+strconv.Itoa(limit))
	}

	if len(params) > 0 {
		url += "?" + stringJoin(params, "&")
	}

	data, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	if candles, ok := data["candles"].([]interface{}); ok {
		var result []map[string]interface{}
		for _, candle := range candles {
			if cmap, ok := candle.(map[string]interface{}); ok {
				result = append(result, cmap)
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("неверный формат данных таблицы")
}

// stringJoin вспомогательная функция для объединения строк
func stringJoin(elems []string, sep string) string {
	switch len(elems) {
	case 0:
		return ""
	case 1:
		return elems[0]
	}

	n := len(sep) * (len(elems) - 1)
	for i := 0; i < len(elems); i++ {
		n += len(elems[i])
	}

	var b strings.Builder
	b.Grow(n)
	b.WriteString(elems[0])
	for _, s := range elems[1:] {
		b.WriteString(sep)
		b.WriteString(s)
	}
	return b.String()
}
