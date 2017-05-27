package main

import (
	"fmt"
	"log"
	"sort"
	"time"
)

const (
	entryTimeFmt = "2006-01-02"
)

type EntryManager struct {
	numTypes      int // number of entry types.
	hashedEntries map[int64]*DatedEntry
	sortedEntries []*DatedEntry
}

type DatedEntry struct {
	Date   time.Time
	totals []float64
	counts []int
}

type byDate []*DatedEntry

func (a byDate) Len() int           { return len(a) }
func (a byDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byDate) Less(i, j int) bool { return a[i].Date.Before(a[j].Date) }

func newDatedEntry(date time.Time, numTypes int) *DatedEntry {
	return &DatedEntry{
		Date:   date,
		totals: make([]float64, numTypes),
		counts: make([]int, numTypes),
	}
}

func (de *DatedEntry) String() string {
	return fmt.Sprintf("%s: %v", de.Date.Format(entryTimeFmt), de.totals)
}

func (de *DatedEntry) merge(other *DatedEntry) {
	for i := 0; i < len(de.totals); i++ {
		de.totals[i] += other.totals[i]
		de.counts[i] += de.counts[i]
	}
}

func (de *DatedEntry) addOne(index int, value float64) {
	de.totals[index] += value
	de.counts[index]++
}

func NewEntryManager(numTypes int) *EntryManager {
	return &EntryManager{
		numTypes:      numTypes,
		hashedEntries: make(map[int64]*DatedEntry),
	}
}

func (em *EntryManager) String() string {
	var str string
	for _, entry := range em.sortedEntries {
		str += entry.String() + "\n"
	}
	return str
}

func (em *EntryManager) AddSample(date time.Time, index int, value float64) {
	if index < 0 || index >= em.numTypes {
		log.Fatalf("index out of range: %d not in [%d, %d]", index, 0, em.numTypes)
	}

	date = date.Round(time.Second)
	timestamp := date.Unix()

	var de *DatedEntry
	var ok bool

	if de, ok = em.hashedEntries[timestamp]; !ok {
		de = newDatedEntry(date, em.numTypes)
		em.hashedEntries[timestamp] = de
	}
	de.addOne(index, value)
}

// Order builds sortedEntries from hashedEntries and sorts it.
// As a final step, hashEntries is deleted.
func (em *EntryManager) Order() {
	em.sortedEntries = make([]*DatedEntry, len(em.hashedEntries))
	count := 0
	for _, entry := range em.hashedEntries {
		em.sortedEntries[count] = entry
		count++
	}
	sort.Sort(byDate(em.sortedEntries))
	em.hashedEntries = nil
}

func (em *EntryManager) ByInterval(interval time.Duration) []*DatedEntry {
	var estSize int64
	numEntries := len(em.sortedEntries)
	if numEntries <= 1 {
		estSize = 1
	} else {
		timeInterval := em.sortedEntries[numEntries-1].Date.Sub(em.sortedEntries[0].Date)
		estSize = timeInterval.Nanoseconds() / interval.Nanoseconds()
	}

	ret := make([]*DatedEntry, 0, estSize)

	var last *DatedEntry
	for _, e := range em.sortedEntries {
		newDate := e.Date.Round(interval)

		if last == nil || last.Date != newDate {
			if last != nil {
				ret = append(ret, last)
			}
			last = newDatedEntry(newDate, em.numTypes)
		}
		last.merge(e)
	}
	if last != nil {
		ret = append(ret, last)
	}

	return ret
}
