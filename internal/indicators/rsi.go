package indicators

import (
	"fmt"
	"math"
)

// RSI calculates the Relative Strength Index
type RSI struct {
	Period int
	Values []float64
	prevAvgGain float64
	prevAvgLoss float64
	initialized bool
}

// NewRSI creates a new RSI indicator
func NewRSI(period int) *RSI {
	return &RSI{
		Period: period,
		Values: make([]float64, 0),
		initialized: false,
	}
}

// Calculate calculates RSI for a slice of closing prices
func (r *RSI) Calculate(prices []float64) ([]float64, error) {
	if len(prices) < r.Period+1 {
		return nil, fmt.Errorf("not enough data points: need at least %d", r.Period+1)
	}

	r.Values = make([]float64, 0, len(prices)-r.Period)
	r.initialized = false

	// Calculate initial average gain and loss
	var avgGain, avgLoss float64
	for i := 1; i <= r.Period; i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			avgGain += change
		} else {
			avgLoss += math.Abs(change)
		}
	}
	avgGain /= float64(r.Period)
	avgLoss /= float64(r.Period)

	// Calculate first RSI
	rs := avgGain / avgLoss
	firstRSI := 100 - (100 / (1 + rs))
	r.Values = append(r.Values, firstRSI)

	r.prevAvgGain = avgGain
	r.prevAvgLoss = avgLoss
	r.initialized = true

	// Calculate subsequent RSI values using smoothed averages
	for i := r.Period + 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]

		var gain, loss float64
		if change > 0 {
			gain = change
		} else {
			loss = math.Abs(change)
		}

		// Smoothed average
		avgGain = ((r.prevAvgGain * float64(r.Period-1)) + gain) / float64(r.Period)
		avgLoss = ((r.prevAvgLoss * float64(r.Period-1)) + loss) / float64(r.Period)

		rs = avgGain / avgLoss
		rsi := 100 - (100 / (1 + rs))

		r.Values = append(r.Values, rsi)
		r.prevAvgGain = avgGain
		r.prevAvgLoss = avgLoss
	}

	return r.Values, nil
}

// GetLatest returns the most recent RSI value
func (r *RSI) GetLatest() (float64, error) {
	if len(r.Values) == 0 {
		return 0, fmt.Errorf("no RSI values calculated")
	}
	return r.Values[len(r.Values)-1], nil
}

// GetPrevious returns the previous RSI value
func (r *RSI) GetPrevious() (float64, error) {
	if len(r.Values) < 2 {
		return 0, fmt.Errorf("not enough RSI values")
	}
	return r.Values[len(r.Values)-2], nil
}

// CrossedAbove returns true if RSI crossed above a threshold
func (r *RSI) CrossedAbove(threshold float64) (bool, error) {
	if len(r.Values) < 2 {
		return false, fmt.Errorf("not enough RSI values")
	}
	prev := r.Values[len(r.Values)-2]
	curr := r.Values[len(r.Values)-1]
	return prev < threshold && curr >= threshold, nil
}

// CrossedBelow returns true if RSI crossed below a threshold
func (r *RSI) CrossedBelow(threshold float64) (bool, error) {
	if len(r.Values) < 2 {
		return false, fmt.Errorf("not enough RSI values")
	}
	prev := r.Values[len(r.Values)-2]
	curr := r.Values[len(r.Values)-1]
	return prev > threshold && curr <= threshold, nil
}

// IsOversold returns true if RSI is in oversold territory
func (r *RSI) IsOversold(threshold float64) (bool, error) {
	current, err := r.GetLatest()
	if err != nil {
		return false, err
	}
	return current <= threshold, nil
}

// IsOverbought returns true if RSI is in overbought territory
func (r *RSI) IsOverbought(threshold float64) (bool, error) {
	current, err := r.GetLatest()
	if err != nil {
		return false, err
	}
	return current >= threshold, nil
}
