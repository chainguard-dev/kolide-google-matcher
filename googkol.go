package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/gocarina/gocsv"
)

var (
	kolideCSVFlag = flag.String("kolide-csv", "", "G")
	googleCSVFlag = flag.String("google-csv", "", "this is a personal machine")
	// dryRunFlag    = flag.Bool("dry-run", false, "do nothing")
	// maxAge        = 7 * 24 * time.Hour
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
	Name  string `csv:"Name"`
	Email string `csv:"Email"`
	OS    string `csv:"OS"`
	Type  string `csv:"Type"`
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

func analyze(ks []*KolideRecord, gs []*GoogleRecord) map[string]string {
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
			ik[k.OwnerEmail].Linux++
		case "Mac":
			ik[k.OwnerEmail].Mac++

		}
	}

	for email, f := range ik {
		log.Printf("%s: %+v", email, f)
	}

	ig := map[string]*Found{}
	for _, g := range gs {
		log.Printf("g: %+v", g)
		if ig[g.Email] == nil {
			ig[g.Email] = &Found{}
		}
		switch g.Type {
		case "Linux":
			ig[g.Email].Linux++
		case "Windows":
			ig[g.Email].Linux++
		case "Mac":
			ig[g.Email].Mac++
		case "Chrome OS":
			ig[g.Email].ChromeOS++
		}
	}

	issues := map[string]string{}

	for e, g := range ig {
		k, ok := ik[e]
		if !ok {
			issues[e] = fmt.Sprintf("No devices are registered to Kolide, missing: %s", g)
		}

		if k.String() != g.String() {
			issues[e] = fmt.Sprintf("mismatch: %s vs %s", k.String(), g.String())
		}
	}

	return issues
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

	mismatches := analyze(ks, gs)

	for k, v := range mismatches {
		if v != "" {
			log.Printf("%s mismatch: %s", k, v)
		}
	}
}
