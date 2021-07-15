package repository

// Implementation defines the interfae needed by the bot for data persistence
type Implementation interface {
	IncrementCommandCount()
	CommandCount() uint64
	SetRiotQuote(puuid string, quote string)
	RiotQuote(puuid string) string
	SetRiotVerified(puuid string, discordID string)
	RiotVerified(puuid string) string
}
