package main

import (
  "fmt"
  "net/http"
  "html/template"

  "database/sql"
  _ "github.com/mattn/go-sqlite3"
  "encoding/json"
  "encoding/xml"
  "net/url"
  "io/ioutil"
)

type query struct {
  Text string
  DBStatus bool
}

// A structure that handles the results of the search
type searchResult struct {
  Title string  `xml:"title,attr"`
  Author string `xml:"author,attr"`
  Year string   `xml:"hyr,attr"`
  ID string     `xml:"wi,attr"`
}

// A slice of the response content
type SearchResponse struct {
  Results []searchResult `xml:"works>work"`
}

// A simple function that checks for error given an error object and a response object
func checkErr(err error, w http.ResponseWriter)  {
  if err != nil{
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}


// A function that communicates with the classify2 api with the user's query
func search(query string) ([]searchResult, error){
  var resp *http.Response
  var err error
  // Checking the response
  if resp, err = http.Get("http://classify.oclc.org/classify2/Classify?&summary=true&title="+url.QueryEscape(query)); err != nil{
     return []searchResult{}, err
  }

  defer resp.Body.Close()
  var body []byte

  // Reading the content of the response
  if body, err = ioutil.ReadAll(resp.Body); err != nil{
    return []searchResult{}, err
  }

  // Parsing the xml response
  var content SearchResponse
  err = xml.Unmarshal(body, &content)

  return content.Results, err
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
  http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
    var results []searchResult
    var err error

    results, err = search(r.FormValue("search"))
    checkErr(err, w)
    encoder := json.NewEncoder(w)
    // Converting the results object into json
    err = encoder.Encode(results)
    checkErr(err, w)
  })

  //  Listining to the port
  fmt.Println(http.ListenAndServe(":8080", nil))
}
