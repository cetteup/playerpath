package main

import (
	"fmt"
	"os"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/cetteup/playerpath/cmd/playerpath/internal/config"
	"github.com/cetteup/playerpath/cmd/playerpath/internal/handler"
	"github.com/cetteup/playerpath/cmd/playerpath/internal/options"
	"github.com/cetteup/playerpath/cmd/playerpath/modify"
	"github.com/cetteup/playerpath/internal/database"
	"github.com/cetteup/playerpath/internal/domain/player/sql"
	"github.com/cetteup/playerpath/internal/domain/provider"
)

var (
	buildVersion = "development"
	buildCommit  = "uncommitted"
	buildTime    = "unknown"
)

func main() {
	version := fmt.Sprintf("playerpath %s (%s) built at %s", buildVersion, buildCommit, buildTime)
	opts := options.Init()

	// Print version and exit
	if opts.Version {
		fmt.Println(version)
		os.Exit(0)
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out:        os.Stdout,
		NoColor:    !opts.ColorizeLogs,
		TimeFormat: time.RFC3339,
	})
	if opts.Debug {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}

	cfg, err := config.LoadConfig(opts.ConfigPath)
	if err != nil {
		log.Fatal().
			Err(err).
			Str("config", opts.ConfigPath).
			Msg("Failed to read config file")
	}

	db := database.Connect(
		cfg.Database.Hostname,
		cfg.Database.DatabaseName,
		cfg.Database.Username,
		cfg.Database.Password,
	)
	defer func() {
		err2 := db.Close()
		if err2 != nil {
			log.Error().
				Err(err2).
				Msg("Failed to close database connection")
		}
	}()

	servers := make(map[string]provider.Provider, len(cfg.Servers))
	for _, server := range cfg.Servers {
		servers[server.IP] = server.Provider
	}

	repository := sql.NewRepository(db)
	h := handler.NewHandler(repository, servers, opts.Provider)
	h.WithModifier(
		modify.HostRequestModifier{},
		modify.InfoQueryRequestModifier{},
		modify.VerificationResponseModifier{},
	)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Use(middleware.Recover())
	e.Use(middleware.TimeoutWithConfig(middleware.TimeoutConfig{
		Timeout: time.Second * 10,
	}))
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogError:     true,
		LogRemoteIP:  true,
		LogMethod:    true,
		LogURI:       true,
		LogStatus:    true,
		LogLatency:   true,
		LogUserAgent: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			log.Info().
				Err(v.Error).
				Str("remote", v.RemoteIP).
				Str("method", v.Method).
				Str("URI", v.URI).
				Int("status", v.Status).
				Str("latency", v.Latency.Truncate(time.Millisecond).String()).
				Str("agent", v.UserAgent).
				Any("provider", c.Get("provider")).
				Msg("request")

			return nil
		},
	}))

	asp := e.Group("/ASP")
	// Requests forwarded based on player provider
	asp.GET("/getplayerinfo.aspx", h.HandleDynamicForward)
	asp.GET("/getawardsinfo.aspx", h.HandleDynamicForward)
	asp.GET("/getunlocksinfo.aspx", h.HandleDynamicForward)
	asp.GET("/getrankinfo.aspx", h.HandleDynamicForward)
	asp.GET("/VerifyPlayer.aspx", h.HandleDynamicForward)
	// Fallback forward to default provider
	asp.Any("/*.aspx", h.HandleStaticForward)

	// API routes
	api := e.Group("/api")
	api.GET("/player/:pid", h.HandleGetPlayer)

	e.Logger.Fatal(e.Start(opts.ListenAddr))
}
