package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tradeai/bot/internal/api"
	"github.com/tradeai/bot/internal/config"
	"github.com/tradeai/bot/internal/logger"
	"github.com/tradeai/bot/internal/strategy"
	"github.com/tradeai/bot/internal/trader"
)

const (
	// Default polling interval
	pollInterval = 30 * time.Second
	// Number of klines to fetch for indicators
	klineLimit = 100
)

// Bot represents the trading bot
type Bot struct {
	cfg       *config.Config
	logger    *logger.CustomLogger
	client    *api.BinanceClient
	strategy  *strategy.TechnicalStrategy
	trader    *trader.Trader
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewBot creates a new Bot instance
func NewBot() (*Bot, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logger
	log, err := logger.New(cfg.Logging.Level, cfg.Logging.File)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize logger: %w", err)
	}

	log.LogInfo("Initializing Binance Trading Bot", nil)

	// Initialize Binance client
	client, err := api.NewBinanceClient(&cfg.API, log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Binance client: %w", err)
	}

	// Ping to verify connection
	if err := client.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to Binance: %w", err)
	}

	log.LogInfo("Connected to Binance API", nil)

	// Initialize strategy
	strategy, err := strategy.NewTechnicalStrategy(cfg, log)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize strategy: %w", err)
	}

	// Initialize trader
	tradingClient := trader.NewTrader(client, &cfg.Trading, log)

	return &Bot{
		cfg:      cfg,
		logger:   log,
		client:   client,
		strategy: strategy,
		trader:   tradingClient,
	}, nil
}

// Start starts the trading bot
func (b *Bot) Start() error {
	b.logger.LogInfo("Starting trading bot...", nil)

	// Create context with cancellation
	b.ctx, b.cancel = context.WithCancel(context.Background())

	// Start trader
	b.trader.Start()

	// Main trading loop
	go b.runLoop()

	b.logger.LogInfo("Trading bot started successfully", nil)
	return nil
}

// Stop stops the trading bot
func (b *Bot) Stop() error {
	b.logger.LogInfo("Stopping trading bot...", nil)

	b.cancel()
	b.trader.Stop()

	// Close any open positions
	if b.trader.HasPosition() {
		b.logger.LogInfo("Closing open position before exit...", nil)
		if err := b.trader.ClosePosition("bot stopped"); err != nil {
			b.logger.LogError(err, "failed to close position")
		}
	}

	b.logger.LogInfo("Trading bot stopped", nil)
	return nil
}

// runLoop runs the main trading loop
func (b *Bot) runLoop() {
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-b.ctx.Done():
			return
		case <-ticker.C:
			b.processTick()
		}
	}
}

// processTick processes a single tick
func (b *Bot) processTick() {
	symbol := b.cfg.Trading.Symbol
	interval := b.cfg.Trading.Interval

	// Fetch klines
	klines, err := b.client.GetKlines(symbol, interval, klineLimit)
	if err != nil {
		b.logger.LogError(err, "failed to fetch klines")
		return
	}

	// Extract closing prices
	prices := make([]float64, len(klines))
	for i, kline := range klines {
		var price float64
		fmt.Sscanf(kline.Close, "%f", &price)
		prices[i] = price
	}

	// Update strategy with prices
	if err := b.strategy.UpdatePrices(prices); err != nil {
		b.logger.LogError(err, "failed to update strategy")
		return
	}

	// Get current price
	currentPrice, err := b.client.GetCurrentPrice(symbol)
	if err != nil {
		b.logger.LogError(err, "failed to get current price")
		return
	}

	// Check if we should close position
	if b.trader.HasPosition() {
		if err := b.checkExitConditions(currentPrice); err != nil {
			b.logger.LogError(err, "failed to check exit conditions")
		}
		return
	}

	// Evaluate entry signal
	signal, reason, err := b.strategy.Evaluate(currentPrice)
	if err != nil {
		b.logger.LogError(err, "failed to evaluate strategy")
		return
	}

	// Execute entry if signal is BUY
	if signal == strategy.SignalBuy {
		b.logger.LogInfo(fmt.Sprintf("Entering position: %s", reason), nil)
		if err := b.trader.OpenPosition(signal, currentPrice); err != nil {
			b.logger.LogError(err, "failed to open position")
		}
	}
}

// checkExitConditions checks if we should exit the current position
func (b *Bot) checkExitConditions(currentPrice float64) error {
	// Check stop loss
	if b.trader.CheckStopLoss(currentPrice) {
		b.logger.LogInfo("Stop loss triggered", nil)
		return b.trader.ClosePosition("stop loss")
	}

	// Check take profit
	if b.trader.CheckTakeProfit(currentPrice) {
		b.logger.LogInfo("Take profit triggered", nil)
		return b.trader.ClosePosition("take profit")
	}

	// Evaluate sell signal
	signal, reason, err := b.strategy.Evaluate(currentPrice)
	if err != nil {
		return err
	}

	if signal == strategy.SignalSell {
		b.logger.LogInfo(fmt.Sprintf("Exit signal: %s", reason), nil)
		return b.trader.ClosePosition("signal")
	}

	// Log position info
	posInfo, _ := b.trader.GetPositionInfo(currentPrice)
	b.logger.LogInfo(posInfo, nil)

	return nil
}

func main() {
	// Initialize bot
	bot, err := NewBot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize bot: %v\n", err)
		os.Exit(1)
	}

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start bot
	if err := bot.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to start bot: %v\n", err)
		os.Exit(1)
	}

	// Wait for shutdown signal
	sig := <-sigChan
	fmt.Printf("\nReceived signal: %v\n", sig)
	fmt.Println("Shutting down...")

	// Stop bot
	if err := bot.Stop(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to stop bot: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Bot stopped gracefully")
}
