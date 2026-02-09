package options

import (
	"flag"

	"github.com/cetteup/playerpath/cmd/importer/internal/registry"
)

type Options struct {
	Version bool

	Debug        bool
	ColorizeLogs bool

	ConfigPath string
	Source     string

	BatchSize int
}

func Init() *Options {
	opts := new(Options)
	flag.BoolVar(&opts.Version, "v", false, "prints the version")
	flag.BoolVar(&opts.Version, "version", false, "prints the version")
	flag.BoolVar(&opts.Debug, "debug", false, "enable debug logging")
	flag.BoolVar(&opts.ColorizeLogs, "colorize-logs", false, "colorize log messages")
	flag.StringVar(&opts.ConfigPath, "config", "config.yaml", "path to YAML config file")
	flag.StringVar(&opts.Source, "source", registry.BaseURL, "bf2.co registry API base URL or path to bf2opendata folder")
	flag.IntVar(&opts.BatchSize, "batch", 1024, "number of players to read from bf2opendata before upserting batch to database")
	flag.Parse()
	return opts
}
