# ETCD demo

## Start etcd cluster

```bash
# start containers
docker compose up -d

# Show cluster members
docker exec etcd1 etcdctl member list

# Which node is the leader right now?
docker exec etcd1 etcdctl endpoint status --cluster -w table

# What happens if you write to etcd1 and read from etcd2/3?
docker exec etcd1 etcdctl put mykey "hello"
# OK
docker exec etcd2 etcdctl get mykey
docker exec etcd3 etcdctl get mykey
# mykey
# hello

# Stop a node and write a vakue
docker stop etcd2
docker exec etcd1 etcdctl put mykey2 "hello"

# Restart and check if value propagated
docker start etcd2
docker exec etcd2 etcdctl get mykey2
```

## Run snippets

```bash
go run ./cmd/01-kv-basic
```
