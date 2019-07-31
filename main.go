package main

import (
  "fmt"
  "net/http"
  "html/template"

  "database/sql"
  _ "github.com/mattn/go-sqlite3"
  "encoding/json"
)

// A structure that handles the results of the search
type searchResult struct {
  Title string
  Author string
  Year string
  ID string
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

  // Estaplishing a connection to our database
  db, _ := sql.Open("sqlite3", "dev.db")


  // Handeling our route
  http.HandleFunc("/",func (w http.ResponseWriter, r *http.Request) {
    // Getting the query
    q := query{Text: "chillis"}
    text := r.FormValue("text")

    // Checking if the query is not empty to replace the default text
    if text != "" {
        q.Text = text
    }

    // checking the status of our connection
    q.DBStatus = db.Ping() == nil
    // Executing or renderin the template providing the query recieved
    err := temps.ExecuteTemplate(w, "index.html", q)
    checkErr(err, w)
  })

  //  Handeling the route to the /search route
  http.HandleFunc("/search", func (w http.ResponseWriter, r *http.Request) {
    results := []searchResult{
      searchResult{"Oliver-Twist", "Charles dicknes", "1985", "QA19.08"},
      searchResult{"Tales of two Cities", "William shakspear", "1589", "ZA19.08"},
      searchResult{"Macbeth", "William Shakespeer", "1763", "FD17.48"},
    }


    encoder := json.NewEncoder(w)
    // Converting the results object into json
    err := encoder.Encode(results)
    checkErr(err, w)
  })

  //  Listining to the port
  fmt.Println(http.ListenAndServe(":8080", nil))
}