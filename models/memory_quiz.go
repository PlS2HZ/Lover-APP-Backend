package models

import "time"

type Memory struct {
	ID         string    `json:"id" db:"id"`
	UserID     string    `json:"user_id" db:"user_id"`
	Category   string    `json:"category" db:"category"`
	Content    string    `json:"content" db:"content"`
	HappenedAt string    `json:"happened_at" db:"happened_at"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
}

type QuizQuestion struct {
	Question     string   `json:"question"`
	Options      []string `json:"options"`
	AnswerIndex  int      `json:"answer_index"`
	SweetComment string   `json:"sweet_comment"`
}
