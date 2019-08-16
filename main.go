package main

import (
  "log"
  "net/http"
  "database/sql"
  _ "github.com/mattn/go-sqlite3"
  "encoding/json"
  "encoding/xml"
  "net/url"
  "io/ioutil"
  "strconv"
  "golang.org/x/crypto/bcrypt"
  "github.com/urfave/negroni"
  "github.com/goincremental/negroni-sessions"
  "github.com/goincremental/negroni-sessions/cookiestore"
  "github.com/yosssi/ace"
  gmux "github.com/gorilla/mux"
  "gopkg.in/gorp.v1"
)

type Book struct {
  PK int64 `db:"pk"`
  Title string `db:"title"`
  Author string `db:"author"`
  Class string  `db:"class"`
  ID string `db:"id"`
}

type page struct {
  Books []Book
  Filter string
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

// User type to be used with the database
type User struct {
    Username string `db:"username"`
    Secret []byte `db:"secret"`
}


// A simple function that checks for error given an error object and a response object
func checkErr(err error, w http.ResponseWriter){
  if err != nil{
    http.Error(w, err.Error(), http.StatusInternalServerError)
    return
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


var database *sql.DB
var dbmap *gorp.DbMap

func initDB()  {
  database, _ = sql.Open("sqlite3", "dev.db")
  dbmap = &gorp.DbMap{Db: database, Dialect: gorp.SqliteDialect{}}

  dbmap.AddTableWithName(Book{}, "books").SetKeys(true, "pk")
  dbmap.CreateTablesIfNotExists()
}


func getBookCollection(books *[]Book, sortCol string, filterCol string, w http.ResponseWriter) bool {
  if sortCol == ""{
    sortCol = "pk"
  }
  var where string
  where = " "
  if filterCol == "fiction"{
    where = " where class between 800 and 900 "
  } else if filterCol == "nonfiction"{
    where = " where class not between 800 and 900 "
  }
  if _, err := dbmap.Select(books, "select * from books"+where+"order by " + sortCol); err!=nil{
    return false
  }
  return true
}

// MAIN FUNCTION
func main() {
  // Estaplishing connection with our database
  initDB()
  mux := gmux.NewRouter()

  // Handeling the main route
  mux.HandleFunc("/",func (w http.ResponseWriter, r *http.Request) {
    sortBy := sessions.GetSession(r).Get("sortBy")
    filterBy := sessions.GetSession(r).Get("Filter")
    template, err := ace.Load("templates/index", "", nil)
    checkErr(err, w)
    // Getting the query
    var sortCol string
    var filterCol string
    if sortBy != nil{
      sortCol = sortBy.(string)
    }
    if filterBy != nil{
      filterCol = filterBy.(string)
    }
    log.Print(filterCol)
    p := page{Books: []Book{}, Filter: filterCol}
    if !getBookCollection(&p.Books, sortCol, p.Filter, w){
      return
    }
    // Executing or renderin the template providing the query recieved
    err = template.Execute(w, p)
    checkErr(err, w)
  }).Methods("GET")


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
  }).Methods("POST")


  // Handeling adding books to the data base
  mux.HandleFunc("/books", func (w http.ResponseWriter, r *http.Request) {
    // Get the data of the selected book
    book, err := findBook(r.FormValue("id"))
    checkErr(err, w)

    // Inseting the book to the database
    b := Book{
      PK: -1,
      Title: book.BookData.Title,
      Author: book.BookData.Author,
      Class: book.Classification.MostPopular,
    }
    err = dbmap.Insert(&b)
    checkErr(err,w)
    err = json.NewEncoder(w).Encode(b)
    checkErr(err, w)
  }).Methods("PUT")

  mux.HandleFunc("/books/{pk}", func (w http.ResponseWriter, r *http.Request) {
    pk, _ := strconv.ParseInt(gmux.Vars(r)["pk"], 10, 64)
    _, err := dbmap.Delete(&Book{pk,"","","",""})
    checkErr(err, w)
    w.WriteHeader(http.StatusOK)
  }).Methods("DELETE")

  mux.HandleFunc("/books", func (w http.ResponseWriter, r *http.Request) {
    var b []Book
    var filterCol string
    filterBy := sessions.GetSession(r).Get("Filter")
    if filterBy != nil{
      filterCol = filterBy.(string)
    }
    sortBy := r.FormValue("sortBy")
    if !getBookCollection(&b, sortBy, filterCol, w){
      return
    }
    sessions.GetSession(r).Set("sortBy",sortBy)

    err := json.NewEncoder(w).Encode(b)
    checkErr(err, w)
  }).Methods("GET")

  mux.HandleFunc("/books/filter", func (w http.ResponseWriter, r *http.Request) {
    sessions.GetSession(r).Set("Filter", r.FormValue("filterBy"))
    var b []Book
    var sortCol string
    sortBy := sessions.GetSession(r).Get("sortBy")
    if sortBy != nil{
      sortCol = sortBy.(string)
    }
    if !getBookCollection(&b, sortCol, r.FormValue("filterBy"), w){
      return
    }
    err := json.NewEncoder(w).Encode(b)
    checkErr(err, w)
  }).Methods("GET")


  // Handeling the login route
  mux.HandleFunc("/login", func (w http.ResponseWriter, r *http.Request) {
    if r.FormValue("register") != "" || r.FormValue("login") != ""{
      http.Redirect(w, r, "/", http.StatusFound)
      return
    }

    template, err := ace.Load("templates/login", "", nil)
    checkErr(err, w)

    err = template.Execute(w,nil)
    checkErr(err, w)
  })

  n := negroni.Classic()
  n.Use(sessions.Sessions("your-Library", cookiestore.New([]byte("this-is-safe"))))
  n.Use(negroni.HandlerFunc(verifyDatabase))
  n.UseHandler(mux)
  //  Listining to the port
  n.Run(":8080")
}
