package main

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"log/slog"
)

func main() {
	http.HandleFunc("/", getAPIStatus)
	
	err := http.ListenAndServe(":3000", nil)
	if errors.Is(err, http.ErrServerClosed) {
		fmt.Printf("Server closed\n")
	} else if err != nil {
		fmt.Printf("Error starting server: %s\n", err)
		os.Exit(1)
	}

}

func getAPIStatus(w http.ResponseWriter, r *http.Request) {
	// Just want to log instance and respond with short status of server
}
