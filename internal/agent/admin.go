package agent

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog"

	"github.com/senhub-io/mcp-server-prtg/internal/admin"
	"github.com/senhub-io/mcp-server-prtg/internal/clientconfig"
	"github.com/senhub-io/mcp-server-prtg/internal/database"
	"github.com/senhub-io/mcp-server-prtg/internal/prtg"
	"github.com/senhub-io/mcp-server-prtg/internal/services/configuration"
	"github.com/senhub-io/mcp-server-prtg/internal/version"
)

type adminRuntime struct {
	config          *configuration.Configuration
	db              *database.DB
	toolsCount      int
	metricsFromDB   bool
	metricsFromPRTG bool
	started         time.Time
	logger          *zerolog.Logger
}

func newAdminHandler(rt adminRuntime) http.Handler {
	return admin.NewHandler(admin.Deps{
		Store:        rt.config,
		Status:       rt.status,
		TestDatabase: rt.testDatabase,
		TestPRTG:     rt.testPRTG,
		ClientConfig: rt.clientConfig,
		Restart:      rt.restart,
		Logger:       rt.logger,
	})
}

func (rt adminRuntime) status(ctx context.Context) any {
	dbStatus := map[string]any{
		"status": "not_configured",
		"host":   rt.config.GetDatabaseHost(),
		"name":   rt.config.GetDatabaseName(),
	}
	var statistics any
	var statisticsErr string
	if rt.db != nil {
		dbStatus["schema"] = rt.db.SchemaFeatures()
		if err := rt.db.Health(ctx); err != nil {
			dbStatus["status"] = "error"
			dbStatus["error"] = err.Error()
		} else {
			dbStatus["status"] = "connected"
			stats, err := rt.db.GetStatistics(ctx)
			if err != nil {
				statisticsErr = err.Error()
			} else {
				statistics = stats
			}
		}
	}
	status := map[string]any{
		"version":     version.Get(),
		"uptime":      time.Since(rt.started).Round(time.Second).String(),
		"tools_count": rt.toolsCount,
		"metrics_sources": map[string]bool{
			"database": rt.metricsFromDB,
			"prtg":     rt.metricsFromPRTG,
		},
		"database": dbStatus,
		"prtg": map[string]any{
			"configured": rt.config.IsPRTGEnabled(),
			"base_url":   rt.config.GetPRTGBaseURL(),
		},
	}
	if statistics != nil {
		status["statistics"] = statistics
	}
	if statisticsErr != "" {
		status["statistics_error"] = statisticsErr
	}
	return status
}

func (rt adminRuntime) testDatabase(ctx context.Context, s configuration.DatabaseSettings) admin.TestResult {
	password := ""
	if s.Password != nil {
		password = *s.Password
	} else {
		if !rt.config.SameDatabaseTarget(s) {
			return admin.TestResult{OK: false, Message: "Enter the database password to test another server, user or SSL mode"}
		}

		password = rt.config.StoredDatabasePassword()
	}
	connStr := configuration.ConnectionString(s.Host, s.Port, s.Name, s.User, password, s.SSLMode)
	db, err := database.New(connStr, rt.logger)
	if err != nil {
		return admin.TestResult{
			OK:      false,
			Message: "Cannot connect to the database",
			Details: []string{err.Error()},
		}
	}
	defer func() {
		if err := db.Close(); err != nil {
			rt.logger.Warn().Err(err).Msg("closing test database connection")
		}
	}()

	details := make([]string, 0, 8)
	features := db.SchemaFeatures()
	for _, name := range []string{"channel_values", "last_fetched", "collection_status", "soft_delete"} {
		yes := "no"
		if features[name] {
			yes = "yes"
		}
		details = append(details, fmt.Sprintf("%s: %s", name, yes))
	}
	stats, err := db.GetStatistics(ctx)
	if err != nil {
		return admin.TestResult{OK: false, Message: "Connected, but statistics query failed", Details: append(details, err.Error())}
	}
	details = append(details, fmt.Sprintf("sensors: %d, devices: %d, groups: %d", stats.TotalSensors, stats.TotalDevices, stats.TotalGroups))
	if stats.ExportCoverage != nil {
		details = append(details, fmt.Sprintf("channel history: %d up to date, %d stale", stats.ExportCoverage.UpToDate, stats.ExportCoverage.Stale))
	}
	if !features["channel_values"] {
		details = append(details, "This database has no channel history: check that it is the Data Exporter target database (prtg_data_exporter_collection), not the configuration database.")
	}
	return admin.TestResult{OK: true, Message: "Connected", Details: details}
}

func (rt adminRuntime) testPRTG(ctx context.Context, s configuration.PRTGSettings) admin.TestResult {
	token := ""
	if s.Token != nil {
		token = *s.Token
	} else {
		if !rt.config.SamePRTGTarget(s) {
			return admin.TestResult{OK: false, Message: "Enter the PRTG API token to test another PRTG URL or without certificate verification"}
		}

		token = rt.config.StoredPRTGToken()
	}
	if s.BaseURL == "" {
		return admin.TestResult{OK: false, Message: "Enter the PRTG URL"}
	}

	if token == "" {
		return admin.TestResult{OK: false, Message: "Enter the PRTG API token", Details: []string{"Use an API key of type Scripting with read access"}}
	}
	client, err := prtg.NewClient(prtg.ClientConfig{
		BaseURL:       s.BaseURL,
		Token:         token,
		Timeout:       time.Duration(s.TimeoutSeconds) * time.Second,
		VerifySSL:     s.VerifySSL,
		Logger:        rt.logger,
		CacheDisabled: true,
		MaxConcurrent: 1,
	})
	if err != nil {
		return admin.TestResult{OK: false, Message: "Cannot reach the PRTG API", Details: []string{err.Error()}}
	}
	if err := client.Ping(ctx); err != nil {
		return admin.TestResult{
			OK:      false,
			Message: "Cannot reach the PRTG API",
			Details: []string{
				err.Error(),
				"Check that the PRTG API v2 is active on the core server (port 1616) and that this machine can reach it",
			},
		}
	}
	counts, err := client.GetObjectCounts(ctx)
	if err != nil {
		return admin.TestResult{
			OK:      false,
			Message: "The PRTG API rejected the token",
			Details: []string{err.Error(), "Use an API key of type Scripting with read access"},
		}
	}
	return admin.TestResult{
		OK:      true,
		Message: "Connected",
		Details: []string{fmt.Sprintf("%+v", counts)},
	}
}

func (rt adminRuntime) clientConfig(host string) (string, string, error) {
	resolved := clientconfig.Host(host)
	_, port, err := net.SplitHostPort(rt.config.GetServerAddress())
	if err != nil {
		return "", resolved, fmt.Errorf("parsing server address %q: %w", rt.config.GetServerAddress(), err)
	}
	var buf strings.Builder
	if err := clientconfig.Render(&buf, resolved, port, rt.config.IsTLSEnabled(), rt.config.GetAPIKey(), rt.config.GetTLSCertFile()); err != nil {
		return "", resolved, fmt.Errorf("rendering client config: %w", err)
	}
	return buf.String(), resolved, nil
}

func (rt adminRuntime) restart() {
	rt.logger.Info().Msg("restart requested from the administration page")
	// exit code 1 so the service manager (Windows recovery / systemd Restart=always) starts the process again
	os.Exit(1)
}
