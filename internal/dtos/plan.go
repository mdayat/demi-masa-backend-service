package dtos

type PlanResponse struct {
	Id               string `json:"id"`
	Type             string `json:"type"`
	Name             string `json:"name"`
	Price            int32  `json:"price"`
	DurationInMonths int16  `json:"duration_in_months"`
	CreatedAt        string `json:"created_at"`
}
