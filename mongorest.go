package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"goji.io"
	"goji.io/pat"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Book struct {
	ISBN    string   `json: isbn`
	Title   string   `json: title`
	Authors []string `json: authors`
	Price   string   `json: price`
}

func ErrorWithJSON(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, "{message: %q}", message)
}

func ResponseWithJSON(w http.ResponseWriter, json []byte, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	w.Write(json)
}

func ensureIndex(s *mgo.Session) {
	session := s.Copy()
	defer session.Close()

	collection := session.DB("store").C("books")

	index := mgo.Index{
		Key:        []string{"ISBN"},
		Unique:     true,
		DropDups:   true,
		Background: true,
		Sparse:     true,
	}
	err := collection.EnsureIndex(index)
	if err != nil {
		panic(err)
	}
}

func allBooks(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		collection := session.DB("store").C("books")

		var books []Book
		err := collection.Find(bson.M{}).All(&books)
		if err != nil {
			ErrorWithJSON(w, "Database error", http.StatusInternalServerError)
			log.Println("Failed to get all books: ", err)
			return
		}

		respBody, err := json.MarshalIndent(books, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		ResponseWithJSON(w, respBody, http.StatusOK)
	}
}

func addBook(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		var book Book
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&book)
		if err != nil {
			ErrorWithJSON(w, "Incorrect Body", http.StatusBadRequest)
			return
		}

		collection := session.DB("store").C("books")
		err = collection.Insert(book)
		if err != nil {
			if mgo.IsDup(err) {
				ErrorWithJSON(w, "a book with this ISBN already exists in the database", http.StatusBadRequest)
				return
			}
			ErrorWithJSON(w, "database error", http.StatusInternalServerError)
			log.Println("failed to insert book: ", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Location", r.URL.Path+"/"+book.ISBN)
		w.WriteHeader(http.StatusCreated)
	}
}

func bookByISBN(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		isbn := pat.Param(r, "isbn")
		collection := session.DB("store").C("books")

		var book Book
		err := collection.Find(bson.M{"isbn:": isbn}).One(&book)
		if err != nil {
			ErrorWithJSON(w, "database error", http.StatusInternalServerError)
			log.Println("failed to find book: ", err)
			return
		}

		if book.ISBN == "" {
			ErrorWithJSON(w, "book not found", http.StatusNotFound)
			return
		}

		respBody, err := json.MarshalIndent(book, "", "  ")
		if err != nil {
			log.Fatal(err)
		}

		ResponseWithJSON(w, respBody, http.StatusOK)
	}
}

func updateBook(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		isbn := pat.Param(r, "isbn")

		var book Book
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&book)
		if err != nil {
			ErrorWithJSON(w, "incorrect body", http.StatusBadRequest)
			return
		}

		collection := session.DB("store").C("books")

		err = collection.Update(bson.M{"isbn:": isbn}, &book)
		if err != nil {
			switch err {
			default:
				ErrorWithJSON(w, "database error", http.StatusInternalServerError)
				log.Println("update book failed: ", err)
				return
			case mgo.ErrNotFound:
				ErrorWithJSON(w, "book not found", http.StatusNotFound)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func deleteBook(s *mgo.Session) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		session := s.Copy()
		defer session.Close()

		isbn := pat.Param(r, "isbn")
		collection := session.DB("store").C("books")

		err := collection.Remove(bson.M{"isbn:": isbn})
		if err != nil {
			switch err {
			default:
				ErrorWithJSON(w, "database error", http.StatusInternalServerError)
				log.Println("failed to delete book: ", err)
				return
			case mgo.ErrNotFound:
				ErrorWithJSON(w, "book not found", http.StatusNotFound)
				return
			}
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func main() {
	// retrieve mongodb service port from environment
	os.Getenv("MONGODB_SERVICE_PORT")
	for _, e := range os.Environ() {
		pair := strings.Split(e, "=")
		fmt.Println(pair[0] + " = " + pair[1])
	}

	// connect to mongodb
	session, err := mgo.Dial("localhost")
	if err != nil {
		panic(err)
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true) // use secondary node for reading if possible
	ensureIndex(session)                 // create an index if it doesn't already exist

	// construct multiplexer
	mux := goji.NewMux()
	mux.HandleFunc(pat.Get("/books"), allBooks(session))
	mux.HandleFunc(pat.Put("/books"), addBook(session))
	mux.HandleFunc(pat.Get("/books/:isbn"), bookByISBN(session))
	mux.HandleFunc(pat.Put("/books/:isbn"), updateBook(session))
	mux.HandleFunc(pat.Delete("/books/:isbn"), deleteBook(session))

	// start listener
	log.Fatal(http.ListenAndServe(":9099", mux))
}
