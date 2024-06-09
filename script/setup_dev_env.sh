#!/bin/bash

export KUBE_IGNORE_TLS_ERROR=true
KUBE_NAMESPACE='secret-dev'

function create_ns() {
    kubectl create ns $KUBE_NAMESPACE || true
    kubectl apply -f - <<EOF
      apiVersion: v1
      kind: Namespace
      metadata:
        name: $KUBE_NAMESPACE
EOF
}

create_ns

function grant_sa() {
    kubectl apply -f - 1>/dev/null <<EOF
        apiVersion: rbac.authorization.k8s.io/v1
        kind: Role
        metadata:
            name: secret-manager
            namespace: $KUBE_NAMESPACE
        rules:
            - apiGroups:
                - ""
              resources:
                - secrets
              verbs:
                - get
                - create
                - list
                - delete
EOF

    kubectl apply -f - 1>/dev/null <<EOF
        apiVersion: rbac.authorization.k8s.io/v1
        kind: RoleBinding
        metadata:
          name: secret-manager
          namespace: $KUBE_NAMESPACE
        roleRef:
          apiGroup: rbac.authorization.k8s.io
          kind: Role
          name: secret-manager
        subjects:
          - kind: ServiceAccount
            name: default
            namespace: $KUBE_NAMESPACE
EOF

    kubectl apply -f - 1>/dev/null <<EOF
      apiVersion: rbac.authorization.k8s.io/v1
      kind: ClusterRoleBinding
      metadata:
        name: secret-manager-admin
        namespace: $KUBE_NAMESPACE
      roleRef:
        apiGroup: rbac.authorization.k8s.io
        kind: ClusterRole
        name: cluster-admin
      subjects:
        - kind: ServiceAccount
          name: default
          namespace: $KUBE_NAMESPACE
EOF
}

function create_sa_token() {
    kubectl apply -f - 1>/dev/null <<EOF
        apiVersion: v1
        kind: Secret
        metadata:
            name: secrets-dev
            namespace: $KUBE_NAMESPACE
            annotations:
                kubernetes.io/service-account.name: default
        type: kubernetes.io/service-account-token
EOF

    kubectl describe -n $KUBE_NAMESPACE secrets/secrets-dev | \
    grep token: | awk '{ print $2 }'
}

function get_apiserver() {
    kubectl config view | \
    grep server: | awk '{ print $2 }'
}

grant_sa

export KUBE_SERVICEACCOUNT_TOKEN=$(create_sa_token)
export KUBE_APISERVER=$(get_apiserver)
