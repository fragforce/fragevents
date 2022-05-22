package df

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
)
