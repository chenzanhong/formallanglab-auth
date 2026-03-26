package model

import "time"

/* 用户 */
type User struct {
	ID        int64     `json:"id" gorm:"primarykey"`
	Name      string    `json:"name" gorm:"column:name"`
	Password  string    `json:"password" gorm:"column:password"`
	Email     string    `json:"email" gorm:"column:email"`
	Role      string    `json:"role" gorm:"column:role"`
	CreatedAt time.Time `json:"created_at" gorm:"column:created_at"`
}

func (u *User) TableName() string {
	return "users"
}
