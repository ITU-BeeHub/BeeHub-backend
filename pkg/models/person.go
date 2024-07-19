package models

import "time"

type Person struct {
	Email             string    `json:"email"`
	Password          string    `json:"password"`
	First_name        string    `json:"first_name"`
	Last_name         string    `json:"last_name"`
	Photo_base64      string    `json:"photo_location"`
	Token             string    `json:"token"`
	Class             string    `json:"class"`
	Faculty           string    `json:"faculty"`
	Department        string    `json:"departmant"`
	Gender            string    `json:"gender"`
	GPA               string    `json:"gpa"`
	Transcript_base64 string    `json:"transcript"`
	LoginTime         time.Time `json:"loginTime"`
}
