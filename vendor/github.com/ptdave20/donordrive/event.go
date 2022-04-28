package donordrive

import "time"

type Event struct {
	EventId         int               `json:"eventId"`
	Type            string            `json:"type"`
	Name            string            `json:"name"`
	Links           map[string]string `json:"links"`
	EndDateUTC      time.Time         `json:"endDateUTC"`
	Venue           string            `json:"venue"`
	City            string            `json:"city"`
	Country         string            `json:"country"`
	Province        string            `json:"province"`
	Timezone        string            `json:"timezone"`
	StartDateUTC    time.Time         `json:"startDateUTC"`
	NumParticipants int               `json:"numParticipants"`
	NumTeams        int               `json:"numTeams'"`
}
