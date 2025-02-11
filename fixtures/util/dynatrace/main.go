package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		var withoutAgentPath bool
		path := req.URL.Path

		if strings.HasPrefix(path, "/without-agent-path") {
			path = strings.TrimPrefix(path, "/without-agent-path")
			withoutAgentPath = true
		}

		switch path {
		case "/v1/deployment/installer/agent/unix/paas-sh/latest":
			context := struct{ URI string }{URI: req.Host}
			t := template.Must(template.New("install.sh").ParseFiles("install.sh"))
			err := t.Execute(w, context)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, err.Error())
				return
			}

		case "/dynatrace-env.sh", "/liboneagentproc.so", "/ruxitagentproc.conf":
			contents, err := os.ReadFile(strings.TrimPrefix(req.URL.Path, "/"))
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprint(w, err.Error())
				return
			}

			fmt.Fprintf(w, "%s", contents)

		case "/manifest.json":
			var payload map[string]interface{}
			file, err := os.Open("manifest.json")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			err = json.NewDecoder(file).Decode(&payload)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}

			if withoutAgentPath {
				payload["technologies"] = map[string]interface{}{
					"process": map[string]interface{}{
						"linux-x86-64": []struct{}{},
					},
				}
			}

			json.NewEncoder(w).Encode(payload)

		case "/v1/deployment/installer/agent/processmoduleconfig":
			fakeConfig, err := os.ReadFile("fake_config.json")
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(err.Error()))
				return
			}
			w.Write(fakeConfig)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", os.Getenv("PORT")), nil))
}
