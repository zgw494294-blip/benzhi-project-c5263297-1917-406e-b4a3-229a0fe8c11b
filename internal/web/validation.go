package web

import (
	"encoding/json"
	"net/http"
	"stage-rigging-clearance/internal/domain"
	"strings"
)

func decodeBody(r *http.Request, v any) error {
	if r.Body == nil {
		return domain.ErrInvalidInput
	}
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		return domain.ErrInvalidInput
	}
	return nil
}
func requireMethod(w http.ResponseWriter, r *http.Request, method string) bool {
	if r.Method != method {
		w.Header().Set("Allow", method)
		http.Error(w, "请求方法不支持", http.StatusMethodNotAllowed)
		return false
	}
	return true
}
func cleanID(v string) string { return strings.TrimSpace(v) }
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}
