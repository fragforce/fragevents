package mondb

type BaseMonitor struct {
	MonitorName string `json:"monitor-name"`
}

type TeamMonitor struct {
	*BaseMonitor
	TeamID int `json:"team-id"`
}

type ParticipantMonitor struct {
	*BaseMonitor
	ParticipantID int `json:"participant-id"`
}
