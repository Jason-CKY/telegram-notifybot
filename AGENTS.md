# AGENTS.md - Coding Agent Guidelines

This document provides guidelines for agentic coding agents working in this repository.

## Project Overview

This is a Go Telegram bot for currency exchange rate notifications. It fetches exchange rate data from the MAS API, stores user settings in Directus, and sends notifications via Telegram.

**Tech Stack:** Go 1.24, Telegram Bot API, Directus (headless CMS), PostgreSQL, Docker

---

## Build, Lint, and Test Commands

### Development

```bash
# Start development server with hot reload (requires Air)
air

# Build binary
go build -o ./tmp/main .

# Run the application
go run main.go
```

### Docker

```bash
make build    # Rebuild all Docker images
make start    # Start containers
make stop     # Stop containers
make destroy  # Stop and remove volumes

# Or with docker compose
docker compose up -d postgres directus  # Start services
docker compose up initialize-db          # Initialize Directus schema
docker compose down -v                   # Stop and remove volumes
```

### Testing

```bash
# Run all tests
go test ./...

# Run tests in a specific package
go test ./internal/utils/...

# Run a single test file
go test ./internal/utils/utils_test.go

# Run a single test function
go test ./internal/utils -run TestFunctionName

# Run tests with verbose output
go test -v ./...

# Run tests with coverage
go test -cover ./...
```

### Linting and Formatting

```bash
# Format code
go fmt ./...

# Vet code for common errors
go vet ./...

# Run static analysis (if golangci-lint is installed)
golangci-lint run
```

---

## Code Style Guidelines

### Import Organization

Group imports in three sections, separated by blank lines:

1. Standard library packages
2. External/third-party packages
3. Local project packages

```go
import (
    "bytes"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    log "github.com/sirupsen/logrus"

    "github.com/Jason-CKY/telegram-notifybot/internal/schemas"
    "github.com/Jason-CKY/telegram-notifybot/internal/utils"
)
```

### Naming Conventions

- **Packages:** lowercase, single word preferred (`core`, `handler`, `schemas`, `utils`)
- **Exported functions/types:** PascalCase (`HandleUpdate`, `ChatSettings`)
- **Unexported functions/variables:** camelCase (`localTimezone`)
- **Constants:** UPPER_CASE for values, PascalCase for typed constants
- **Interfaces:** PascalCase with `-er` suffix for single-method interfaces
- **Acronyms:** Keep consistent casing (`HTTP`, `API`, `ID` -> `ChatID`)

### Struct and Type Definitions

```go
type CurrencySubscription struct {
    ID                   string    `json:"id,omitempty"`
    ChatID               int64     `json:"chat_id"`
    Currency             string    `json:"currency"`
    ThresholdAbove       *float64  `json:"threshold_above"`
    ThresholdBelow       *float64  `json:"threshold_below"`
    Interval             *float64  `json:"interval"`
    LastNotifiedRate     float64   `json:"last_notified_rate"`
    LastNotificationTime time.Time `json:"last_notification_time"`
    Enabled              bool      `json:"enabled"`
}
```

- Use PascalCase for struct fields
- Use snake_case for JSON tags (API compatibility)
- Use `omitempty` for optional fields (UUID, nullable values)
- Use pointers for nullable fields (`*float64`)

### Error Handling

- Return errors from functions rather than panicking (except in initialization)
- Log errors with context using logrus
- Wrap errors with descriptive messages

```go
if err != nil {
    log.Error(err)
    return
}

// For HTTP errors
if res.StatusCode != 200 {
    return nil, fmt.Errorf("error fetching data: %v", string(body))
}

// Initialization errors can panic
if err != nil {
    panic(err)
}
```

### Logging

Use logrus for structured logging:

```go
log.Info("connecting to telegram bot")
log.Errorf("error processing request: %v", err)
log.Debugf("querying %v", endpoint)

// Configure logrus in main
log.SetReportCaller(true)
log.SetFormatter(&log.TextFormatter{
    FullTimestamp:          true,
    DisableLevelTruncation: true,
})
```

### Environment Variables

Use `internal/utils` helper functions for environment variable access:

```go
// String value (panics if not set)
utils.LookupEnvString("DIRECTUS_HOST")

// String array (comma-separated)
utils.LookupEnvStringArray("ALLOWED_USERNAMES")

// Integer value
utils.LookupEnvInt("PORT")
```

### HTTP Client Pattern

```go
req, httpErr := http.NewRequest(http.MethodGet, endpoint, nil)
req.Header.Set("User-Agent", "Telegram-NotifyBot/1.0")
req.Header.Set("Authorization", fmt.Sprintf("Bearer %v", utils.DirectusToken))
if httpErr != nil {
    return nil, httpErr
}
client := &http.Client{}
res, httpErr := client.Do(req)
if httpErr != nil {
    return nil, httpErr
}
defer res.Body.Close()
body, _ := io.ReadAll(res.Body)
if res.StatusCode != 200 {
    return nil, fmt.Errorf("status code %v error: %v", res.StatusCode, string(body))
}
```

### Custom JSON Marshaling

When custom JSON marshaling is needed, use the alias pattern to avoid recursion. Required for converting between Go types and JSON string representations (e.g., int64 chat_id to string):

```go
func (cs CurrencySubscription) MarshalJSON() ([]byte, error) {
    type Alias CurrencySubscription
    aux := &struct {
        ChatID string `json:"chat_id"`
        *Alias
    }{
        ChatID: strconv.FormatInt(cs.ChatID, 10),
        Alias:  (*Alias)(&cs),
    }
    return json.Marshal(aux)
}

func (cs *CurrencySubscription) UnmarshalJSON(data []byte) error {
    type Alias CurrencySubscription
    aux := &struct {
        ChatID string `json:"chat_id"`
        *Alias
    }{
        Alias: (*Alias)(cs),
    }
    if err := json.Unmarshal(data, &aux); err != nil {
        return err
    }
    chatID, err := strconv.ParseInt(aux.ChatID, 10, 64)
    if err != nil {
        return err
    }
    cs.ChatID = chatID
    return nil
}
```

---

## Project Structure

```
.
├── main.go                         # Application entry point, bot initialization
├── internal/
│   ├── core/
│   │   ├── scheduler.go            # Hourly FX notification scheduler
│   │   ├── fx_api.go               # MAS API client wrapper
│   │   └── fx_chart.go             # Chart generation using go-charts
│   ├── handler/
│   │   ├── router.go               # Command routing and dispatcher
│   │   └── fx_handler.go           # All FX command handlers
│   ├── schemas/
│   │   ├── chat_settings.go        # ChatSettings model and CRUD
│   │   ├── currency_subscription.go # CurrencySubscription model and CRUD
│   │   └── exchange_rate.go        # MAS API response types, rate fetching
│   └── utils/
│       ├── common.go               # Global vars, constants, help message
│       └── utils.go                # Helper functions (env lookup, etc.)
├── scripts/
│   └── directus/
│       └── build-tables.sh         # Directus schema initialization script
├── docker-compose.yml              # Docker services (postgres, directus, init)
├── Makefile                        # Docker convenience commands
├── .air.toml                       # Air hot-reload configuration
├── .env.example                    # Environment variables template
├── AGENTS.md                       # This file
└── README.md                       # Project documentation
```

---

## Components

### Core Package (`internal/core`)

**scheduler.go**
- `StartFXScheduler()` - Runs hourly, checks all active subscriptions
- Fetches current rates, compares against thresholds/intervals
- Sends Telegram notifications when conditions are met

**fx_api.go**
- `GetFXRate(currency string)` - Get current rate for a currency
- `GetFXRates(currency string, months int)` - Get historical rates

**fx_chart.go**
- `GenerateFXChart(rates []HistoricalRate, currency string)` - Generate PNG chart

### Handler Package (`internal/handler`)

**router.go**
- `HandleUpdate(update tgbotapi.Update)` - Main update dispatcher
- Routes commands to appropriate handlers

**fx_handler.go**
- `HandleFXCommand()` - `/fx <currency>`
- `HandleFXChartCommand()` - `/fx_chart <currency> [months]`
- `HandleFXSubscribeCommand()` - `/fx_subscribe <currency> --above/--below <rate>`
- `HandleFXIntervalCommand()` - `/fx_interval <currency> <interval>`
- `HandleFXListCommand()` - `/fx_list`
- `HandleFXUnsubscribeCommand()` - `/fx_unsubscribe <currency>`

### Schemas Package (`internal/schemas`)

**chat_settings.go**
- `ChatSettings` struct with `MarshalJSON`/`UnmarshalJSON`
- CRUD methods: `Create()`, `GetChatSettings(chatID)`

**currency_subscription.go**
- `CurrencySubscription` struct with UUID primary key
- CRUD methods: `Create()`, `Update()`, `Delete()`
- Query methods: `GetCurrencySubscription()`, `GetCurrencySubscriptionsByChatID()`, `GetAllActiveSubscriptions()`
- Notification logic: `ShouldNotifyForThreshold()`, `ShouldNotifyForInterval()`, `GetNotificationMessage()`

**exchange_rate.go**
- `MASResponse` - API response wrapper
- `ExchangeRateRecord` - Single rate record
- `HistoricalRate` - Simplified rate struct
- `FetchExchangeRates()`, `GetRate()`, `GetHistoricalRates()`

### Utils Package (`internal/utils`)

**common.go**
- Global variables: `LogLevel`, `DirectusHost`, `DirectusToken`, `BotToken`, `WhitelistedUsernames`
- Constants: `HELP_MESSAGE`, `DEFAULT_TIMEZONE`, `SupportedCurrencies`
- Helper: `IsCurrencySupported()`

**utils.go**
- `LookupEnvString()`, `LookupEnvStringArray()`, `LookupEnvInt()`

---

## Telegram Bot Commands

| Command | Description |
|---------|-------------|
| `/help` | Show all available commands |
| `/start` | Register with the bot |
| `/fx <currency>` | Show current exchange rate |
| `/fx_chart <currency> [months]` | Show historical chart (default: 12) |
| `/fx_subscribe <currency> --above <rate>` | Notify when rate >= threshold |
| `/fx_subscribe <currency> --below <rate>` | Notify when rate <= threshold |
| `/fx_interval <currency> <interval>` | Notify every X SGD change |
| `/fx_list` | List all subscriptions |
| `/fx_unsubscribe <currency>` | Remove subscription |

**Supported Currencies:** USD, EUR, GBP, JPY, MYR, HKD, AUD, KRW, TWD, IDR, THB, CNY, INR, PHP

---

## Key Dependencies

- `github.com/go-telegram-bot-api/telegram-bot-api/v5` - Telegram Bot API
- `github.com/sirupsen/logrus` - Structured logging
- `github.com/joho/godotenv` - Environment variable loading
- `github.com/vicanso/go-charts/v2` - Chart generation
- `github.com/stretchr/testify` - Testing utilities (indirect)

---

## Directus Collections

### notifybot_chat_settings

| Field | Type | Notes |
|-------|------|-------|
| chat_id | string | Primary key |
| date_created | timestamp | Auto-generated |

### notifybot_currency_subscriptions

| Field | Type | Notes |
|-------|------|-------|
| id | uuid | Primary key (auto-generated) |
| chat_id | string | Telegram chat ID |
| currency | string | Currency code |
| threshold_above | float | Nullable |
| threshold_below | float | Nullable |
| interval | float | Nullable |
| last_notified_rate | float | Default: 0 |
| last_notification_time | timestamp | Nullable |
| enabled | boolean | Default: true |
| date_created | timestamp | Auto-generated |
| date_updated | timestamp | Auto-updated |

---

## Notes

- Do not add comments unless requested
- Follow existing patterns in the codebase
- Default timezone is `Asia/Singapore`
- User authentication via whitelisted Telegram usernames
- Threshold notifications are one-time (auto-remove after triggered)
- Interval notifications persist until manually removed
- FX scheduler runs every hour
- MAS API provides monthly (end-of-month) exchange rates
- JPY, KRW are quoted per 100 units; IDR per 1000 units - code handles division automatically
