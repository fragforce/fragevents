package donordrive

type Milestone struct {
	Description     string            `json:"description"`
	FundraisingGoal float64           `json:"fundraisingGoal"`
	IsActive        bool              `json:"isActive"`
	IsComplete      bool              `json:"isComplete"`
	Links           map[string]string `json:"links"`
	MilestoneID     string            `json:"milestoneID"`
	EndDateUTC      string            `json:"endDateUTC"`
	StartDateUTC    string            `json:"startDateUTC"`
}
