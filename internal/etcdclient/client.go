package etcdclient

import (
	"log"
	"time"

	clientv3 "go.etcd.io/etcd/client/v3"
)

func NewClient() *clientv3.Client {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints: []string{
			"http://localhost:2379",
			"http://localhost:2479",
			"http://localhost:2579",
		},
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		log.Fatal(err)
	}

	return cli
}
