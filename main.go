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

	qrcode "github.com/skip2/go-qrcode"
	qrdecode "github.com/tuotoo/qrcode"
)

var tpl *template.Template
var records [][]string
var redemption_data [][]string
var png []byte
var err error
var staffID string

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
	http.HandleFunc("/decodeQR", decodeQRHandler)
	http.HandleFunc("/result", resultHandler)
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
	staffID = r.FormValue("id")
	http.Redirect(
		w,
		r,
		"/result",
		http.StatusSeeOther,
	)
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

func decodeQRHandler(w http.ResponseWriter, r *http.Request) {

	file, _, err := r.FormFile("dropQR")
	if err != nil {
		http.Error(w, "Upload failed", 400)
		return
	}
	defer file.Close()

	tmp, _ := os.CreateTemp("", "qr*.png")
	defer os.Remove(tmp.Name())

	io.Copy(tmp, file)

	qrFile, _ := os.Open(tmp.Name())

	result, err := qrdecode.Decode(qrFile)
	if err != nil {
		http.Error(w, "Decode failed", 400)
		return
	}

	// w.Write([]byte("Decoded QR: " + result.Content))
	staffID = result.Content
	http.Redirect(
		w,
		r,
		"/result",
		http.StatusSeeOther,
	)
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	team_name := ""
	for _, record := range records {
		if staffID == record[0] {
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
