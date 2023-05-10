package main

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type day struct {
	Day   time.Time
	Url   string
	Stage string
	Bands []band
}

type band struct {
	Name  string
	Start time.Time
	End   time.Time
}

func (d *day) getBands(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "h4" {
		for _, a := range n.Attr {
			if a.Key == "class" && a.Val == "act-schedule__title" {
				d.Stage = n.FirstChild.Data
			}
		}
	}

	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "class" && a.Val == "act-schedule__acts-act" {
				name := n.FirstChild.NextSibling.FirstChild.Data
				start := n.FirstChild.NextSibling.NextSibling.NextSibling.FirstChild.Data
				end := n.FirstChild.NextSibling.NextSibling.NextSibling.NextSibling.NextSibling.FirstChild.Data
				d.addBand(name, start, end)
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		d.getBands(c)
	}
}

func (d *day) toTime(str string) time.Time {
	t := strings.Split(str, ".")
	h, _ := strconv.ParseInt(t[0], 0, 0)
	m, _ := strconv.ParseInt(t[1], 0, 0)

	result := time.Date(d.Day.Year(), d.Day.Month(), d.Day.Day(), int(h), int(m), 0, 0, time.UTC)

	if result.Before(d.Day) {
		result = result.AddDate(0, 0, 1)
	}

	return result
}

func (d *day) addBand(name string, start string, end string) {
	d.Bands = append(d.Bands, band{name, d.toTime(start), d.toTime(end)})
}

func (d *day) retrieveSchedule() {
	resp, err := http.Get(d.Url)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	doc, err := html.Parse(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	d.getBands(doc)
}

func main() {
	bands := make([]band, 0)
	days := []day{
		{time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC), "https://www.graspop.be/nl/line-up/donderdag/schedule", "", bands},
		{time.Date(2023, 6, 16, 12, 0, 0, 0, time.UTC), "https://www.graspop.be/nl/line-up/vrijdag/schedule", "", bands},
		{time.Date(2023, 6, 17, 12, 0, 0, 0, time.UTC), "https://www.graspop.be/nl/line-up/zaterdag/schedule", "", bands},
		{time.Date(2023, 6, 18, 12, 0, 0, 0, time.UTC), "https://www.graspop.be/nl/line-up/zondag/schedule", "", bands},
	}

	t, err := template.ParseFiles("schedule_tmpl.html")
	if err != nil {
		log.Fatal(err)
	}

	for _, d := range days {
		d.retrieveSchedule()
		if err = t.Execute(os.Stdout, d); err != nil {
			log.Fatal(err)
		}
	}
}
