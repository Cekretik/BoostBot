package main

import (
	"gorm.io/gorm"
)

type UserState struct {
	gorm.Model
	UserID     int64 `gorm:"column:user_id" json:"user_id"`
	Subscribed bool  `gorm:"column:subscribed" json:"subscribed"`
	ChannelID  int64 `gorm:"column:channel_id" json:"channel_id"`
}
