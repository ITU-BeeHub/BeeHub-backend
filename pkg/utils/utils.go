package utils

import (
	"fmt"

	godotenv "github.com/joho/godotenv"
)

// Loads environment variables from a .env file.
func LoadEnvVariables() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}
