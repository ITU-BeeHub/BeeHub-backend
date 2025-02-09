package beepicker

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/ITU-BeeHub/BeeHub-backend/pkg"
	utils "github.com/ITU-BeeHub/BeeHub-backend/pkg/utils"
	"github.com/gin-gonic/gin"

	"github.com/go-resty/resty/v2"
)

var (
	cache          []map[string]string // Cache verisi
	cacheTimestamp time.Time           // Cache zaman damgası
	cacheMutex     sync.Mutex          // Cache erişimi için mutex
)

const raw_repo_URL = "https://raw.githubusercontent.com/ITU-BeeHub/BeeHub-courseScraper/main/public"
const most_recent_URL = "https://raw.githubusercontent.com/ITU-BeeHub/BeeHub-courseScraper/main/public/most_recent.txt"
const course_codes_URL = "https://raw.githubusercontent.com/ITU-BeeHub/BeeHub-courseScraper/main/public/course_codes.json"
const kepler_picker_url = "https://obs.itu.edu.tr/api/ders-kayit/v21"

type Service struct {
	personManager *pkg.PersonManager
}

func NewService(personManager *pkg.PersonManager) *Service {
	return &Service{personManager: personManager}
}

func (s *Service) CourseService() ([]map[string]string, error) {
	cacheMutex.Lock()
	defer cacheMutex.Unlock()

	// Cache'in geçerliliğini kontrol et
	if time.Since(cacheTimestamp) < 5*time.Minute && cache != nil {
		return cache, nil
	}

	// Cache güncel değilse yeni veriyi çek
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

	var convertedData []map[string]string
	for _, item := range data {
		convertedItem := make(map[string]string)
		for key, value := range item {
			convertedItem[key] = fmt.Sprintf("%v", value)
		}
		convertedData = append(convertedData, convertedItem)
	}

	// Veriyi cache'e kaydet ve zaman damgasını güncelle
	cache = convertedData
	cacheTimestamp = time.Now()

	return convertedData, nil
}

func MergeCourseJsons(course_codes []string, newest_folder string) ([]map[string]interface{}, error) {
	base_url := raw_repo_URL + "/" + newest_folder + "/"

	var allCourses []map[string]interface{}

	courseDataChan := make(chan []map[string]interface{})
	var wg sync.WaitGroup

	for _, course_code := range course_codes {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			resp, err := http.Get(base_url + code + ".json")
			if resp.StatusCode != http.StatusOK {
				log.Printf("Failed to retrieve JSON for course code %s: %s", code, resp.Status)
				return
			}
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

			// Create a struct to capture the new response format
			var result struct {
				DersProgramList []map[string]interface{} `json:"dersProgramList"`
			}

			// Unmarshal into the new struct to access the list
			err = json.Unmarshal(body, &result)
			if err != nil {
				log.Println("Error unmarshaling course json:", err)
				log.Println("Faulty JSON:", string(body))
				return
			}

			// Send the array from the "dersProgramList" to the channel
			courseDataChan <- result.DersProgramList
		}(course_code)
	}

	go func() {
		wg.Wait()
		close(courseDataChan)
	}()

	// Collect all the courses from the channel
	for courses := range courseDataChan {
		allCourses = append(allCourses, courses...)
	}

	return allCourses, nil
}

func getCourseCodes() ([]string, error) {
	resp, err := http.Get(course_codes_URL)
	if err != nil {
		return []string{}, err
	}
	defer resp.Body.Close()

	var course_codes_response []map[string]interface{}
	course_codes_bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return []string{}, err
	}
	err = json.Unmarshal(course_codes_bytes, &course_codes_response)
	if err != nil {
		return []string{}, err
	}

	var course_codes []string
	for _, course := range course_codes_response {
		if code, ok := course["dersBransKodu"].(string); ok {
			course_codes = append(course_codes, code)
		}
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

func (s *Service) PickService(courses []CourseRequest, c *gin.Context) error {
	client := resty.New()
	token := s.personManager.GetToken()
	pickResults := make(map[string]map[string]interface{})

	// Flag to track whether any course has been successfully picked
	anySuccess := false

	// Initialize a queue with the top-level courses
	queue := []CourseRequest{}
	queue = append(queue, courses...)

	// Map to keep track of courses that have been processed
	processedCRNs := make(map[string]bool)

	// Retry counter
	retryCount := 0

	for len(queue) > 0 {
		// Extract CRNs for the current batch
		var currentBatchCRNs []string
		crnToCourseMap := make(map[string]CourseRequest)

		for _, course := range queue {
			if !processedCRNs[course.CRN] {
				currentBatchCRNs = append(currentBatchCRNs, course.CRN)
				crnToCourseMap[course.CRN] = course
				processedCRNs[course.CRN] = true
			}
		}

		if len(currentBatchCRNs) == 0 {
			break // No new CRNs to process
		}

		// Send batch request
		resp, err := sendCourseRequestBatch(client, currentBatchCRNs, token)
		if err != nil {
			log.Printf("Error sending batch request: %v", err)
			return err
		}

		// Parse response
		result, err := parsePickResponse(resp)
		if err != nil {
			log.Printf("Error parsing pick response: %v", err)
			return err
		}

		// Stream the response to the client (SSE)
		for crn, res := range result {
			pickResults := map[string]interface{}{"crn": crn, "result": res}
			jsonData, _ := json.Marshal(pickResults)
			c.Writer.Write(jsonData)
			c.Writer.Write([]byte("\n"))
		}
		c.Writer.Flush()

		// Process results
		nextQueue := []CourseRequest{} // Queue for the next batch
		retryNeeded := false

		for crn, res := range result {
			pickResults[crn] = res
			statusCode := int(res["statusCode"].(float64))
			resultCode := res["resultCode"].(string)

			course := crnToCourseMap[crn]

			// If course was successfully picked
			if statusCode == 0 {
				anySuccess = true // Mark that at least one course was successfully picked
			}

			// If pick failed with "NULLParam-CheckOgrenciKayitZamaniKontrolu"
			// we will retry the same CRN in the next batch
			// This block will execute if user clicked button a bit early.
			if statusCode != 0 && resultCode == "NULLParam-CheckOgrenciKayitZamaniKontrolu" {
				// Retry the same CRN in the next batch if no course has been successfully taken yet
				if !anySuccess && retryCount < 3 {
					nextQueue = append(nextQueue, course)
					retryNeeded = true
					processedCRNs[course.CRN] = false // Mark the CRN as not processed as it will be tried again.
				} else {
					// Proceed to reserves if any
					nextQueue = append(nextQueue, course.Reserves...)
				}
			} else if statusCode != 0 {
				// For other failures, proceed to reserves if any
				nextQueue = append(nextQueue, course.Reserves...)
			}
		}

		// If we retried and anySuccess is still false, increment retry count
		if retryNeeded && !anySuccess {
			retryCount++
			// If we reached maximum retry limit, stop retrying
			if retryCount >= 3 {
				log.Printf("Reached maximum retry attempts with no success.")
				break
			}
		}

		// Prepare the queue for the next batch
		queue = nextQueue

		// Rate limiting: Sleep for 3.1 seconds before the next batch
		time.Sleep(3100 * time.Millisecond)
	}

	return nil
}

func sendCourseRequestBatch(client *resty.Client, crns []string, token string) (*resty.Response, error) {
	headers := map[string]string{
		"accept":        "application/json, text/plain, */*",
		"authorization": "Bearer " + token,
		"origin":        "https://obs.itu.edu.tr",
		"referer":       "https://obs.itu.edu.tr/ogrenci/DersKayitIslemleri/DersKayit",
	}

	payload := map[string]interface{}{
		"ECRN": crns,
		"SCRN": []string{},
	}

	resp, err := client.R().
		SetHeaders(headers).
		SetBody(payload).
		Post(kepler_picker_url)

	return resp, err
}

func parsePickResponse(resp *resty.Response) (map[string]map[string]interface{}, error) {
	if resp.StatusCode() != http.StatusOK {
		return nil, fmt.Errorf("non-200 status code received: %d", resp.StatusCode())
	}

	var result struct {
		EcrnResultList []map[string]interface{} `json:"ecrnResultList"`
		ScrnResultList []map[string]interface{} `json:"scrnResultList"`
	}

	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %v", err)
	}

	pickResults := make(map[string]map[string]interface{})
	errorCodes := utils.GetErrorCodes()

	for _, ecrnResult := range result.EcrnResultList {
		crn := ecrnResult["crn"].(string)
		resultCode := ecrnResult["resultCode"].(string)
		statusCode := ecrnResult["statusCode"].(float64) // Assuming statusCode is a float64

		// Check if resultCode exists in errorCodes, otherwise fall back to successResult if statusCode is 0
		if errorCodes[resultCode] == "" {
			if statusCode == 0 {
				ecrnResult["resultData"] = fmt.Sprintf(errorCodes["successResult"], crn)
			} else {
				ecrnResult["resultData"] = fmt.Sprintf(errorCodes["VAL01"], crn)
			}
		} else {
			// Check if error message contains a %s for crn substitution
			if utils.ContainsPlaceholder(errorCodes[resultCode], "%s") {
				ecrnResult["resultData"] = fmt.Sprintf(errorCodes[resultCode], crn)
			} else {
				ecrnResult["resultData"] = errorCodes[resultCode]
			}
		}

		pickResults[crn] = ecrnResult
	}

	return pickResults, nil
}
