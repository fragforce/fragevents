package mondb

import (
	"encoding/json"
	"github.com/fragforce/fragevents/lib/df"
)

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

func NewTeamMonitorFromJSON(data []byte) (*TeamMonitor, error) {
	ret := TeamMonitor{}
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}

func NewParticipantMonitorFromJSON(data []byte) (*ParticipantMonitor, error) {
	ret := ParticipantMonitor{}
	if err := json.Unmarshal(data, &ret); err != nil {
		return nil, err
	}
	return &ret, nil
}
