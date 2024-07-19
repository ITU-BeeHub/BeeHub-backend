package models

// PersonDTO struct with selected attributes
type PersonDTO struct {
	Email        string `json:"email"`
	First_name   string `json:"first_name"`
	Last_name    string `json:"last_name"`
	Faculty      string `json:"faculty"`
	Department   string `json:"department"`
	GPA          string `json:"gpa"`
	Photo_base64 string `json:"photo"`
	Class        string `json:"class"`
}

func ToPersonDTO(person Person) PersonDTO {
	return PersonDTO{
		Email:        person.Email,
		First_name:   person.First_name,
		Last_name:    person.Last_name,
		Faculty:      person.Faculty,
		Department:   person.Department,
		GPA:          person.GPA,
		Photo_base64: person.Photo_base64,
		Class:        person.Class,
	}
}
