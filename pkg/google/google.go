package google

type Client struct {
	apiKey string
}

func NewClient(apiKey string) *Client {
	return &Client{apiKey: apiKey}
}

// TODO: what is format here
type Device struct {
	Name     string `json:"Name"`
	Email    string `json:"Email"`
	OS       string `json:"OS"`
	Type     string `json:"Type"`
	LastSync string `json:"Last Sync"`
}

// TODO: actually implement
func (c *Client) GetAllDevices() ([]Device, error) {
	allDevices := []Device{}
	return allDevices, nil
}
