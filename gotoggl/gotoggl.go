// Package gotoggl provides access to the Toggl API.
package gotoggl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"
        "strings"
        "io"
)

var _ = json.Unmarshal
var _ = log.Print
var _ = time.Time{}
var _ = fmt.Print

const (
	TogglApi   = "https://api.track.toggl.com/api/v9/"
	ReportsApi = "https://toggl.com/reports/api/v2/"
	UserAgent  = "github.com/roessland/gotoggl"
)

// TimeEntry contains the data returned for a single time entry.
type TimeEntry struct {
	Description string
	ProjectId   int `json:"pid"`
	Start       time.Time
	Duration    time.Duration
	Tags        []string
}

type Me struct {
	Id          int
	DefaultWorkspaceId int `json:"default_workspace_id"`
}

// TimeEntryResponse is a wrapper for the data returned by /time_entries
type TimeEntryResponse struct {
	Data TimeEntry
}

// TimeEntriesResponse is an alias for []TimeEntry. For convenience.
type TimeEntriesResponse []TimeEntry

// TimeEntriesService accesses /time_entries
type TimeEntriesService struct {
	client *Client
}

type searchRequest struct {
        EndDate    string  `json:"end_date"`
        StartDate  string  `json:"start_date"`
}

type searchTimeEntry struct {
	Description string
	ProjectId   int `json:"project_id"`
	Billable    bool
	Start       time.Time
	TagIds      []int `json:"tag_ids"`
        TimeEntries []searchTimeEntryDetail `json:"time_entries"`
}

type searchTimeEntryDetail struct {
	Start       time.Time
	Seconds   int `json:"seconds"`
}

type TagItem struct {
	Id int `json:"id"`
	Name string `json:"name"`
} 

// Range returns time entries started in a specific time range. Only the first
// 1000 found time entries are returned. There is no pagination.
func (tes *TimeEntriesService) Range(start, end time.Time) ([]TimeEntry, error) {
        // On récupère le workspace par défaut
        me := Me{}
	pathMe := fmt.Sprintf("me")
	errMe := tes.client.GET(pathMe, &me)
	if errMe != nil {
		return nil, fmt.Errorf("Couldn't get me: %v\n", errMe)
	}

        fmt.Println(fmt.Sprintf("Default workspace ID found : %d", me.DefaultWorkspaceId))


        // on liste tous les ids
        allTags := []TagItem{}
        errTags := tes.client.GET(fmt.Sprintf("workspaces/%d/tags", me.DefaultWorkspaceId), &allTags)
	if errTags != nil {
		return nil, fmt.Errorf("Couldn't get tags: %v\n", errTags)
	}

        // on prépare le body pour récupérer les time entries
        aa := searchRequest{
                EndDate: end.Format("2006-01-02"),
                StartDate: start.Format("2006-01-02"),
        }

        s, errM := json.Marshal(aa)
        if errM != nil {
                return nil, errM

        }

        // on recherche les time entries
	searchTimeEntries := []searchTimeEntry{}
	path := fmt.Sprintf("workspace/%d/search/time_entries", me.DefaultWorkspaceId)
	err := tes.client.POSTOnV3(path, strings.NewReader(string(s)), &searchTimeEntries)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get time entries: %v\n", err)
	}


        // on met en forme les timeEntries
        timeEntries := []TimeEntry{}

	for _, searchTimeEntry := range searchTimeEntries {
          te := TimeEntry{}

          te.Description = searchTimeEntry.Description
          te.ProjectId = searchTimeEntry.ProjectId

	  for _, searchTimeEntryDetail := range searchTimeEntry.TimeEntries {
             te.Start = searchTimeEntryDetail.Start
             d, _ := time.ParseDuration(fmt.Sprintf("%ds", searchTimeEntryDetail.Seconds))
             te.Duration = d
          }

          var tags []string
          for _, tagId := range searchTimeEntry.TagIds {
            var tagName string
            for _, allTag := range allTags {
              if (allTag.Id == tagId) {
                tagName = allTag.Name
              }
            }
            if (0 == len(tagName)) {
              panic(fmt.Sprintf("Name not found for tag id %d", tagId))
            }

            tags = append(tags, tagName)
          }

          te.Tags = tags

          timeEntries = append(timeEntries, te)
        }


	return timeEntries, nil
}

type User struct {
	ApiToken              string `json:"api_token"`
	DefaultWorkspaceId    int    `json:"default_wid"`
	Email                 string
	JqueryTimeofdayFormat string `json:"jquery_timeofday_format"`
	JqueryDateFormat      string `json:"jquery_date_format"`
	TimeofdayFormat       string `json:"timeofday_format"`
	DateFormat            string `json:"date_format"`
	StoreStartAndStopTime bool   `json:"store_start_and_stop_time"`
	BeginningOfWeek       int    `json:"beginning_of_week"`
	Language              string
	ImageUrl              string `json:"image_url"`
	SidebarPiechart       bool   `json:"sidebar_piechart"`
	At                    time.Time
	NewBlogPost           struct {
		Title    string
		Url      string
		Category string
		PubDate  string `json:"pub_date"`
	} `json:"new_blog_post"`
	SendProductEmails      bool `json:"send_product_emails"`
	SendWeeklyReport       bool `json:"send_weekly_report"`
	SendTimerNotifications bool `json:"send_timer_notifications"`
	OpenIdEnabled          bool `json:"openid_enabled"`
	Timezone               string
}

type UserResponse struct {
	Data User
}

// TimeEntriesService accesses current user data
type MeService struct {
	client *Client
}

// Get returns details of current user
func (ms *MeService) Get() (User, error) {
	userResp := UserResponse{}
	err := ms.client.GET("me", &userResp)
	if err != nil {
		return User{}, fmt.Errorf("Couldn't get time entries: %v\n", err)
	}
	return userResp.Data, nil
}

// Client accesses the Toggl API using a given API key.
type Client struct {
	client      *http.Client
	ApiKey      string
	TimeEntries *TimeEntriesService
	Me          *MeService
}

// NewClient creates a new Toggl API client using an API key.
func NewClient(apiKey string) *Client {
	c := &Client{
		client: &http.Client{},
		ApiKey: apiKey,
	}
	c.TimeEntries = &TimeEntriesService{client: c}
	c.Me = &MeService{client: c}
	return c
}

func (c *Client) GET(path string, response interface{}) error {
        return c.genericDoRequest("GET", TogglApi, path, nil, response)
}

func (c *Client) POSTOnV3(path string, body io.Reader, response interface{}) error {
        return c.genericDoRequest("POST", "https://api.track.toggl.com/reports/api/v3/", path, body, response)
}

func (c *Client) genericDoRequest(method string, endpoint string, path string, body io.Reader, response interface{}) error {
	if len(path) > 0 && path[0] == '/' {
		log.Print("Warning: Do not include / at the start of path")
	}

	req, _ := http.NewRequest(method, endpoint + path, body)

	req.SetBasicAuth(c.ApiKey, "api_token")
	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("GET couldn't do request %v: %v\n", path, err)
	}
	defer func() {
		resp.Body.Close()
	}()
	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("GET to %v couldn't read response body: %v\n", req.URL, err)
	}
	if len(buf) == 0 {
		return fmt.Errorf("GET to %v response had length zero.\n", req.URL)
	}
	if err := json.Unmarshal(buf, &response); err != nil {
		return fmt.Errorf("GET couldn't unmarshal response: %v (Response was %v)\n", err, string(buf))
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 400 {
		return fmt.Errorf("GET got wrong status code %v\n", resp.Status)
	}
	return nil
}



