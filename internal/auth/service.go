package auth

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"

	"github.com/ITU-BeeHub/BeeHub-backend/pkg"
	"github.com/ITU-BeeHub/BeeHub-backend/pkg/config"
	models "github.com/ITU-BeeHub/BeeHub-backend/pkg/models"
	"golang.org/x/net/html"
)

type URLs struct {
	Token           string
	Photo           string
	GpaAndGradeBase string
	TermList        string
	PersonalInfo    string
	Transcript      string
	BaseURL         string
}

var apiURLs = URLs{
	Token:           "https://obs.itu.edu.tr/ogrenci/auth/jwt",
	Photo:           "https://obs.itu.edu.tr/api/ogrenci/OgrenciFotograf",
	GpaAndGradeBase: "https://obs.itu.edu.tr/api/ogrenci/AkademikDurum",
	TermList:        "https://obs.itu.edu.tr/api/ogrenci/DonemListesi/",
	PersonalInfo:    "https://obs.itu.edu.tr/api/ogrenci/KisiselBilgiler",
	Transcript:      "https://obs.itu.edu.tr/api/ogrenci/Belgeler/TranskriptIngilizceOnizleme",
	BaseURL:         "https://obs.itu.edu.tr",
}

type Service struct {
	personManager *pkg.PersonManager
	configManager *config.ConfigManager
	sessionMu     sync.RWMutex
	sessionClient *http.Client
	subsessionMu  sync.RWMutex
	subsessionID  string
}

func NewService(personManager *pkg.PersonManager) *Service {
	return &Service{
		personManager: personManager,
		configManager: config.GetGlobalConfigManager(),
	}
}

func NewServiceWithConfig(personManager *pkg.PersonManager, configManager *config.ConfigManager) *Service {
	return &Service{
		personManager: personManager,
		configManager: configManager,
	}
}

func (s *Service) setSessionClient(client *http.Client) {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	s.sessionClient = client
}

func (s *Service) getSessionClient() *http.Client {
	s.sessionMu.RLock()
	defer s.sessionMu.RUnlock()
	return s.sessionClient
}

func (s *Service) setSubsessionID(id string) {
	s.subsessionMu.Lock()
	defer s.subsessionMu.Unlock()
	s.subsessionID = id
}

func (s *Service) getSubsessionID() string {
	s.subsessionMu.RLock()
	defer s.subsessionMu.RUnlock()
	return s.subsessionID
}

func (s *Service) makeRequestWithClient(client *http.Client, method, url string, body io.Reader, headers map[string]string) (*http.Response, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}

	userAgent := s.configManager.GetRandomUserAgent()
	if userAgent == "" {
		userAgent = "Mozilla/5.0"
	}
	req.Header.Set("User-Agent", userAgent)
	if headers == nil || headers["Accept"] == "" {
		req.Header.Set("Accept", "application/json, text/plain, */*")
	}
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
	s.setSessionClient(client)
	if subsession, err := s.fetchPortalSubsession(client); err == nil && subsession != "" {
		s.setSubsessionID(subsession)
	}
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
	if err := s.fetchPhoto(person); err != nil {
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

	infoResponse := make(map[string]interface{})
	if err := decodeJSONWithStatus(resp, &infoResponse); err != nil {
		return err
	}

	if kisiselBilgiler, ok := infoResponse["kisiselBilgiler"].(map[string]interface{}); ok {
		updatePersonFromInfo(person, kisiselBilgiler)
	}
	return nil
}

func (s *Service) fetchPhoto(person *models.Person) error {
	photoURL := apiURLs.Photo
	if cfg, err := s.configManager.GetConfig(); err == nil {
		if cfg.URLs.ProfilePhotoURL != "" {
			photoURL = cfg.URLs.ProfilePhotoURL
		}
	}

	photoHeaders := map[string]string{
		"Accept": "image/avif,image/webp,image/png,image/svg+xml,image/*;q=0.8,*/*;q=0.5",
	}

	client := s.getSessionClient()
	if client == nil {
		client = &http.Client{}
	}

	subsession := s.getSubsessionID()
	if subsession == "" {
		if fetched, err := s.fetchPortalSubsession(client); err == nil && fetched != "" {
			subsession = fetched
			s.setSubsessionID(fetched)
		}
	}

	if strings.Contains(photoURL, "{subsession}") {
		photoURL = strings.ReplaceAll(photoURL, "{subsession}", subsession)
	} else if photoURL == "" && subsession != "" {
		photoURL = "https://portal.itu.edu.tr/services/ui/photo.aspx?subsession=" + subsession
	}

	if photoURL == "" || strings.Contains(photoURL, "{subsession}") {
		return nil
	}

	resp, err := s.makeRequestWithClient(client, "GET", photoURL, nil, photoHeaders)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Photo is optional; don't fail profile if unavailable.
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
			return nil
		}
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("photo request failed: %d %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading photo response: %w", err)
	}

	if len(imageBytes) == 0 {
		return fmt.Errorf("empty photo response")
	}

	person.Photo_base64 = base64.StdEncoding.EncodeToString(imageBytes)
	return nil
}

func (s *Service) fetchPortalSubsession(client *http.Client) (string, error) {
	resp, err := s.makeRequestWithClient(client, "GET", "https://portal.itu.edu.tr/apps/default/", nil, nil)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading portal response: %w", err)
	}
	subsession := extractSubsessionFromText(string(bodyBytes))
	if subsession == "" {
		return "", fmt.Errorf("subsession not found")
	}
	return subsession, nil
}

func extractSubsessionFromText(text string) string {
	const key = "subsession="
	index := strings.Index(text, key)
	if index == -1 {
		return ""
	}
	start := index + len(key)
	end := start
	for end < len(text) {
		ch := text[end]
		if (ch >= '0' && ch <= '9') ||
			(ch >= 'a' && ch <= 'f') ||
			(ch >= 'A' && ch <= 'F') ||
			ch == '-' {
			end++
			continue
		}
		break
	}
	if end == start {
		return ""
	}
	return text[start:end]
}

func (s *Service) fetchAcademicInfo(person *models.Person, headers map[string]string) error {
	termListURL := apiURLs.TermList
	academicStatusBase := apiURLs.GpaAndGradeBase
	if cfg, err := s.configManager.GetConfig(); err == nil {
		if cfg.URLs.TermList != "" {
			termListURL = cfg.URLs.TermList
		}
		if cfg.URLs.AcademicStatusBase != "" {
			academicStatusBase = cfg.URLs.AcademicStatusBase
		}
	}

	termID, err := s.fetchLatestAcademicTermID(termListURL, headers)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/%d", academicStatusBase, termID)
	resp, err := s.makeRequestWithClient(&http.Client{}, "GET", url, nil, headers)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	classResponse := make(map[string]interface{})
	if err := decodeJSONWithStatus(resp, &classResponse); err != nil {
		return err
	}

	if academicInfo, ok := classResponse["akademikDurum"].(map[string]interface{}); ok {
		updatePersonAcademicInfo(person, academicInfo)
	}
	return nil
}

type termListResponse struct {
	StudentTerms []termInfo `json:"ogrenciDonemListesi"`
}

type termInfo struct {
	AcademicTermID     int  `json:"akademikDonemId"`
	IsLatestAcademicID bool `json:"sonAkademikDurumDonemId"`
}

func (s *Service) fetchLatestAcademicTermID(termListURL string, headers map[string]string) (int, error) {
	resp, err := s.makeRequestWithClient(&http.Client{}, "GET", termListURL, nil, headers)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var termResponse termListResponse
	if err := decodeJSONWithStatus(resp, &termResponse); err != nil {
		return 0, err
	}

	if len(termResponse.StudentTerms) == 0 {
		return 0, fmt.Errorf("term list is empty")
	}

	for _, term := range termResponse.StudentTerms {
		if term.IsLatestAcademicID {
			return term.AcademicTermID, nil
		}
	}

	latest := termResponse.StudentTerms[0].AcademicTermID
	for _, term := range termResponse.StudentTerms[1:] {
		if term.AcademicTermID > latest {
			latest = term.AcademicTermID
		}
	}

	return latest, nil
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

func decodeJSONWithStatus(resp *http.Response, target interface{}) error {
	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("error reading response body: %w", err)
	}
	if len(bodyBytes) == 0 {
		return fmt.Errorf("empty response body")
	}
	if err := json.Unmarshal(bodyBytes, target); err != nil {
		return fmt.Errorf("error decoding response body: %w", err)
	}
	return nil
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
	return bytes.Contains(body, []byte("Öğrenci sistemine devam etmek istediğiniz kimliğinizi seçmeniz")) ||
		bytes.Contains(body, []byte("identity-card")) ||
		bytes.Contains(body, []byte("SetIdentity"))
}

func extractActiveIdentity(body []byte) (*Identity, error) {
	doc, err := html.Parse(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	var identity *Identity
	var first *Identity
	var second *Identity
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

				link := findLink(n)
				if link == "" {
					return
				}

				// Parse the link to extract identity information
				u, err := url.Parse(link)
				if err != nil {
					return
				}

				candidate := &Identity{
					ID:        u.Query().Get("id"),
					StudentNo: u.Query().Get("ogrNo"),
					ReturnURL: u.Query().Get("returnURL"),
				}

				if status := findStatus(n); status == "Aktif" {
					candidate.Status = "Aktif"
					identity = candidate
					return
				}

				if first == nil {
					first = candidate
				} else if second == nil {
					second = candidate
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	if identity == nil && second != nil {
		identity = second
	}
	if identity == nil && first != nil {
		identity = first
	}

	if identity == nil {
		return nil, fmt.Errorf("no identity found")
	}

	return identity, nil
}

func (s *Service) LogoutService() {
	s.personManager.ClearPerson()
	s.setSessionClient(nil)
	s.setSubsessionID("")
}
