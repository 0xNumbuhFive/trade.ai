package trader

import (
	"fmt"
	"sync"
	"time"
	"github.com/tradeai/bot/internal/api"
	"github.com/tradeai/bot/internal/config"
	"github.com/tradeai/bot/internal/logger"
	"github.com/tradeai/bot/internal/strategy"
)

// Position represents a trading position
type Position struct {
	Symbol      string
	Side        strategy.Signal
	EntryPrice  float64
	Quantity    float64
	StopLoss    float64
	TakeProfit  float64
	OrderID     int64
	OpenedAt    time.Time
}

// Trader manages order execution
type Trader struct {
	client     *api.BinanceClient
	logger    *logger.CustomLogger
	cfg       *config.TradingConfig
	position  *Position
	mu        sync.RWMutex
	started   bool
}

// NewTrader creates a new Trader instance
func NewTrader(client *api.BinanceClient, cfg *config.TradingConfig, log *logger.CustomLogger) *Trader {
	return &Trader{
		client:    client,
		logger:    log,
		cfg:       cfg,
		position:  nil,
		started:   false,
	}
}

// HasPosition returns true if there is an open position
func (t *Trader) HasPosition() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.position != nil
}

// GetPosition returns the current position
func (t *Trader) GetPosition() *Position {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.position
}

// OpenPosition opens a new position
func (t *Trader) OpenPosition(side strategy.Signal, price float64) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.position != nil {
		return fmt.Errorf("position already exists")
	}

	// Place market order
	sideStr := "BUY"
	if side == strategy.SignalSell {
		sideStr = "SELL"
	}

	resp, err := t.client.PlaceOrder(t.cfg.Symbol, sideStr, "MARKET", t.cfg.Quantity)
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	// Get current price if market order filled
	entryPrice := price
	if resp.Price != "" {
		fmt.Sscanf(resp.Price, "%f", &entryPrice)
	}

	// Calculate stop loss and take profit
	var stopLoss, takeProfit float64
	if side == strategy.SignalBuy {
		stopLoss = entryPrice * (1 - t.cfg.StopLossPct/100)
		takeProfit = entryPrice * (1 + t.cfg.TakeProfitPct/100)
	} else {
		stopLoss = entryPrice * (1 + t.cfg.StopLossPct/100)
		takeProfit = entryPrice * (1 - t.cfg.TakeProfitPct/100)
	}

	t.position = &Position{
		Symbol:     t.cfg.Symbol,
		Side:       side,
		EntryPrice: entryPrice,
		Quantity:   t.cfg.Quantity,
		StopLoss:   stopLoss,
		TakeProfit: takeProfit,
		OrderID:    resp.OrderID,
		OpenedAt:   time.Now(),
	}

	t.logger.LogOrder(sideStr, t.cfg.Symbol, t.cfg.Quantity, entryPrice, "opened")
	return nil
}

// ClosePosition closes the current position
func (t *Trader) ClosePosition(reason string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.position == nil {
		return fmt.Errorf("no position to close")
	}

	// Determine closing side
	sideStr := "SELL"
	if t.position.Side == strategy.SignalSell {
		sideStr = "BUY"
	}

	// Place closing order
	resp, err := t.client.PlaceOrder(t.cfg.Symbol, sideStr, "MARKET", t.cfg.Quantity)
	if err != nil {
		return fmt.Errorf("failed to close position: %w", err)
	}

	// Get closing price
	var closingPrice float64
	if resp.Price != "" {
		fmt.Sscanf(resp.Price, "%f", &closingPrice)
	}

	// Calculate profit/loss
	var pnl float64
	if t.position.Side == strategy.SignalBuy {
		pnl = (closingPrice - t.position.EntryPrice) * t.position.Quantity
	} else {
		pnl = (t.position.EntryPrice - closingPrice) * t.position.Quantity
	}

	t.logger.LogOrder(sideStr, t.cfg.Symbol, t.cfg.Quantity, closingPrice, fmt.Sprintf("closed - %s", reason))
	t.logger.LogInfo(fmt.Sprintf("Position closed: PnL = %.2f USDT", pnl), nil)

	// Clear position
	t.position = nil

	return nil
}

// CheckStopLoss checks if stop loss is triggered
func (t *Trader) CheckStopLoss(currentPrice float64) bool {
	if !t.HasPosition() {
		return false
	}

	pos := t.GetPosition()
	if pos.Side == strategy.SignalBuy {
		return currentPrice <= pos.StopLoss
	}
	return currentPrice >= pos.StopLoss
}

// CheckTakeProfit checks if take profit is triggered
func (t *Trader) CheckTakeProfit(currentPrice float64) bool {
	if !t.HasPosition() {
		return false
	}

	pos := t.GetPosition()
	if pos.Side == strategy.SignalBuy {
		return currentPrice >= pos.TakeProfit
	}
	return currentPrice <= pos.TakeProfit
}

// Start starts the trader
func (t *Trader) Start() {
	t.mu.Lock()
	t.started = true
	t.mu.Unlock()
}

// Stop stops the trader
func (t *Trader) Stop() {
	t.mu.Lock()
	t.started = false
	t.mu.Unlock()
}

// IsStarted returns true if the trader is running
func (t *Trader) IsStarted() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.started
}

// GetPnL calculates the current profit/loss
func (t *Trader) GetPnL(currentPrice float64) (float64, error) {
	pos := t.GetPosition()
	if pos == nil {
		return 0, fmt.Errorf("no position")
	}

	if pos.Side == strategy.SignalBuy {
		return (currentPrice - pos.EntryPrice) * pos.Quantity, nil
	}
	return (pos.EntryPrice - currentPrice) * pos.Quantity, nil
}

// GetPositionInfo returns formatted position information
func (t *Trader) GetPositionInfo(currentPrice float64) (string, error) {
	pos := t.GetPosition()
	if pos == nil {
		return "No open position", nil
	}

	pnl, _ := t.GetPnL(currentPrice)
	duration := time.Since(pos.OpenedAt).Round(time.Second)

	return fmt.Sprintf("Symbol: %s | Side: %s | Entry: %.2f | Current: %.2f | PnL: %.2f | Duration: %s",
		pos.Symbol, pos.Side, pos.EntryPrice, currentPrice, pnl, duration), nil
}

// ValidatePosition validates if position is still valid
func (t *Trader) ValidatePosition() error {
	pos := t.GetPosition()
	if pos == nil {
		return nil
	}

	// Check if order was filled
	order, err := t.client.GetOrderStatus(pos.Symbol, pos.OrderID)
	if err != nil {
		return fmt.Errorf("failed to validate position: %w", err)
	}

	// Status: NEW, PARTIALLY_FILLED, FILLED, CANCELED, PENDING_CANCEL, REJECTED, EXPIRED
	if order.Status == "FILLED" || order.Status == "PARTIALLY_FILLED" {
		return nil
	}

	return fmt.Errorf("order not filled: status = %s", order.Status)
}
