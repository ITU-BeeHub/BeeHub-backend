package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/ITU-BeeHub/BeeHub-backend/addDropBot/documents" // Buradaki dosya yolunu proje yapınıza göre düzenleyin

	"github.com/kardianos/service"
)

const (
	baseURL         = "https://www.sis.itu.edu.tr/TR/ogrenci/ders-programi/ders-programi.php?seviye=LS"
	availabilityURL = "https://www.sis.itu.edu.tr/TR/ogrenci/ders-programi/ders-kontenjan.php?crn="

	checkInterval = 16 * time.Minute
)

type Course struct {
	CRN        string `json:"crn"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Capacity   string `json:"capacity"`
	Enrolement string `json:"enrolement"`
}

var Token string = ""
var Duration time.Time

type program struct{}

var logger service.Logger

func (p *program) Start(s service.Service) error {
	if logger != nil {
		logger.Info("Starting BeeHub Bot Service...")
	}
	go p.run()
	return nil
}

func (p *program) run() {

	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	documentsDir, err := documents.GetDocumentsDir() // GetDocumentsDir fonksiyonunu doğru paketten çağırın
	if err != nil {
		log.Fatalf("Failed to get Documents directory: %v", err)
	}
	crnFilePath := filepath.Join(documentsDir, ".crns.txt")

	crns, err := readCRNsFromFile(crnFilePath)
	if err != nil {
		log.Fatalf("Failed to read CRNs from file: %v", err)
	}
	credentialsFilePath := filepath.Join(documentsDir, ".credentials.txt")
	email, password, err := readCredentialsFromFile(credentialsFilePath)

	fmt.Println("CRNs:", crns)
	if err != nil {
		log.Fatalf("Error reading CRNs from file: %v", err)

	}
	allCourses := []Course{}

	if Token == "" || time.Since(Duration) > 5*time.Hour {
		token, err := LoginService(email, password)
		if err != nil {
			log.Fatalf("Error logging in: %v", err)
		}
		Token = token
		Duration = time.Now()
	}
	resp, err := SendCourseRequestsToCRNs(crns)
	if err != nil {
		log.Fatalf("Error sending course requests: %v", err)
	}
	for i := 0; i < len(crns); i++ {
		for _, result := range resp.ECRNResultList {
			if crns[i] == result.CRN {
				if result.ResultCode == "successResult" || result.ResultCode == "VAL03" {
					crns = append(crns[:i], crns[i+1:]...)
				} else {
					log.Printf("Error adding course %s: %s", result.CRN, result.ResultCode)
				}
			}

		}
	}
	checkStop(crns)
	for {

		courseCodes, err := FetchCourses()
		if err != nil {
			log.Fatalf("Error fetching courses: %v", err)
		}

		for _, courseCode := range courseCodes {
			courses, err := FetchCoursePage(courseCode)
			if err != nil {
				log.Printf("Error fetching course page for %s: %v", courseCode, err)
				continue
			}
			allCourses = append(allCourses, courses...)
		}

		availableCourses, err := CheckCourseAvailability(allCourses, crns)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Available courses: %v", availableCourses)
		if Token == "" || time.Since(Duration) > 5*time.Hour {
			token, err := LoginService(email, password)
			if err != nil {
				log.Fatalf("Error logging in: %v", err)
			}
			Token = token
			Duration = time.Now()
		}
		resp, err := SendCourseRequests(availableCourses)
		if err != nil {
			log.Fatalf("Error sending course requests: %v", err)
		}
		for i := 0; i < len(crns); i++ {
			for _, result := range resp.ECRNResultList {
				if crns[i] == result.CRN {
					if result.ResultCode == "successResult" || result.ResultCode == "VAL03" {
						crns = append(crns[:i], crns[i+1:]...)
					} else {
						log.Printf("Error adding course %s: %s", result.CRN, result.ResultCode)
					}
				}

			}
		}
		checkStop(crns)
		// Wait for the next tick
		<-ticker.C

	}
}

func (p *program) Stop(s service.Service) error {
	if logger != nil {
		logger.Info("Stopping BeeHub Bot Service...")
	}
	return nil
}

func main() {
	svcConfig := &service.Config{
		Name:        "BeeHubBotService",
		DisplayName: "BeeHub Bot Service",
		Description: "This service checks for course availability periodically.",
	}

	prg := &program{}
	s, err := service.New(prg, svcConfig)
	if err != nil {
		log.Fatal(err)
	}

	logger, err = s.Logger(nil)
	if err != nil {
		log.Fatal(err)
	}

	err = s.Run()
	if err != nil {
		logger.Error(err)
	}
}

// readCRNsFromFile reads CRN codes from a specified file
func readCRNsFromFile(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var crns []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		crns = append(crns, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return crns, nil
}

func readCredentialsFromFile(filepath string) (string, string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", "", err
	}
	defer file.Close()

	var email, password string
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		email = scanner.Text()
	}
	if scanner.Scan() {
		password = scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		return "", "", err
	}

	return email, password, nil
}

func checkStop(crns []string) {
	if len(crns) == 0 {
		log.Fatal("All courses have been added.")
	}
}
