package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"chainguard.dev/googkol/pkg/google"
	"chainguard.dev/googkol/pkg/kolide"
)

var (
	endpointsCSVFlag = flag.String("endpoints-csv", "", "Path to Google Endpoints CSV file")

	kolideAPIKey = os.Getenv("KOLIDE_API_KEY")
	//	googleAPIKey     = os.Getenv("GOOGLE_API_KEY")
	slackWebhookURL  = os.Getenv("SLACK_WEBHOOK_URL")
	maxAge           = 5 * 24 * time.Hour
	googleDateFormat = "January 2, 2006 at 3:04 PM MST"
)

func analyze(ks []kolide.Device, gs []google.Device) (map[string]string, error) {
	kDevices := map[string]map[string][]kolide.Device{}

	for _, k := range ks {
		email := k.AssignedOwner.Email
		log.Printf("k: %+v", k)
		if kDevices[email] == nil {
			kDevices[email] = map[string][]kolide.Device{
				"Windows": []kolide.Device{},
				"macOS":   []kolide.Device{},
				"Linux":   []kolide.Device{},
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

	gDevices := map[string]map[string][]google.Device{}
	inScope := 0

	for _, g := range gs {
		// Empty record
		if g.Name == "" {
			continue
		}
		log.Printf("g: %+v", g)

		seen, err := time.Parse(googleDateFormat, g.LastSync)
		if err != nil {
			return nil, fmt.Errorf("parse error for %s: %w", g.LastSync, err)
		}

		if time.Since(seen) > maxAge {
			continue
		}

		inScope++
		email := g.Email
		if gDevices[email] == nil {
			gDevices[email] = map[string][]google.Device{
				"Windows":  []google.Device{},
				"macOS":    []google.Device{},
				"Linux":    []google.Device{},
				"ChromeOS": []google.Device{},
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

	log.Printf("Google: found %d in-scope devices", inScope)
	issues := map[string]string{}

	for email, gOS := range gDevices {
		kOS, ok := kDevices[email]
		if !ok {
			issues[email] = fmt.Sprintf("No devices are registered to Kolide, missing: %+v", gOS)
			continue
		}

		mismatches := []string{}
		for _, os := range []string{"Linux", "macOS", "Windows"} {
			if len(gOS[os]) > len(kOS[os]) {
				gDevs := []string{}
				for _, gd := range gOS[os] {
					gDevs = append(gDevs, gd.String())
				}

				kDevs := []string{}
				for _, kd := range kOS[os] {
					kDevs = append(kDevs, kd.String())
				}

				text := fmt.Sprintf("Google sees %d %s devices, Kolide sees %d\nGoogle:\n  %s\nKolide:\n  %s\n",
					len(gOS[os]), os, len(kOS[os]), strings.Join(kDevs, ", \n  "), strings.Join(gDevs, ", \n  "))
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
