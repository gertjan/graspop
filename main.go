package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/html"
)

type Day struct {
	Day   time.Time
	Url   string
	Stage string
	Bands []Band
}

type Band struct {
	Name  string
	Stage string
	Start time.Time
	End   time.Time
}

func (b Band) StartStr() string {
	return b.Start.Format("time-1504")
}

func (b Band) EndStr() string {
	return b.End.Format("time-1504")
}

func (b Band) IntervalStr() string {
	return fmt.Sprintf("%v - %v", b.Start.Format("15:04"), b.End.Format("15:04"))
}

func (d *Day) getBands(n *html.Node) {
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

				d.addBand(name, d.toTime(start), d.toTime(end))
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		d.getBands(c)
	}
}

func (d *Day) toTime(str string) time.Time {
	t := strings.Split(str, ".")
	h, _ := strconv.ParseInt(t[0], 0, 0)
	m, _ := strconv.ParseInt(t[1], 0, 0)

	result := time.Date(d.Day.Year(), d.Day.Month(), d.Day.Day(), int(h), int(m), 0, 0, time.UTC)

	if result.Before(d.Day) {
		result = result.AddDate(0, 0, 1)
	}

	return result
}

func (d *Day) addBand(name string, start time.Time, end time.Time) {
	// filters
	if d.Stage == "Classic Rock Café" {
		return
	}

	if d.Stage == "Metal Dome" {
		if start.Hour() < 4 {
			return
		}
	}

	if name == "Golden Moment" {
		return
	}

	d.Bands = append(d.Bands, Band{
		Name:  name,
		Stage: d.Stage,
		Start: start,
		End:   end,
	})
}

func (d *Day) retrieveSchedule() {
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

var stages = []string{
	"South Stage", "North Stage", "Marquee", "Jupiler Stage", "Metal Dome",
}

func (d *Day) findBand(stage string, index int) Band {
	i := 0
	for _, b := range d.Bands {
		if b.Stage == stage {
			if i == index {
				return b
			}
			i++
		}
	}

	return Band{}
}

func (d *Day) ToTable() [][]string {
	table := make([][]string, 0)
	table = append(table, stages)

	found := false
	for i := 0; ; i++ {
		row := make([]string, len(stages))
		found = false
		for j := 0; j < len(stages); j++ {
			b := d.findBand(stages[j], i)
			if b.Name != "" {
				found = true
				row[j] = fmt.Sprintf("%s %s", b.IntervalStr(), b.Name)
			}
		}
		if found {
			table = append(table, row)
		} else {
			return table
		}
	}
}

func (s Schedule) GetTitle(d Day) string {
	return strings.ToTitle(strings.TrimSuffix(strings.TrimPrefix(d.Url, "https://www.graspop.be/nl/line-up/"), "/schedule"))
}

func (s Schedule) GetTime() []string {
	last := s.Days[0].Day.Add(16 * time.Hour).Add(45 * time.Minute)

	times := make([]string, 0)
	for t := s.Days[0].Day; t.Before(last); t = t.Add(5 * time.Minute) {
		times = append(times, t.Format("time-1504"))
	}

	return times
}

func (s Schedule) GetDisplayTimes() []string {
	last := s.Days[0].Day.Add(13 * time.Hour).Add(30 * time.Minute)

	times := make([]string, 0)
	for t := s.Days[0].Day; t.Before(last); t = t.Add(30 * time.Minute) {
		times = append(times, t.Format("time-1504"))
	}

	return times
}

type Schedule struct {
	Days     []*Day
	Footnote string
}

func (s Schedule) GetStageIndex(stageName string) string {
	switch stageName {
	case "South Stage":
		return "stage-1"
	case "North Stage":
		return "stage-2"
	case "Marquee":
		return "stage-3"
	case "Jupiler Stage":
		return "stage-4"
	case "Metal Dome":
		return "stage-5"
	default:
		log.Fatal(stageName)
		return ""
	}

}

func execTemplate(s Schedule, tmpl string, outName string) {
	t, err := template.ParseFiles(tmpl)
	if err != nil {
		log.Fatal(err)
	}

	out, _ := os.Create(outName)
	if err = t.Execute(out, s); err != nil {
		log.Fatal(err)
	}
}

func main() {
	footnote := time.Now().Format("Retrieved from https://www.graspop.be - 2006-01-02 15:04")

	bands := make([]Band, 0)
	days := []*Day{
		{time.Date(2024, 6, 20, 12, 0, 0, 0, time.UTC), "https://www.graspop.be/nl/line-up/donderdag/schedule", "", bands},
		{time.Date(2024, 6, 21, 12, 0, 0, 0, time.UTC), "https://www.graspop.be/nl/line-up/vrijdag/schedule", "", bands},
		{time.Date(2024, 6, 22, 12, 0, 0, 0, time.UTC), "https://www.graspop.be/nl/line-up/zaterdag/schedule", "", bands},
		{time.Date(2024, 6, 23, 12, 0, 0, 0, time.UTC), "https://www.graspop.be/nl/line-up/zondag/schedule", "", bands},
	}

	for _, d := range days {
		d.retrieveSchedule()
	}

	s := Schedule{
		Days:     days,
		Footnote: footnote,
	}

	for _, d := range s.Days {
		for _, b := range d.Bands {
			log.Println(b)
		}
	}

	execTemplate(s, "schedule_tmpl.html", "index.html")
	execTemplate(s, "compact_tmpl.html", "compact.html")
}
