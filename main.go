package main

import (
	"fmt"
	"html/template"
	"net/http"
)

var tpl *template.Template

func handler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Redeem server running")
}

func main() {
	tpl, _ = template.ParseGlob("templates/*.html")
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/getId", getIdHandler)
	http.ListenAndServe(":8080", nil)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tpl.ExecuteTemplate(w, "index.html", nil)
}
func getIdHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.FormValue("id"))
}
