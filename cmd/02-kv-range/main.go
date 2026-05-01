package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mabhi256/etcd-demo/internal/etcdclient"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	cli := etcdclient.NewClient()
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cli.Put(ctx, "/config/app/version", "1.0.0")
	cli.Put(ctx, "/config/app/port", "8080")
	res1, _ := cli.Put(ctx, "/config/db/host", "localhost")

	pretty, _ := json.MarshalIndent(res1, "", "  ")
	fmt.Printf("PUT /config: %s\n\n", pretty)

	//-------------------------------------------
	// Prefix
	//-------------------------------------------
	prefix := "/config/app"
	fmt.Printf("GET '%s' prefix:\n", prefix)
	res2, err := cli.Get(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		log.Fatal(err)
	}

	for _, kv := range res2.Kvs {
		fmt.Printf("%s -> %s\n", kv.Key, kv.Value)
	}

	//-------------------------------------------
	// prefix + keys-only
	//-------------------------------------------
	fmt.Printf("\nGET prefix + keys-only:\n")
	res3, err := cli.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithKeysOnly())
	if err != nil {
		log.Fatal(err)
	}

	for _, kv := range res3.Kvs {
		fmt.Printf("%s\n", kv.Key)
	}

	//-------------------------------------------
	// prefix + count-only
	//-------------------------------------------
	fmt.Printf("\nGET prefix + count-only:\n")
	res4, err := cli.Get(ctx, prefix, clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%s: %d\n", prefix, res4.Count)

	//-------------------------------------------
	// prefix + delete
	//-------------------------------------------
	fmt.Printf("\nDEL prefix:\n")
	res, err := cli.Delete(ctx, prefix, clientv3.WithPrefix())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("  deleted: %d\n\n", res.Deleted)

	//-------------------------------------------
	// range (of shard)
	//-------------------------------------------
	for i := range 10 {
		job := fmt.Sprintf("/jobs/%03d", i)
		task := fmt.Sprintf("{\"task\":\"resize-image\",\"file\":\"img%03d.png\"}", i)
		cli.Put(ctx, job, task)
	}

	fmt.Printf("\nGET range:\n")
	res5, err := cli.Get(ctx, "/jobs/001", clientv3.WithRange("/jobs/004"))
	if err != nil {
		log.Fatal(err)
	}

	for _, kv := range res5.Kvs {
		fmt.Printf("%s -> %s\n", kv.Key, kv.Value)
	}

	// Get range till end
	fmt.Printf("\nGET range till end:\n")
	res5, err = cli.Get(ctx, "/jobs/006", clientv3.WithRange("/jobs/\xff"))
	if err != nil {
		log.Fatal(err)
	}

	for _, kv := range res5.Kvs {
		fmt.Printf("%s -> %s\n", kv.Key, kv.Value)
	}

	//-------------------------------------------
	// from-key (no upper bound, will bleed into other keys)
	//-------------------------------------------
	fmt.Printf("\nGET from-key:\n")
	cli.Put(ctx, "bleeding", "true")
	res6, err := cli.Get(ctx, "/jobs/007", clientv3.WithFromKey())
	if err != nil {
		log.Fatal(err)
	}

	for _, kv := range res6.Kvs {
		fmt.Printf("%s -> %s\n", kv.Key, kv.Value)
	}

	//-------------------------------------------
	// from-key + limit
	//-------------------------------------------
	fmt.Printf("\nGET from-key + limit:\n")
	res7, _ := cli.Get(ctx, "/jobs/004", clientv3.WithFromKey(), clientv3.WithLimit(3))
	for _, kv := range res7.Kvs {
		fmt.Printf("%s -> %s\n", kv.Key, kv.Value)
	}

	// Get next batch/page
	fmt.Printf("\nGET next batch from last key:\n")
	len := len(res7.Kvs)
	lastKey := string(res7.Kvs[len-1].Key)
	res8, _ := cli.Get(ctx, lastKey+"\x00", clientv3.WithFromKey(), clientv3.WithLimit(3))
	for _, kv := range res8.Kvs {
		fmt.Printf("%s -> %s\n", kv.Key, kv.Value)
	}
}
