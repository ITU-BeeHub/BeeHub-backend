package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/kardianos/service"
)

const (
	baseURL         = "https://www.sis.itu.edu.tr/TR/ogrenci/ders-programi/ders-programi.php?seviye=LS"
	availabilityURL = "https://www.sis.itu.edu.tr/TR/ogrenci/ders-programi/ders-kontenjan.php?crn="
	repoRootDir     = "C:\\Program Files (x86)\\BeeHub"
	checkInterval   = 1 * time.Minute
)

type Course struct {
	CRN        string `json:"crn"`
	Code       string `json:"code"`
	Name       string `json:"name"`
	Capacity   string `json:"capacity"`
	Enrolement string `json:"enrolement"`
}

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

	for {
		select {
		case <-ticker.C:
			courseCodes, err := fetchCourses()
			if err != nil {
				log.Fatalf("Error fetching courses: %v", err)
			}

			crns := []string{"30366"} // Example CRNs provided by user
			allCourses := []Course{}

			for _, courseCode := range courseCodes {
				courses, err := fetchCoursePage(courseCode)
				if err != nil {
					log.Printf("Error fetching course page for %s: %v", courseCode, err)
					continue
				}
				allCourses = append(allCourses, courses...)
			}

			availableCourses, err := checkCourseAvailability(allCourses, crns)
			if err != nil {
				log.Fatal(err)
			}

			outputPath := filepath.Join(repoRootDir, "data.json")
			file, err := os.Create(outputPath)
			if err != nil {
				log.Fatalf("Error creating output file: %v", err)
			}

			jsonData, err := json.MarshalIndent(availableCourses, "", "  ")
			if err != nil {
				log.Fatalf("Error marshalling JSON data: %v", err)
			}

			_, err = file.Write(jsonData)
			if err != nil {
				log.Fatalf("Error writing to file: %v", err)
			}

			logger.Info("Available courses saved to ", outputPath)
			logger.Info(time.Now().Clock())

			file.Close()
		}
	}
}

func (p *program) Stop(s service.Service) error {
	if logger != nil {
		logger.Info("Stopping BeeHub Bot Service...")
	}
	return nil
}

func fetchCoursePage(courseCode string) ([]Course, error) {
	url := fmt.Sprintf("%s&derskodu=%s", baseURL, courseCode)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch course page: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	table := doc.Find("div.table-responsive")
	courses := []Course{}
	headers := []string{}

	table.Find("tr").Each(func(i int, s *goquery.Selection) {
		if i == 1 {
			s.Find("td, th").Each(func(_ int, cell *goquery.Selection) {
				headers = append(headers, strings.TrimSpace(cell.Text()))
			})
		} else if i > 1 {
			course := Course{}
			s.Find("td, th").Each(func(idx int, cell *goquery.Selection) {
				switch headers[idx] {
				case "CRN":
					course.CRN = strings.TrimSpace(cell.Text())
				case "Course Code":
					course.Code = strings.TrimSpace(cell.Text())
				case "Course Title":
					course.Name = strings.TrimSpace(cell.Text())
				case "Capacity":
					course.Capacity = strings.TrimSpace(cell.Text())
				case "Enrolled":
					course.Enrolement = strings.TrimSpace(cell.Text())
				}
			})
			courses = append(courses, course)
		}
	})

	return courses, nil
}

func fetchCourses() ([]string, error) {
	resp, err := http.Get(baseURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch courses: %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, err
	}

	var courseCodes []string
	doc.Find("option").Each(func(i int, s *goquery.Selection) {
		value, exists := s.Attr("value")
		if exists && len(value) == 3 {
			courseCodes = append(courseCodes, value)
		}
	})

	return courseCodes, nil
}

func checkCourseAvailability(courses []Course, crns []string) ([]Course, error) {
	availableCourses := []Course{}
	for _, course := range courses {
		for _, crn := range crns {
			if course.CRN == crn && course.Capacity != course.Enrolement {
				availableCourses = append(availableCourses, course)
			}
		}
	}
	return availableCourses, nil
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
