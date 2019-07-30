package main

import (
  "fmt"
  "net/http"
  "html/template"
)

// A query structure for the queries to be passed through
type query struct {
  Text string
}

// A simple function that checks for error given an error object and a response object
func checkErr(err error, w http.ResponseWriter)  {
  if err != nil{
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}



func main() {
  // Parsing all the templates we have
  temps :=template.Must(template.ParseFiles("templates/index.html"))

  // Handeling our route
  http.HandleFunc("/",func (w http.ResponseWriter, r *http.Request) {
    // Getting the query
    q := query{Text: "chillis"}
    text := r.FormValue("text")

    // Checking if the query is not empty to replace the default text
    if text != "" {
        q.Text = text
    }

    // Executing or renderin the template providing the query recieved
    err := temps.ExecuteTemplate(w, "index.html", q)
    checkErr(err, w)
  })

  //  Listining to the port
  fmt.Println(http.ListenAndServe(":8080", nil))
}
