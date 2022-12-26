package main

import "fmt"
import "time"
import "./gotoggl"
import "github.com/mattn/go-redmine"
import "encoding/json"
import "net/http"
import "strings"
import "errors"
import "strconv"
import "regexp"
import "flag"
import "os"
import "crypto/tls"

func main() {
	boolRun := flag.Bool("run", false, "Send TimeEntries to redmine")
	flag.Parse()

	args := flag.Args()
	if len(args) != 1 {
		panic("Date argument is missing")
	}

	location, err := time.LoadLocation("UTC")
	if err != nil {
		panic(err)
	}

	parsedDateArgument, err := time.ParseInLocation("2006-01-02", args[0], location)
	if err != nil {
		panic(err)
	}

	year, err := strconv.Atoi(parsedDateArgument.Format("2006"))
	if err != nil {
		panic(err)
	}

	month := parsedDateArgument.Month()

	day, err := strconv.Atoi(parsedDateArgument.Format("02"))
	if err != nil {
		panic(err)
	}

	togglApiKey := os.Getenv("T2R_TOGGL_API_KEY")
	if 0 == len(togglApiKey) {
		panic("Env var T2R_TOGGL_API_KEY has not been set")
	}

	client := gotoggl.NewClient(togglApiKey)
	service := client.TimeEntries
	start := time.Date(year, month, day, 0, 0, 0, 0, location)
	end := time.Date(year, month, day, 23, 59, 0, 0, location)

	fmt.Printf("Search for time entries on " + parsedDateArgument.Format("2006-01-02") + "\n")

	timeEntries, err := service.Range(start, end)
	if err != nil {
		panic(err)
	}

	redmineEndpoint := os.Getenv("T2R_REDMINE_ENDPOINT")
	if 0 == len(redmineEndpoint) {
		panic("Env var T2R_REDMINE_ENDPOINT has not been set")
	}

	redmineApiKey := os.Getenv("T2R_REDMINE_API_KEY")
	if 0 == len(redmineApiKey) {
		panic("Env var T2R_REDMINE_API_KEY has not been set")
	}

	togglProjectIdString := os.Getenv("T2R_TOGGL_PROJECT_ID")
	if 0 == len(togglProjectIdString) {
		panic("Env var T2R_TOGGL_PROJECT_ID has not been set")
	}

        // on évite une erreur x509: certificate signed by unknown authority
        // mais ça serait mieux de passer par une variable d'environnement
        http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}

	togglProjectId, err := strconv.Atoi(togglProjectIdString)
	if err != nil {
		panic(err)
	}

	redmineClient := redmine.NewClient(redmineEndpoint, redmineApiKey)

	activities, err := redmineClient.TimeEntryActivities()
	if err != nil {
		panic(err)
	}

	activitiesArray := make(map[string]int)
	for _, activity := range activities {
		activitiesArray[activity.Name] = activity.Id
	}

        filter := redmine.NewFilter()
        filter.AddPair("user_id", "me")
        filter.AddPair("spent_on", start.Format("2006-01-02"))
        existingTimeEntries, err := redmineClient.TimeEntriesWithFilter(*filter)
	if err != nil {
		panic(err)
	}

        if (0 != len(existingTimeEntries)) {
          panic("There is already existing time entries for this day")
        }

	var totalHours float32

	for _, timeEntry := range timeEntries {
		if timeEntry.ProjectId != int(togglProjectId) {
			continue
		}

		var issueId string
		var comment string
		r := regexp.MustCompile(`(?P<Ticket>\d+).*-(?P<Comment>.*)?`)
		result := r.FindStringSubmatch(timeEntry.Description)
		if 0 == len(result) {
			r := regexp.MustCompile(`(?P<Ticket>\d+).*`)
			result := r.FindStringSubmatch(timeEntry.Description)
			if 0 == len(result) {
				panic("Issue number not found : " + timeEntry.Description)
			} else {
				issueId = result[1]
			}
		} else {
			issueId = result[1]
			comment = strings.TrimSpace(result[2])
		}

		activityId := findActivityId(activitiesArray, timeEntry.Tags)
		if activityId == 0 {
			panic("No activity found in tags" + fmt.Sprintf("%+v", timeEntry.Tags) + " for " + timeEntry.Description + "\n")
		}

		aa := TimeEntryRequestParameters{
			IssueId:    issueId,
			Hours:      float32(timeEntry.Duration.Hours()),
			SpentOn:    timeEntry.Start.Format("2006-01-02"),
			ActivityId: fmt.Sprint(activityId),
			Comment:    comment,
		}

		totalHours = totalHours + float32(timeEntry.Duration.Hours())

		fmt.Printf("%s\n", displayableEntry(aa))

		if *boolRun {
			tre, err := CreateTimeEntry(redmineClient, redmineEndpoint, redmineApiKey, aa)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Time entry %d created\n", tre.Id)
		}
	}
	fmt.Printf("Total hours : %f\n", totalHours)

}

type errorsResult struct {
	Errors []string `json:"errors"`
}

type timeEntryResult struct {
	TimeEntry TimeEntry `json:"time_entry"`
}

type timeEntryRequest struct {
	TimeEntry TimeEntryRequestParameters `json:"time_entry"`
}

type TimeEntry struct {
	Id int `json:"id"`
}

type TimeEntryRequestParameters struct {
	IssueId    string  `json:"issue_id"`
	SpentOn    string  `json:"spent_on"`
	Hours      float32 `json:"hours"`
	ActivityId string  `json:"activity_id"`
	Comment    string  `json:"comments"`
}

func findActivityId(activitiesArray map[string]int, tags []string) int {
	for _, tag := range tags {
		i := activitiesArray[tag]
		if i != 0 {
			return i
		}
	}
	return 0
}

func displayableEntry(timeEntry TimeEntryRequestParameters) string {
	s, err := json.Marshal(timeEntry)
	if err != nil {
		//return nil, err
	}
	so := fmt.Sprintf("%s", s)
	return so
}

func CreateTimeEntry(c *redmine.Client, endpoint string, apiKey string, aa TimeEntryRequestParameters) (*TimeEntry, error) {

	bb := timeEntryRequest{
		TimeEntry: aa,
	}

	s, err := json.Marshal(bb)
	if err != nil {
		return nil, err

	}

	req, err := http.NewRequest("POST", endpoint+"/time_entries.json?key="+apiKey, strings.NewReader(string(s)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	decoder := json.NewDecoder(res.Body)
	var r timeEntryResult
	if res.StatusCode != 201 {
		var er errorsResult
		err = decoder.Decode(&er)
		if err == nil {
			err = errors.New(strings.Join(er.Errors, "\n"))
		}
	} else {
		err = decoder.Decode(&r)
	}
	if err != nil {
		return nil, err
	}
	return &r.TimeEntry, nil
}
