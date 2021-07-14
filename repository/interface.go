package repository

// Implementation defines the interfae needed by the bot for data persistence
type Implementation interface {
	IncrementCommandCount()
	CommandCount() uint64
	SetRiotQuote(string, string)
	RiotQuote(string) string
	SetRiotVerified(string, string)
	// TODO: this should only take the summoner ID after cleaning up verify
	// verify should not require any args. Check the memory map - if discrod ID in there, then they are verifying
	// if discord ID isn't there, send them the message and a code if they supplied args
	RiotVerified(string, string) bool
}
