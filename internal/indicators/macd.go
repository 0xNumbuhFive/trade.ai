package indicators

import (
	"fmt"
)

// MACD calculates the Moving Average Convergence Divergence
type MACD struct {
	FastPeriod   int
	SlowPeriod   int
	SignalPeriod int
	Values      []MACDValue
	initialized bool
}

// MACDValue holds MACD calculation results
type MACDValue struct {
	MACD       float64
	Signal     float64
	Histogram  float64
}

// NewMACD creates a new MACD indicator
func NewMACD(fastPeriod, slowPeriod, signalPeriod int) *MACD {
	return &MACD{
		FastPeriod:   fastPeriod,
		SlowPeriod:   slowPeriod,
		SignalPeriod: signalPeriod,
		Values:       make([]MACDValue, 0),
		initialized:  false,
	}
}

// Calculate calculates MACD for a slice of closing prices
func (m *MACD) Calculate(prices []float64) ([]MACDValue, error) {
	if len(prices) < m.SlowPeriod+m.SignalPeriod {
		return nil, fmt.Errorf("not enough data points: need at least %d", m.SlowPeriod+m.SignalPeriod)
	}

	m.Values = make([]MACDValue, 0)

	// Calculate fast and slow EMAs
	fastEMA := calculateEMA(prices, m.FastPeriod)
	slowEMA := calculateEMA(prices, m.SlowPeriod)

	// Calculate MACD line (fast EMA - slow EMA)
	macdLine := make([]float64, len(prices))
	for i := range fastEMA {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	// Calculate signal line (EMA of MACD)
	signalLine := calculateMACDSignal(macdLine, m.SignalPeriod)

	// Calculate histogram and values
	startIdx := m.SlowPeriod - 1
	for i := startIdx; i < len(macdLine); i++ {
		signalIdx := i - startIdx
		if signalIdx >= len(signalLine) {
			break
		}
		histogram := macdLine[i] - signalLine[signalIdx]
		m.Values = append(m.Values, MACDValue{
			MACD:      macdLine[i],
			Signal:    signalLine[signalIdx],
			Histogram: histogram,
		})
	}

	m.initialized = true
	return m.Values, nil
}

// calculateEMA calculates Exponential Moving Average
func calculateEMA(prices []float64, period int) []float64 {
	multiplier := float64(2) / float64(period+1)
	ema := make([]float64, len(prices))

	// First EMA is SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	ema[period-1] = sum / float64(period)

	// Calculate subsequent EMAs
	for i := period; i < len(prices); i++ {
		ema[i] = (prices[i]-ema[i-1])*multiplier + ema[i-1]
	}

	return ema
}

// calculateMACDSignal calculates the signal line (EMA) of MACD
func calculateMACDSignal(macdLine []float64, period int) []float64 {
	multiplier := float64(2) / float64(period+1)
	signal := make([]float64, len(macdLine))

	// First signal is SMA of first 'period' MACD values
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += macdLine[i]
	}
	signal[period-1] = sum / float64(period)

	// Calculate subsequent signals
	for i := period; i < len(macdLine); i++ {
		signal[i] = (macdLine[i]-signal[i-1])*multiplier + signal[i-1]
	}

	return signal
}

// GetLatest returns the most recent MACD value
func (m *MACD) GetLatest() (MACDValue, error) {
	if len(m.Values) == 0 {
		return MACDValue{}, fmt.Errorf("no MACD values calculated")
	}
	return m.Values[len(m.Values)-1], nil
}

// GetPrevious returns the previous MACD value
func (m *MACD) GetPrevious() (MACDValue, error) {
	if len(m.Values) < 2 {
		return MACDValue{}, fmt.Errorf("not enough MACD values")
	}
	return m.Values[len(m.Values)-2], nil
}

// MACDCrossedAbove returns true if MACD crossed above signal line
func (m *MACD) MACDCrossedAbove() (bool, error) {
	if len(m.Values) < 2 {
		return false, fmt.Errorf("not enough MACD values")
	}
	prev := m.Values[len(m.Values)-2]
	curr := m.Values[len(m.Values)-1]
	return prev.MACD < prev.Signal && curr.MACD >= curr.Signal, nil
}

// MACDCrossedBelow returns true if MACD crossed below signal line
func (m *MACD) MACDCrossedBelow() (bool, error) {
	if len(m.Values) < 2 {
		return false, fmt.Errorf("not enough MACD values")
	}
	prev := m.Values[len(m.Values)-2]
	curr := m.Values[len(m.Values)-1]
	return prev.MACD > prev.Signal && curr.MACD <= curr.Signal, nil
}

// IsPositive returns true if MACD is positive (bullish)
func (m *MACD) IsPositive() (bool, error) {
	current, err := m.GetLatest()
	if err != nil {
		return false, err
	}
	return current.MACD > 0, nil
}

// IsNegative returns true if MACD is negative (bearish)
func (m *MACD) IsNegative() (bool, error) {
	current, err := m.GetLatest()
	if err != nil {
		return false, err
	}
	return current.MACD < 0, nil
}
