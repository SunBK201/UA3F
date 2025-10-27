package config

import "flag"

type Config struct {
	BindAddr             string
	Port                 int
	LogLevel             string
	PayloadUA            string
	UAPattern            string
	EnablePartialReplace bool
}

func Parse() (*Config, bool) {
	var (
		bindAddr  string
		port      int
		loglevel  string
		payloadUA string
		uaPattern string
		partial   bool
		showVer   bool
	)

	flag.StringVar(&bindAddr, "b", "127.0.0.1", "bind address (default: 127.0.0.1)")
	flag.IntVar(&port, "p", 1080, "port")
	flag.StringVar(&payloadUA, "f", "FFF", "User-Agent")
	flag.StringVar(&uaPattern, "r", "", "UA-Pattern")
	flag.BoolVar(&partial, "s", false, "Enable Regex Partial Replace")
	flag.StringVar(&loglevel, "l", "info", "Log level (default: info)")
	flag.BoolVar(&showVer, "v", false, "show version")
	flag.Parse()

	cfg := &Config{
		BindAddr:             bindAddr,
		Port:                 port,
		LogLevel:             loglevel,
		PayloadUA:            payloadUA,
		UAPattern:            uaPattern,
		EnablePartialReplace: partial,
	}
	return cfg, showVer
}
