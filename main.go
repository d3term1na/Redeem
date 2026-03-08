package main

import (
	"database/sql"
	"encoding/csv"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"

	qrcode "github.com/skip2/go-qrcode"
	qrdecode "github.com/tuotoo/qrcode"
)

var tpl *template.Template
var records [][]string
var png []byte
var err error
var staffID string
var DB *sql.DB

func initDB() {
	connStr := "host=localhost user=postgres password=DanielChan12! dbname=redeemdb sslmode=disable"

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	DB = db

	_, err4 := DB.Exec("TRUNCATE TABLE staff, teams CASCADE")
	if err4 != nil {
		panic(err4)
	}

	file, _ := os.Open("redeem_sample.csv")
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
		var exists bool
		team_name := row[1]
		err = DB.QueryRow(
			"SELECT EXISTS(SELECT 1 FROM teams WHERE team_name=$1)",
			team_name,
		).Scan(&exists)
		if err != nil {
			panic(err)
		}
		if !exists {
			_, err1 := DB.Exec(
				"INSERT INTO teams (team_name, redeemed, redeemed_at) VALUES ($1,$2,$3)",
				team_name,
				false,
				nil,
			)
			if err1 != nil {
				panic(err1)
			}
		}
		_, err2 := DB.Exec(
			"INSERT INTO staff (staff_id, team_name, created_at) VALUES ($1,$2,$3)",
			row[0],
			team_name,
			row[2],
		)
		if err2 != nil {
			panic(err2)
		}
	}
}

func main() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	initDB()

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
	tpl.ExecuteTemplate(w, "index.html", nil)
}

func staffHandler(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "staff.html", nil)
}

func adminHandler(w http.ResponseWriter, r *http.Request) {
	type Team struct {
		Name       string
		Redeemed   bool
		RedeemedAt *time.Time
	}

	rows, err := DB.Query("SELECT team_name, redeemed, redeemed_at FROM teams")
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	var redemption_data []Team

	for rows.Next() {
		var t Team
		var redeemed_at_millis *int64
		err := rows.Scan(&t.Name, &t.Redeemed, &redeemed_at_millis)
		if err != nil {
			panic(err)
		}
		if redeemed_at_millis != nil {
			ts := time.UnixMilli(*redeemed_at_millis)
			t.RedeemedAt = &ts
		} else {
			t.RedeemedAt = nil
		}
		redemption_data = append(redemption_data, t)
	}

	tpl.ExecuteTemplate(w, "admin.html", redemption_data)
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

	staffID = result.Content
	http.Redirect(
		w,
		r,
		"/result",
		http.StatusSeeOther,
	)
}

func resultHandler(w http.ResponseWriter, r *http.Request) {

	type Result struct {
		Valid    bool
		Redeemed bool
		Team     string
	}

	var team_name string
	var valid bool
	var redeemed bool
	var res Result

	err = DB.QueryRow(
		"SELECT team_name FROM staff WHERE staff_id = $1",
		staffID,
	).Scan(&team_name)

	if team_name == "" {
		valid = false
		redeemed = true
	} else {
		valid = true
		err = DB.QueryRow(
			"SELECT redeemed FROM teams WHERE team_name = $1",
			team_name,
		).Scan(&redeemed)
		if !redeemed {
			_, err1 := DB.Exec(
				"UPDATE teams SET redeemed = $1, redeemed_at = $2 WHERE team_name = $3",
				true,
				time.Now().UnixMilli(),
				team_name,
			)
			if err1 != nil {
				panic(err1)
			}
		}
	}
	res.Valid = valid
	res.Redeemed = redeemed
	res.Team = team_name
	tpl.ExecuteTemplate(w, "result.html", res)
}
