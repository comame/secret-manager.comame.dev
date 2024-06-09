package main

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
)

type k8sSecretDatabase struct {
	apiServer      string
	token          string
	namespace      string
	ignoreTLSError bool
}

func (d *k8sSecretDatabase) Save(s secret) error {
	if err := s.validate(); err != nil {
		return err
	}

	if s.Namespace != d.namespace {
		return errors.New("ネームスペースが Kubernetes のものと一致しない")
	}

	secret := convertSecretToK8sSecret(s)

	j, err := json.Marshal(secret)
	if err != nil {
		return err
	}

	b := bytes.NewReader(j)
	_, status, err := d.request(k8sSecretsEndpoint(d.namespace), http.MethodPost, b)
	if err != nil {
		return err
	}

	if status != http.StatusCreated {
		return errors.New("201 created 以外が返った")
	}

	return nil
}

func (d *k8sSecretDatabase) Get(namespace, name string) (*secret, error) {
	if namespace != d.namespace {
		return nil, errors.New("ネームスペースが Kubernetes のものと一致しない")
	}

	secrets, err := d.List(namespace)
	if err != nil {
		return nil, err
	}

	for _, s := range secrets {
		if s.Name == name {
			log.Printf("get secret %s %s", namespace, name)
			return &s, nil
		}
	}

	return nil, errors.New("secrets がない")
}

func (d *k8sSecretDatabase) List(namespace string) ([]secret, error) {
	if namespace != d.namespace {
		return nil, errors.New("ネームスペースが Kubernetes のものと一致しない")
	}

	if !isValidSecretIdentifier(namespace) {
		return nil, errors.New("無効な namespace")
	}

	u := k8sSecretsEndpoint(namespace) + "?labelSelector=app=secret-manager.comame.dev"
	res, status, err := d.request(u, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	if status != http.StatusOK {
		return nil, errors.New("200 ok 以外が返った")
	}

	var kl k8sSecretList
	if err := json.Unmarshal(res, &kl); err != nil {
		return nil, err
	}

	var l []secret
	for _, v := range kl.Items {
		s, err := convertK8sSecretToSecret(v)
		if err != nil {
			return nil, err
		}
		l = append(l, *s)
	}

	return l, nil
}

func (d *k8sSecretDatabase) ListAllNamespaceForAdmin() ([]secret, error) {
	u := "/api/v1/secrets?labelSelector=app=secret-manager.comame.dev"
	res, status, err := d.request(u, http.MethodGet, nil)
	if err != nil {
		return nil, err
	}

	if status != http.StatusOK {
		return nil, errors.New("200 ok 以外が返った")
	}

	log.Println("list all-secrets")

	var kl k8sSecretList
	if err := json.Unmarshal(res, &kl); err != nil {
		return nil, err
	}

	var l []secret
	for _, v := range kl.Items {
		s, err := convertK8sSecretToSecret(v)
		s.Value = ""
		if err != nil {
			return nil, err
		}

		l = append(l, *s)
	}

	return l, nil
}

// トークンを露出しないように Stringer を実装
func (d k8sSecretDatabase) String() string {
	return fmt.Sprintf("{%s %s %v}", d.apiServer, d.namespace, d.ignoreTLSError)
}

// トークンを露出しないように GoStringer を実装
func (d k8sSecretDatabase) GoString() string {
	return fmt.Sprintf("k8sSecretDatabase{apiServer:\"%s\", namespace:\"%s\", ignoreTLSError:%v}", d.apiServer, d.namespace, d.ignoreTLSError)
}

type k8sServiceAccountToken struct {
	Namespace string `json:"kubernetes.io/serviceaccount/namespace"`
}

// Kubernetes の Secret に保存するデータベースを作成する。
// Secret.Namespace はそのまま Kubernetes のネームスペースに対応する。`token` は Kubernetes API へのリクエストに使う Bearer トークン。
//
// 読み込まれる環境変数
// KUBE_APISERVER: Kubernetes API の URL。
// KUBE_IGNORE_TLS_ERROR: Kubernetes API へのリクエスト時、TLS の証明書エラーを無視する。
//
//	token: Kubernetes API へのリクエストに使う Bearer トークン
func createK8sSecretDatabase(token string) database {
	d := new(k8sSecretDatabase)

	d.token = token

	d.apiServer = os.Getenv("KUBE_APISERVER")
	if d.apiServer == "" {
		// クラスター内部の API を使用する
		d.apiServer = "https://kubernetes.default.svc.cluster.local"
	}

	// TODO: 可能ならばクラスター内部の CA を使用する
	ig := os.Getenv("KUBE_IGNORE_TLS_ERROR")
	if ig != "" {
		d.ignoreTLSError = true
	}

	jwt := strings.Split(token, ".")
	if len(jwt) != 3 {
		// トークンが不正なとき、結局 Kubernetes API が Unauthorized を返すので、ゼロ値を入れておく
		return d
	}
	plb, err := base64.RawURLEncoding.DecodeString(jwt[1])
	if err != nil {
		return d
	}
	var pl k8sServiceAccountToken
	if err := json.Unmarshal([]byte(plb), &pl); err != nil {
		return d
	}
	d.namespace = pl.Namespace

	return d
}

func (d *k8sSecretDatabase) request(path string, method string, body io.Reader) ([]byte, int, error) {
	client := new(http.Client)

	if d.ignoreTLSError {
		client.Transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true,
			},
		}
	}

	req, err := http.NewRequest(method, d.apiServer+path, body)
	if err != nil {
		return nil, 0, err
	}

	req.Header.Set("Authorization", "Bearer "+d.token)
	req.Header.Set("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, 0, errors.Join(err, errors.New("kubernetes api の呼び出しに失敗"))
	}

	b, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, 0, err
	}

	return b, res.StatusCode, nil
}

type k8sSecretList struct {
	Items []k8sSecret `json:"items"`
}

type k8sSecret struct {
	ApiVersion string            `json:"apiVersion"`
	Data       map[string]string `json:"data"`
	Immutable  bool              `json:"immutable"`
	Metadata   k8sMetadata       `json:"metadata"`
	Type       string            `json:"type"`
}

type k8sMetadata struct {
	Labels    map[string]string `json:"labels"`
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
}

func k8sSecretsEndpoint(namespace string) string {
	return fmt.Sprintf("/api/v1/namespaces/%s/secrets", namespace)
}

func convertSecretToK8sSecret(s secret) k8sSecret {
	return k8sSecret{
		ApiVersion: "v1",
		Metadata: k8sMetadata{
			Name: fmt.Sprintf("secret-manager--%s--%s", s.Namespace, s.Name),
			Labels: map[string]string{
				"app":                            "secret-manager.comame.dev",
				"secret-manager.comame.dev/id":   s.ID,
				"secret-manager.comame.dev/name": s.Name,
				"secret-manager.comame.dev/type": string(s.Type),
			},
			Namespace: s.Namespace,
		},
		Immutable: true,
		Data: map[string]string{
			"value": base64.StdEncoding.EncodeToString([]byte(s.Value)),
		},
	}
}

func convertK8sSecretToSecret(s k8sSecret) (*secret, error) {
	id, ok := s.Metadata.Labels["secret-manager.comame.dev/id"]
	if !ok {
		return nil, fmt.Errorf("secret %s に必要な label がない", s.Metadata.Name)
	}
	name, ok := s.Metadata.Labels["secret-manager.comame.dev/name"]
	if !ok {
		return nil, fmt.Errorf("secret %s に必要な label がない", s.Metadata.Name)
	}
	typ, ok := s.Metadata.Labels["secret-manager.comame.dev/type"]
	if !ok {
		return nil, fmt.Errorf("secret %s に必要な label がない", s.Metadata.Name)
	}

	if typ != secretTypePlain {
		return nil, fmt.Errorf("secret %s の type が不正", s.Metadata.Name)
	}

	v64, ok := s.Data["value"]
	if !ok {
		return nil, fmt.Errorf("secret %s の data に value: がない", s.Metadata.Name)
	}
	b, err := base64.StdEncoding.DecodeString(v64)
	if err != nil {
		return nil, errors.Join(fmt.Errorf("secret %s の data をでコードできなかった", s.Metadata.Name), err)
	}

	return &secret{
		ID:        id,
		Name:      name,
		Namespace: s.Metadata.Namespace,
		Type:      secretType(typ),
		Value:     string(b),
	}, nil
}
