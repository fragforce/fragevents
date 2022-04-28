package donordrive

type Donation struct {
	Amount         float64 `json:"amount"`
	AvatarImageURL string  `json:"avatarImageURL"`
	CreatedDateUTC string  `json:"createdDateUTC"`
	DisplayName    string  `json:"displayName"`
	DonationID     string  `json:"donationID"`
	DonorID        string  `json:"donorId"`
	EventID        int     `json:"eventID"`
	IncentiveID    string  `json:"incentiveID"`
	Message        string  `json:"message"`
	ParticipantID  int     `json:"participantID"`
	TeamID         int     `json:"teamID"`
}
