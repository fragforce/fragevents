package df

// Keep these in here to help avoid circular deps

const (
	// 	Monitor types - Names
	MonitorNameTeam        = "Team"
	MonitorNameParticipant = "Participant"
	// 	Request Types - URL Pattern
	RTypeTeam        = "team"
	RTypeParticipant = "participant"
	//	Redis Pool Names & Redis Database Numbers
	//	The defaults are set in `df/redis.go`
	RPoolGroupCache   = "groupcache"
	RPoolGroupCacheDB = 2
	RPoolMonitoring   = "monitoring"
	RPoolMonitoringDB = 3
	//	Kafka Header Keys
	KHeaderKeyTeamID    = "team-id"
	KHeaderKeyTeamName  = "team-name"
	KHeaderKeyEventID   = "event-id"
	KHeaderKeyEventName = "event-name"
	KHeaderKeyFetchedAt = "fetched-at"
)
