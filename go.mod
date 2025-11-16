module pos-gate

go 1.25.0

require (
	// System Tray (requires CGO)
	fyne.io/systray v1.11.0

	// Encryption utilities
	github.com/ProtonMail/gopenpgp/v2 v2.7.5
	github.com/go-chi/chi/v5 v5.1.0

	// Validation
	github.com/go-playground/validator/v10 v10.28.0

	// HTTP Client with retry
	github.com/go-resty/resty/v2 v2.16.5
	// Core HTTP Framework (choose one)
	github.com/gofiber/fiber/v3 v3.0.0-rc.2

	// JWT Token handling
	github.com/golang-jwt/jwt/v5 v5.2.2

	// Utilities
	github.com/google/uuid v1.6.0

	// Windows Service Management
	github.com/kardianos/service v1.2.2
	github.com/knadh/koanf/parsers/json v1.0.0
	github.com/knadh/koanf/providers/env v1.0.0
	github.com/knadh/koanf/providers/file v1.1.2

	// Configuration Management
	github.com/knadh/koanf/v2 v2.1.2

	// Error handling
	github.com/pkg/errors v0.9.1

	// Database migrations
	github.com/pressly/goose/v3 v3.22.1
	github.com/robfig/cron/v3 v3.0.1
	github.com/spf13/cobra v1.8.1

	// GUI Framework
	github.com/wailsapp/wails/v2 v2.9.2

	// Structured Logging
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.42.0

	// Windows-specific utilities
	golang.org/x/sys v0.36.0

	// Database
	modernc.org/sqlite v1.33.1
)

require (
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	// Indirect dependencies will be added automatically by go mod tidy
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/text v0.29.0 // indirect
)

require (
	github.com/andybalholm/brotli v1.2.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.10 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/gofiber/schema v1.6.0 // indirect
	github.com/gofiber/utils/v2 v2.0.0-rc.1 // indirect
	github.com/klauspost/compress v1.18.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/tinylib/msgp v1.4.0 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasthttp v1.65.0 // indirect
	golang.org/x/net v0.44.0 // indirect
)

// Optional: Add replace directives for local development
// replace github.com/yourcompany/shared => ../shared
