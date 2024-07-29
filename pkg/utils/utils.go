package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	godotenv "github.com/joho/godotenv"
)

// Error codes from Kepler
func GetErrorCodes() map[string]string {
	return map[string]string{
		"successResult": "The operation for the course with CRN %s has been successfully completed.",
		"errorResult":   "No operation was completed in this process group.",
		"error":         "An error occurred during the operation.",
		"VAL01":         "The course with CRN %s cannot be added due to a problem.",
		"VAL02":         "The course with CRN %s cannot be added due to 'Enrollment Time Hold'.",
		"VAL03":         "The course with CRN %s could not be taken again because it was taken this semester.",
		"VAL04":         "The course with CRN %s could not be taken because it was not included in the lesson plan.",
		"VAL05":         "The course with CRN %s cannot be added as the maximum number of credits allowed for this term is exceeded.",
		"VAL06":         "The course with CRN %s cannot be added as the enrollment limit has been reached and there is no quota left.",
		"VAL07":         "The course with CRN %s cannot be re-added because this course has been completed before with an AA grade.",
		"VAL08":         "The course with CRN %s could not be taken because your program is not among the programs that can take this course.",
		"VAL09":         "The course with CRN %s cannot be added due to a time conflict with another course.",
		"VAL10":         "No action has been taken because you are not registered for the course with CRN %s this semester.",
		"VAL11":         "The course with CRN %s cannot be added as its prerequisites are not met.",
		"VAL12":         "The course with CRN %s is not offered in the respective semester.",
		"VAL13":         "The course with CRN %s has been temporarily disabled.",
		"VAL14":         "The system is temporarily disabled.",
		"VAL15":         "You can send a maximum of 12 CRN parameters.",
		"VAL16":         "You currently have an ongoing transaction; try again later.",
		"VAL18":         "The course with CRN %s could not be taken due to 'Attribute Hold'.",
		"VAL19":         "The course with CRN %s could not be taken because it is an undergraduate course.",
		"VAL20":         "You can leave only 1 course per semester.",
		"CRNListEmpty":  "The course with CRN %s is not available during the course selection period.",
		"CRNNotFound":   "The course with CRN %s is not available during the course selection period.",
		"ERRLoad":       "This service is temporarily unavailable.",
		"NULLParam-CheckOgrenciKayitZamaniKontrolu": "The course with CRN %s cannot be added due to 'Enrollment Time Hold'.",
	}
}


type Schedule struct {
	Name string `json:"name"`
	ECRN []int  `json:"ECRN"`
	SCRN []int  `json:"SCRN"`
}

type ScheduleList struct {
	Schedules []Schedule `json:"schedules"`
}

// Loads environment variables from a .env file.
func LoadEnvVariables() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
}

// GetUserSchedules reads the schedules.json file and returns the schedules of the user. (All schedules are returned)
func GetUserSchedules() (ScheduleList, error) {

	path, err := getProjectBasePath()
	if err != nil {
		log.Fatal("Error getting project base path")
		return ScheduleList{}, err
	}

	// open schedules.json
	file, err := os.Open(filepath.Join(path, "schedules.json"))
	if err != nil {
		if os.IsNotExist(err) {
			// Create the file if it doesn't exist
			file, err = os.Create(filepath.Join(path, "schedules.json"))
			if err != nil {
				log.Fatal("Error creating schedules.json")
				return ScheduleList{}, err
			}

			// Write an empty JSON object to the file
			_, err = file.Write([]byte("{\"schedules\":[]}"))
			if err != nil {
				log.Fatal("Error writing schedules.json")
				return ScheduleList{}, err
			}

			// Close file after writing and reopen it so that the file pointer is at the beginning
			// yes, this is a hack but it works
			file.Close()
			file, err = os.Open(filepath.Join(path, "schedules.json"))
			if err != nil {
				log.Fatal("Error opening schedules.json")
				return ScheduleList{}, err
			}

		} else {
			log.Fatal("Error opening schedules.json")
			return ScheduleList{}, err
		}
	}
	defer file.Close()

	// Read the contents of the file
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal("Error reading schedules.json")
		return ScheduleList{}, err
	}

	// Parse the JSON data into a ScheduleList struct
	var scheduleList ScheduleList
	err = json.Unmarshal(data, &scheduleList)
	if err != nil {
		log.Fatal("Error parsing schedules.json")
		return ScheduleList{}, err
	}

	return scheduleList, nil
}

// GetUserSchedule reads the schedules.json file and returns the schedule with the given name.
func GetUserSchedule(schedule_name string) (Schedule, error) {
	scheduleList, err := GetUserSchedules()
	if err != nil {
		return Schedule{}, err
	}

	// Find the schedule with the given name
	var user_schedule Schedule
	for _, schedule := range scheduleList.Schedules {
		if schedule.Name == schedule_name {
			user_schedule = schedule
			break
		}
	}

	// if the schedule is not found, return an error
	if user_schedule.Name == "" {
		return Schedule{}, fmt.Errorf("schedule not found")
	}

	return user_schedule, nil
}

// SaveUserSchedule saves the given schedule to the schedules.json file. (Overwrites the schedule with the same name)
func SaveUserSchedule(schedule Schedule) error {
	path, err := getProjectBasePath()
	if err != nil {
		log.Fatal("Error getting project base path")
		return err
	}

	// open schedules.json
	file, err := os.Open(filepath.Join(path, "schedules.json"))
	if err != nil {
		log.Fatal("Error opening schedules.json")
		return err
	}
	defer file.Close()

	// Read the contents of the file
	data, err := io.ReadAll(file)
	if err != nil {
		log.Fatal("Error reading schedules.json")
		return err
	}

	// Parse the JSON data into a ScheduleList struct
	var scheduleList ScheduleList
	err = json.Unmarshal(data, &scheduleList)
	if err != nil {
		log.Fatal("Error parsing schedules.json")
		return err
	}

	// Find the schedule with the given name
	var found bool
	for i, s := range scheduleList.Schedules {
		if s.Name == schedule.Name {
			scheduleList.Schedules[i] = schedule
			found = true
			break
		}
	}

	// if the schedule is not found, append it to the list
	if !found {
		scheduleList.Schedules = append(scheduleList.Schedules, schedule)
	}

	// Marshal the ScheduleList struct back to JSON
	newData, err := json.Marshal(scheduleList)
	if err != nil {
		log.Fatal("Error marshaling schedules.json")
		return err
	}

	// Write the new JSON data back to the file
	err = os.WriteFile(filepath.Join(path, "schedules.json"), newData, 0644)
	if err != nil {
		log.Fatal("Error writing schedules.json")
		return err
	}

	return nil
}

// Saves the given ScheduleList to the schedules.json file. (Overwrites the file)
func SaveUserSchedules(scheduleList ScheduleList) error {
	path, err := getProjectBasePath()
	if err != nil {
		log.Fatal("Error getting project base path")
		return err
	}

	// Marshal the ScheduleList struct to JSON
	data, err := json.Marshal(scheduleList)
	if err != nil {
		log.Fatal("Error marshaling schedules.json")
		return err
	}

	// Write the JSON data to the file
	err = os.WriteFile(filepath.Join(path, "schedules.json"), data, 0644)
	if err != nil {
		log.Fatal("Error writing schedules.json")
		return err
	}

	return nil
}

// Returns the base path of the project.
func getProjectBasePath() (string, error) {
	basePath, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return basePath, nil
}
