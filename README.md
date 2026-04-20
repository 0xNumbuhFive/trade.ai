# Binance Trading Bot

A high-performance Go-based cryptocurrency trading bot that uses Technical Analysis indicators (RSI, MACD, Moving Averages) to execute real trades on Binance exchange.

## Features

- **Technical Analysis**: Uses RSI, MACD, and EMA indicators for trading signals
- **Risk Management**: Configurable stop-loss and take-profit
- **Real-time Monitoring**: Live position tracking and logging
- **Graceful Shutdown**: Safely closes positions on exit
- **Configurable**: All settings via YAML config and environment variables

## Requirements

- Go 1.21+
- Binance account with API keys
- Git

## Installation

1. **Clone the repository**

   ```bash
   cd /Users/eugenemutai/Development/trade.ai
   ```

2. **Install dependencies**

   ```bash
   go mod download
   ```

3. **Set up API credentials**

   Copy the example environment file:

   ```bash
   cp .env.example .env
   ```

   Edit `.env` and add your Binance API credentials:

   ```env
   BINANCE_API_KEY=your_api_key_here
   BINANCE_SECRET_KEY=your_secret_key_here
   ```

4. **Configure trading parameters**

   Edit `config/config.yaml` to adjust:
   - Trading pair (default: BTCUSDT)
   - Order quantity
   - Stop-loss and take-profit percentages
   - Indicator periods
   - Logging level

## Configuration

### Trading Settings (`config.yaml`)

```yaml
trading:
  symbol: "BTCUSDT" # Trading pair
  quantity: 0.01 # Order quantity
  stop_loss_pct: 2.0 # Stop loss percentage
  take_profit_pct: 5.0 # Take profit percentage
  interval: "1m" # Candle interval

indicators:
  rsi_period: 14
  rsi_oversold: 30.0
  rsi_overbought: 70.0
  macd_fast: 12
  macd_slow: 26
  macd_signal: 9
  ema_period: 20
```

### Environment Variables

| Variable             | Description             |
| -------------------- | ----------------------- |
| `BINANCE_API_KEY`    | Your Binance API key    |
| `BINANCE_SECRET_KEY` | Your Binance secret key |

## Running the Bot

```bash
go run cmd/bot/main.go
```

The bot will:

1. Connect to Binance API
2. Fetch historical klines
3. Start the trading loop
4. Monitor positions and execute trades

Press `Ctrl+C` to stop the bot gracefully.

## Trading Strategy

### Entry Signals

**BUY** when:

- RSI crosses above 30 (oversold recovery) AND
- MACD crosses above signal line AND
- Price above EMA

**SELL** when:

- RSI crosses below 70 (overbought) AND
- MACD crosses below signal line AND
- Price below EMA

### Exit Conditions

- Stop-loss triggered
- Take-profit triggered
- Opposite signal generated

## Security Notes

1. **API Permissions**: Create API keys with only trading permissions (enable "Enable Spot & Margin Trading")
2. **Never share keys**: Keep your API keys private
3. **Start small**: Test with small amounts first
4. **Monitor**: Always monitor the bot during trading

## Project Structure

```
trade.ai/
├── cmd/bot/main.go           # Application entry point
├── internal/
│   ├── api/binance.go       # Binance API client
│   ├── config/config.go     # Configuration management
│   ├── indicators/           # Technical indicators
│   │   ├── rsi.go
│   │   ├── macd.go
│   │   └── moving_average.go
│   ├── logger/logger.go     # Logging utilities
│   ├── strategy/technical.go # Trading strategy
│   └── trader/executor.go   # Order execution
├── config/config.yaml       # Configuration file
├── .env.example             # Environment template
└── README.md
```

## Troubleshooting

### Common Issues

**"BINANCE_API_KEY and BINANCE_SECRET_KEY must be set"**

- Make sure you have a `.env` file with your credentials

**"Failed to connect to Binance"**

- Check your internet connection
- Verify API keys are correct
- Ensure API key has proper permissions

**"Failed to place order"**

- Check your account balance
- Verify order quantity meets minimum requirements
- Ensure trading is enabled on your Binance account

## Disclaimer

This bot is for educational purposes. Cryptocurrency trading involves significant risk. Always understand the risks before trading with real money.

## License

MIT License
