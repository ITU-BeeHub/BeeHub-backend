package pkg

import (
	"sync"
	"time"

	"github.com/ITU-BeeHub/BeeHub-backend/pkg/models"
)

type PersonManager struct {
	mu     sync.Mutex
	person *models.Person
}

func NewPersonManager() *PersonManager {
	return &PersonManager{
		person: &models.Person{},
	}
}

func (pm *PersonManager) GetPerson() *models.Person {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.person
}

func (pm *PersonManager) UpdatePerson(person *models.Person) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.person = person
}

func (pm *PersonManager) UpdateToken(token string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.person.Token = token
}

func (pm *PersonManager) UpdateLoginTime() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.person.LoginTime = time.Now()
}

func (pm *PersonManager) GetLoginTime() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.person.LoginTime.String()
}

func (pm *PersonManager) GetToken() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.person.Token
}
func (pm *PersonManager) GetEmail() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.person.Email
}
func (pm *PersonManager) SetEmail(mail string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.person.Email = mail
}

func (pm *PersonManager) GetPhoto() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.person.Photo_base64
}

func (pm *PersonManager) GetClass() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.person.Class
}

func (pm *PersonManager) GetFirstName() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.person.First_name
}

func (pm *PersonManager) GetLastName() string {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	return pm.person.Last_name
}

func (pm *PersonManager) SetPhoto(photo string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.person.Photo_base64 = photo
}

func (pm *PersonManager) SetClass(class string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.person.Class = class
}

func (pm *PersonManager) SetFirstName(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.person.First_name = name
}

func (pm *PersonManager) SetLastName(name string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.person.Last_name = name
}

func (pm *PersonManager) SetPassword(password string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.person.Password = password
}
