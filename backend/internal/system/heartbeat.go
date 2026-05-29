package system

import (
	"context"
	"time"

	"attendance/config"
	"attendance/internal/domain"
	"attendance/internal/repository"

	"go.uber.org/zap"
)

type HeartbeatMonitor struct {
	cfg    *config.Config
	rp     repository.AdminRepository
	logger *zap.SugaredLogger
}

func NewHeartbeatMonitor(
	cfg *config.Config,
	rp repository.AdminRepository,
	logger *zap.SugaredLogger,
) *HeartbeatMonitor {
	return &HeartbeatMonitor{cfg: cfg, rp: rp, logger: logger}
}

func (m *HeartbeatMonitor) Start(ctx context.Context) {
	go m.detectStartupOutage(ctx)
	go m.loop(ctx)
}

func (m *HeartbeatMonitor) detectStartupOutage(ctx context.Context) {
	location, err := time.LoadLocation(m.cfg.BusinessTimezone)
	if err != nil {
		m.logger.Errorf("heartbeat timezone is invalid: %v", err)
		return
	}

	now := time.Now().In(location)
	heartbeat, err := m.rp.GetSystemHeartbeat(ctx)
	if err != nil {
		m.logger.Errorf("failed to read system heartbeat: %v", err)
		return
	}

	if heartbeat != nil {
		lastSeen := heartbeat.LastSeenAt.In(location)
		if now.Sub(lastSeen) > m.cfg.OutageThreshold {
			input := m.outageInput(lastSeen, now, location)
			if _, err := m.rp.CreateSystemOutage(ctx, input); err != nil {
				m.logger.Errorf("failed to create system outage: %v", err)
			} else {
				m.logger.Infow(
					"system outage detected",
					"startedAt", input.StartedAt,
					"endedAt", input.EndedAt,
					"impactsWorkHours", input.ImpactsWorkHours,
				)
			}
		}
	}

	if err := m.rp.UpdateSystemHeartbeat(ctx, now); err != nil {
		m.logger.Errorf("failed to update system heartbeat: %v", err)
	}
}

func (m *HeartbeatMonitor) loop(ctx context.Context) {
	ticker := time.NewTicker(m.cfg.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := m.rp.UpdateSystemHeartbeat(ctx, time.Now().UTC()); err != nil {
				m.logger.Errorf("failed to update system heartbeat: %v", err)
			}
		}
	}
}

func (m *HeartbeatMonitor) outageInput(
	startedAt time.Time,
	endedAt time.Time,
	location *time.Location,
) domain.CreateSystemOutageInput {
	impacts := m.impactsWorkHours(startedAt, endedAt, location)
	var affected *time.Time
	if impacts {
		date := time.Date(
			startedAt.In(location).Year(),
			startedAt.In(location).Month(),
			startedAt.In(location).Day(),
			0, 0, 0, 0,
			location,
		)
		affected = &date
	}

	return domain.CreateSystemOutageInput{
		StartedAt:            startedAt,
		EndedAt:              endedAt,
		Reason:               "server_unavailable",
		AffectedBusinessDate: affected,
		ImpactsWorkHours:     impacts,
	}
}

func (m *HeartbeatMonitor) impactsWorkHours(
	startedAt time.Time,
	endedAt time.Time,
	location *time.Location,
) bool {
	startClock, err := time.Parse("15:04", m.cfg.OutageImpactStart)
	if err != nil {
		return false
	}
	endClock, err := time.Parse("15:04", m.cfg.OutageImpactEnd)
	if err != nil {
		return false
	}

	for day := normalizeDay(startedAt.In(location)); !day.After(normalizeDay(endedAt.In(location))); day = day.AddDate(0, 0, 1) {
		windowStart := time.Date(day.Year(), day.Month(), day.Day(), startClock.Hour(), startClock.Minute(), 0, 0, location)
		windowEnd := time.Date(day.Year(), day.Month(), day.Day(), endClock.Hour(), endClock.Minute(), 0, 0, location)
		if startedAt.Before(windowEnd) && endedAt.After(windowStart) {
			return true
		}
	}

	return false
}

func normalizeDay(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}
