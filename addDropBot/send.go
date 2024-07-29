package main

import (
	"encoding/json"

	"github.com/go-resty/resty/v2"
)

const apiURL = "https://kepler-beta.itu.edu.tr/api/ders-kayit/v21"

type Result struct {
	CRN        string `json:"crn"`
	ResultCode string `json:"resultCode"`
}

type Response struct {
	ECRNResultList []Result `json:"ecrnResultList"`
	SCRNResultList []Result `json:"scrnResultList"`
}

func SendCourseRequests(courses []Course) (*Response, error) {
	client := resty.New()
	headers := map[string]string{
		"accept":        "application/json, text/plain, */*",
		"authorization": "Bearer  " + Token,
		"origin":        "https://kepler-beta.itu.edu.tr",
		"referer":       "https://kepler-beta.itu.edu.tr/ogrenci/DersKayitIslemleri/DersKayit",
	}

	crns := []string{}
	for _, course := range courses {
		crns = append(crns, course.CRN)
	}

	payload := map[string]interface{}{
		"ECRN": crns,       // Example CRNs to be added
		"SCRN": []string{}, // Example CRNs to be deleted
	}

	resp, err := client.R().SetHeaders(headers).SetBody(payload).Post(apiURL)

	if err != nil {
		return nil, err
	}
	var response Response
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}

func SendCourseRequestsToCRNs(crns []string) (*Response, error) {
	client := resty.New()
	headers := map[string]string{
		"accept":        "application/json, text/plain, */*",
		"authorization": "Bearer  " + Token,
		"origin":        "https://kepler-beta.itu.edu.tr",
		"referer":       "https://kepler-beta.itu.edu.tr/ogrenci/DersKayitIslemleri/DersKayit",
	}

	payload := map[string]interface{}{
		"ECRN": crns,       // Example CRNs to be added
		"SCRN": []string{}, // Example CRNs to be deleted
	}

	resp, err := client.R().SetHeaders(headers).SetBody(payload).Post(apiURL)

	if err != nil {
		return nil, err
	}
	var response Response
	err = json.Unmarshal(resp.Body(), &response)
	if err != nil {
		return nil, err
	}

	return &response, nil
}
