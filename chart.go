package main

import (
	"io"
	"log"
	"os"
	"time"

	chart "github.com/wcharczuk/go-chart"
)

func writeChart(filename string, data []*summary) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("could not create file %s: %v", filename, err)
	}
	defer f.Close()
	drawChart(data, f)
}

func drawChart(data []*summary, writer io.Writer) {
	dates := make([]time.Time, len(data))
	total := make([]float64, len(data))
	linux := make([]float64, len(data))
	darwin := make([]float64, len(data))
	windows := make([]float64, len(data))
	source := make([]float64, len(data))
	for i, s := range data {
		date, err := time.Parse(timeFmt, s.Date)
		if err != nil {
			log.Fatalf("could not parse %s: %v", s.Date, err)
		}
		dates[i] = date
		total[i] = (float64)(s.Total)
		linux[i] = (float64)(s.TypeCounts[linuxDownload])
		darwin[i] = (float64)(s.TypeCounts[darwinDownload])
		windows[i] = (float64)(s.TypeCounts[windowsDownload])
		source[i] = (float64)(s.TypeCounts[sourceDownload])
	}

	graph := chart.Chart{
		XAxis: chart.XAxis{
			Style: chart.Style{Show: true},
		},
		YAxis: chart.YAxis{
			Style: chart.Style{Show: true},
		},
		Background: chart.Style{
			Padding: chart.Box{Top: 20, Left: 20},
		},
		Series: []chart.Series{
			chart.TimeSeries{Name: "Total", XValues: dates, YValues: total},
			chart.TimeSeries{Name: "Linux", XValues: dates, YValues: linux},
			chart.TimeSeries{Name: "Darwin", XValues: dates, YValues: darwin},
			chart.TimeSeries{Name: "Windows", XValues: dates, YValues: windows},
			chart.TimeSeries{Name: "Source", XValues: dates, YValues: source},
		},
	}

	//note we have to do this as a separate step because we need a reference to graph
	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}
	err := graph.Render(chart.PNG, writer)
	if err != nil {
		log.Fatalf("failed to render: %v", err)
	}
}
