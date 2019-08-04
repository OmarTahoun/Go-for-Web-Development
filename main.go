package main

import (
  "net/http"
  "database/sql"
  _ "github.com/mattn/go-sqlite3"
  "encoding/json"
  "encoding/xml"
  "net/url"
  "io/ioutil"
  "github.com/codegangsta/negroni"
  "github.com/yosssi/ace"
)

var database *sql.DB

type book struct {
  pk int
  Title string
  Author string
  classification string
}
type page struct {
  Books []book{}
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


// A middle ware to check if the database connection is stable or not.
func verifyDatabase(w http.ResponseWriter, r *http.Request, next http.HandlerFunc)  {
  if err := database.Ping(); err != nil{
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
  }
  next(w,r)
}


// Sending a query to the API
func callAPI(url string) ([]byte, error)  {
  var resp *http.Response
  var err error
  // Checking the response
  if resp, err = http.Get(url); err != nil{
     return []byte{}, err
  }
  defer resp.Body.Close()
  return ioutil.ReadAll(resp.Body)
}


// A function that communicates with the classify2 api with the user's query
func search(query string) ([]SearchResult, error){
  var content SearchResponse
  body, err := callAPI("http://classify.oclc.org/classify2/Classify?summary=true&title=" + url.QueryEscape(query))
  if err != nil{
    return []SearchResult{}, err
  }
  // Parsing the xml response
  err = xml.Unmarshal(body, &content)
  // Returning the books information to be added to the table
  return content.Results, err
}


// Find a book by a given id (when user clicks on book from search resul)
func findBook(id string) (BookResponse, error) {
  var content BookResponse
  body, err := callAPI("http://classify.oclc.org/classify2/Classify?summary=true&owi=" + url.QueryEscape(id))
  if err != nil{
    return content, err
  }
  // Parsing the xml response
  err = xml.Unmarshal(body, &content)
  // Returning the book's information to be added to the database
  return content, err
}



// MAIN FUNCTION
func main() {
  // Estaplishing connection with our database
  database, _ = sql.Open("sqlite3", "./dev.db")
  mux := http.NewServeMux()

  // Handeling the main route
  mux.HandleFunc("/",func (w http.ResponseWriter, r *http.Request) {
    template, err := ace.Load("templates/index", "", nil)
    checkErr(err, w)
    // Getting the query
    p := page{Books []book{}}
    text := r.FormValue("text")
    // Checking if the query is not empty to replace the default text
    if text != "" {
        q.Text = text
    }
    // checking the status of our connection
    q.DBStatus = database.Ping() == nil
    // Executing or renderin the template providing the query recieved
    err = template.Execute(w, q)
    checkErr(err, w)
  })


  //  Handeling the searche route
  mux.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
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
  mux.HandleFunc("/books/add", func (w http.ResponseWriter, r *http.Request) {
    // Get the data of the selected book
    book, err := findBook(r.FormValue("id"))
    checkErr(err, w)

    // Inseting the book to the database
    statement, err := database.Prepare("INSERT INTO books (pk, title, author, id, class) VALUES (?, ?, ?, ?, ?)")
    checkErr(err, w)

    statement.Exec(nil, book.BookData.Title, book.BookData.Author, book.BookData.ID, book.Classification.MostPopular)
    checkErr(err,w)
  })



  n := negroni.Classic()
  n.Use(negroni.HandlerFunc(verifyDatabase))
  n.UseHandler(mux)
  //  Listining to the port
  n.Run(":8080")
}
