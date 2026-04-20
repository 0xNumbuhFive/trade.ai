package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config holds all configuration for the trading bot
type Config struct {
	Trading  TradingConfig  `mapstructure:"trading"`
	Indicators IndicatorsConfig `mapstructure:"indicators"`
	API      APIConfig      `mapstructure:"api"`
	Logging  LoggingConfig  `mapstructure:"logging"`
}

// TradingConfig holds trading-specific configuration
type TradingConfig struct {
	Symbol         string  `mapstructure:"symbol"`
	Quantity       float64 `mapstructure:"quantity"`
	StopLossPct    float64 `mapstructure:"stop_loss_pct"`
	TakeProfitPct   float64 `mapstructure:"take_profit_pct"`
	Interval       string  `mapstructure:"interval"`
}

// IndicatorsConfig holds technical indicator configuration
type IndicatorsConfig struct {
	RSIPeriod     int     `mapstructure:"rsi_period"`
	RSIOversold   float64 `mapstructure:"rsi_oversold"`
	RSIOverbought float64 `mapstructure:"rsi_overbought"`
	MACDFast      int     `mapstructure:"macd_fast"`
	MACDSlow      int     `mapstructure:"macd_slow"`
	MACDSignal    int     `mapstructure:"macd_signal"`
	EMAPeriods    []int  `mapstructure:"ema_periods"`
}

// APIConfig holds API configuration
type APIConfig struct {
	BaseURL        string `mapstructure:"base_url"`
	TimeoutSeconds int    `mapstructure:"timeout_seconds"`
	APIKey         string `mapstructure:"api_key"`
	SecretKey      string `mapstructure:"secret_key"`
}

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// Load loads configuration from file and environment variables
func Load() (*Config, error) {
	// Load .env file if exists
	_ = godotenv.Load()

	// Set default values
	viper.SetDefault("trading.symbol", "BTCUSDT")
	viper.SetDefault("trading.quantity", 0.01)
	viper.SetDefault("trading.stop_loss_pct", 2.0)
	viper.SetDefault("trading.take_profit_pct", 5.0)
	viper.SetDefault("trading.interval", "1m")

	viper.SetDefault("indicators.rsi_period", 14)
	viper.SetDefault("indicators.rsi_oversold", 30.0)
	viper.SetDefault("indicators.rsi_overbought", 70.0)
	viper.SetDefault("indicators.macd_fast", 12)
	viper.SetDefault("indicators.macd_slow", 26)
	viper.SetDefault("indicators.macd_signal", 9)
	viper.SetDefault("indicators.ema_periods", []int{10, 20, 30})

	viper.SetDefault("api.base_url", "https://api.binance.com")
	viper.SetDefault("api.timeout_seconds", 30)

	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.file", "logs/bot.log")

	// Environment variables take precedence
	viper.BindEnv("api.api_key", "BINANCE_API_KEY")
	viper.BindEnv("api.secret_key", "BINANCE_SECRET_KEY")

	// Look for config file
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		// Config file is optional, continue without it
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Printf("Warning: Error reading config file: %v\n", err)
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	if cfg.API.APIKey == "" || cfg.API.SecretKey == "" {
		return nil, fmt.Errorf("BINANCE_API_KEY and BINANCE_SECRET_KEY must be set")
	}

	// Ensure log directory exists
	if cfg.Logging.File != "" {
		logDir := cfg.Logging.File[:len(cfg.Logging.File)-len("bot.log")]
		if err := os.MkdirAll(logDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create log directory: %w", err)
		}
	}

	return &cfg, nil
}
