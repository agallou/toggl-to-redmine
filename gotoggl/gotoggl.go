// Package gotoggl provides access to the Toggl API.
package gotoggl

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"
)

var _ = json.Unmarshal
var _ = log.Print
var _ = time.Time{}
var _ = fmt.Print

const (
	TogglApi   = "https://www.toggl.com/api/v8/"
	ReportsApi = "https://toggl.com/reports/api/v2/"
	UserAgent  = "github.com/roessland/gotoggl"
)

// Duration encapsulates the standard Duration in an anonymous field. Toggl
// returns durations in seconds, but time.Duration uses nanoseconds. Therefore
// we have to implement a custom UnmarshalJSON.
type Duration struct{ time.Duration }

// UnmarshalJSON loads a Toggl duration into a Go duration. Toggl durations are
// given in seconds.
func (d *Duration) UnmarshalJSON(data []byte) error {
	seconds, err := strconv.ParseInt(string(data), 10, 64)
	if err != nil {
		fmt.Errorf("Couldn't unmarshal toggl.Duration: %v\n", err)
	}
	d.Duration = time.Duration(seconds * int64(time.Second))
	return nil
}

// TimeEntry contains the data returned for a single time entry.
type TimeEntry struct {
	Id          int
	Description string
	WorkspaceId int `json:"wid"`
	ProjectId   int `json:"pid"`
	Guid        string
	Billable    bool
	Start       time.Time
	Stop        time.Time
	Duration    Duration
	DurOnly     bool
	UserId      int    `json:"uid"`
	CreatedWith string `json:"created_with"`
	Tags        []string
	At          string
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

// Get returns details of a single time entry
func (tes *TimeEntriesService) Get(id int) (TimeEntry, error) {
	panic("Get() Not yet implemented")
	return TimeEntry{}, nil
}

// Current returns running time entry
func (tes *TimeEntriesService) Current() (TimeEntry, error) {
	panic("Current() not yet implemented")
	return TimeEntry{}, nil
}

// Range returns time entries started in a specific time range. Only the first
// 1000 found time entries are returned. There is no pagination.
func (tes *TimeEntriesService) Range(start, end time.Time) ([]TimeEntry, error) {
	timeEntries := []TimeEntry{}
	t0 := start.Format(time.RFC3339)
	t1 := end.Format(time.RFC3339)
	path := fmt.Sprintf("time_entries?start_date=%s&end_date=%s", t0, t1)
	err := tes.client.GET(path, &timeEntries)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get time entries: %v\n", err)
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

// GET does a GET operation to the main API (not the reports API) and
// unmarshals the result into the given interface.
func (c *Client) GET(path string, response interface{}) error {
	if len(path) > 0 && path[0] == '/' {
		log.Print("Warning: Do not include / at the start of path")
	}
	req, _ := http.NewRequest("GET", TogglApi+path, nil)
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

/*

type TogglTimeEntry struct {
	Id          int
	Description string
	WorkspaceId int `json:"wid"`
	ProjectId   int `json:"pid"`
	Guid        string
	Billable    bool
	Start       time.Time
	Stop        time.Time
	Duration    int
	DurOnly     bool
	UserId      int    `json:"uid"`
	CreatedWith string `json:"created_with"`
	Tags        []string
	At          string
}

type TogglTimeEntryResponse struct {
	Data TogglTimeEntry
}

type TogglProject struct {
	ID            int
	GUID          string
	WID           int
	CID           int
	Name          string
	Billable      bool
	IsPrivate     bool `json:"is_private"`
	Active        bool
	Template      bool
	At            time.Time
	CreatedAt     time.Time `json:"created_at"`
	Color         string
	AutoEstimates bool `json:"auto_estimates"`
	ActualHours   int  `json:"actual_hours"`
}

type TogglProjectResponse struct {
	Data TogglProject
}

type TogglProjectSummary struct {
	Id int
	// Items []???
	Time  int // Duration in milliseconds
	Title struct {
		Client   string
		Color    string
		HexColor string `json:"hex_color"`
		Project  string
	}
	// TotalCurrencies []Currency `json:"total_currencies"`
}

type TogglProjectSummariesResponse struct {
	Data []TogglProjectSummary
}
*/
