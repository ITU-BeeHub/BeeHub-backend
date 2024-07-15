package auth

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/cookiejar"
	"os"

	"github.com/joho/godotenv"
)

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

	// İstek gönder
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Yönlendirme URL'sini bul
	if len(resp.Request.Response.Request.URL.String()) < 0 {
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

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	return string(body), nil

}
