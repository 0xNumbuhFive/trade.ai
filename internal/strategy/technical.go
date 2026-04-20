package strategy

import (
	"fmt"
	"github.com/tradeai/bot/internal/config"
	"github.com/tradeai/bot/internal/indicators"
	"github.com/tradeai/bot/internal/logger"
)

// Signal represents a trading signal
type Signal string

const (
	SignalBuy  Signal = "BUY"
	SignalSell Signal = "SELL"
	SignalHold Signal = "HOLD"
)

// TechnicalStrategy implements a technical analysis based trading strategy
type TechnicalStrategy struct {
	cfg       *config.Config
	logger   *logger.CustomLogger
	rsi      *indicators.RSI
	macd     *indicators.MACD
	ema      *indicators.MovingAverage
	prices   []float64
	lastSig  Signal
}

// NewTechnicalStrategy creates a new technical analysis strategy
func NewTechnicalStrategy(cfg *config.Config, log *logger.CustomLogger) (*TechnicalStrategy, error) {
	indicatorsCfg := cfg.Indicators

	return &TechnicalStrategy{
		cfg:    cfg,
		logger: log,
		rsi:    indicators.NewRSI(indicatorsCfg.RSIPeriod),
		macd:   indicators.NewMACD(indicatorsCfg.MACDFast, indicatorsCfg.MACDSlow, indicatorsCfg.MACDSignal),
		ema:    indicators.NewEMA(indicatorsCfg.EMAPeriod),
		prices: make([]float64, 0),
		lastSig: SignalHold,
	}, nil
}

// UpdatePrices updates the price data and recalculates indicators
func (s *TechnicalStrategy) UpdatePrices(prices []float64) error {
	s.prices = prices

	// Calculate RSI
	rsiValues, err := s.rsi.Calculate(prices)
	if err != nil {
		return fmt.Errorf("failed to calculate RSI: %w", err)
	}
	_ = rsiValues

	// Calculate MACD
	macdValues, err := s.macd.Calculate(prices)
	if err != nil {
		return fmt.Errorf("failed to calculate MACD: %w", err)
	}
	_ = macdValues

	// Calculate EMA
	emaValues, err := s.ema.Calculate(prices)
	if err != nil {
		return fmt.Errorf("failed to calculate EMA: %w", err)
	}
	_ = emaValues

	return nil
}

// Evaluate evaluates the current market conditions and returns a trading signal
func (s *TechnicalStrategy) Evaluate(currentPrice float64) (Signal, string, error) {
	if err := s.UpdatePrices(s.prices); err != nil {
		return SignalHold, "", err
	}

	indicatorsCfg := s.cfg.Indicators

	// Check RSI conditions
	rsiOverbought, err := s.rsi.IsOverbought(indicatorsCfg.RSIOverbought)
	if err != nil {
		return SignalHold, "", err
	}
	rsiOversold, err := s.rsi.IsOversold(indicatorsCfg.RSIOversold)
	if err != nil {
		return SignalHold, "", err
	}

	rsiCrossedAbove30, _ := s.rsi.CrossedAbove(indicatorsCfg.RSIOversold)
	rsiCrossedBelow70, _ := s.rsi.CrossedBelow(indicatorsCfg.RSIOverbought)

	// Check MACD conditions
	macdCrossedAbove, _ := s.macd.MACDCrossedAbove()
	macdCrossedBelow, _ := s.macd.MACDCrossedBelow()

	// Check EMA conditions
	priceAboveEMA, _ := s.ema.IsPriceAbove(currentPrice)
	priceBelowEMA, _ := s.ema.IsPriceBelow(currentPrice)

	// Get indicator values for logging
	rsiVal, _ := s.rsi.GetLatest()
	macdVal, _ := s.macd.GetLatest()
	emaVal, _ := s.ema.GetLatest()

	// Log current indicators
	s.logger.LogIndicator(s.cfg.Trading.Symbol, rsiVal, macdVal.MACD, macdVal.Signal, emaVal)

	// Evaluate signals
	signal := SignalHold
	reason := ""

	// BUY conditions:
	// - RSI crossed above oversold (30) OR
	// - MACD crossed above signal line
	// - Price above EMA
	buyConditions := 0
	if rsiCrossedAbove30 {
		buyConditions++
		reason += "RSI crossed above 30; "
	}
	macdCrossedAbove, _ = s.macd.MACDCrossedAbove()
	if macdCrossedAbove {
		buyConditions++
		reason += "MACD crossed above signal; "
	}
	if priceAboveEMA {
		buyConditions++
		reason += "Price above EMA; "
	}

	if buyConditions >= 2 {
		signal = SignalBuy
		s.lastSig = SignalBuy
		s.logger.LogSignal(string(SignalBuy), s.cfg.Trading.Symbol, reason)
		return SignalBuy, reason, nil
	}

	// SELL conditions:
	// - RSI crossed below overbought (70) OR
	// - MACD crossed below signal line
	// - Price below EMA
	sellConditions := 0
	reason = ""

	rsiCrossedBelow70, _ = s.rsi.CrossedBelow(indicatorsCfg.RSIOverbought)
	if rsiCrossedBelow70 {
		sellConditions++
		reason += "RSI crossed below 70; "
	}
	macdCrossedBelow, _ = s.macd.MACDCrossedBelow()
	if macdCrossedBelow {
		sellConditions++
		reason += "MACD crossed below signal; "
	}
	if priceBelowEMA {
		sellConditions++
		reason += "Price below EMA; "
	}

	if sellConditions >= 2 {
		signal = SignalSell
		s.lastSig = SignalSell
		s.logger.LogSignal(string(SignalSell), s.cfg.Trading.Symbol, reason)
		return SignalSell, reason, nil
	}

	// If RSI is overbought, consider selling
	if rsiOverbought && sellConditions >= 1 {
		signal = SignalSell
		s.lastSig = SignalSell
		s.logger.LogSignal(string(SignalSell), s.cfg.Trading.Symbol, reason)
		return SignalSell, reason, nil
	}

	// If RSI is oversold and we don't have a position, consider buying
	if rsiOversold && buyConditions >= 1 {
		signal = SignalBuy
		s.lastSig = SignalBuy
		s.logger.LogSignal(string(SignalBuy), s.cfg.Trading.Symbol, reason)
		return SignalBuy, reason, nil
	}

	signal = SignalHold
	s.logger.LogSignal(string(signal), s.cfg.Trading.Symbol, "No clear signal")
	return signal, "No clear signal", nil
}

// GetLastSignal returns the last generated signal
func (s *TechnicalStrategy) GetLastSignal() Signal {
	return s.lastSig
}

// GetIndicators returns current indicator values
func (s *TechnicalStrategy) GetIndicators() (rsi, macd, signal, ema float64, err error) {
	rsiVal, err := s.rsi.GetLatest()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	macdVal, err := s.macd.GetLatest()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	emaVal, err := s.ema.GetLatest()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	return rsiVal, macdVal.MACD, macdVal.Signal, emaVal, nil
}

// CalculateStopLoss calculates the stop loss price
func (s *TechnicalStrategy) CalculateStopLoss(entryPrice float64, side Signal) float64 {
	stopLossPct := s.cfg.Trading.StopLossPct

	if side == SignalBuy {
		return entryPrice * (1 - stopLossPct/100)
	}

	return entryPrice * (1 + stopLossPct/100)
}

// CalculateTakeProfit calculates the take profit price
func (s *TechnicalStrategy) CalculateTakeProfit(entryPrice float64, side Signal) float64 {
	takeProfitPct := s.cfg.Trading.TakeProfitPct

	if side == SignalBuy {
		return entryPrice * (1 + takeProfitPct/100)
	}

	return entryPrice * (1 - takeProfitPct/100)
}

// ShouldStopLoss checks if stop loss should be triggered
func (s *TechnicalStrategy) ShouldStopLoss(entryPrice, currentPrice float64, side Signal) bool {
	stopLossPrice := s.CalculateStopLoss(entryPrice, side)

	if side == SignalBuy {
		return currentPrice <= stopLossPrice
	}

	return currentPrice >= stopLossPrice
}

// ShouldTakeProfit checks if take profit should be triggered
func (s *TechnicalStrategy) ShouldTakeProfit(entryPrice, currentPrice float64, side Signal) bool {
	takeProfitPrice := s.CalculateTakeProfit(entryPrice, side)

	if side == SignalBuy {
		return currentPrice >= takeProfitPrice
	}

	return currentPrice <= takeProfitPrice
}
