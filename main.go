package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "OK")
	})

	fmt.Println("Go service running on port 8081")
	http.ListenAndServe(":8081", nil)
}
