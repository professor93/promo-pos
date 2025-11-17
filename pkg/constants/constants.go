package constants

const (
	// Application metadata
	AppName        = "POSService"
	AppDisplayName = "POS Background Service"
	AppDescription = "Windows POS Service with offline-first architecture"

	// File paths (Windows style, will be converted at runtime)
	DefaultDataDir   = `%PROGRAMDATA%\POSService`
	DefaultConfigDir = `%PROGRAMDATA%\POSService`
	DefaultLogDir    = `%PROGRAMDATA%\POSService\logs`

	// File names
	ConfigFileName   = "config.enc"
	DatabaseFileName = "data.db"

	// Default configuration values
	DefaultPort           = 8080
	DefaultSyncInterval   = 59 // seconds
	DefaultMaxOfflineHours = 24
	DefaultLogLevel       = "info"

	// HTTP Server settings
	DefaultMaxConcurrentConnections = 100
	DefaultRateLimitPerMinute      = 100
	DefaultRequestTimeout          = 30 // seconds

	// Sync settings
	DefaultSyncRetryMax         = 5
	DefaultSyncRetryBackoffBase = 2 // seconds

	// Service settings
	WindowsServiceName        = "POSService"
	WindowsServiceDisplayName = "POS Background Service"
	WindowsServiceDescription = "Windows POS Service with offline-first architecture and encrypted data storage"

	// Registry paths (Windows) / File paths (Linux)
	RegistryBasePath = `SOFTWARE\POSService`

	// Offline grace period
	OfflineGracePeriodHours = 24
)
