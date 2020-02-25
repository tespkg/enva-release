package envs

import "tespkg.in/kit/log"

type Args struct {
	ListenAddr string

	Dsn string

	// The logging options to use
	LoggingOptions *log.Options
}

func DefaultArgs() *Args {
	return &Args{
		ListenAddr: ":9112",
		Dsn:        "http://localhost:8502/envs",

		LoggingOptions: log.DefaultOptions(),
	}
}
