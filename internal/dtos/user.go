package dtos

type UserRequest struct {
	Email     string `json:"email" validate:"omitempty,email"`
	Password  string `json:"password" validate:"omitempty,min=8"`
	Username  string `json:"username" validate:"omitempty,min=2"`
	Latitude  string `json:"latitude" validate:"omitempty,latitude"`
	Longitude string `json:"longitude" validate:"omitempty,longitude"`
}

type UserSubscription struct {
	Id        string `json:"id"`
	PlanId    string `json:"plan_id"`
	PaymentId string `json:"payment_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type UserResponse struct {
	Id           string            `json:"id"`
	Email        string            `json:"email"`
	Name         string            `json:"name"`
	Latitude     float64           `json:"latitude"`
	Longitude    float64           `json:"longitude"`
	City         string            `json:"city"`
	Timezone     string            `json:"timezone"`
	CreatedAt    string            `json:"created_at"`
	Subscription *UserSubscription `json:"subscription"`
}
