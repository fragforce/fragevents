package df

import (
	"encoding/json"
	"github.com/ptdave20/donordrive"
	"time"
)

type CachedTeam struct {
	donordrive.Team `json:"team"`
	FetchedAt       time.Time `json:"fetched-at"` // Use team.GetFetchedAt()
	RawData         []byte    `json:"-"`          // Raw copy of json data - if we already have it
	RawTeamData     []byte    `json:"-"`          // Raw copy of json data - if we already have it - For team only, not cached
}

func (c *CachedTeam) GetFetchedAt() string {
	return c.FetchedAt.UTC().Format(time.RFC3339Nano)
}

//GetRawData fetches the raw data, recreating if not set
func (c *CachedTeam) GetRawData() ([]byte, error) {
	if c.RawData == nil {
		raw, err := json.Marshal(c)
		if err != nil {
			return nil, err
		}
		c.RawData = raw
	}
	return c.RawData, nil
}

//GetRawTeamData fetches the data for the team only
func (c *CachedTeam) GetRawTeamData() ([]byte, error) {
	if c.RawTeamData == nil {
		raw, err := json.Marshal(c.Team)
		if err != nil {
			return nil, err
		}
		c.RawTeamData = raw
	}
	return c.RawTeamData, nil
}

type CachedParticipants struct {
	Participants []donordrive.Participant `json:"participants"`
	Count        int                      `json:"count"`      // Number of participants
	FetchedAt    time.Time                `json:"fetched-at"` // Use team.GetFetchedAt()
	RawData      []byte                   `json:"-"`          // Raw copy of json data - if we already have it
}

func (c *CachedParticipants) GetFetchedAt() string {
	return c.FetchedAt.UTC().Format(time.RFC3339Nano)
}

//GetRawData fetches the raw data, recreating if not set
func (c *CachedParticipants) GetRawData() ([]byte, error) {
	if c.RawData == nil {
		raw, err := json.Marshal(c)
		if err != nil {
			return nil, err
		}
		c.RawData = raw
	}
	return c.RawData, nil
}

type CachedParticipant struct {
	donordrive.Participant `json:"participant"`
	FetchedAt              time.Time `json:"fetched-at"` // Use team.GetFetchedAt()
	RawData                []byte    `json:"-"`          // Raw copy of json data - if we already have it
	RawParticipantData     []byte    `json:"-"`          // Raw copy of json data - if we already have it - For Participant only, not cached
}

func (c *CachedParticipant) GetFetchedAt() string {
	return c.FetchedAt.UTC().Format(time.RFC3339Nano)
}

//GetRawData fetches the raw data, recreating if not set
func (c *CachedParticipant) GetRawData() ([]byte, error) {
	if c.RawData == nil {
		raw, err := json.Marshal(c)
		if err != nil {
			return nil, err
		}
		c.RawData = raw
	}
	return c.RawData, nil
}

//GetRawParticipantData fetches the data for the team only
func (c *CachedParticipant) GetRawParticipantData() ([]byte, error) {
	if c.RawParticipantData == nil {
		raw, err := json.Marshal(c.Participant)
		if err != nil {
			return nil, err
		}
		c.RawParticipantData = raw
	}
	return c.RawParticipantData, nil
}
