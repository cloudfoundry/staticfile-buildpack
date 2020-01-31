package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
)

func main() {
	var application struct {
		ApplicationURIs []string `json:"application_uris"`
	}

	err := json.Unmarshal([]byte(os.Getenv("VCAP_APPLICATION")), &application)
	if err != nil {
		log.Fatalf("failed to parse VCAP_APPLICATION: %s", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		switch req.URL.Path {
		case "/v1/deployment/installer/agent/unix/paas-sh/latest":
			context := struct{ URI string }{URI: application.ApplicationURIs[0]}
			t := template.Must(template.New("install.sh").ParseFiles("install.sh"))
			err := t.Execute(w, context)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

		case "/manifest.json", "/dynatrace-env.sh", "/liboneagentproc.so":
			contents, err := ioutil.ReadFile(strings.TrimPrefix(req.URL.Path, "/"))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			w.Write(contents)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))
}
