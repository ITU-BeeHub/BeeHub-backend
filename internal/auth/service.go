package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/ITU-BeeHub/BeeHub-backend/pkg"
	models "github.com/ITU-BeeHub/BeeHub-backend/pkg/models"
	"golang.org/x/net/html"
)

const token_url = "https://kepler-beta.itu.edu.tr/ogrenci/auth/jwt"
const photo_url = "https://kepler-beta.itu.edu.tr/api/ogrenci/OgrenciFotograf"
const gpa_and_grade_url = "https://kepler-beta.itu.edu.tr/api/ogrenci/AkademikDurum/759"
const personal_info_url = "https://kepler-beta.itu.edu.tr/api/ogrenci/KisiselBilgiler/"
const transcript_url = "https://kepler-beta.itu.edu.tr/api/ogrenci/Belgeler/TranskriptIngilizceOnizleme"

type Service struct {
	personManager *pkg.PersonManager
}

func NewService(personManager *pkg.PersonManager) *Service {
	return &Service{personManager: personManager}
}

func (s *Service) LoginService(email, password string) (string, error) {

	// Cookie jar oluştur
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}

	// HTTP client oluştur ve cookie jar ekle
	client := &http.Client{
		Jar: jar,
	}

	// İlk GET isteği için headers tanımla
	req, err := http.NewRequest("GET", "https://kepler-beta.itu.edu.tr", nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "BeeHub")
	// İstek gönder
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Yönlendirme URL'sini bul
	if len(resp.Request.Response.Request.URL.String()) == 0 {
		fmt.Println("No redirect found")
	}
	loginURL := resp.Request.Response.Request.URL.String()

	// İlk GET isteği için headers tanımla
	req, err = http.NewRequest("GET", loginURL, nil)
	if err != nil {
		log.Fatal(err)
	}

	// İstek gönder
	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// HTML'i ayrıştır
	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// HTML'den form verilerini çıkar
	var viewstate, viewstategenerator, eventvalidation string
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			for _, attr := range n.Attr {
				if attr.Key == "name" {
					switch attr.Val {
					case "__VIEWSTATE":
						for _, a := range n.Attr {
							if a.Key == "value" {
								viewstate = a.Val
							}
						}
					case "__VIEWSTATEGENERATOR":
						for _, a := range n.Attr {
							if a.Key == "value" {
								viewstategenerator = a.Val
							}
						}
					case "__EVENTVALIDATION":
						for _, a := range n.Attr {
							if a.Key == "value" {
								eventvalidation = a.Val
							}
						}
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	// Form verilerini ve POST isteği için headers tanımla
	formData := url.Values{
		"__EVENTTARGET":                        {""},
		"__EVENTARGUMENT":                      {""},
		"__VIEWSTATE":                          {viewstate},
		"__VIEWSTATEGENERATOR":                 {viewstategenerator},
		"__EVENTVALIDATION":                    {eventvalidation},
		"ctl00$ContentPlaceHolder1$hfAppName":  {"Öğrenci Bilgi Sistemi"},
		"ctl00$ContentPlaceHolder1$tbUserName": {email},
		"ctl00$ContentPlaceHolder1$tbPassword": {password},
		"ctl00$ContentPlaceHolder1$btnLogin":   {"Giriş / Login"},
	}

	req, err = http.NewRequest("POST", loginURL, strings.NewReader(formData.Encode()))
	if err != nil {
		log.Fatal(err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// POST isteğini gönder
	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Yanıtı kontrol et
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("bad status")
	}

	req, err = http.NewRequest("GET", token_url, nil)
	if err != nil {
		log.Fatal(err)
	}

	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// check if user is logged in
	if !isLoggedIn(body) {
		return "", fmt.Errorf("login failed")
	}
	person := s.personManager.GetPerson()
	s.personManager.SetEmail(email)
	s.personManager.SetPassword(password)
	s.personManager.UpdateLoginTime()
	s.personManager.UpdateToken(string(body))
	s.personManager.UpdatePerson(person)
	return string(body), nil

}

// Returns true if user is logged in
//
// Works by checking if the response is a html document
// If response is a html document then user is not logged in
func isLoggedIn(body []byte) bool {

	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		log.Fatal(err)
	}

	// Check if the response contains a login form
	var hasLoginForm bool
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "form" {
			hasLoginForm = true
			return
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return !hasLoginForm
}

func (s *Service) ProfileService(person *models.Person) (models.PersonDTO, error) {
	token := "Bearer " + s.personManager.GetToken()
	jar, err := cookiejar.New(nil)
	if err != nil {
		log.Fatal(err)
	}
	// HTTP client oluştur ve cookie jar ekle
	client := &http.Client{
		Jar: jar,
	}

	// İlk GET isteği için headers tanımla
	req, err := http.NewRequest("GET", personal_info_url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("User-Agent", "BeeHub")
	req.Header.Set("Authorization", token)
	// İstek gönder
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var info_response map[string]interface{}
	err = json.Unmarshal(body, &info_response)
	if err != nil {
		log.Fatal(err)

	}
	// Accessing the nested map and the adSoyad field
	if kisiselBilgiler, ok := info_response["kisiselBilgiler"].(map[string]interface{}); ok {

		name, ok := kisiselBilgiler["adSoyad"].(string)
		if ok {
			// Splitting the name into first and last name
			names := strings.Split(name, " ")
			if len(names) >= 2 {

				person.First_name = names[0]
				person.Last_name = names[1]
			}
		}
		email, ok := kisiselBilgiler["ePosta"].(string)
		if ok {
			person.Email = email
		}
		department, ok := kisiselBilgiler["bolumAdiEN"].(string)
		if ok {
			person.Department = department
		}
		faculty, ok := kisiselBilgiler["fakulteEN"].(string)
		if ok {
			person.Faculty = faculty
		}
	} else {
		fmt.Println("kisiselBilgiler field is not a map or not found")
	}

	// Fotoğraf isteği için headers tanımla
	req, err = http.NewRequest("GET", photo_url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", token)
	// İstek gönder
	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	var photoResponse map[string]interface{}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(body, &photoResponse)
	if err != nil {
		log.Fatal(err)
	}
	photoBase64, ok := photoResponse["base64Fotograf"].(string)
	if ok {
		person.Photo_base64 = photoBase64
	}

	// GPA ve sınıf isteği için headers tanımla
	req, err = http.NewRequest("GET", gpa_and_grade_url, nil)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", token)
	// İstek gönder
	resp, err = client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var classResponse map[string]interface{}
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	err = json.Unmarshal(body, &classResponse)
	if err != nil {
		log.Fatal(err)

	}

	if academicInfo, ok := classResponse["akademikDurum"].(map[string]interface{}); ok {
		class, ok := academicInfo["sinifSeviye"].(string)
		if ok {
			person.Class = string(class)[0:1]
		}
		gpa, ok := academicInfo["genelNotOrtalamasi"].(float64)
		if ok {
			person.GPA = fmt.Sprintf("%.2f", gpa)
		}
	}

	// Person güncelle ve DTO oluştur
	s.personManager.UpdatePerson(person)
	personDTO := models.ToPersonDTO(*person)
	return personDTO, nil
}
