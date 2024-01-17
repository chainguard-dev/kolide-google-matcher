package google

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
)

var (
	googleDateFormat = "January 2, 2006 at 3:04 PM MST"
	timeFormat       = "Jan 2, 2006"
)

type Client struct {
	csvPath string
}

func New(csvPath string) *Client {
	return &Client{csvPath: csvPath}
}

// Device is the struct returned by the CSV output at https://admin.google.com/ac/devices/list?default=true&category=desktop.
type Device struct {
	Name       string `csv:"Name"`
	Email      string `csv:"Email"`
	OS         string `csv:"OS"`
	Type       string `csv:"Type"`
	LastSync   string `csv:"Last Sync"`
	FirstSync  string `csv:"First Sync"`
	DeviceName string `csv:"Device Name"`
	HostName   string `csv:"Host Name"`

	FirstSyncTime time.Time
	LastSyncTime  time.Time
}

func (d *Device) String() string {
	name := fmt.Sprintf("%s [%s]", d.DeviceName, d.OS)
	return fmt.Sprintf("%-45.45s %s — %s", name, d.FirstSyncTime.Format(timeFormat), d.LastSyncTime.Format(timeFormat))
}

func (c *Client) GetAllDevices() ([]Device, error) {
	bs, err := os.ReadFile(c.csvPath)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	if len(bs) < 128 {
		return nil, fmt.Errorf("%s appears to be incomplete - only %d bytes found", c.csvPath, (bs))
	}

	ds := []Device{}
	err = gocsv.UnmarshalBytes(bs, &ds)
	if len(ds) == 0 {
		firstLine := bytes.Split(bs, []byte("\n"))
		return nil, fmt.Errorf("no valid CSV rows found in %s, first line is:\n%s", c.csvPath, firstLine)
	}

	for i, d := range ds {
		if d.DeviceName == "" {
			return nil, fmt.Errorf("row is missing Device Name column: %s - %+v", d.DeviceName, d)
		}
		if d.OS == "" {
			d.OS = d.Type
		}

		// At some point Google switched the ASCII space for a unicode short space
		d.FirstSync = strings.ReplaceAll(d.FirstSync, "\xe2\x80\xaf", " ")
		d.LastSync = strings.ReplaceAll(d.LastSync, "\xe2\x80\xaf", " ")
		ts, err := time.Parse(googleDateFormat, d.LastSync)
		if err != nil {
			log.Printf("LastSync: parse error for %s: %v", d.LastSync, err)
		} else {
			d.LastSyncTime = ts
		}

		ts, err = time.Parse(googleDateFormat, d.FirstSync)
		if err != nil {
			log.Printf("FirstSync: parse error for %s: %v", d.FirstSync, err)
		} else {
			d.FirstSyncTime = ts
		}
		ds[i] = d
	}

	return ds, err
}
