package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type Person struct {
	ID        string   `json:id,omitempty`
	Firstname string   `json:firstname,omitempty`
	Lastname  string   `json:lastname,omitempty`
	Address   *Address `json:address,omitempty`
}

type Address struct {
	City    string `json:city,omitempty`
	Country string `json:country,omitempty`
}

// globale Variable, da wir keine Datenbank nutzen
var people []Person

func GetPersonEndpoint(w http.ResponseWriter, req *http.Request) {
	// Rueckgabe einer einzelnen Person anhand ihrer ID
	params := mux.Vars(req)
	for _, item := range people {
		if item.ID == params["id"] {
			json.NewEncoder(w).Encode(item)
			return
		}
	}
	json.NewEncoder(w).Encode(&Person{}) // return empty Person (newborn)
}

func GetPeopleEndpoint(w http.ResponseWriter, req *http.Request) {
	// Rueckgabe des kompletten people arrays
	json.NewEncoder(w).Encode(people)
}

func CreatePersonEndpoint(w http.ResponseWriter, req *http.Request) {
	// wandelt json eingabe in einen person struct um
	params := mux.Vars(req)
	var person Person
	_ = json.NewDecoder(req.Body).Decode(&person)
	person.ID = params["id"]
	people = append(people, person)
	json.NewEncoder(w).Encode(people)
}

func DeletePersonEndpoint(w http.ResponseWriter, req *http.Request) {
	params := mux.Vars(req)
	for index, item := range people {
		if item.ID == params["id"] {
			people = append(people[:index], people[index+1:]...) // erzeugt eine neue Liste, die die aktuelle person auslaesst
			// 0 bis index und index + 1 bis ende
			break
		}
	}
	json.NewEncoder(w).Encode(people)
}

func main() {
	router := mux.NewRouter()

	// generate test data
	people = append(people, Person{ID: "1", Firstname: "Michael", Lastname: "Leimenmeier", Address: &Address{City: "Dortmund", Country: "Germany"}})
	people = append(people, Person{ID: "2", Firstname: "Sascha Mario", Lastname: "Klein", Address: &Address{City: "Bochum", Country: "Germany"}})
	people = append(people, Person{ID: "3", Firstname: "Taran"})
	people = append(people, Person{ID: "4", Firstname: "Anju"})

	router.HandleFunc("/people", GetPeopleEndpoint).Methods("GET")
	router.HandleFunc("/people/{id}", GetPersonEndpoint).Methods("GET")
	router.HandleFunc("/people/{id}", CreatePersonEndpoint).Methods("POST")
	router.HandleFunc("/people/{id}", DeletePersonEndpoint).Methods("DELETE")

	log.Fatal(http.ListenAndServe(":1234", router))
}
