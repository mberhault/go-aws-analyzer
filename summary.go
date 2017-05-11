package main

import (
	"fmt"
	"log"
	"sort"
	"time"
)

const (
	timeFmt = "2006-01-02"
)

var (
	dayMap = map[string]*summary{}
	daily  byDate
	weekly byDate
)

type summary struct {
	Date       string // in YYYY-MM-DD format.
	TypeCounts [numDownloadTypes]int
	Total      int
}

func (s *summary) increment(s2 *summary) {
	s.Total += s2.Total
	for i := 0; i < numDownloadTypes; i++ {
		s.TypeCounts[i] += s2.TypeCounts[i]
	}
}

func (s *summary) String() string {
	return fmt.Sprintf("%s:%d", s.Date, s.Total)
}

type byDate []*summary

func (a byDate) Len() int           { return len(a) }
func (a byDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byDate) Less(i, j int) bool { return a[i].Date < a[j].Date }

func buildOrdered() {
	daily = make(byDate, len(dayMap), len(dayMap))
	count := 0
	for _, entry := range dayMap {
		daily[count] = entry
		count++
	}
	sort.Sort(daily)

	// Build weekly list.
	var last *summary
	weekly = make(byDate, 0, len(daily)/7)
	for _, s := range daily {
		date, err := time.Parse(timeFmt, s.Date)
		if err != nil {
			log.Fatal(err)
		}

		weekStart := date.AddDate(0, 0, -(int)(date.Weekday()))
		weekStr := weekStart.Format(timeFmt)

		if last == nil || last.Date != weekStr {
			if last != nil {
				weekly = append(weekly, last)
			}
			last = &summary{Date: weekStr}
		}
		last.increment(s)
	}
	if last != nil {
		weekly = append(weekly, last)
	}
}
