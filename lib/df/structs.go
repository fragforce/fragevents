package df

import (
	"github.com/ptdave20/donordrive"
	"time"
)

type CachedTeam struct {
	donordrive.Team `json:"team"`
	FetchedAt       time.Time `json:"fetched-at"` // Use team.GetFetchedAt()
	RawData         []byte    `json:"-"`          // Raw copy of data - just in case
}

func (c *CachedTeam) GetFetchedAt() string {
	return c.FetchedAt.UTC().Format(time.RFC3339Nano)
}
