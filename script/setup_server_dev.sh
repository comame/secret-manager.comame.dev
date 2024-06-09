# 開発時、サーバの起動に必要な環境変数を準備する
#
# source scripts/setup_server_dev.sh
# go run .

function get_apiserver() {
    kubectl config view | \
    grep server: | awk '{ print $2 }'
}

export KUBE_APISERVER=$(get_apiserver)
export KUBE_IGNORE_TLS_ERROR=true
