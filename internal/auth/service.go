package auth

import (
	"errors"
	"fmt"
	"log"
	"net/http"
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
	// İlk GET isteği için headers tanımla
	client := &http.Client{}
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
	if len(resp.Request.Response.Request.URL.String()) > 0 {
		loginURL := resp.Request.Response.Request.URL.String()
		return loginURL, nil
	} else {
		return "", errors.New("login URL not found")
	}

}
