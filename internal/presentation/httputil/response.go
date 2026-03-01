package httputil

import (
	"encoding/json"
	"html/template"
	"net/http"
)

func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func Error(w http.ResponseWriter, status int, errCode, desc string) {
	resp := map[string]string{"error": errCode}
	if desc != "" {
		resp["error_description"] = desc
	}
	JSON(w, status, resp)
}

func HTML(w http.ResponseWriter, tmpl *template.Template, name string, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	tmpl.ExecuteTemplate(w, name, data)
}

func DecodeJSON(r *http.Request, v any) error {
	return json.NewDecoder(r.Body).Decode(v)
}
