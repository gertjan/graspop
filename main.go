package main

import (
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/net/html"
)

type Day struct {
	Day           time.Time
	Url           string
	Stage         string
	Bands         []Band
	FilteredBands []Band
}

type FilteredStage struct {
	Name  string
	Bands []Band
}

func (d *Day) FilteredStages() []FilteredStage {
	stageMap := make(map[string][]Band)
	stageOrder := make([]string, 0)
	for _, b := range d.FilteredBands {
		if _, ok := stageMap[b.Stage]; !ok {
			stageOrder = append(stageOrder, b.Stage)
		}
		stageMap[b.Stage] = append(stageMap[b.Stage], b)
	}
	result := make([]FilteredStage, 0, len(stageOrder))
	for _, name := range stageOrder {
		result = append(result, FilteredStage{Name: name, Bands: stageMap[name]})
	}
	return result
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

func getAttrVal(node *html.Node, key string) string {
	for _, attr := range node.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func (d *Day) getBands(n *html.Node) {
	if n.Type == html.ElementNode && n.Data == "section" {
		if getAttrVal(n, "class") == "act-schedule__stage" {
			d.Stage = getAttrVal(n, "id")
		}
	}

	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "class" && a.Val == "act-schedule__acts-act" {
				r := n.FirstChild.NextSibling.FirstChild.NextSibling
				name := r.FirstChild.Data
				start := r.NextSibling.NextSibling.FirstChild.Data
				end := r.NextSibling.NextSibling.NextSibling.NextSibling.FirstChild.Data

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
	b := Band{
		Name:  name,
		Stage: d.Stage,
		Start: start,
		End:   end,
	}
	if d.Stage == "Classic Rock Café" {
		d.FilteredBands = append(d.FilteredBands, b)
		return
	}
	d.Bands = append(d.Bands, b)
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

func (s Schedule) GetTimeForDay(d Day) []string {
	var last time.Time
	for _, b := range d.Bands {
		if b.End.After(last) {
			last = b.End
		}
	}
	if last.IsZero() {
		last = d.Day.Add(16*time.Hour + 45*time.Minute)
	}
	times := make([]string, 0)
	for t := d.Day; t.Before(last); t = t.Add(5 * time.Minute) {
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

func (d *Day) FullTitle() string {
	return d.Day.Format("Monday 2 January 2006")
}

type Schedule struct {
	Days     []*Day
	Footnote string
	QRCode   template.URL
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

	days := []*Day{
		{Day: time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC), Url: "https://www.graspop.be/nl/line-up/donderdag/schedule"},
		{Day: time.Date(2026, 6, 19, 12, 0, 0, 0, time.UTC), Url: "https://www.graspop.be/nl/line-up/vrijdag/schedule"},
		{Day: time.Date(2026, 6, 20, 12, 0, 0, 0, time.UTC), Url: "https://www.graspop.be/nl/line-up/zaterdag/schedule"},
		{Day: time.Date(2026, 6, 21, 12, 0, 0, 0, time.UTC), Url: "https://www.graspop.be/nl/line-up/zondag/schedule"},
	}

	for _, d := range days {
		d.retrieveSchedule()
	}

	qrBytes, err := qrcode.Encode("https://gertjan.github.io/graspop", qrcode.Medium, 128)
	if err != nil {
		log.Fatal(err)
	}
	qrDataURI := template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(qrBytes))

	s := Schedule{
		Days:     days,
		Footnote: footnote,
		QRCode:   qrDataURI,
	}

	for _, d := range s.Days {
		for _, b := range d.Bands {
			log.Println(b)
		}
	}

	execTemplate(s, "schedule_tmpl.html", "index.html")

	log.Println("Done")
}
