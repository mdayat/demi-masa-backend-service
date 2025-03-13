package dtos

type SubscriptionResponse struct {
	Id        string `json:"id"`
	PlanId    string `json:"plan_id"`
	PaymentId string `json:"payment_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}
