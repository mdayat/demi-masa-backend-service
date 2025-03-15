package dtos

type CreateInvoiceRequest struct {
	CouponCode    string `json:"coupon_code"`
	CustomerName  string `json:"customer_name" validate:"required"`
	CustomerEmail string `json:"customer_email" validate:"required,email"`
	Plan          struct {
		Id               string `json:"id" validate:"required,uuid"`
		Type             string `json:"type" validate:"required"`
		Name             string `json:"name" validate:"required"`
		Price            int    `json:"price" validate:"required"`
		DurationInMonths int    `json:"duration_in_months" validate:"required"`
	} `json:"plan"`
}

type InvoiceResponse struct {
	Id          string `json:"id"`
	PlanId      string `json:"plan_id"`
	RefId       string `json:"ref_id"`
	CouponCode  string `json:"coupon_code"`
	TotalAmount int32  `json:"total_amount"`
	QrUrl       string `json:"qr_url"`
	ExpiresAt   string `json:"expires_at"`
	CreatedAt   string `json:"created_at"`
}

type PaymentResponse struct {
	Id         string `json:"id"`
	InvoiceId  string `json:"invoice_id"`
	AmountPaid int32  `json:"amount_paid"`
	Status     string `json:"status"`
	CreatedAt  string `json:"created_at"`
}

type TripayCallbackRequest struct {
	Reference   string `json:"reference"`
	MerchantRef string `json:"merchant_ref"`
	TotalAmount int    `json:"total_amount"`
	Status      string `json:"status"`
}

type TripayOrderItem struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	Name     string `json:"name"`
	Price    int    `json:"price"`
	Quantity int    `json:"quantity"`
}

type TripayTransactionRequest struct {
	Method        string            `json:"method"`
	MerchantRef   string            `json:"merchant_ref"`
	Amount        int               `json:"amount"`
	CustomerName  string            `json:"customer_name"`
	CustomerEmail string            `json:"customer_email"`
	OrderItems    []TripayOrderItem `json:"order_items"`
	Signature     string            `json:"signature"`
}

type TripayTransactionResponse struct {
	Reference   string `json:"reference"`
	Amount      int    `json:"amount"`
	ExpiredTime int    `json:"expired_time"`
	QrURL       string `json:"qr_url"`
}
