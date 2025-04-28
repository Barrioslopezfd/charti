package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func main() {
	http.HandleFunc("GET /api/chart/", handlerChart)
	log.Println("Listening to port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handlerChart(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Path
	chart_url := strings.TrimPrefix(url, "/api/chart/")

	f, err := getIndexFile(chart_url)
	if err != nil {
		responseWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	var infoList []map[string]string

	for _, charts := range f.Entries {
		for _, chart := range charts {
			if len(chart.URLs) < 1 {
				continue
			}
			downloadTgzFile(chart.URLs[0])
			idx := strings.LastIndex(chart.URLs[0], "/")
			val, err := renderHelmChart("./charts/" + chart.URLs[0][idx:])
			if err != nil {
				responseWithError(w, http.StatusInternalServerError, err.Error())
				return
			}
			infoList, err = getContainerImages(val)
			if err != nil {
				responseWithError(w, http.StatusInternalServerError, err.Error())
				return
			}

			infoList, err = downloadDockerImages(infoList)
			if err != nil {
				responseWithError(w, http.StatusInternalServerError, err.Error())
				return
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(infoList); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func responseWithError(w http.ResponseWriter, code int, msg string) {
	type retError struct {
		Error string `json:"error"`
	}
	datErr := retError{
		Error: msg,
	}
	datWrong, err := json.Marshal(datErr)
	if err != nil {
		log.Printf("Error marshaling json: %v", err)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(500)
		return
	}
	log.Printf("%s", msg)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(datWrong)
}
