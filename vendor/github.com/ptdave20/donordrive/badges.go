package donordrive

type Badge struct {
	BadgeCode       string `json:"badgeCode"`
	BadgeImageURL   string `json:"badgeImageURL"`
	Description     string `json:"description"`
	Title           string `json:"title"`
	UnlockedDateUTC string `json:"unlockedDateUTC"`
}
