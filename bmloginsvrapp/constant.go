package main

const (
	loginopstart = 10000
)

type UserLoginExtendInfo struct {
	DonateMoney int32 `json:"DonateMoney"`
	DonateLeft  int32 `json:"DonateLeft"`
	SystemGift  []int `json:"SystemGift"`
}
