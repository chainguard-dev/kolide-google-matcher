package kolide

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

var timeFormat = "Jan 2, 2006"

type Client struct {
	apiKey string
}

func New(apiKey string) *Client {
	return &Client{apiKey: apiKey}
}

type Device struct {
	ID                   int       `json:"id"`
	Name                 string    `json:"name"`
	OwnedBy              string    `json:"owned_by"`
	Privacy              string    `json:"privacy"`
	Platform             string    `json:"platform"`
	EnrolledAt           time.Time `json:"enrolled_at"`
	LastSeenAt           time.Time `json:"last_seen_at"`
	OperatingSystem      string    `json:"operating_system"`
	IssueCount           int       `json:"issue_count"`
	ResolvedIssueCount   int       `json:"resolved_issue_count"`
	FailureCount         int       `json:"failure_count"`
	ResolvedFailureCount int       `json:"resolved_failure_count"`
	PrimaryUserName      string    `json:"primary_user_name"`
	HardwareModel        string    `json:"hardware_model"`
	HardwareVendor       string    `json:"hardware_vendor"`
	LauncherVersion      string    `json:"launcher_version"`
	OsqueryVersion       string    `json:"osquery_version"`
	Serial               string    `json:"serial"`
	AssignedOwner        struct {
		ID        int    `json:"id"`
		OwnerType string `json:"owner_type"`
		Name      string `json:"name"`
		Email     string `json:"email"`
	} `json:"assigned_owner"`
	KolideMdm              interface{} `json:"kolide_mdm"`
	Note                   interface{} `json:"note"`
	NoteHTML               interface{} `json:"note_html"`
	OperatingSystemDetails struct {
		DeviceID     int         `json:"device_id"`
		Platform     string      `json:"platform"`
		Name         string      `json:"name"`
		Codename     string      `json:"codename"`
		Version      string      `json:"version"`
		Build        string      `json:"build"`
		MajorVersion int         `json:"major_version"`
		MinorVersion int         `json:"minor_version"`
		PatchVersion int         `json:"patch_version"`
		Ubr          interface{} `json:"ubr"`
		ReleaseID    interface{} `json:"release_id"`
	} `json:"operating_system_details"`
	RemoteIP        string      `json:"remote_ip"`
	Location        interface{} `json:"location"`
	ProductImageURL string      `json:"product_image_url"`
}

func (d *Device) String() string {
	name := fmt.Sprintf("%s (%s)", d.Name, d.OperatingSystem)
	return fmt.Sprintf("%-60.60s [%s to %s]", name, d.EnrolledAt.Format(timeFormat), d.LastSeenAt.Format(timeFormat))
}

type getAllDevicesResponse struct {
	Devices    []Device `json:"data"`
	Pagination struct {
		Next          string `json:"next"`
		NextCursor    string `json:"next_cursor"`
		CurrentCursor string `json:"current_cursor"`
		Count         int    `json:"count"`
	}
}

func (c *Client) GetAllDevices() ([]Device, error) {
	allDevices := []Device{}
	var nextCursor string
	for {
		url := "https://k2.kolide.com/api/v0/devices"
		if nextCursor != "" {
			url = fmt.Sprintf("%s?cursor=%s", url, nextCursor)
		}
		bearer := fmt.Sprintf("Bearer %s", c.apiKey)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Add("Authorization", bearer)
		httpClient := &http.Client{}
		r, err := httpClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			return nil, err
		}
		var response getAllDevicesResponse
		if err = json.Unmarshal(body, &response); err != nil {
			return nil, err
		}
		for _, device := range response.Devices {
			d := device
			//			log.Printf("kolide device: %v", d)
			allDevices = append(allDevices, d)
		}
		if response.Pagination.NextCursor == "" {
			break
		}
		nextCursor = response.Pagination.NextCursor
	}
	return allDevices, nil
}
