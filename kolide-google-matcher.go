package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/slack-go/slack"

	"github.com/chainguard-dev/kolide-google-matcher/pkg/google"
	"github.com/chainguard-dev/kolide-google-matcher/pkg/kolide"
	"github.com/chainguard-dev/kolide-google-matcher/pkg/mismatch"
)

var (
	endpointsCSVFlag   = flag.String("endpoints-csv", "", "Path to Google Endpoints CSV file")
	maxNoLoginDuration = flag.Duration("max-nologin-duration", 20*24*time.Hour, "maximum amount of time someone can go without logging in")
	maxCheckinOffset   = flag.Duration("max-checkin-offset", 72*time.Hour, "maximum amount of time a checkin is expected after logging in")

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

	problems := mismatch.Analyze(ks, gs, *maxNoLoginDuration, *maxCheckinOffset)
	if len(problems) > 0 {
		fmt.Println("")
		fmt.Printf("%d accounts have problems:\n", len(problems))
		fmt.Println("=========================================================================")
	}

	count := 0
	emails := []string{}
	for k := range problems {
		emails = append(emails, k)
	}
	sort.Strings(emails)

	for _, email := range emails {
		count++
		fmt.Printf("[%d] %s\n", count, email)
		fmt.Printf("    PROBLEM: %s\n\n", problems[email])
	}

	// If SLACK_WEBHOOK_URL set in environment, send a copy of the output to Slack
	if slackWebhookURL != "" {
		log.Println("---\nAttempting to send output to provided Slack webhook...")
		lines := []string{}
		for k, v := range problems {
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
