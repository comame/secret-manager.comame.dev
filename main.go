package main

import (
	"errors"
	"fmt"
	"log"
	"os"
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

func init() {
}

func main() {
	databaseInstance := createK8sSecretDatabase(os.Getenv("KUBE_SERVICEACCOUNT_TOKEN"))
	log.Println(databaseInstance)
	log.Printf("%#v", databaseInstance)

	if err := databaseInstance.Save(secret{
		ID:        "id",
		Name:      "test-secret",
		Namespace: "secret-dev",
		Type:      secretTypePlain,
		Value:     "super secret string!",
	}); err != nil {
		log.Println(err)
	}

	if err := databaseInstance.Save(secret{
		ID:        "id",
		Name:      "test-secret-2",
		Namespace: "secret-dev",
		Type:      secretTypePlain,
		Value:     "super secret string!",
	}); err != nil {
		log.Println(err)
	}

	sarr, err := databaseInstance.List("secret-dev")
	log.Println(sarr, err)

	s, err := databaseInstance.Get("secret-dev", "test-secret")
	log.Println(s, err)
	log.Printf("%#v", s)
}
