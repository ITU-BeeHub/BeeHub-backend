package beepicker

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"sync"

	utils "github.com/ITU-BeeHub/BeeHub-backend/pkg/utils"
)

const raw_repo_URL = "https://raw.githubusercontent.com/ITU-BeeHub/BeeHub-courseScraper/main/public"
const most_recent_URL = "https://raw.githubusercontent.com/ITU-BeeHub/BeeHub-courseScraper/main/public/most_recent.txt"
const course_codes_URL = "https://raw.githubusercontent.com/ITU-BeeHub/BeeHub-courseScraper/main/public/course_codes.json"

type Service struct {
}

func NewService() *Service {
	return &Service{}
}

func (s *Service) CourseService() ([]map[string]string, error) {

	folder, err := getNewestFolder()
	if err != nil {
		return nil, errors.New("error getting newest folder")
	}

	course_codes, err := getCourseCodes()
	if err != nil {
		return nil, errors.New("error getting course codes")
	}

	data, err := MergeCourseJsons(course_codes, folder)
	if err != nil {
		return nil, errors.New("error getting course data")
	}

	return data, nil
}

// Returns the schedules of the user
func (s *Service) SchedulesService() (utils.ScheduleList, error) {

	// Get the schedules
	schedules, err := utils.GetUserSchedules()
	if err != nil {
		return utils.ScheduleList{}, err
	}

	return schedules, nil
}

// Saves the schedule of the user in the schedules.json file
// If the schedule already exists, updates the schedule (overwrites the old one)
// If the schedule does not exist, creates a new schedule
func (s *Service) ScheduleSaveService(schedule_name string, ecrn []int, scrn []int) error {

	// Get the schedules
	schedules, err := utils.GetUserSchedules()
	if err != nil {
		return err
	}

	is_found := false
	// Find the schedule with the same name
	for i, schedule := range schedules.Schedules {
		if schedule.Name == schedule_name {
			// Update the schedule
			schedules.Schedules[i].ECRN = ecrn
			schedules.Schedules[i].SCRN = scrn
			is_found = true
			break
		}
	}
	if !is_found {
		// Create a new schedule
		schedules.Schedules = append(schedules.Schedules, utils.Schedule{Name: schedule_name, ECRN: ecrn, SCRN: scrn})
	}

	// Save the schedules
	err = utils.SaveUserSchedules(schedules)
	if err != nil {
		return err
	}

	return nil
}

func MergeCourseJsons(course_codes []string, newest_folder string) ([]map[string]string, error) {
	// Merges all course jsons into one json and returns it as a slice of maps

	base_url := raw_repo_URL + "/" + newest_folder + "/"

	var allCourses []map[string]string

	// Create a channel to receive the course data
	courseDataChan := make(chan []map[string]string)

	// Create a wait group to wait for all goroutines to finish
	var wg sync.WaitGroup

	for _, course_code := range course_codes {

		// Start a goroutine to fetch each course code information
		wg.Add(1)
		go func(code string) {
			defer wg.Done()

			// Get course json
			resp, err := http.Get(base_url + code + ".json")
			if err != nil {
				log.Println("Error getting course json:", err)
				return
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Println("Error reading course json:", err)
				return
			}

			var courses []map[string]string
			err = json.Unmarshal(body, &courses)
			if err != nil {
				log.Println("Error unmarshaling course json:", err)
				return
			}

			courseDataChan <- courses
		}(course_code)
	}

	// Start a goroutine to close the channel when all goroutines are done
	go func() {
		wg.Wait()
		close(courseDataChan)
	}()

	// Collect the course data from the channel
	for courses := range courseDataChan {
		allCourses = append(allCourses, courses...)
	}

	return allCourses, nil
}

func getCourseCodes() ([]string, error) {
	// Get course codes
	resp, err := http.Get(course_codes_URL)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	// Append json elements to the course_codes slice
	course_codesBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return []string{}, err
	}

	var course_codes []string
	err = json.Unmarshal(course_codesBytes, &course_codes)
	if err != nil {
		return []string{}, err
	}

	return course_codes, nil
}

func getNewestFolder() (string, error) {
	// Gets the most recent folder name
	resp, err := http.Get(most_recent_URL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	most_recent_file_name, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(most_recent_file_name), nil
}
