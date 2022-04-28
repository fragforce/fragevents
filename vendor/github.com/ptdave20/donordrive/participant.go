package donordrive

type Participant struct {
	AvatarImageURL  string            `json:"avatarImageURL"`
	CampaignDate    string            `json:"campaignDate"`
	CampaignName    string            `json:"campaignName"`
	CreatedDateUTC  string            `json:"createdDateUTC"`
	DisplayName     string            `json:"displayName"`
	EventId         int               `json:"eventId"`
	EventName       string            `json:"eventName"`
	FundraisingGoal float64           `json:"fundraisingGoal"`
	IsTeamCaptain   bool              `json:"isTeamCaptain"`
	NumAwardBadges  int               `json:"numAwardBadges"`
	Links           map[string]string `json:"links"`
	ParticipantId   int               `json:"participantId"`
	StreamIsLive    bool              `json:"streamIsLive"`
	SumDonations    float64           `json:"sumDonations"`
	SumPledges      float64           `json:"sumPledges"`
	TeamId          int               `json:"teamId"`
	TeamName        string            `json:"teamName"`
}
