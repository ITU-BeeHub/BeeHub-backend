package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func FetchCoursePage(courseCode string) ([]Course, error) {
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

func FetchCourses() ([]string, error) {
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

func CheckCourseAvailability(courses []Course, crns []string) ([]Course, error) {
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
