package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"regexp"
)

type secret struct {
	ID        string     `json:"id"`
	Name      string     `json:"name"`
	Namespace string     `json:"namespace"`
	Type      secretType `json:"type"`
	Value     string     `json:"value"`
}

// シークレットを露出しないように Stringer を実装
func (s secret) String() string {
	return fmt.Sprintf("%s_%s", s.Namespace, s.Name)
}

// シークレットを露出しないように GoStringer を実装
func (s secret) GoString() string {
	return fmt.Sprintf("secret{Name:\"%s\", Namespace:\"%s\", Type:\"%s\"}", s.Name, s.Namespace, s.Type)
}

type secretType string

const (
	secretTypePlain = "plain"
)

func isValidSecretIdentifier(text string) bool {
	r := regexp.MustCompile(`^[a-zA-Z]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)
	return r.MatchString(text)
}

func (s *secret) validate() error {
	if !isValidSecretIdentifier(s.ID) {
		return errors.New("invalid letter in id")
	}
	if !isValidSecretIdentifier(s.Name) {
		return errors.New("invalid letter in name")
	}
	if !isValidSecretIdentifier(s.Namespace) {
		return errors.New("invalid letter in namespace")
	}

	return nil
}

func main() {
	http.HandleFunc("GET /v1/secrets/{namespace}/{name}", func(w http.ResponseWriter, r *http.Request) {
		ns := r.PathValue("namespace")
		name := r.PathValue("name")

		// trim "Bearer "
		authHeader := r.Header.Get("Authorization")
		if len(authHeader) < 8 {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("{}\n"))
			return
		}
		token := authHeader[7:]

		db := createK8sSecretDatabase(token)
		secret, err := db.Get(ns, name)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("{}\n"))
			return
		}

		js, err := json.Marshal(secret)
		if err != nil {
			log.Println(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("{}\n"))
			return
		}

		w.Write(js)
		w.Write([]byte("\n"))
	})

	log.Println("start 0.0.0.0:8080")
	http.ListenAndServe(":8080", nil)
}
