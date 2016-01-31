package cross

type KnownLog struct {
	// all we care about
	Description string
	Key         string
	URL         string
}

type Config struct {
	Logs        []KnownLog
	DatabaseURI string
	StatsdURI   string
}
