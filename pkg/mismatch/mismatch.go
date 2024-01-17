package mismatch

import (
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/chainguard-dev/kolide-google-matcher/pkg/google"
	"github.com/chainguard-dev/kolide-google-matcher/pkg/kolide"
	"github.com/dustin/go-humanize"
)

var (
	maxAge          = 6 * 24 * time.Hour
	inactiveUserAge = 21 * 24 * time.Hour
	maxCheckinDelta = 14 * 24 * time.Hour
	// Chrome lies about the OS version in the user agent string
	chromeUserAgentmacOS = "10.15.7"
	timeFormat           = "Jan 2 2006"

	versionRE = regexp.MustCompile(`\d[\.\d]+`)
)

func mobileDeviceName(s string) bool {
	return strings.Contains(s, "iPhone")
}

// isMismatchAcceptable determines if a mismatch is acceptable or not
func isMismatchAcceptable(gs []google.Device, ks []kolide.Device) bool {
	if len(ks) > 1 {
		log.Printf("multiple kolide devices: %v", ks)
		return false
	}

	//	if len(gs) > 2 {
	//		return false
	//	}

	if len(ks) == 0 {
		return false
	}

	k := ks[0]

	// We only apply special logic for macOS, as it's where the confusion lies.
	if k.Platform != "darwin" {
		log.Printf("non-darwin: %+v", k)
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
			log.Printf("%s does not contain %s", g.DeviceName, shortest)
			acceptable = false
		}

		// It's just Chrome
		if strings.Contains(g.OS, chromeUserAgentmacOS) {
			if g.HostName == "" {
				continue
			}
		}

		if mobileDeviceName(g.HostName) {
			log.Printf("mobile device that Google thinks is %s: %s", g.OS, g.HostName)
			continue
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
			log.Printf("%s exceeded end checkin delta check: %s -- %+v", g.DeviceName, endDelta, g)
			continue
		}

		// Fuzzy matching, as Kolide may show fuller versions, such as "13.4.1 (c)"
		gOS := versionRE.FindString(g.OS)
		kOS := versionRE.FindString(k.OperatingSystemDetails.Version)

		if gOS != kOS {
			log.Printf("failed OS version match check: Google has %q, Kolide has %q", g.OS, k.OperatingSystemDetails.Version)
			acceptable = false
			continue
		}

		if g.HostName == "" {
			// log.Printf("%s has no hostname registered in Google, skipping hostname cross-reference", g.DeviceName)
			continue
		}

		if !similarHostname(g.HostName, k.Name) {
			log.Printf("dissimilar hostname: Google is %s, Kolide is %s -- %+v", g.HostName, k.Name, g)
			acceptable = false
			continue
		}
	}

	if !acceptable {
		log.Printf("found probable mismatch:")
		log.Printf("google: %+v", gs)
		log.Printf("kolide: %+v", ks)
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
	lastCheckin := map[string]time.Time{}

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

		if k.LastSeenAt.After(lastCheckin[email]) {
			lastCheckin[email] = k.LastSeenAt
		}

		kDevices[email][os] = append(kDevices[email][os], k)
	}

	log.Printf("Kolide: found %d devices", len(ks))

	gDevices := map[string]map[string][]google.Device{}
	inScope := 0
	lastLogin := map[string]time.Time{}

	for _, g := range gs {
		// Empty record
		if g.Name == "" {
			continue
		}

		//	log.Printf("g: %+v", g)
		if lastLogin[g.Email].Before(g.LastSyncTime) {
			lastLogin[g.Email] = g.LastSyncTime
		}

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
			// log.Printf("Ignoring type %s (%s)", g.Type, g.OS)
			inScope--
		}
		gDevices[email][os] = append(gDevices[email][os], g)
	}

	log.Printf("Google: found %d devices that have logged in within %s", inScope, maxAge)

	issues := map[string]string{}

	for email, t := range lastLogin {
		if time.Since(t) > maxNoLogin {
			devices := len(kDevices[email])
			if devices > 0 {
				lc := lastCheckin[email]
				text := fmt.Sprintf("%s is an inactive account with %d Kolide devices - no logins since %s (%s). Last Kolide checkin was %s (%s)", email, devices, t.Format(timeFormat), humanize.Time(t), lc.Format(timeFormat), humanize.Time(lc))
				issues[email] = text
			}
		}
	}

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
			text := fmt.Sprintf("%d device in Google and none in Kolide\n    %s",
				len(gDevs), strings.Join(gDevs, "\n    "))
			issues[email] = text
			continue
		}

		mismatches := []string{}

		// Include all operating systems here
		allKDevs := []string{}
		for _, kds := range kOS {
			for _, kd := range kds {
				allKDevs = append(allKDevs, kd.String())
			}
		}

		allGDevs := []string{}
		for _, gds := range gOS {
			for _, gd := range gds {
				allGDevs = append(allGDevs, gd.String())
			}
		}
		newestCheckin := time.Time{}
		for _, os := range []string{"Linux", "macOS", "Windows"} {

			kDevs := []string{}
			for _, kd := range kOS[os] {
				if kd.LastSeenAt.After(newestCheckin) {
					newestCheckin = kd.LastSeenAt
				}
				kDevs = append(kDevs, kd.String())
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
				text := fmt.Sprintf("%d %s device(s) in Google, %d in Kolide\n    Google:\n      %s\n    Kolide:\n      %s",
					len(gOS[os]), os, len(kOS[os]), strings.Join(allGDevs, "\n      "), strings.Join(allKDevs, "\n      "))
				mismatches = append(mismatches, text)
				issues[email] = strings.Join(mismatches, "\n")
			}
		}

		offset := newestLogin.Sub(newestCheckin)
		if offset > maxCheckinOffset {
			issues[email] = fmt.Sprintf("Kolide agent is broken or uninstalled!\n    Latest Kolide check-in: %s (%s)\n    Latest Google login:    %s (%s)",
				newestCheckin.Format(timeFormat), humanize.Time(newestCheckin),
				newestLogin.Format(timeFormat), humanize.Time(newestLogin))
		}
	}

	return issues
}
