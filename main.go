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
type SearchResult struct {
  Title string  `xml:"title,attr"`
  Author string `xml:"author,attr"`
  Year string   `xml:"hyr,attr"`
  ID string     `xml:"owi,attr"`
}


// A structure that holds the book search result (When searching for a book by ID)
type BookResponse struct {
  BookData struct {
    Title string `xml:"title,attr"`
    Author string `xml:"author,attr"`
    ID string `xml:"owi,attr"`
  } `xml:"work"`
  Classification struct {
    MostPopular string `xml:"sfa,attr"`
  } `xml:"recommendations>ddc>mostPopular"`
}


// A slice of the response content
type SearchResponse struct {
  Results []SearchResult `xml:"works>work"`
}


// A simple function that checks for error given an error object and a response object
func checkErr(err error, w http.ResponseWriter)  {
  if err != nil{
    http.Error(w, err.Error(), http.StatusInternalServerError)
  }
}

// A function that communicates with the classify2 api with the user's query
func search(query string) ([]SearchResult, error){
  var resp *http.Response
  var err error
  // Checking the response
  if resp, err = http.Get("http://classify.oclc.org/classify2/Classify?summary=true&title="+url.QueryEscape(query)); err != nil{
     return []SearchResult{}, err
  }

  defer resp.Body.Close()
  var body []byte

  // Reading the content of the response
  if body, err = ioutil.ReadAll(resp.Body); err != nil{
    return []SearchResult{}, err
  }

  // Parsing the xml response
  var content SearchResponse
  err = xml.Unmarshal(body, &content)

  return content.Results, err
}


// Find a book by a given id (when user clicks on book from search resul)
func findBook(id string) (BookResponse, error) {
  var resp *http.Response
  var err error
  // Checking the response
  if resp, err = http.Get("http://classify.oclc.org/classify2/Classify?summary=true&owi="+url.QueryEscape(id)); err != nil{
     return BookResponse{}, err
  }

  defer resp.Body.Close()
  var body []byte

  // Reading the content of the response
  if body, err = ioutil.ReadAll(resp.Body); err != nil{
    return BookResponse{}, err
  }

  // Parsing the xml response
  var content BookResponse
  err = xml.Unmarshal(body, &content)

  // Returning the book's information to be added to the database
  return content, err
}


func main() {
  // Parsing all the templates we have
  temps :=template.Must(template.ParseFiles("templates/index.html"))
  database, _ := sql.Open("sqlite3", "./dev.db")

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
    q.DBStatus = database.Ping() == nil
    // Executing or renderin the template providing the query recieved
    err := temps.ExecuteTemplate(w, "index.html", q)
    checkErr(err, w)
  })

  //  Handeling the route to the /search route
  http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
    var results []SearchResult
    var err error

    results, err = search(r.FormValue("search"))
    checkErr(err, w)
    encoder := json.NewEncoder(w)
    // Converting the results object into json
    err = encoder.Encode(results)
    checkErr(err, w)
  })

  // Handeling adding books to the data base
  http.HandleFunc("/books/add", func (w http.ResponseWriter, r *http.Request) {
    // Get the data of the selected book
    book, err := findBook(r.FormValue("id"))
    checkErr(err, w)

    err = database.Ping()
    checkErr(err, w)

    // Inseting the book to the database
    statement, err := database.Prepare("INSERT INTO books (pk, title, author, id, class) VALUES (?, ?, ?, ?, ?)")
    checkErr(err, w)

    statement.Exec(nil, book.BookData.Title, book.BookData.Author, book.BookData.ID, book.Classification.MostPopular)
    checkErr(err,w)
  })
  //  Listining to the port
  fmt.Println(http.ListenAndServe(":8080", nil))
}
