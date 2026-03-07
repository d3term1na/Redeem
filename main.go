package main

import (
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/skip2/go-qrcode"
)

var tpl *template.Template
var records [][]string
var redemption_data [][]string
var png []byte
var err error

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	file, _ := os.Open("redeem_sample.csv")
	defer file.Close()
	records, _ = csv.NewReader(file).ReadAll()
	records = records[1:]

	file, _ = os.Open("redemption_data.csv")
	defer file.Close()
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

		timestamp, err := strconv.ParseInt(row[1], 10, 64)
		if err != nil {
			panic(err)
		}
		t := time.UnixMilli(timestamp)
		formatted := t.Local().Format("2006-01-02 15:04:05")
		redemption_data = append(redemption_data, []string{row[0], formatted})
	}
	fmt.Println(redemption_data[0][0])
	tpl = template.Must(template.ParseGlob("templates/*.html"))

	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/staff", staffHandler)
	http.HandleFunc("/admin", adminHandler)
	http.HandleFunc("/getId", getIdHandler)
	http.HandleFunc("/getQR", getQRHandler)
	http.HandleFunc("/downloadQR", downloadQRHandler)
	http.ListenAndServe(":8080", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "index.html", redemption_data)
}

func staffHandler(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "staff.html", nil)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "admin.html", nil)
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
	result := true
	for _, row := range redemption_data {
		if row[0] == team_name {
			result = false
		}
	}
	if result {
		t := time.Now()
		millis_val := t.UnixMilli()
		millis_str := strconv.Itoa(int(millis_val))
		redemption_data = append(redemption_data, []string{team_name, millis_str})
		if err := addToCSV("redemption_data.csv", []string{team_name, millis_str}); err != nil {
			log.Fatalf("Error appending to CSV: %v", err)
		}
	}
	tpl.ExecuteTemplate(w, "result.html", result)
}

func getQRHandler(w http.ResponseWriter, r *http.Request) {
	queryId := r.FormValue("id")
	png, err = qrcode.Encode(queryId, qrcode.Medium, 256)
	if err != nil {
		http.Error(w, "QR generation failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Write(png)
}

func downloadQRHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Content-Disposition", "attachment; filename=staff_qr.png")
	w.Write(png)
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

func generateQR(staffId string) error {
	return qrcode.WriteFile(staffId, qrcode.Medium, 256, "staff_qr.png")
}
