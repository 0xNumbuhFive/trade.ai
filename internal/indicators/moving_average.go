package indicators

import (
	"fmt"
)

// MovingAverage calculates Simple and Exponential Moving Averages
type MovingAverage struct {
	Period  int
	Type    MAType
	Values  []float64
}

// MAType defines the type of moving average
type MAType string

const (
	MATypeSMA MAType = "SMA"
	MATypeEMA MAType = "EMA"
)

// NewSMA creates a new Simple Moving Average indicator
func NewSMA(period int) *MovingAverage {
	return &MovingAverage{
		Period: period,
		Type:   MATypeSMA,
		Values: make([]float64, 0),
	}
}

// NewEMA creates a new Exponential Moving Average indicator
func NewEMA(period int) *MovingAverage {
	return &MovingAverage{
		Period: period,
		Type:   MATypeEMA,
		Values: make([]float64, 0),
	}
}

// Calculate calculates SMA for a slice of closing prices
func (ma *MovingAverage) Calculate(prices []float64) ([]float64, error) {
	if len(prices) < ma.Period {
		return nil, fmt.Errorf("not enough data points: need at least %d", ma.Period)
	}

	ma.Values = make([]float64, 0)

	if ma.Type == MATypeSMA {
		return ma.calculateSMA(prices)
	}

	return ma.calculateEMA(prices)
}

// calculateSMA calculates Simple Moving Average
func (ma *MovingAverage) calculateSMA(prices []float64) ([]float64, error) {
	for i := ma.Period - 1; i < len(prices); i++ {
		var sum float64
		for j := i - ma.Period + 1; j <= i; j++ {
			sum += prices[j]
		}
		ma.Values = append(ma.Values, sum/float64(ma.Period))
	}
	return ma.Values, nil
}

// calculateEMA calculates Exponential Moving Average
func (ma *MovingAverage) calculateEMA(prices []float64) ([]float64, error) {
	multiplier := float64(2) / float64(ma.Period+1)

	// First EMA is SMA
	var sum float64
	for i := 0; i < ma.Period; i++ {
		sum += prices[i]
	}
	ma.Values = append(ma.Values, sum/float64(ma.Period))

	// Calculate subsequent EMAs
	for i := ma.Period; i < len(prices); i++ {
		ema := (prices[i]-ma.Values[len(ma.Values)-1])*multiplier + ma.Values[len(ma.Values)-1]
		ma.Values = append(ma.Values, ema)
	}

	return ma.Values, nil
}

// GetLatest returns the most recent moving average value
func (ma *MovingAverage) GetLatest() (float64, error) {
	if len(ma.Values) == 0 {
		return 0, fmt.Errorf("no moving average values calculated")
	}
	return ma.Values[len(ma.Values)-1], nil
}

// IsPriceAbove returns true if current price is above the moving average
func (ma *MovingAverage) IsPriceAbove(currentPrice float64) (bool, error) {
	maValue, err := ma.GetLatest()
	if err != nil {
		return false, err
	}
	return currentPrice > maValue, nil
}

// IsPriceBelow returns true if current price is below the moving average
func (ma *MovingAverage) IsPriceBelow(currentPrice float64) (bool, error) {
	maValue, err := ma.GetLatest()
	if err != nil {
		return false, err
	}
	return currentPrice < maValue, nil
}

// CrossedAbove returns true if price crossed above the moving average
func (ma *MovingAverage) CrossedAbove(currentPrice float64) (bool, error) {
	if len(ma.Values) < 2 {
		return false, fmt.Errorf("not enough moving average values")
	}
	prevMA := ma.Values[len(ma.Values)-2]
	currMA := ma.Values[len(ma.Values)-1]

	// We need prices at the same timestamps as MA values
	// This requires the caller to provide appropriate price data
	return prevMA < currentPrice && currMA >= currentPrice, nil
}

// CrossedBelow returns true if price crossed below the moving average
func (ma *MovingAverage) CrossedBelow(currentPrice float64) (bool, error) {
	if len(ma.Values) < 2 {
		return false, fmt.Errorf("not enough moving average values")
	}
	prevMA := ma.Values[len(ma.Values)-2]
	currMA := ma.Values[len(ma.Values)-1]

	return prevMA > currentPrice && currMA <= currentPrice, nil
}

// GetAll returns all calculated moving average values
func (ma *MovingAverage) GetAll() []float64 {
	return ma.Values
}

// CalculateBatchEMAs calculates multiple EMAs at once for different periods
func CalculateBatchEMAs(prices []float64, periods []int) (map[int][]float64, error) {
	result := make(map[int][]float64)

	for _, period := range periods {
		ema := NewEMA(period)
		values, err := ema.Calculate(prices)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate EMA for period %d: %w", period, err)
		}
		result[period] = values
	}

	return result, nil
}

// CalculateBatchSMAs calculates multiple SMAs at once for different periods
func CalculateBatchSMAs(prices []float64, periods []int) (map[int][]float64, error) {
	result := make(map[int][]float64)

	for _, period := range periods {
		sma := NewSMA(period)
		values, err := sma.Calculate(prices)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate SMA for period %d: %w", period, err)
		}
		result[period] = values
	}

	return result, nil
}
