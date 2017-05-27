package main

import (
	"html/template"
	"io"
	"log"
	"os"
	"time"

	chart "github.com/wcharczuk/go-chart"
)

type RenderSpec struct {
	Title         string
	TimeFormat    string
	Data          []*DatedEntry
	Titles        []string
	DataFormatter func(*DatedEntry) []float64
	NumToDisplay  int
}

func (rs RenderSpec) FormatData(de *DatedEntry) []float64 {
	return rs.DataFormatter(de)
}

const (
	templateText = `
	  <center>
	  {{- $spec := .}}
	  <h3>{{.Title}}</h3> 
	  <table border=1 style="border-collapse:collapse">
			<tr>
			  <td>Date</td>
				{{- range $spec.Titles}}
				  <td>{{.}}</td>
				{{- end}}
			</tr>
			{{- range $_, $d := $spec.Data}}
				<tr>
					<td><b>{{$d.Date.Format $spec.TimeFormat}}</b></td>
					{{- $filteredData := ($spec.FormatData $d)}}
					{{- range $filteredData}}
					  <td>{{.}}</td>
					{{- end}}
				</tr>
			{{- end}}
		</table>
		</center>
		<br>`
)

func RenderTemplate(w io.Writer, spec RenderSpec) {
	num := spec.NumToDisplay
	length := len(spec.Data)
	if length < num {
		num = length
	}
	newSpec := spec
	newSpec.Data = make([]*DatedEntry, num)
	for i := 0; i < num; i++ {
		newSpec.Data[i] = spec.Data[length-1-i]
	}

	t, err := template.New("view").Parse(templateText)
	if err != nil {
		log.Fatalf("error parsing template: %v", err)
	}
	if err := t.Execute(w, newSpec); err != nil {
		log.Fatalf("could not execute template: %v", err)
	}
}

func RenderChart(w io.Writer, spec RenderSpec) {
	num := spec.NumToDisplay
	length := len(spec.Data)
	if length < num {
		num = length
	}
	newSpec := spec
	newSpec.Data = spec.Data[length-1-num:]

	titles := newSpec.Titles
	numTypes := len(titles)
	numPoints := len(newSpec.Data)
	dates := make([]time.Time, numPoints)
	dataSet := make([][]float64, numTypes)
	for i := 0; i < numTypes; i++ {
		dataSet[i] = make([]float64, numPoints)
	}
	for i, s := range newSpec.Data {
		dates[i] = s.Date
		formattedData := newSpec.FormatData(s)
		for j := 0; j < numTypes; j++ {
			dataSet[j][i] = formattedData[j]
		}
	}

	graph := chart.Chart{
		XAxis:      chart.XAxis{Style: chart.Style{Show: true}},
		YAxis:      chart.YAxis{Style: chart.Style{Show: true}},
		Background: chart.Style{Padding: chart.Box{Top: 20, Left: 20}},
		Series:     make([]chart.Series, numTypes),
	}
	for i := 0; i < numTypes; i++ {
		graph.Series[i] = chart.TimeSeries{
			Name:    titles[i],
			XValues: dates,
			YValues: dataSet[i],
		}
	}

	// note we have to do this as a separate step because we need a reference to graph
	graph.Elements = []chart.Renderable{
		chart.Legend(&graph),
	}

	if err := graph.Render(chart.PNG, w); err != nil {
		log.Fatalf("failed to render: %v", err)
	}
}

func RenderChartToFile(filename string, spec RenderSpec) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatalf("could not create summary files: %v", err)
	}
	RenderChart(f, spec)

	if err := f.Close(); err != nil {
		log.Fatalf("failed to close %s: %v", filename, err)
	}
}
