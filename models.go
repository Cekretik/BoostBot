package main

import (
	"time"

	"gorm.io/gorm"
)

type UserState struct {
	gorm.Model
	UserID     int64 `gorm:"column:user_id" json:"user_id"`
	Subscribed bool  `gorm:"column:subscribed" json:"subscribed"`
	ChannelID  int64 `gorm:"column:channel_id" json:"channel_id"`
}

type Category struct {
	gorm.Model
	Name string `gorm:"column:name" json:"name"`
	ID   string `gorm:"column:category_id" json:"id"`
}

type Subcategory struct {
	gorm.Model
	Name       string `gorm:"column:name" json:"name"`
	ID         string `gorm:"column:subcategory_id" json:"id"`
	CategoryID string `gorm:"column:category_id" json:"categoryId"`
}

type Service struct {
	gorm.Model
	ID               int        `gorm:"column:id" json:"id"`
	Name             string     `gorm:"column:name" json:"name"`
	CategoryID       string     `gorm:"column:category_id" json:"categoryId"`
	Min              int        `gorm:"column:min" json:"min"`
	Max              int        `gorm:"column:max" json:"max"`
	Dripfeed         bool       `gorm:"column:dripfeed" json:"dripfeed"`
	Refill           bool       `gorm:"column:refill" json:"refill"`
	Cancel           bool       `gorm:"column:cancel" json:"cancel"`
	ServiceID        string     `gorm:"column:service_id" json:"serviceId"`
	Rate             float64    `gorm:"column:rate" json:"rate"`
	Type             string     `gorm:"column:type" json:"type"`
	ServiceType      string     `gorm:"column:service_type" json:"serviceType"`
	AverageTimestamp *time.Time `gorm:"column:average_timestamp" json:"averageTimestamp"`
	CreatedAt        time.Time  `gorm:"column:created_at" json:"createdAt"`
	UpdatedAt        time.Time  `gorm:"column:updated_at" json:"updatedAt"`
}
