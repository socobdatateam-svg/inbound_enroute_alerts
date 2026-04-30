package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port                  string
	SeaTalkAppID          string
	SeaTalkAppSecret      string
	SeaTalkSigningSecret  string
	AdminToken            string
	GoogleCredentials     string
	GoogleCredentialsJSON string
	SheetID               string
	TabName               string
	CaptureRange          string
	BotConfigTab          string
	BotName               string
	ReportLink            string
	Timezone              string
	EnableScheduledSends  bool
	EnableChangeSends     bool
	WatchTab              string
	WatchCell             string
	WatchPollSeconds      int
	ChangeSettleSeconds   int
	ImageFormat           string
	PNGDPI                int
	PNGMaxWidth           int
	WorkDir               string
}

func Load() (Config, error) {
	if err := loadDotEnv(".env"); err != nil {
		return Config{}, err
	}

	cfg := Config{
		Port:                 getenv("PORT", "8080"),
		SheetID:              getenv("SHEET_ID", "1LiSwe5XABNPSPIhdK-Hu7S8VjWEsjPmUHmnU3toGXhc"),
		TabName:              getenv("TAB_NAME", "Compliance Tracker"),
		CaptureRange:         getenv("CAPTURE_RANGE", "A1:X80"),
		BotConfigTab:         getenv("BOT_CONFIG_TAB", "bot_config"),
		BotName:              getenv("BOT_NAME", "Bot Workstation"),
		ReportLink:           getenv("REPORT_LINK", "https://docs.google.com/spreadsheets/d/1hYCkLL9Z4UR3WeKFuCDsOYch5v1FxmTJLGtk8UG_yyI/edit?gid=2001886446#gid=2001886446"),
		Timezone:             getenv("APP_TIMEZONE", "Asia/Manila"),
		ImageFormat:          getenv("IMAGE_FORMAT", "png"),
		PNGDPI:               mustInt("PNG_DPI", 300),
		PNGMaxWidth:          mustInt("PNG_MAX_WIDTH", 2400),
		WorkDir:              getenv("WORK_DIR", "tmp"),
		EnableScheduledSends: getenv("ENABLE_SCHEDULED_SENDS", "false") == "true",
		EnableChangeSends:    getenv("ENABLE_CHANGE_SENDS", "true") == "true",
		WatchTab:             getenv("WATCH_TAB", "Summary Sheet (In progress)"),
		WatchCell:            getenv("WATCH_CELL", "AE6"),
		WatchPollSeconds:     mustInt("WATCH_POLL_SECONDS", 1),
		ChangeSettleSeconds:  mustInt("CHANGE_SETTLE_SECONDS", 7),
	}
	cfg.SeaTalkAppID = os.Getenv("SEATALK_APP_ID")
	cfg.SeaTalkAppSecret = os.Getenv("SEATALK_APP_SECRET")
	cfg.SeaTalkSigningSecret = os.Getenv("SEATALK_SIGNING_SECRET")
	cfg.AdminToken = os.Getenv("ADMIN_TOKEN")
	cfg.GoogleCredentials = os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
	cfg.GoogleCredentialsJSON = os.Getenv("GOOGLE_CREDENTIALS_JSON")

	for name, value := range map[string]string{
		"SEATALK_APP_ID":         cfg.SeaTalkAppID,
		"SEATALK_APP_SECRET":     cfg.SeaTalkAppSecret,
		"SEATALK_SIGNING_SECRET": cfg.SeaTalkSigningSecret,
	} {
		if value == "" {
			return Config{}, fmt.Errorf("%s is required", name)
		}
	}
	if cfg.GoogleCredentials == "" && cfg.GoogleCredentialsJSON == "" {
		return Config{}, fmt.Errorf("GOOGLE_APPLICATION_CREDENTIALS or GOOGLE_CREDENTIALS_JSON is required")
	}

	if cfg.ImageFormat != "png" && cfg.ImageFormat != "jpg" && cfg.ImageFormat != "jpeg" {
		return Config{}, fmt.Errorf("IMAGE_FORMAT must be png or jpg")
	}
	if cfg.PNGDPI <= 0 {
		return Config{}, fmt.Errorf("PNG_DPI must be greater than 0")
	}
	if cfg.PNGMaxWidth <= 0 {
		return Config{}, fmt.Errorf("PNG_MAX_WIDTH must be greater than 0")
	}
	if cfg.WatchPollSeconds <= 0 {
		return Config{}, fmt.Errorf("WATCH_POLL_SECONDS must be greater than 0")
	}
	if cfg.ChangeSettleSeconds <= 0 {
		return Config{}, fmt.Errorf("CHANGE_SETTLE_SECONDS must be greater than 0")
	}
	return cfg, nil
}

func loadDotEnv(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", path, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimPrefix(scanner.Text(), "\ufeff"))
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.Trim(value, `"'`)
		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from %s: %w", key, path, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	return nil
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func mustInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
