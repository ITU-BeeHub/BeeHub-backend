package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	"github.com/ITU-BeeHub/BeeHub-backend/pkg"
	models "github.com/ITU-BeeHub/BeeHub-backend/pkg/models"
	"golang.org/x/net/html"
)

type URLs struct {
	Token        string
	Photo        string
	GpaAndGrade  string
	PersonalInfo string
	Transcript   string
	BaseURL      string
}

var apiURLs = URLs{
	Token:        "https://obs.itu.edu.tr/ogrenci/auth/jwt",
	Photo:        "https://obs.itu.edu.tr/api/ogrenci/OgrenciFotograf",
	GpaAndGrade:  "https://obs.itu.edu.tr/api/ogrenci/AkademikDurum/759",
	PersonalInfo: "https://obs.itu.edu.tr/api/ogrenci/KisiselBilgiler",
	Transcript:   "https://obs.itu.edu.tr/api/ogrenci/Belgeler/TranskriptIngilizceOnizleme",
	BaseURL:      "https://obs.itu.edu.tr",
}

type Service struct {
	personManager *pkg.PersonManager
}

func NewService(personManager *pkg.PersonManager) *Service {
	return &Service{
		personManager: personManager,
	}
}

func (s *Service) makeRequestWithClient(client *http.Client, method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("User-Agent", "BeeHub")
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request: %w", err)
	}

	return resp, nil
}

// Identity structure to hold student identity information
type Identity struct {
	ID         string
	Department string
	StudentNo  string
	Status     string
	ReturnURL  string
}

func (s *Service) LoginService(email, password string) (string, error) {
	// Clear any existing session
	s.LogoutService()

	// Create new client for each login attempt
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	// Initial GET request to get the login page
	resp, err := s.makeRequestWithClient(client, "GET", apiURLs.BaseURL, nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	loginURL := resp.Request.Response.Request.URL.String()

	// Get login form
	resp, err = s.makeRequestWithClient(client, "GET", loginURL, nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	formData, err := extractFormData(resp.Body, email, password)
	if err != nil {
		return "", fmt.Errorf("error extracting form data: %w", err)
	}

	// Login POST request
	headers := map[string]string{"Content-Type": "application/x-www-form-urlencoded"}
	resp, err = s.makeRequestWithClient(client, "POST", loginURL, strings.NewReader(formData.Encode()), headers)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Check if we got identity selection page
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	// If we have identity selection page
	if isIdentitySelectionPage(body) {
		identity, err := extractActiveIdentity(body)
		if err != nil {
			return "", fmt.Errorf("error extracting identity: %w", err)
		}

		// Make request to set identity
		identityURL := fmt.Sprintf("/login/SetIdentity?id=%s&returnURL=%s&yetkiAnahtari=ogrenci&ogrNo=%s",
			identity.ID, identity.ReturnURL, identity.StudentNo)

		resp, err = s.makeRequestWithClient(client, "GET", apiURLs.BaseURL+identityURL, nil, nil)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()
	}

	// Get JWT token
	resp, err = s.makeRequestWithClient(client, "GET", apiURLs.Token, nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	tokenBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading token body: %w", err)
	}

	if !isLoggedIn(tokenBody) {
		return "", fmt.Errorf("login failed")
	}

	token := string(tokenBody)
	s.updatePersonInfo(email, password, token)
	return token, nil
}

func (s *Service) ProfileService(person *models.Person) (models.PersonDTO, error) {
	token := "Bearer " + s.personManager.GetToken()
	headers := map[string]string{"Authorization": token}

	// Get personal info
	if err := s.fetchPersonalInfo(person, headers); err != nil {
		return models.PersonDTO{}, err
	}

	// Get photo
	if err := s.fetchPhoto(person, headers); err != nil {
		return models.PersonDTO{}, err
	}

	// Get academic info
	if err := s.fetchAcademicInfo(person, headers); err != nil {
		return models.PersonDTO{}, err
	}

	s.personManager.UpdatePerson(person)
	return models.ToPersonDTO(*person), nil
}

func (s *Service) fetchPersonalInfo(person *models.Person, headers map[string]string) error {
	resp, err := s.makeRequestWithClient(&http.Client{}, "GET", apiURLs.PersonalInfo, nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var infoResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&infoResponse); err != nil {
		return err
	}

	if kisiselBilgiler, ok := infoResponse["kisiselBilgiler"].(map[string]interface{}); ok {
		updatePersonFromInfo(person, kisiselBilgiler)
	}
	return nil
}

func (s *Service) fetchPhoto(person *models.Person, headers map[string]string) error {
	resp, err := s.makeRequestWithClient(&http.Client{}, "GET", apiURLs.Photo, nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var photoResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&photoResponse); err != nil {
		return err
	}

	if photoBase64, ok := photoResponse["base64Fotograf"].(string); ok {
		person.Photo_base64 = photoBase64
	}
	return nil
}

func (s *Service) fetchAcademicInfo(person *models.Person, headers map[string]string) error {
	resp, err := s.makeRequestWithClient(&http.Client{}, "GET", apiURLs.GpaAndGrade, nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var classResponse map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&classResponse); err != nil {
		return err
	}

	if academicInfo, ok := classResponse["akademikDurum"].(map[string]interface{}); ok {
		updatePersonAcademicInfo(person, academicInfo)
	}
	return nil
}

// Helper functions
func extractFormData(body io.Reader, email, password string) (url.Values, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, err
	}

	formFields := map[string]string{}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "input" {
			name, value := "", ""
			for _, attr := range n.Attr {
				if attr.Key == "name" {
					name = attr.Val
				}
				if attr.Key == "value" {
					value = attr.Val
				}
			}
			if name != "" {
				formFields[name] = value
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return url.Values{
		"__VIEWSTATE":                          {formFields["__VIEWSTATE"]},
		"__VIEWSTATEGENERATOR":                 {formFields["__VIEWSTATEGENERATOR"]},
		"__EVENTVALIDATION":                    {formFields["__EVENTVALIDATION"]},
		"__EVENTTARGET":                        {""},
		"__EVENTARGUMENT":                      {""},
		"ctl00$ContentPlaceHolder1$hfAppName":  {"Öğrenci Bilgi Sistemi"},
		"ctl00$ContentPlaceHolder1$tbUserName": {email},
		"ctl00$ContentPlaceHolder1$tbPassword": {password},
		"ctl00$ContentPlaceHolder1$btnLogin":   {"Giriş / Login"},
	}, nil
}

func (s *Service) updatePersonInfo(email, password, token string) {
	person := s.personManager.GetPerson()
	s.personManager.SetEmail(email)
	s.personManager.SetEmail(email)
	s.personManager.SetPassword(password)
	s.personManager.UpdateLoginTime()
	s.personManager.UpdateToken(token)
	s.personManager.UpdatePerson(person)
}

func updatePersonFromInfo(person *models.Person, info map[string]interface{}) {
	if name, ok := info["adSoyad"].(string); ok {
		names := strings.Split(name, " ")
		if len(names) >= 2 {
			person.First_name = names[0]
			person.Last_name = names[1]
		}
	}

	if department, ok := info["bolumAdiEN"].(string); ok {
		person.Department = department
	}
	if faculty, ok := info["fakulteEN"].(string); ok {
		person.Faculty = faculty
	}
}

func updatePersonAcademicInfo(person *models.Person, info map[string]interface{}) {
	if class, ok := info["sinifSeviye"].(string); ok {
		person.Class = string(class)[0:1]
	}
	if gpa, ok := info["genelNotOrtalamasi"].(float64); ok {
		person.GPA = fmt.Sprintf("%.2f", gpa)
	}
}

func isLoggedIn(body []byte) bool {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return false
	}

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

func isIdentitySelectionPage(body []byte) bool {
	return bytes.Contains(body, []byte("Öğrenci sistemine devam etmek istediğiniz kimliğinizi seçmeniz"))
}

func extractActiveIdentity(body []byte) (*Identity, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var identity *Identity
	var f func(*html.Node)
	f = func(n *html.Node) {
		if identity != nil {
			return
		}

		if n.Type == html.ElementNode && n.Data == "div" {
			// Check if this is an identity card
			isIdentityCard := false
			for _, attr := range n.Attr {
				if attr.Key == "class" && strings.Contains(attr.Val, "identity-card") {
					isIdentityCard = true
					break
				}
			}

			if isIdentityCard {
				// Find the link that contains the identity information
				var findLink func(*html.Node) string
				findLink = func(n *html.Node) string {
					if n.Type == html.ElementNode && n.Data == "a" {
						for _, attr := range n.Attr {
							if attr.Key == "href" {
								return attr.Val
							}
						}
					}
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if link := findLink(c); link != "" {
							return link
						}
					}
					return ""
				}

				// Find status text
				var findStatus func(*html.Node) string
				findStatus = func(n *html.Node) string {
					if n.Type == html.ElementNode && n.Data == "td" {
						if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
							if strings.Contains(n.FirstChild.Data, "Aktif") {
								return "Aktif"
							}
						}
					}
					for c := n.FirstChild; c != nil; c = c.NextSibling {
						if status := findStatus(c); status != "" {
							return status
						}
					}
					return ""
				}

				if status := findStatus(n); status == "Aktif" {
					if link := findLink(n); link != "" {
						// Parse the link to extract identity information
						u, err := url.Parse(link)
						if err != nil {
							return
						}

						identity = &Identity{
							ID:        u.Query().Get("id"),
							StudentNo: u.Query().Get("ogrNo"),
							ReturnURL: u.Query().Get("returnURL"),
							Status:    "Aktif",
						}
						return
					}
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if identity == nil {
		return nil, fmt.Errorf("no active identity found")
	}

	return identity, nil
}

func (s *Service) LogoutService() {
	s.personManager.ClearPerson()
}
