package watcher

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"
)

type Config struct {
	SheetID      string
	TabName      string
	CaptureRange string
	BotConfigTab string
	BotName      string
	ReportLink   string
	Timezone     string
	WatchTab     string
	WatchCell    string
	PollInterval time.Duration
	SettleDelay  time.Duration
}

type Sheets interface {
	Values(context.Context, string, string) ([][]string, error)
	GroupIDs(context.Context, string) ([]string, error)
	SheetGID(context.Context, string) (int64, error)
	Token(context.Context) (string, error)
}

type SeaTalk interface {
	SendGroupText(context.Context, string, string, bool) error
	SendImage(context.Context, string, string) error
}

type Renderer interface {
	Capture(context.Context, string, int64, string, string) (string, error)
}

type Watcher struct {
	cfg      Config
	sheets   Sheets
	seatalk  SeaTalk
	renderer Renderer
	mu       sync.Mutex
	alerting bool
}

var scheduledSendHours = []int{0, 4, 6, 10, 13, 15, 18, 21}

func New(cfg Config, sheets Sheets, seatalk SeaTalk, renderer Renderer) *Watcher {
	return &Watcher{cfg: cfg, sheets: sheets, seatalk: seatalk, renderer: renderer}
}

func (w *Watcher) RunSchedule(ctx context.Context) {
	location, err := time.LoadLocation(w.cfg.Timezone)
	if err != nil {
		log.Printf("scheduled send timezone %q invalid, using local timezone: %v", w.cfg.Timezone, err)
		location = time.Local
	}
	log.Printf("scheduled sends enabled for %s at 6AM, 10AM, 1PM, 3PM, 6PM, 9PM, 12MN, 4AM", location)

	for {
		now := time.Now().In(location)
		next := nextScheduledSend(now, scheduledSendHours)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			log.Printf("scheduled send triggered for %s", next.Format("3:04PM Jan-02"))
			w.runAlert(ctx)
		}
	}
}

func (w *Watcher) RunValueWatch(ctx context.Context) {
	pollInterval := w.cfg.PollInterval
	if pollInterval <= 0 {
		pollInterval = 1 * time.Second
	}
	settleDelay := w.cfg.SettleDelay
	if settleDelay <= 0 {
		settleDelay = 7 * time.Second
	}
	log.Printf("change sends enabled: watching %s!%s every %s; sending %s after changes settle", w.cfg.WatchTab, w.cfg.WatchCell, pollInterval, settleDelay)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	var lastValue string
	baselineSet := false
	var settleTimer *time.Timer
	var settleC <-chan time.Time
	defer func() {
		if settleTimer != nil {
			settleTimer.Stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			value, err := w.watchValue(ctx)
			if err != nil {
				log.Printf("watch %s!%s: %v", w.cfg.WatchTab, w.cfg.WatchCell, err)
				continue
			}
			if !baselineSet {
				lastValue = value
				baselineSet = true
				log.Printf("watch baseline set for %s!%s", w.cfg.WatchTab, w.cfg.WatchCell)
				continue
			}
			if value == lastValue {
				continue
			}
			lastValue = value
			log.Printf("detected change in %s!%s; waiting %s before sending", w.cfg.WatchTab, w.cfg.WatchCell, settleDelay)
			if settleTimer == nil {
				settleTimer = time.NewTimer(settleDelay)
				settleC = settleTimer.C
				continue
			}
			if !settleTimer.Stop() {
				select {
				case <-settleTimer.C:
				default:
				}
			}
			settleTimer.Reset(settleDelay)
			settleC = settleTimer.C
		case <-settleC:
			log.Printf("settle delay passed for %s!%s; sending report", w.cfg.WatchTab, w.cfg.WatchCell)
			w.runAlert(ctx)
			settleC = nil
		}
	}
}

func nextScheduledSend(now time.Time, hours []int) time.Time {
	for _, hour := range hours {
		next := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, now.Location())
		if next.After(now) {
			return next
		}
	}
	return time.Date(now.Year(), now.Month(), now.Day()+1, hours[0], 0, 0, 0, now.Location())
}

func (w *Watcher) runAlert(parent context.Context) {
	w.mu.Lock()
	if w.alerting {
		w.mu.Unlock()
		log.Printf("alert already running; skipping overlapping trigger")
		return
	}
	w.alerting = true
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		w.alerting = false
		w.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	if err := w.alert(ctx); err != nil {
		log.Printf("alert: %v", err)
	}
}

func (w *Watcher) SendNow(ctx context.Context) error {
	w.mu.Lock()
	if w.alerting {
		w.mu.Unlock()
		return fmt.Errorf("alert already running")
	}
	w.alerting = true
	w.mu.Unlock()
	defer func() {
		w.mu.Lock()
		w.alerting = false
		w.mu.Unlock()
	}()

	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	return w.alert(ctx)
}

func (w *Watcher) alert(ctx context.Context) error {
	gid, err := w.sheets.SheetGID(ctx, w.cfg.TabName)
	if err != nil {
		return err
	}
	token, err := w.sheets.Token(ctx)
	if err != nil {
		return err
	}
	image, err := w.renderer.Capture(ctx, w.cfg.SheetID, gid, w.cfg.CaptureRange, token)
	if err != nil {
		return err
	}
	groupIDs, err := w.sheets.GroupIDs(ctx, w.cfg.BotConfigTab)
	if err != nil {
		return err
	}
	if len(groupIDs) == 0 {
		return fmt.Errorf("no SeaTalk group IDs found in %s!A2:A", w.cfg.BotConfigTab)
	}
	text, err := w.alertText(ctx)
	if err != nil {
		return err
	}
	for _, groupID := range groupIDs {
		if err := w.seatalk.SendGroupText(ctx, groupID, text, true); err != nil {
			log.Printf("send text to %s: %v", groupID, err)
			continue
		}
		if err := w.seatalk.SendImage(ctx, groupID, image); err != nil {
			log.Printf("send image to %s: %v", groupID, err)
			continue
		}
		log.Printf("sent text and report image to %s", groupID)
	}
	return nil
}

func (w *Watcher) alertText(ctx context.Context) (string, error) {
	linehaulWindow, err := w.cell(ctx, "O1")
	if err != nil {
		return "", err
	}
	v3, err := w.cellFromTab(ctx, "enrroute_consodata", "V3")
	if err != nil {
		return "", err
	}
	v4, err := w.cellFromTab(ctx, "enrroute_consodata", "V4")
	if err != nil {
		return "", err
	}
	v5, err := w.cellFromTab(ctx, "enrroute_consodata", "V5")
	if err != nil {
		return "", err
	}

	now := w.now()
	hour := now.Hour()
	if hour >= 8 && hour < 17 {
		return formatDailyUpdateAlert(now), nil
	}
	return formatLinehaulAlert(linehaulWindow, now, v3, v4, v5), nil
}

func formatDailyUpdateAlert(now time.Time) string {
	return fmt.Sprintf("<mention-tag target=\"seatalk://user?id=0\"/> En Route, Docked & On-Queue Update as of %s", now.Format("3:04PM"))
}

func formatLinehaulAlert(linehaulWindow string, now time.Time, v3, v4, v5 string) string {
	v3 = strings.TrimPrefix(strings.TrimPrefix(v3, "**"), "*")
	v3 = strings.TrimSuffix(strings.TrimSuffix(v3, "**"), "*")
	return fmt.Sprintf("<mention-tag target=\"seatalk://user?id=0\"/> IB Expected Linehauls to Arrive within %s including Late Units as of %s Update.\n\n<b>%s</b>\n%s\n%s", linehaulWindow, now.Format("3:04PM"), v3, v4, v5)
}

func (w *Watcher) now() time.Time {
	location, err := time.LoadLocation(w.cfg.Timezone)
	if err != nil {
		return time.Now()
	}
	return time.Now().In(location)
}

func (w *Watcher) cell(ctx context.Context, cell string) (string, error) {
	values, err := w.sheets.Values(ctx, w.cfg.TabName, cell)
	if err != nil {
		return "", fmt.Errorf("read %s!%s: %w", w.cfg.TabName, cell, err)
	}
	if len(values) == 0 || len(values[0]) == 0 {
		return "", nil
	}
	return values[0][0], nil
}

func (w *Watcher) cellFromTab(ctx context.Context, tabName, cell string) (string, error) {
	values, err := w.sheets.Values(ctx, tabName, cell)
	if err != nil {
		return "", fmt.Errorf("read %s!%s: %w", tabName, cell, err)
	}
	if len(values) == 0 || len(values[0]) == 0 {
		return "", nil
	}
	return values[0][0], nil
}

func (w *Watcher) watchValue(ctx context.Context) (string, error) {
	values, err := w.sheets.Values(ctx, w.cfg.WatchTab, w.cfg.WatchCell)
	if err != nil {
		return "", fmt.Errorf("read %s!%s: %w", w.cfg.WatchTab, w.cfg.WatchCell, err)
	}
	return firstCell(values), nil
}

func firstCell(values [][]string) string {
	if len(values) == 0 || len(values[0]) == 0 {
		return ""
	}
	return values[0][0]
}
