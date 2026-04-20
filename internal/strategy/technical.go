package strategy

import (
	"fmt"
	"sort"

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
	emas     map[int]*indicators.MovingAverage
	prices   []float64
	lastSig  Signal
}

// NewTechnicalStrategy creates a new technical analysis strategy
func NewTechnicalStrategy(cfg *config.Config, log *logger.CustomLogger) (*TechnicalStrategy, error) {
	indicatorsCfg := cfg.Indicators

	// Initialize multiple EMAs
	emas := make(map[int]*indicators.MovingAverage)
	for _, period := range indicatorsCfg.EMAPeriods {
		emas[period] = indicators.NewEMA(period)
	}

	return &TechnicalStrategy{
		cfg:    cfg,
		logger: log,
		rsi:    indicators.NewRSI(indicatorsCfg.RSIPeriod),
		macd:   indicators.NewMACD(indicatorsCfg.MACDFast, indicatorsCfg.MACDSlow, indicatorsCfg.MACDSignal),
		emas:   emas,
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

	// Calculate all EMAs
	for period, ema := range s.emas {
		_, err := ema.Calculate(prices)
		if err != nil {
			return fmt.Errorf("failed to calculate EMA(%d): %w", period, err)
		}
	}

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

	// Check EMA conditions - price relative to all EMAs
	// Get sorted periods for consistent ordering
	periods := make([]int, 0, len(s.emas))
	for period := range s.emas {
		periods = append(periods, period)
	}
	sort.Ints(periods)

	// Check if price is above all EMAs (bullish) or below all EMAs (bearish)
	priceAboveAll := true
	priceBelowAll := true
	emaValues := make(map[int]float64)
	for _, period := range periods {
		ema := s.emas[period]
		above, _ := ema.IsPriceAbove(currentPrice)
		below, _ := ema.IsPriceBelow(currentPrice)
		emaValues[period], _ = ema.GetLatest()
		if !above {
			priceAboveAll = false
		}
		if !below {
			priceBelowAll = false
		}
	}

	// Get indicator values for logging (use shortest EMA for logging)
	shortestPeriod := periods[0]
	emaVal := emaValues[shortestPeriod]
	rsiVal, _ := s.rsi.GetLatest()
	macdVal, _ := s.macd.GetLatest()

	// Log current indicators
	s.logger.LogIndicator(s.cfg.Trading.Symbol, rsiVal, macdVal.MACD, macdVal.Signal, emaVal)

	// Evaluate signals
	signal := SignalHold
	reason := ""

	// BUY conditions:
	// - RSI crossed above oversold (30) OR
	// - MACD crossed above signal line
	// - Price above shortest EMA (or all EMAs for stronger signal)
	buyConditions := 0
	if rsiCrossedAbove30 {
		buyConditions++
		reason += "RSI crossed above 30; "
	}
	if macdCrossedAbove {
		buyConditions++
		reason += "MACD crossed above signal; "
	}
	// Use price above shortest EMA as primary condition
	if priceAboveAll {
		buyConditions++
		reason += "Price above all EMAs; "
	} else if priceAboveEMA(shortestPeriod, currentPrice, s.emas) {
		buyConditions++
		reason += fmt.Sprintf("Price above EMA(%d); ", shortestPeriod)
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
	if priceBelowAll {
		sellConditions++
		reason += "Price below all EMAs; "
	} else if priceBelowEMA(shortestPeriod, currentPrice, s.emas) {
		sellConditions++
		reason += fmt.Sprintf("Price below EMA(%d); ", shortestPeriod)
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

// GetIndicators returns current indicator values (using shortest EMA)
func (s *TechnicalStrategy) GetIndicators() (rsi, macd, signal, ema float64, err error) {
	rsiVal, err := s.rsi.GetLatest()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	macdVal, err := s.macd.GetLatest()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	// Get shortest period EMA for backward compatibility
	if len(s.emas) == 0 {
		return 0, 0, 0, 0, fmt.Errorf("no EMAs configured")
	}
	periods := make([]int, 0, len(s.emas))
	for period := range s.emas {
		periods = append(periods, period)
	}
	sort.Ints(periods)
	shortest := periods[0]
	emaVal, err := s.emas[shortest].GetLatest()
	if err != nil {
		return 0, 0, 0, 0, err
	}

	return rsiVal, macdVal.MACD, macdVal.Signal, emaVal, nil
}

// GetIndicatorsAll returns all EMA values for monitoring
func (s *TechnicalStrategy) GetIndicatorsAll() (rsi float64, macdVal, signalVal indicators.MACDValue, emas map[int]float64, err error) {
	rsi, err = s.rsi.GetLatest()
	if err != nil {
		return 0, indicators.MACDValue{}, indicators.MACDValue{}, nil, err
	}

	macdVal, err = s.macd.GetLatest()
	if err != nil {
		return 0, indicators.MACDValue{}, indicators.MACDValue{}, nil, err
	}

	signalVal = macdVal

	emas = make(map[int]float64)
	for period, ema := range s.emas {
		val, err := ema.GetLatest()
		if err != nil {
			return 0, indicators.MACDValue{}, indicators.MACDValue{}, nil, err
		}
		emas[period] = val
	}

	return rsi, macdVal, signalVal, emas, nil
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

// Helper function to check if price is above a specific EMA
func priceAboveEMA(period int, currentPrice float64, emas map[int]*indicators.MovingAverage) bool {
	ema, exists := emas[period]
	if !exists {
		return false
	}
	above, _ := ema.IsPriceAbove(currentPrice)
	return above
}

// Helper function to check if price is below a specific EMA
func priceBelowEMA(period int, currentPrice float64, emas map[int]*indicators.MovingAverage) bool {
	ema, exists := emas[period]
	if !exists {
		return false
	}
	below, _ := ema.IsPriceBelow(currentPrice)
	return below
}
