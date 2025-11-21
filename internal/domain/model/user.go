package model

/* 用户 */
type User struct {
	ID       int64  `json:"id" gorm:"primarykey"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Token    string `json:"token"`
	Email    string `json:"email"`
}
