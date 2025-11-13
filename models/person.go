package models

import "time"

type Person struct {
	ID           uint      `json:"id,omitempty"`
	Name         string    `json:"name"`
	LastName     string    `json:"last_name"`
	Email        string    `json:"email"`
	Password     string    `json:"password"`
	Description  string    `json:"description,omitempty"`
	ProfileImage string    `json:"profile_image,omitempty"`
	Phone        string    `json:"phone,omitempty"`
	Country      string    `json:"country,omitempty"`
	Linkedin     string    `json:"linkedin,omitempty"`
	Github       string    `json:"github,omitempty"`
	Portfolio    string    `json:"portfolio,omitempty"`
	IsMentor     bool      `json:"is_mentor,omitempty"`
	IsMentee     bool      `json:"is_mentee,omitempty"`
	IsActive     bool      `json:"is_active,omitempty"`
	CreatedAt    time.Time `json:"created_at,omitempty"`
	UpdatedAt    time.Time `json:"updated_at,omitempty"`
}
