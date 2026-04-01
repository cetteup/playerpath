package options

import (
	"flag"
	"time"
)

type Options struct {
	Version bool

	Debug        bool
	ColorizeLogs bool

	ConfigPath string

	Interval  time.Duration
	BatchSize int
}

func Init() *Options {
	opts := new(Options)
	flag.BoolVar(&opts.Version, "v", false, "prints the version")
	flag.BoolVar(&opts.Version, "version", false, "prints the version")
	flag.BoolVar(&opts.Debug, "debug", false, "enable debug logging")
	flag.BoolVar(&opts.ColorizeLogs, "colorize-logs", false, "colorize log messages")
	flag.StringVar(&opts.ConfigPath, "config", "config.yaml", "path to YAML config file")
	flag.DurationVar(&opts.Interval, "interval", 5*time.Minute, "interval for importing players")
	flag.IntVar(&opts.BatchSize, "batch", 1000, "number of players to batch-upsert to database")
	flag.Parse()
	return opts
}
