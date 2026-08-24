package config

import "time"

type Settings struct {
	Config    Config
	StartedAt time.Time
	Version   string
}

func NewSettings(c Config, version string, now time.Time) Settings {
	return Settings{Config: c, StartedAt: now.UTC(), Version: version}
}

func (s Settings) Public() map[string]any {
	return map[string]any{"version": s.Version, "started_at": s.StartedAt, "arrayops_lane": s.Config.Addr, "worker_interval": s.Config.WorkerInterval.String()}
}
