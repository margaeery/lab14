package main

type Writer interface {
	Write(value any) error
	Flush() error
	Close() error
}

type LeagueWriter interface {
	Write(league League) error
}
