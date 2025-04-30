// SPDX-FileCopyrightText: 2025 Danila Gorelko <hello@danilax86.space>
//
// SPDX-License-Identifier: MIT

package handlers

import (
	"mr-metrics/internal/web"
	"net/http"
)

func handleStyle(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")

	css, err := web.GetStyleCSS()
	if err != nil {
		http.Error(w, "Failed to read style.css", http.StatusInternalServerError)
		return
	}

	_, err = w.Write(css)
	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
		return
	}
}
