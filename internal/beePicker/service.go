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

func (s *Service) PickService(courseCodes []string) (map[string]map[string]interface{}, error) {
	client := resty.New()

	responses, err := sendCourseRequests(client, courseCodes, s.personManager.GetToken())
	if err != nil {
		return nil, fmt.Errorf("error sending course requests: %v", err)
	}
	return mergePickResponses(responses)
}

func sendCourseRequests(client *resty.Client, courses []string, token string) ([]*resty.Response, error) {
	var responses []*resty.Response
	var errors []error
	headers := map[string]string{
		"accept":        "application/json, text/plain, */*",
		"authorization": "Bearer  " + token,
		"origin":        "https://obs.itu.edu.tr",
		"referer":       "https://obs.itu.edu.tr/ogrenci/DersKayitIslemleri/DersKayit",
	}
	payload := map[string]interface{}{
		"ECRN": courses,    // Example CRNs to be added
		"SCRN": []string{}, // Example CRNs to be deleted
	}
	for i := 0; i < 5; i++ {
		resp, err := client.R().
			SetHeaders(headers).
			SetBody(payload).
			Post(kepler_picker_url)

		if err != nil {
			errors = append(errors, err)
			continue
		}
		responses = append(responses, resp)
		fmt.Println()
		fmt.Println(responses)
		// Saniyede bir istek göndermek için bekleme
		time.Sleep(3100 * time.Millisecond)
	}

	if len(errors) > 0 {
		return nil, fmt.Errorf("errors occurred while sending course requests: %v", errors)
	}

	return responses, nil
}

func mergePickResponses(responses []*resty.Response) (map[string]map[string]interface{}, error) {
	pickResults := make(map[string]map[string]interface{})
	errorCodes := utils.GetErrorCodes()

	for _, resp := range responses {
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

		for _, ecrnResult := range result.EcrnResultList {
			crn := ecrnResult["crn"].(string)
			statusCode := int(ecrnResult["statusCode"].(float64))
			resultCode := ecrnResult["resultCode"].(string)

			// Populate the resultData field using GetErrorCodes
			ecrnResult["resultData"] = fmt.Sprintf(errorCodes[resultCode], crn)

			// Check if the CRN is already in the map
			if existingResult, exists := pickResults[crn]; exists {
				// Keep the one with statusCode = 0 (success)
				if existingStatusCode := int(existingResult["statusCode"].(float64)); existingStatusCode != 0 && statusCode == 0 {
					pickResults[crn] = ecrnResult
				}
			} else {
				// Add the CRN to the map
				pickResults[crn] = ecrnResult
			}
		}

		for _, scrnResult := range result.ScrnResultList {
			crn := scrnResult["crn"].(string)
			statusCode := int(scrnResult["statusCode"].(float64))
			resultCode := scrnResult["resultCode"].(string)

			// Populate the resultData field using GetErrorCodes
			scrnResult["resultData"] = fmt.Sprintf(errorCodes[resultCode], crn)

			// Check if the CRN is already in the map
			if existingResult, exists := pickResults[crn]; exists {
				// Keep the one with statusCode = 0 (success)
				if existingStatusCode := int(existingResult["statusCode"].(float64)); existingStatusCode != 0 && statusCode == 0 {
					pickResults[crn] = scrnResult
				}
			} else {
				// Add the CRN to the map
				pickResults[crn] = scrnResult
			}
		}
	}

	return pickResults, nil
}
