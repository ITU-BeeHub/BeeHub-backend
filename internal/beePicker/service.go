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
	"github.com/ITU-BeeHub/BeeHub-backend/pkg/config"
	utils "github.com/ITU-BeeHub/BeeHub-backend/pkg/utils"

	"github.com/go-resty/resty/v2"
)

var (
	cache          []map[string]string // Cache verisi
	cacheTimestamp time.Time           // Cache zaman damgası
	cacheMutex     sync.Mutex          // Cache erişimi için mutex
)

// Ders listesi URL'leri
const (
	rawRepoURL     = "https://raw.githubusercontent.com/ITU-BeeHub/BeeHub-courseScraper/main/public"
	mostRecentURL  = "https://raw.githubusercontent.com/ITU-BeeHub/BeeHub-courseScraper/main/public/most_recent.txt"
	courseCodesURL = "https://raw.githubusercontent.com/ITU-BeeHub/BeeHub-courseScraper/main/public/course_codes.json"
)

// Service beePicker servisini temsil eder
type Service struct {
	personManager *pkg.PersonManager
	configManager *config.ConfigManager
}

// NewService yeni bir Service oluşturur
func NewService(personManager *pkg.PersonManager) *Service {
	return &Service{
		personManager: personManager,
		configManager: config.GetGlobalConfigManager(),
	}
}

// NewServiceWithConfig özel ConfigManager ile Service oluşturur
func NewServiceWithConfig(personManager *pkg.PersonManager, configManager *config.ConfigManager) *Service {
	return &Service{
		personManager: personManager,
		configManager: configManager,
	}
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
	base_url := rawRepoURL + "/" + newest_folder + "/"

	var allCourses []map[string]interface{}

	courseDataChan := make(chan []map[string]interface{})
	var wg sync.WaitGroup

	for _, course_code := range course_codes {
		wg.Add(1)
		go func(code string) {
			defer wg.Done()
			resp, err := http.Get(base_url + code + ".json")
			if err != nil {
				log.Println("Error getting course json:", err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusNotFound {
				// This branch code is not present in the latest scrape; skip silently.
				return
			}
			if resp.StatusCode != http.StatusOK {
				log.Printf("Failed to retrieve JSON for course code %s: %s", code, resp.Status)
				return
			}

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
	resp, err := http.Get(courseCodesURL)
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
	resp, err := http.Get(mostRecentURL)
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

// PickService ders ekleme ve silme işlemlerini gerçekleştirir
// courses: Eklenecek dersler (rezerveler dahil)
// dropCRNs: Silinecek ders CRN'leri (SCRN)
func (s *Service) PickService(courses []CourseRequest, dropCRNs []string) (map[string]map[string]interface{}, error) {
	client := resty.New()
	token := s.personManager.GetToken()

	queue := append([]CourseRequest{}, courses...)
	allResponses := []*resty.Response{}

	dropSent := false
	maxAttempts := 5
	attempt := 0

	for attempt < maxAttempts && (len(queue) > 0 || (!dropSent && len(dropCRNs) > 0)) {
		attempt++
		var currentBatchCRNs []string
		crnToCourseMap := make(map[string]CourseRequest)

		for _, course := range queue {
			currentBatchCRNs = append(currentBatchCRNs, course.CRN)
			crnToCourseMap[course.CRN] = course
		}

		var currentDropCRNs []string
		if !dropSent && len(dropCRNs) > 0 {
			currentDropCRNs = dropCRNs
			dropSent = true
		}

		if len(currentBatchCRNs) == 0 && len(currentDropCRNs) == 0 {
			break
		}

		resp, err := s.sendCourseRequestBatch(client, currentBatchCRNs, currentDropCRNs, token)
		if err != nil {
			return nil, err
		}
		allResponses = append(allResponses, resp)

		batchResult, err := parseRawPickResponse(resp)
		if err != nil {
			return nil, err
		}

		nextQueue := []CourseRequest{}

		for _, ecrnResult := range batchResult.EcrnResultList {
			crn, _ := ecrnResult["crn"].(string)
			course, ok := crnToCourseMap[crn]
			if !ok {
				continue
			}

			statusCode := toInt(ecrnResult["statusCode"])
			if statusCode != 0 && len(course.Reserves) > 0 {
				nextQueue = append(nextQueue, course.Reserves...)
			}
		}

		queue = nextQueue
		if attempt < maxAttempts && len(queue) > 0 {
			time.Sleep(3050 * time.Millisecond)
		}
	}

	return mergePickResponses(allResponses)
}

// sendCourseRequestBatch tek bir batch isteği gönderir
func (s *Service) sendCourseRequestBatch(client *resty.Client, addCRNs []string, dropCRNs []string, token string) (*resty.Response, error) {
	headers := s.configManager.GetHeaders(token)

	payload := map[string]interface{}{
		"ECRN": addCRNs,
		"SCRN": dropCRNs,
	}

	if addCRNs == nil {
		payload["ECRN"] = []string{}
	}
	if dropCRNs == nil {
		payload["SCRN"] = []string{}
	}

	url := s.configManager.GetCoursePickerURL()

	resp, err := client.R().
		SetHeaders(headers).
		SetBody(payload).
		Post(url)

	return resp, err
}

type pickResponse struct {
	EcrnResultList []map[string]interface{} `json:"ecrnResultList"`
	ScrnResultList []map[string]interface{} `json:"scrnResultList"`
}

func parseRawPickResponse(resp *resty.Response) (pickResponse, error) {
	var result pickResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return result, fmt.Errorf("error unmarshaling response: %v", err)
	}
	return result, nil
}

func toInt(value interface{}) int {
	switch v := value.(type) {
	case float64:
		return int(v)
	case int:
		return v
	case int32:
		return int(v)
	case int64:
		return int(v)
	case json.Number:
		i, _ := v.Int64()
		return int(i)
	case string:
		var i int
		_, _ = fmt.Sscanf(v, "%d", &i)
		return i
	default:
		return 0
	}
}

// mergePickResponses birden fazla yanıtı birleştirir
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

		// ECRN (ekleme) sonuçlarını işle
		for _, ecrnResult := range result.EcrnResultList {
			crn := ecrnResult["crn"].(string)
			statusCode := int(ecrnResult["statusCode"].(float64))
			resultCode := ecrnResult["resultCode"].(string)

			// resultData alanını doldur
			if errorCodes[resultCode] != "" {
				if utils.ContainsPlaceholder(errorCodes[resultCode], "%s") {
					ecrnResult["resultData"] = fmt.Sprintf(errorCodes[resultCode], crn)
				} else {
					ecrnResult["resultData"] = errorCodes[resultCode]
				}
			} else {
				ecrnResult["resultData"] = fmt.Sprintf(errorCodes["VAL01"], crn)
			}

			// İşlem tipini ekle
			ecrnResult["action"] = "add"

			// CRN zaten map'te varsa, başarılı olanı tut
			if existingResult, exists := pickResults[crn]; exists {
				if existingStatusCode := int(existingResult["statusCode"].(float64)); existingStatusCode != 0 && statusCode == 0 {
					pickResults[crn] = ecrnResult
				}
			} else {
				pickResults[crn] = ecrnResult
			}
		}

		// SCRN (silme) sonuçlarını işle
		for _, scrnResult := range result.ScrnResultList {
			crn := scrnResult["crn"].(string)
			statusCode := int(scrnResult["statusCode"].(float64))
			resultCode := scrnResult["resultCode"].(string)

			// resultData alanını doldur
			if errorCodes[resultCode] != "" {
				if utils.ContainsPlaceholder(errorCodes[resultCode], "%s") {
					scrnResult["resultData"] = fmt.Sprintf(errorCodes[resultCode], crn)
				} else {
					scrnResult["resultData"] = errorCodes[resultCode]
				}
			} else if statusCode == 0 {
				scrnResult["resultData"] = fmt.Sprintf("CRN %s olan ders başarıyla bırakıldı.", crn)
			} else {
				scrnResult["resultData"] = fmt.Sprintf(errorCodes["VAL01"], crn)
			}

			// İşlem tipini ekle
			scrnResult["action"] = "drop"

			// CRN zaten map'te varsa, başarılı olanı tut
			// SCRN için ayrı key kullan (crn_drop) çakışmayı önlemek için
			dropKey := crn + "_drop"
			if existingResult, exists := pickResults[dropKey]; exists {
				if existingStatusCode := int(existingResult["statusCode"].(float64)); existingStatusCode != 0 && statusCode == 0 {
					pickResults[dropKey] = scrnResult
				}
			} else {
				pickResults[dropKey] = scrnResult
			}
		}
	}

	return pickResults, nil
}
