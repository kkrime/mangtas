package main

import (
	"fmt"
	"io"

	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

var (
	validWordRegex = regexp.MustCompile(`^[a-zA-Z,]+$`)
)

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/", homePage)
	router.HandleFunc("/v1/addwords", addWords)
	log.Fatal(http.ListenAndServe(":8080", router))
}

func homePage(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to the mangtas text count service homepage!")
}

func addWords(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	input, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Internal Error")
		return
	}

	if validWordRegex.Match(input) == false {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, "Please make sure all words contain only [A-Ba-b] and seperated by commas (,)")
		return
	}

	words := strings.Split(string(input), ",")
	err = addWordsDB(words)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, err.Error())
		return
	}

	fmt.Fprintf(w, "Words successfully added Wo0p!")
}
