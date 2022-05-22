package mondb

import "github.com/fragforce/fragevents/lib/df"

func NewBaseMonitor(monName string) *BaseMonitor {
	return &BaseMonitor{
		MonitorName: monName,
	}
}

func NewTeamMonitor(teamID int) *TeamMonitor {
	return &TeamMonitor{
		BaseMonitor: NewBaseMonitor(df.MonitorNameTeam),
		TeamID:      teamID,
	}
}

func NewParticipantMonitor(participantID int) *ParticipantMonitor {
	return &ParticipantMonitor{
		BaseMonitor:   NewBaseMonitor(df.MonitorNameParticipant),
		ParticipantID: participantID,
	}
}
