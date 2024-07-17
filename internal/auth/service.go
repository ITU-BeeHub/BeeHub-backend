package auth

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"golang.org/x/net/html"
)

const token_url = "https://kepler-beta.itu.edu.tr/ogrenci/auth/jwt"

func LoginService() (string, error) {
	// .env dosyasını yükle
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// ITU_USERNAME ve ITU_PASSWORD değerlerini oku
	username := os.Getenv("ITU_USERNAME")
	password := os.Getenv("ITU_PASSWORD")

	if username == "" || password == "" {
		log.Fatal("ITU_USERNAME or ITU_PASSWORD not found in .env file")
	} else {
		fmt.Println("ITU_USERNAME:", username)
	}

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
	fmt.Println("Login URL:", loginURL)

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
		"ctl00$ContentPlaceHolder1$tbUserName": {username},
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
