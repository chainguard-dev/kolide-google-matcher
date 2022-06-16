package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/slack-go/slack"
)

var (
	slackWebhookURL  = os.Getenv("SLACK_WEBHOOK_URL")
	kolideCSVFlag    = flag.String("kolide-csv", "", "G")
	googleCSVFlag    = flag.String("google-csv", "", "this is a personal machine")
	maxAge           = 4 * 24 * time.Hour
	googleDateFormat = "January 2, 2006 at 3:04 PM MST"
)

type KolideRecord struct {
	ID         int    `csv:"ID"`
	DeviceName string `csv:"Device Name"`
	Type       string `csv:"Type"`
	OS         string `csv:"OS"`
	OwnerEmail string `csv:"Owner Email"`
}

func parseKolideCSV(path string) ([]*KolideRecord, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()
	ks := []*KolideRecord{}
	err = gocsv.UnmarshalFile(f, &ks)
	return ks, err
}

type GoogleRecord struct {
	Name     string `csv:"Name"`
	Email    string `csv:"Email"`
	OS       string `csv:"OS"`
	Type     string `csv:"Type"`
	LastSync string `csv:"Last Sync"`
}

func parseGoogleCSV(path string) ([]*GoogleRecord, error) {
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()
	gs := []*GoogleRecord{}
	err = gocsv.UnmarshalFile(f, &gs)
	return gs, err
}

type Found struct {
	Mac      int
	Linux    int
	Windows  int
	ChromeOS int
}

func (f *Found) String() string {
	if f == nil {
		return ""
	}

	ds := []string{}

	if f.Mac > 0 {
		ds = append(ds, fmt.Sprintf("%d macOS devices", f.Mac))
	}

	if f.Linux > 0 {
		ds = append(ds, fmt.Sprintf("%d Linux devices", f.Linux))
	}

	if f.Windows > 0 {
		ds = append(ds, fmt.Sprintf("%d Windows devices", f.Windows))
	}

	return strings.Join(ds, ", ")
}

func analyze(ks []*KolideRecord, gs []*GoogleRecord) (map[string]string, error) {
	ik := map[string]*Found{}

	for _, k := range ks {
		log.Printf("k: %+v", k)
		if ik[k.OwnerEmail] == nil {
			ik[k.OwnerEmail] = &Found{}
		}
		switch k.Type {
		case "LinuxDevice":
			ik[k.OwnerEmail].Linux++
		case "WindowsDevice":
			ik[k.OwnerEmail].Windows++
		case "Mac":
			ik[k.OwnerEmail].Mac++
		}
	}

	for email, f := range ik {
		log.Printf("%s: %+v", email, f)
	}

	ig := map[string]*Found{}
	inScope := 0
	for _, g := range gs {
		log.Printf("g: %+v", g)

		seen, err := time.Parse(googleDateFormat, g.LastSync)
		if err != nil {
			return nil, fmt.Errorf("parse error for %s: %w", g.LastSync, err)
		}

		if time.Since(seen) > maxAge {
			continue
		}

		inScope++

		if ig[g.Email] == nil {
			ig[g.Email] = &Found{}
		}
		switch g.Type {
		case "Linux":
			ig[g.Email].Linux++
		case "Windows":
			ig[g.Email].Windows++
		case "Mac":
			ig[g.Email].Mac++
		case "Chrome OS":
			ig[g.Email].ChromeOS++
		default:
			log.Printf("Ignoring type %s (%s)", g.Type, g.OS)
			inScope--
		}
	}

	log.Printf("Google: found %d in-scope devices", inScope)
	issues := map[string]string{}

	for e, g := range ig {
		k, ok := ik[e]
		if !ok {
			gs := g.String()
			// If no Kolide-enrollable devices are found, skip this line.
			if gs != "" {
				issues[e] = fmt.Sprintf("No devices are registered to Kolide, missing: %s", g)
			}
			continue
		}

		mismatches := []string{}
		if g.Linux > k.Linux {
			mismatches = append(mismatches, fmt.Sprintf("Google sees %d Linux devices, Kolide sees %d", g.Linux, k.Linux))
		}

		if g.Mac > k.Mac {
			mismatches = append(mismatches, fmt.Sprintf("Google sees %d macOS devices, Kolide sees %d", g.Mac, k.Mac))
		}

		if g.Windows > k.Windows {
			mismatches = append(mismatches, fmt.Sprintf("Google sees %d Windows devices, Kolide sees %d", g.Windows, k.Windows))
		}

		if len(mismatches) > 0 {
			issues[e] = strings.Join(mismatches, ", ")
		}
	}

	return issues, nil
}

func main() {
	flag.Parse()

	ks, err := parseKolideCSV(*kolideCSVFlag)
	if err != nil {
		log.Fatalf("kolide: %v", err)
	}

	gs, err := parseGoogleCSV(*googleCSVFlag)
	if err != nil {
		log.Fatalf("google: %v", err)
	}

	mismatches, err := analyze(ks, gs)
	if err != nil {
		log.Fatalf("analyze: %v", err)
	}

	for k, v := range mismatches {
		if v != "" {
			log.Printf("%s mismatch: %s", k, v)
		}
	}

	// If SLACK_WEBHOOK_URL set in environment, send a copy of the output to Slack
	if slackWebhookURL != "" {
		log.Println("Attempting to send output to provided Slack webhook...")
		lines := []string{}
		for k, v := range mismatches {
			if v != "" {
				lines = append(lines, fmt.Sprintf("%s mismatch: %s", k, v))
			}
		}
		msg := slack.WebhookMessage{
			Text: strings.Join(lines, "\n\n"),
		}
		if err := slack.PostWebhook(slackWebhookURL, &msg); err != nil {
			log.Fatalf("posting slack webhook: %v", err)
		}
		log.Println("Success.")
	}
}
