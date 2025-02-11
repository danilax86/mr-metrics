package handlers

import (
	"io"
	"net/http"
	"os"
)

func handleStyle(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")

	file, err := os.Open("./internal/web/style.css")
	if err != nil {
		http.Error(w, "Failed to open file", http.StatusInternalServerError)
		return
	}
	defer file.Close()

	_, err = io.Copy(w, file)
	if err != nil {
		http.Error(w, "Failed to copy file", http.StatusInternalServerError)
		return
	}
}
