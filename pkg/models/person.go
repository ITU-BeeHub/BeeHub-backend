package models

type Person struct {
	email    string
	password string
}

func (p *Person) GetEmail() string {
	return p.email
}

func (p *Person) GetPassword() string {
	return p.password
}

func NewPerson(email, password string) *Person {
	return &Person{
		email:    email,
		password: password,
	}
}
