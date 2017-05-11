package main

import (
	"fmt"
	"html/template"
	"io"
	"log"
)

const (
	templateText = `
	  <table border=1 style="border-collapse:collapse">
			<tr>
			  <td>Date</td>
			  <td>Total</td>
			  <td>Linux</td>
			  <td>Darwin</td>
			  <td>Windows</td>
			  <td>Source</td>
			</tr>
		  {{range .}}
				<tr>
					<td><b>{{.Date}}</b></td>
					<td><b>{{.Total}}</b></td>
					<td>{{index .TypeCounts 1}}</td>
					<td>{{index .TypeCounts 2}}</td>
					<td>{{index .TypeCounts 3}}</td>
					<td>{{index .TypeCounts 4}}</td>
				</tr>
			{{end}}
		</table>
		<br>`
)

func writeHTML(w io.Writer, title string, data []*summary) {
	t, err := template.New("view").Parse(templateText)
	if err != nil {
		log.Fatalf("error parsing template: %v", err)
	}
	if _, err := fmt.Fprintf(w, "<h3>%s</h3>\n", title); err != nil {
		log.Fatalf("could not write title: %v", err)
	}
	if err := t.Execute(w, data); err != nil {
		log.Fatalf("could not execute template: %v", err)
	}
}
