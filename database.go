package main

type database interface {
	Save(s secret) error
	Get(namespace, name string) (*secret, error)
	List(namespace string) ([]secret, error)
	ListAllNamespace() ([]secret, error)
}
