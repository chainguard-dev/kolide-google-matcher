package mismatch

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/chainguard-dev/kolide-google-matcher/pkg/google"
	"github.com/chainguard-dev/kolide-google-matcher/pkg/kolide"
)

var (
	maxAge          = 5 * 24 * time.Hour
	maxCheckinDelta = 14 * 24 * time.Hour
	// Chrome lies about the OS version in the user agent string
	chromeUserAgentmacOS = "10.15.7"
)

// isMismatchAcceptable determines if a mismatch is acceptable or not
func isMismatchAcceptable(gs []google.Device, ks []kolide.Device) bool {
	if len(ks) > 1 {
		return false
	}

	if len(gs) > 2 {
		return false
	}

	if len(ks) == 0 {
		return false
	}

	k := ks[0]

	// We only apply special logic for macOS, as it's where the confusion lies.
	if k.Platform != "darwin" {
		return false
	}

	shortest := gs[0].DeviceName
	for _, g := range gs {
		if len(g.DeviceName) < len(shortest) {
			shortest = g.DeviceName
		}
	}

	// Check hostname likelihood
	acceptable := true
	for _, g := range gs {
		// To be acceptable, all devices must share the same prefix
		if !strings.HasPrefix(g.DeviceName, shortest) {
			acceptable = false
		}

		startDelta := g.FirstSyncTime.Sub(k.EnrolledAt).Abs()
		if startDelta > maxCheckinDelta {
			acceptable = false
			log.Printf("%s exceeded start checkin delta check: %s", g.DeviceName, startDelta)
			continue
		}

		endDelta := g.LastSyncTime.Sub(k.LastSeenAt).Abs()
		if endDelta > maxCheckinDelta {
			acceptable = false
			log.Printf("%s exceeded end checkin delta check: %s", g.DeviceName, endDelta)
			continue
		}

		// It's just Chrome
		if strings.Contains(g.OS, chromeUserAgentmacOS) && g.HostName == "" {
			continue
		}

		if !strings.Contains(g.OS, k.OperatingSystemDetails.Version) {
			log.Printf("failed OS version match check: Google is %s, Kolide is %s", g.OS, k.OperatingSystemDetails.Version)
			acceptable = false
			continue
		}

		if g.HostName == "" {
			log.Printf("%s has no hostname, skipping hostname cross-reference", g.DeviceName)
			continue
		}

		if !similarHostname(g.HostName, k.Name) {
			log.Printf("failed hostname check: Google is %s, Kolide is %s", g.HostName, k.Name)
			acceptable = false
			continue
		}
	}

	return acceptable
}

func shortModelName(m string) string {
	m = strings.ReplaceAll(m, "macbook-pro", "MBP")
	m = strings.ReplaceAll(m, "macbook-air", "MBA")
	m = strings.ReplaceAll(m, "mac-pro", "MP")
	return m
}

func similarHostname(a string, b string) bool {
	a, _, _ = strings.Cut(a, ".")
	b, _, _ = strings.Cut(b, ".")
	a = shortModelName(strings.ToLower(a))
	b = shortModelName(strings.ToLower(b))
	return strings.EqualFold(a, b)
}

// Analyze finds mismatches between the devices registered within Kolide and those registered within Google
func Analyze(ks []kolide.Device, gs []google.Device, maxNoLogin time.Duration, maxCheckinOffset time.Duration) map[string]string {
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

	log.Printf("Google: found %d devices that have logged in within %s", inScope, maxAge)
	issues := map[string]string{}

	for email, gOS := range gDevices {
		kOS, ok := kDevices[email]

		gDevs := []string{}
		newestLogin := time.Time{}

		for _, gds := range gOS {
			for _, gd := range gds {
				gDevs = append(gDevs, gd.String())
				if gd.LastSyncTime.After(newestLogin) {
					newestLogin = gd.LastSyncTime
				}
			}
		}

		if !ok {
			text := fmt.Sprintf("Google sees %d devices, Kolide sees 0!\nGoogle:\n  %s\n\n",
				len(gDevs), strings.Join(gDevs, "\n  "))
			issues[email] = text
			continue
		}

		mismatches := []string{}
		allKDevs := []string{}
		newestCheckin := time.Time{}
		for _, os := range []string{"Linux", "macOS", "Windows"} {

			kDevs := []string{}
			for _, kd := range kOS[os] {
				if kd.LastSeenAt.After(newestCheckin) {
					newestCheckin = kd.LastSeenAt
				}
				kDevs = append(kDevs, kd.String())
				allKDevs = append(allKDevs, kd.String())
			}

			gDevs = []string{}
			for _, gd := range gOS[os] {
				gDevs = append(gDevs, gd.String())
			}
			if len(gOS[os]) > len(kOS[os]) {
				acceptable := isMismatchAcceptable(gOS[os], kOS[os])
				if acceptable {
					continue
				}
				text := fmt.Sprintf("Google sees %d %s devices, Kolide sees %d\nGoogle:\n  %s\nKolide:\n  %s\n\n",
					len(gOS[os]), os, len(kOS[os]), strings.Join(gDevs, "\n  "), strings.Join(kDevs, "\n  "))
				mismatches = append(mismatches, text)
				issues[email] = strings.Join(mismatches, "\n")
			}
		}

		if len(allKDevs) > 0 && time.Since(newestLogin) > maxNoLogin {
			issues[email] = fmt.Sprintf("%d Kolide device(s), but has not logged into Google since %s", len(allKDevs), newestLogin)
		}

		offset := newestLogin.Sub(newestCheckin)
		if offset > maxCheckinOffset {
			issues[email] = fmt.Sprintf("Latest Kolide check-in was %s - %s before their last Google login", newestCheckin, offset)
		}
	}

	return issues
}
