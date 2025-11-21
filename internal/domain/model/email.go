package model

type KafkaEmailEvent struct {
	To          string `json:"to"`
	Subject     string `json:"subject"`
	ContentType string `json:"content_type"` // e.g., "text/html"
	Body        string `json:"body"`
	// Type        EmailType `json:"type"` // e.g., "register", "reset"
	Timestamp int64 `json:"timestamp"`
}

// type EmailType string

// const (
// 	EmailTypeRegister EmailType = "register"
// 	EmailTypeResetPwd EmailType = "resetPwd"
// )
