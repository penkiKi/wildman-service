package system

import (
	"context"
	"os"
	"runtime"
	"time"

	"wildman-service/internal/config"
)

type Database interface {
	PingContext(ctx context.Context) error
}

type Check struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

type ReadinessChecks struct {
	Database      Check `json:"database"`
	DataDirectory Check `json:"dataDirectory"`
}

type Readiness struct {
	Status string          `json:"status"`
	Checks ReadinessChecks `json:"checks"`
	Time   string          `json:"time"`
}

type Info struct {
	Service     string `json:"service"`
	Environment string `json:"environment"`
	GoVersion   string `json:"goVersion"`
	OS          string `json:"os"`
	Arch        string `json:"arch"`
}

type Service struct {
	database Database
	config   config.Config
}

func NewService(database Database, cfg config.Config) *Service {
	return &Service{database: database, config: cfg}
}

func (s *Service) Readiness(ctx context.Context) (Readiness, bool) {
	checks := ReadinessChecks{
		Database:      checkDatabase(ctx, s.database),
		DataDirectory: checkDirectory(s.config.DataDir),
	}
	ready := checks.Database.Status == "ok" &&
		checks.DataDirectory.Status == "ok"
	status := "ready"
	if !ready {
		status = "not_ready"
	}

	return Readiness{
		Status: status,
		Checks: checks,
		Time:   time.Now().UTC().Format(time.RFC3339),
	}, ready
}

func (s *Service) Info() Info {
	return Info{
		Service:     "wildman-service",
		Environment: s.config.Environment,
		GoVersion:   runtime.Version(),
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
	}
}

func checkDatabase(ctx context.Context, database Database) Check {
	if database == nil {
		return Check{Status: "error", Message: "数据库未初始化"}
	}
	if err := database.PingContext(ctx); err != nil {
		return Check{Status: "error", Message: "数据库不可用"}
	}
	return Check{Status: "ok"}
}

func checkDirectory(path string) Check {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Check{Status: "error", Message: "目录不存在"}
		}
		return Check{Status: "error", Message: "目录不可访问"}
	}
	if !info.IsDir() {
		return Check{Status: "error", Message: "路径不是目录"}
	}
	return Check{Status: "ok"}
}
