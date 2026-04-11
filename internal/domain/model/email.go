package model

type KafkaEmailEvent struct {
	To          string `json:"to"`           // 接收人的邮箱
	Subject     string `json:"subject"`      // 邮件主题
	ContentType string `json:"content_type"` // 例如 "text/html"
	Body        string `json:"body"`         // 邮件正文内容
}
