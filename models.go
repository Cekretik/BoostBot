package main

import (
	"gorm.io/gorm"
)

type UserState struct {
	gorm.Model
	UserID     int64      `gorm:"column:user_id" json:"user_id"`
	UserName   string     `gorm:"column:user_name" json:"user_name"`
	Subscribed bool       `gorm:"column:subscribed" json:"subscribed"`
	ChannelID  int64      `gorm:"column:channel_id" json:"channel_id"`
	Balance    float64    `gorm:"column:balance" json:"balance"`
	Favorites  []Services `gorm:"many2many:user_favorites;"`
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

type Services struct {
	gorm.Model
	ID         int         `gorm:"column:id" json:"id"`
	Name       string      `gorm:"column:name" json:"name"`
	CategoryID string      `gorm:"column:category_id" json:"categoryId"`
	Min        int         `gorm:"column:min" json:"min"`
	Max        int         `gorm:"column:max" json:"max"`
	Dripfeed   bool        `gorm:"column:dripfeed" json:"dripfeed"`
	Refill     bool        `gorm:"column:refill" json:"refill"`
	Cancel     bool        `gorm:"column:cancel" json:"cancel"`
	ServiceID  string      `gorm:"column:service_id" json:"serviceId"`
	Rate       float64     `gorm:"column:rate" json:"rate"`
	Type       string      `gorm:"column:type" json:"type"`
	Users      []UserState `gorm:"many2many:user_favorites;"`
}

// Struct for POST orders
type Order struct {
	ID           int    `json:"id"`
	ChatID       string `json:"userId"`
	ServiceID    string `json:"serviceId"`
	Link         string `json:"link"`
	Quantity     int    `json:"quantity"`
	Keywords     string `json:"keywords"`
	Comments     string `json:"comments"`
	Usernames    string `json:"usernames"`
	Hashtags     string `json:"hashtags"`
	Hashtag      string `json:"hashtag"`
	Username     string `json:"username"`
	AnswerNumber int    `json:"answerNumber"`
	Min          int    `json:"min"`
	Max          int    `json:"max"`
	Delay        int    `json:"delay"`
}

// Struct of users who have orders
type ServiceDetails struct {
	ID          int     `json:"id"`
	ServiceID   int     `json:"serviceId"`
	Cost        float64 `json:"cost"`
	ServiceType string  `json:"serviceType"`
	Link        string  `json:"link"`
	Quantity    int     `json:"quantity"`
	Status      string  `json:"status"`
	Charge      float64 `json:"charge"`
	StartCount  int     `json:"startCount"`
	Remains     int     `json:"remains"`
}

type UserOrders struct {
	gorm.Model
	ChatID      string  `gorm:"column:user_id" json:"userId"`
	OrderID     int     `gorm:"column:order_id" json:"id"`
	ServiceID   string  `gorm:"column:service_id" json:"serviceId"`
	Cost        float64 `gorm:"column:cost" json:"cost"`
	ServiceType string  `gorm:"column:service_type" json:"serviceType"`
	Link        string  `gorm:"column:link" json:"link"`
	Quantity    int     `gorm:"column:quantity" json:"quantity"`
	Status      string  `gorm:"column:status" json:"status"`
	Charge      float64 `gorm:"column:charge" json:"charge"`
	StartCount  int     `gorm:"column:start_count" json:"startCount"`
	Remains     int     `gorm:"column:remains" json:"remains"`
}

type RefundedOrder struct {
	OrderID uint `gorm:"primaryKey"`
}

type Payments struct {
	ChatID  int     `gorm:"column:user_id" json:"userId"`
	OrderID string  `gorm:"column:order_id" json:"order_id"`
	Amount  float64 `gorm:"column:amount" json:"amount"`
	Url     string  `gorm:"column:url" json:"url"`
	Status  string  `gorm:"column:status" json:"status"`
	Type    string  `gorm:"column:type" json:"type"`
}
