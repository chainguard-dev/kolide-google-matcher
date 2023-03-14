package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/slack-go/slack"

	"github.com/chainguard-dev/kolide-google-matcher/pkg/google"
	"github.com/chainguard-dev/kolide-google-matcher/pkg/kolide"
	"github.com/chainguard-dev/kolide-google-matcher/pkg/mismatch"
)

var (
	endpointsCSVFlag = flag.String("endpoints-csv", "", "Path to Google Endpoints CSV file")

	kolideAPIKey = os.Getenv("KOLIDE_API_KEY")
	//	googleAPIKey     = os.Getenv("GOOGLE_API_KEY")
	slackWebhookURL = os.Getenv("SLACK_WEBHOOK_URL")
)

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

	mismatches := mismatch.Analyze(ks, gs)
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
