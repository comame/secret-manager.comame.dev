package main

type database interface {
	Save(s secret) error
	Get(namespace, name string) (*secret, error)
	List(namespace string) ([]secret, error)
	// 管理画面向けにすべてのネームスペースのシークレットを返す。強い権限で呼び出されるので、シークレットの値は伏せる。
	ListAllNamespaceForAdmin() ([]secret, error)
}
