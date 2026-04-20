package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/tradeai/bot/internal/config"
	"github.com/tradeai/bot/internal/logger"
)

// BinanceClient wraps the Binance API client using direct HTTP
type BinanceClient struct {
	client     *http.Client
	logger    *logger.CustomLogger
	cfg       *config.APIConfig
	baseURL   string
}

// NewBinanceClient creates a new Binance client
func NewBinanceClient(cfg *config.APIConfig, log *logger.CustomLogger) (*BinanceClient, error) {
	return &BinanceClient{
		client: &http.Client{
			Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second,
		},
		logger:  log,
		cfg:     cfg,
		baseURL: cfg.BaseURL,
	}, nil
}

// generateSignature creates the HMAC SHA256 signature
func (b *BinanceClient) generateSignature(queryString string) string {
	h := hmac.New(sha256.New, []byte(b.cfg.SecretKey))
	h.Write([]byte(queryString))
	return hex.EncodeToString(h.Sum(nil))
}

// request makes an authenticated API request
func (b *BinanceClient) request(method, endpoint string, params map[string]string) ([]byte, error) {
	// Build query string
	var queryString string
	for k, v := range params {
		if queryString != "" {
			queryString += "&"
		}
		queryString += k + "=" + url.QueryEscape(v)
	}

	// Add timestamp to params if not present
	if _, ok := params["timestamp"]; !ok {
		timestamp := strconv.FormatInt(time.Now().UnixMilli(), 10)
		if queryString != "" {
			queryString += "&"
		}
		queryString += "timestamp=" + timestamp
	}

	// Generate signature
	signature := b.generateSignature(queryString)
	queryString += "&signature=" + signature

	// Build URL
	apiURL := b.baseURL + endpoint + "?" + queryString

	// Create request
	req, err := http.NewRequest(method, apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Add("X-MBX-APIKEY", b.cfg.APIKey)
	req.Header.Add("Content-Type", "application/json")

	// Do request
	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API request failed: %s - %s", resp.Status, string(body))
	}

	return body, nil
}

// GetKlines returns klines/candlestick data
func (b *BinanceClient) GetKlines(symbol, interval string, limit int) ([]Kline, error) {
	params := map[string]string{
		"symbol":   symbol,
		"interval": interval,
		"limit":    strconv.Itoa(limit),
	}

	data, err := b.request("GET", "/api/v3/klines", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get klines: %w", err)
	}

	// Parse response - klines returns array of arrays
	var rawData [][]interface{}
	if err := json.Unmarshal(data, &rawData); err != nil {
		return nil, fmt.Errorf("failed to parse klines: %w", err)
	}

	klines := make([]Kline, len(rawData))
	for i, k := range rawData {
		openTime := int64(k[0].(float64))
		open := k[1].(string)
		high := k[2].(string)
		low := k[3].(string)
		close := k[4].(string)
		volume := k[5].(string)
		closeTime := int64(k[6].(float64))

		klines[i] = Kline{
			OpenTime:     openTime,
			Open:        open,
			High:        high,
			Low:         low,
			Close:       close,
			Volume:      volume,
			CloseTime:   closeTime,
		}
	}

	return klines, nil
}

// GetCurrentPrice returns the current price for a symbol
func (b *BinanceClient) GetCurrentPrice(symbol string) (float64, error) {
	params := map[string]string{
		"symbol": symbol,
	}

	data, err := b.request("GET", "/api/v3/ticker/price", params)
	if err != nil {
		return 0, fmt.Errorf("failed to get current price: %w", err)
	}

	var resp struct {
		Price string `json:"price"`
	}

	if err := json.Unmarshal(data, &resp); err != nil {
		return 0, err
	}

	price, err := strconv.ParseFloat(resp.Price, 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}

// PlaceOrder places a market order
func (b *BinanceClient) PlaceOrder(symbol, side, orderType string, quantity float64) (*OrderResponse, error) {
	params := map[string]string{
		"symbol":          symbol,
		"side":           side,
		"type":           orderType,
		"quantity":       strconv.FormatFloat(quantity, 'f', 8, 64),
		"timeInForce":    "GTC",
	}

	data, err := b.request("POST", "/api/v3/order", params)
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	var resp OrderResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Ping checks the connection to the API
func (b *BinanceClient) Ping() error {
	params := map[string]string{}

	_, err := b.request("GET", "/api/v3/ping", params)
	return err
}

// GetOrderStatus returns the status of an order
func (b *BinanceClient) GetOrderStatus(symbol string, orderID int64) (*OrderResponse, error) {
	params := map[string]string{
		"symbol":  symbol,
		"orderId": strconv.FormatInt(orderID, 10),
	}

	data, err := b.request("GET", "/api/v3/order", params)
	if err != nil {
		return nil, fmt.Errorf("failed to get order status: %w", err)
	}


	var resp OrderResponse
	if err := json.Unmarshal(data, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// Kline represents a candlestick
type Kline struct {
	OpenTime  int64
	Open      string
	High      string
	Low       string
	Close     string
	Volume    string
	CloseTime int64
}

// OrderResponse represents an order response
type OrderResponse struct {
	OrderID        int64   `json:"orderId"`
	Symbol         string  `json:"symbol"`
	Side           string  `json:"side"`
	Status         string  `json:"status"`
	ClientOrderID  string  `json:"clientOrderId"`
	Price          string  `json:"price"`
	OrigQty        string  `json:"origQty"`
	ExecutedQty    string  `json:"executedQty"`
	CumulativeQuoteQty string `json:"cummulativeQuoteQty"`
}

// AccountResponse represents account info
type AccountResponse struct {
	Balances []Balance `json:"balances"`
}

// Balance represents a wallet balance
type Balance struct {
	Asset  string `json:"asset"`
	Free   string `json:"free"`
	Locked string `json:"locked"`
}
