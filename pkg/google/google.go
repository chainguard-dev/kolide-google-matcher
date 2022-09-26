package google

import (
	"fmt"
	"os"

	"github.com/gocarina/gocsv"
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
}

func (d *Device) String() string {
	return fmt.Sprintf("%s (%s) [%s - %s]", d.DeviceName, d.OS, d.FirstSync, d.LastSync)
}

func (c *Client) GetAllDevices() ([]Device, error) {
	f, err := os.OpenFile(c.csvPath, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()
	ds := []Device{}
	err = gocsv.UnmarshalFile(f, &ds)
	return ds, err
}
