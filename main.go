package main

import (
	"encoding/json"
	"fmt"
	"io"

	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

var (
	// regex to validate input (csv)
	wordValidationRegex = regexp.MustCompile(`^[a-zA-Z,]+$`)
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", homePage).Methods("GET")
	router.HandleFunc("/v1/addwords", addWords).Methods("POST")
	router.HandleFunc("/v1/gettopwords", getTopWords).Methods("GET")
	log.Fatal(http.ListenAndServe(":8080", router))
}

// welcome page/health check
func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the mangtas word count service homepage!")
}

// add words to the service
// works in a all or nothing manner
func addWords(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// try and get body from input
	input, err := io.ReadAll(r.Body)
	if err != nil {
		// send internal error
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Error")
		log.Error(err)
		return
	}

	// validate input
	if len(input) == 0 || wordValidationRegex.Match(input) == false {
		// send bad request
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Please make sure all words contain only [A-Ba-b] and are seperated by commas (,)")
		return
	}

	// split input into []string
	words := strings.Split(string(input), ",")

	// needed for corner case...
	wordsCountMap := make(map[string]int)
	for _, word := range words {
		// skip empty word
		if word == "" {
			continue
		}
		// make word all lower case
		lowerCaseWord := strings.ToLower(word)
		wordsCountMap[lowerCaseWord] += 1
	}

	// attempt to add words to db
	err = addWordsDB(wordsCountMap)
	if err != nil {
		// send internal error
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	// return success message
	fmt.Fprintf(w, "Word(s) successfully added Wo0p!")
}

// out struct for sending data out
type out struct {
	Message string
	// set output as interface{} so out can be used for any type of data we want to output to user
	Output interface{}
}

type wordOut struct {
	Word  string
	Count int
}

// get top 10 most frequently sent words
func getTopWords(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	// try and get body from input
	topWords, err := getTopWordsDB()
	if err != nil {
		// send internal error
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Error")
		log.Error(err)
		return
	}

	// out message struct
	output := out{Message: "These are the top words for the mangtas word count service!", Output: topWords}

	// marshal to json
	var outJson []byte
	outJson, err = json.Marshal(&output)

	// send out message
	fmt.Fprintf(w, string(outJson))
}
