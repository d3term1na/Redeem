package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"time"
)

var tpl *template.Template
var records [][]string
var redemption_data []string

func main() {

	file, _ := os.Open("redeem_sample.csv")
	defer file.Close()
	records, _ = csv.NewReader(file).ReadAll()
	records = records[1:]

	file, _ = os.Open("redemption_data.csv")
	defer file.Close()
	// redemption_data, _ = csv.NewReader(file).ReadAll()
	// redemption_data = redemption_data[1:]
	reader := csv.NewReader(file)
	reader.Read()
	for {
		row, err := reader.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			log.Fatalf("Error while reading CSV file: %s", err)
		}

		// Process the data (e.g., print the fields)
		redemption_data = append(redemption_data, row[0])
		// Access individual fields by index
		// fmt.Printf("Field 1: %s, Field 2: %s\n", record[0], record[1])
	}

	tpl = template.Must(template.ParseGlob("templates/*.html"))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/getId", getIdHandler)
	http.ListenAndServe(":8080", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "index.html", nil)
}
func getIdHandler(w http.ResponseWriter, r *http.Request) {
	queryId := r.FormValue("id")
	team_name := ""
	for _, record := range records {
		if queryId == record[0] {
			team_name = record[1]
			break
		}
	}
	var result bool
	if slices.Contains(redemption_data, team_name) {
		result = false
	} else {
		redemption_data = append(redemption_data, team_name)
		t := time.Now()
		millis_val := t.UnixMilli()
		millis_str := strconv.Itoa(int(millis_val))
		if err := addToCSV("redemption_data.csv", []string{team_name, millis_str}); err != nil {
			log.Fatalf("Error appending to CSV: %v", err)
			fmt.Println("hi")
		}
		result = true
	}
	tpl.ExecuteTemplate(w, "result.html", result)
}

func addToCSV(fileName string, record []string) error {
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("Failed to open file: %w", err)
	}

	defer file.Close()

	writer := csv.NewWriter(file)
	err = writer.Write(record)
	if err != nil {
		return fmt.Errorf("Failed to write record: %w", err)
	}

	writer.Flush()

	if err = writer.Error(); err != nil {
		return fmt.Errorf("Failed to flush data: %w", err)
	}

	return nil
}
