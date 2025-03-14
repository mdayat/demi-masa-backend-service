package dtos

type PrayerRequest struct {
	Status string `json:"status" validate:"omitempty,oneof=pending on_time late missed"`
}

type PrayerResponse struct {
	Id     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
	Year   int16  `json:"year"`
	Month  int16  `json:"month"`
	Day    int16  `json:"day"`
}
