package dtos

type CouponResponse struct {
	Code               string `json:"code"`
	InfluencerUsername string `json:"influencer_username"`
	Quota              int16  `json:"quota"`
	CreatedAt          string `json:"created_at"`
}
