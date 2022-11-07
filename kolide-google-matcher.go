package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"github.com/chainguard-dev/kolide-google-matcher/pkg/google"
	"github.com/chainguard-dev/kolide-google-matcher/pkg/kolide"
)

var (
	endpointsCSVFlag = flag.String("endpoints-csv", "", "Path to Google Endpoints CSV file")

	kolideAPIKey = os.Getenv("KOLIDE_API_KEY")
	//	googleAPIKey     = os.Getenv("GOOGLE_API_KEY")
	slackWebhookURL = os.Getenv("SLACK_WEBHOOK_URL")
	maxAge          = 5 * 24 * time.Hour
)

func analyze(ks []kolide.Device, gs []google.Device) (map[string]string, error) {
	kDevices := map[string]map[string][]kolide.Device{}

	for _, k := range ks {
		email := k.AssignedOwner.Email
		// log.Printf("k: %+v", k)
		if kDevices[email] == nil {
			kDevices[email] = map[string][]kolide.Device{
				"Windows": {},
				"macOS":   {},
				"Linux":   {},
			}
		}

		os := ""

		switch k.Platform {
		case "windows":
			os = "Windows"
		case "darwin":
			os = "macOS"
		default:
			// Assume Linux, this could be various values (arch, rhel, etc.)
			os = "Linux"
		}

		kDevices[email][os] = append(kDevices[email][os], k)
	}

	log.Printf("Kolide: found %d devices", len(ks))

	gDevices := map[string]map[string][]google.Device{}
	inScope := 0

	for _, g := range gs {
		// Empty record
		if g.Name == "" {
			continue
		}
		//	log.Printf("g: %+v", g)

		if time.Since(g.LastSyncTime) > maxAge {
			continue
		}

		inScope++
		email := g.Email
		if gDevices[email] == nil {
			gDevices[email] = map[string][]google.Device{
				"Windows":  {},
				"macOS":    {},
				"Linux":    {},
				"ChromeOS": {},
			}
		}

		os := ""
		switch g.Type {
		case "Linux":
			os = "Linux"
		case "Windows":
			os = "Windows"
		case "Mac":
			os = "macOS"
		case "Chrome OS":
			os = "ChromeOS"
		default:
			log.Printf("Ignoring type %s (%s)", g.Type, g.OS)
			inScope--
		}
		gDevices[email][os] = append(gDevices[email][os], g)
	}

	log.Printf("Google: found %d devices", inScope)
	issues := map[string]string{}

	for email, gOS := range gDevices {
		kOS, ok := kDevices[email]

		gDevs := []string{}

		for _, gds := range gOS {
			for _, gd := range gds {
				gDevs = append(gDevs, gd.String())
			}
		}

		if !ok {
			text := fmt.Sprintf("Google sees %d devices, Kolide sees 0!\nGoogle:\n  %s\n\n",
				len(gDevs), strings.Join(gDevs, "\n  "))
			issues[email] = text
			continue
		}

		mismatches := []string{}
		for _, os := range []string{"Linux", "macOS", "Windows"} {

			kDevs := []string{}
			for _, kd := range kOS[os] {
				kDevs = append(kDevs, kd.String())
			}

			gDevs = []string{}
			for _, gd := range gOS[os] {
				gDevs = append(gDevs, gd.String())
			}
			if len(gOS[os]) > len(kOS[os]) {
				text := fmt.Sprintf("Google sees %d %s devices, Kolide sees %d\nGoogle:\n  %s\nKolide:\n  %s\n\n",
					len(gOS[os]), os, len(kOS[os]), strings.Join(gDevs, "\n  "), strings.Join(kDevs, "\n  "))
				mismatches = append(mismatches, text)
				issues[email] = strings.Join(mismatches, "\n")
			}
		}
	}

	return issues, nil
}

func main() {
	flag.Parse()

	if kolideAPIKey == "" {
		log.Fatal("Missing KOLIDE_API_KEY. Exiting.")
	}

	if *endpointsCSVFlag == "" {
		log.Fatal("--endpoints-csv is mandatory: download from https://admin.google.com/ac/devices/list?default=true&category=desktop")
	}

	ks, err := kolide.New(kolideAPIKey).GetAllDevices()
	if err != nil {
		log.Fatalf("kolide: %v", err)
	}
	gs, err := google.New(*endpointsCSVFlag).GetAllDevices()
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

	log.Printf("found %d total mismatches", len(mismatches))

	// If SLACK_WEBHOOK_URL set in environment, send a copy of the output to Slack
	if slackWebhookURL != "" {
		log.Println("---\nAttempting to send output to provided Slack webhook...")
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
