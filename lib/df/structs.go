package df

import (
	"github.com/ptdave20/donordrive"
	"time"
)

type CachedTeam struct {
	donordrive.Team
	FetchedAt time.Time `json:"fetched-at"`
}
