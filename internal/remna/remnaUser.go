package remna

import "time"

type RemnaUserRequest struct {
	Username             string     `json:"username,omitempty"`
	UUID                 string     `json:"uuid,omitempty"`
	Status               string     `json:"status,omitempty"`
	Email                string     `json:"email,omitempty"`
	Tag                  string     `json:"tag,omitempty"`
	TelegramId           uint       `json:"telegramId,omitempty"`
	HwidDeviceLimit      uint       `json:"hwidDeviceLimit,omitempty"`
	ExpireAt             *time.Time `json:"expireAt,omitempty"`
	ActiveInternalSquads []string   `json:"activeInternalSquads,omitempty"`
}

type RemnaUserResponse struct {
	UUID                 string          `json:"uuid"`
	Username             string          `json:"username"`
	Status               string          `json:"status"`
	ExpireAt             time.Time       `json:"expireAt"`
	TelegramId           uint            `json:"telegramId"`
	Email                string          `json:"email"`
	HwidDeviceLimit      uint            `json:"hwidDeviceLimit"`
	SubscriptionUrl      string          `json:"subscriptionUrl"`
	Tag                  string          `json:"tag"`
	TrafficLimitStrategy string          `json:"traficLimitStrategy"`
	ActiveInternalSquads []InternalSquad `json:"activeInternalSquads"`
}

type InternalSquad struct {
	UUID string `json:"uuid"`
	Name string `json:"name"`
}

type AllUserResponse struct {
	Total uint
	Users []RemnaUserResponse
}

type AllUserResponseRoot struct {
	Response AllUserResponse
}
