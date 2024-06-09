# シークレットの取得を試す

function get_token() {
    kubectl describe secrets secret-store | \
    grep token: | awk '{ print $2 }'
}

function create_secret() {
    value=$(echo -n 'secret!' | base64)
    kubectl apply -f - <<EOF
        apiVersion: v1
        kind: Secret
        type: Opaque
        metadata:
            name: dummy-secret
            labels:
                app: secret-manager.comame.dev
                secret-manager.comame.dev/name: dummy-secret
                secret-manager.comame.dev/type: plain
                secret-manager.comame.dev/id: dummy-secret
        data:
            value: $value
EOF
}

function delete_secret() {
    kubectl delete secret/dummy-secret
}

token=$(get_token)

create_secret

curl -H "Authorization: Bearer $token" http://localhost:8080/v1/secrets/default/dummy-secret

# 権限がないので失敗する
curl -H "Authorization: Bearer $token" http://localhost:8080/v1/secrets/other-namespace/will-fail

delete_secret
