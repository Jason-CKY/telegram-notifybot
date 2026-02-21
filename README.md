# Telegram Currency Exchange Rate Notification Bot

A Go Telegram bot for currency exchange rate notifications against SGD (Singapore Dollar). It fetches exchange rate data from the MAS (Monetary Authority of Singapore) API, stores user settings in Directus, and sends notifications via Telegram.

## Features

- Real-time exchange rate queries against SGD
- Historical exchange rate charts
- Threshold-based notifications (above/below)
- Interval-based notifications (rate change by X SGD)
- Hourly scheduler for checking rates
- User authentication via whitelisted Telegram usernames

## Supported Currencies

USD, EUR, GBP, JPY, MYR, HKD, AUD, KRW, TWD, IDR, THB, CNY, INR, PHP

## Bot Commands

| Command | Description |
|---------|-------------|
| `/help` | Show all available commands |
| `/start` | Register with the bot |
| `/fx <currency>` | Show current exchange rate |
| `/fx_chart <currency> [months]` | Show historical chart (default: 12 months) |
| `/fx_subscribe <currency> --above <rate>` | Notify when rate goes above threshold |
| `/fx_subscribe <currency> --below <rate>` | Notify when rate goes below threshold |
| `/fx_interval <currency> <interval>` | Notify every X SGD change |
| `/fx_list` | List all your subscriptions |
| `/fx_unsubscribe <currency>` | Remove subscription for currency |

### Examples

```
/fx USD                    # Show current USD/SGD rate
/fx_chart EUR 6            # Show EUR/SGD chart for last 6 months
/fx_subscribe USD --above 1.40   # Notify when USD goes above 1.40 SGD
/fx_subscribe EUR --below 1.45   # Notify when EUR goes below 1.45 SGD
/fx_interval JPY 0.01      # Notify when JPY changes by 0.01 SGD
/fx_list                   # List all your subscriptions
/fx_unsubscribe USD        # Remove USD subscription
```

## Tech Stack

- **Go 1.24** - Backend language
- **Telegram Bot API** - Bot framework
- **Directus** - Headless CMS for data storage
- **PostgreSQL** - Database (via Directus)
- **Docker** - Containerization
- **MAS API** - Exchange rate data source

## Dependencies

- [Docker](https://www.docker.com/) / Docker Desktop
- [Docker Compose](https://docs.docker.com/compose/)
- [Go v1.24+](https://go.dev/doc/install)
- [Air](https://github.com/cosmtrek/air) - Hot reload for development

## Quickstart (Development)

### 1. Clone and Configure

```bash
cp .env.example .env
```

Edit `.env` with your credentials:

```env
TELEGRAM_BOT_TOKEN=your_bot_token
DIRECTUS_HOST=http://localhost:8055
DIRECTUS_TOKEN=your_directus_token
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=directus
ALLOWED_USERNAMES=username1,username2
LOG_LEVEL=info
```

### 2. Start Services

```bash
# Start PostgreSQL and Directus
make start

# Or with docker compose
docker compose up -d postgres directus

# Wait for Directus to be ready, then initialize schema
docker compose up initialize-db
```

### 3. Run the Bot

```bash
# With hot reload (recommended for development)
air

# Or run directly
go run main.go
```

## Docker Commands

```bash
make build    # Rebuild all Docker images
make start    # Start containers
make stop     # Stop containers
make destroy  # Stop and remove volumes
```

## Project Structure

```
.
├── main.go                         # Application entry point
├── internal/
│   ├── core/
│   │   ├── scheduler.go            # FX notification scheduler
│   │   ├── fx_api.go               # MAS API client
│   │   └── fx_chart.go             # Chart generation
│   ├── handler/
│   │   ├── router.go               # Command routing
│   │   └── fx_handler.go           # FX command handlers
│   ├── schemas/
│   │   ├── chat_settings.go        # Chat settings CRUD
│   │   ├── currency_subscription.go # Subscription CRUD
│   │   └── exchange_rate.go        # MAS API response types
│   └── utils/
│       ├── common.go               # Global vars, constants
│       └── utils.go                # Helper functions
├── scripts/
│   └── directus/
│       └── build-tables.sh         # Directus schema initialization
├── docker-compose.yml              # Docker services definition
├── Makefile                        # Docker commands
├── .air.toml                       # Air hot-reload configuration
├── AGENTS.md                       # Coding agent guidelines
└── README.md                       # This file
```

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
| currency | string | Currency code (USD, EUR, etc.) |
| threshold_above | float | Nullable - notify when rate >= value |
| threshold_below | float | Nullable - notify when rate <= value |
| interval | float | Nullable - notify when rate changes by X |
| last_notified_rate | float | Last rate user was notified at |
| last_notification_time | timestamp | Last notification timestamp |
| enabled | boolean | Subscription active status |
| date_created | timestamp | Auto-generated |
| date_updated | timestamp | Auto-updated |

## API Reference

### MAS Exchange Rate API

The bot uses the Monetary Authority of Singapore's open data API:

- **Endpoint:** `https://eservices.mas.gov.sg/api/action/datastore/search.json`
- **Resource ID:** `10eafb90-11f2-4e4c-ab72-7b6d1e5e7c30`
- **Data:** Monthly end-of-day exchange rates (SGD per unit of foreign currency)

### Special Rate Handling

Some currencies are quoted per 100 or 1000 units:
- **JPY, KRW:** Per 100 units
- **IDR:** Per 1000 units

The code automatically handles division to return SGD per single unit.

## Development

### Build & Test

```bash
# Build
go build -o ./tmp/main .

# Run tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Vet code
go vet ./...

# Format code
go fmt ./...
```

### Linting

```bash
golangci-lint run
```

## Notes

- Default timezone is `Asia/Singapore`
- Threshold notifications are one-time (auto-remove after triggered)
- Interval notifications persist until manually removed
- FX scheduler runs every hour
- MAS data is updated monthly (end of month rates)

## License

MIT
