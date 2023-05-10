package main

import (
	"log"
	"strconv"
	"strings"
	"testing"
	_ "testing"
	"time"
)

func TestBand(te *testing.T) {
	d := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC)

	t := strings.Split("00.20", ".")
	h, _ := strconv.ParseInt(t[0], 0, 0)
	m, _ := strconv.ParseInt(t[1], 0, 0)

	start := time.Date(d.Year(), d.Month(), d.Day(), int(h), int(m), 0, 0, time.UTC)
	if start.Before(d) {
		start = start.AddDate(0, 0, 1)
	}

	log.Println(start.Format(time.RFC3339))
}
