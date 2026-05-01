package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/mabhi256/etcd-demo/internal/etcdclient"
)

func main() {
	cli := etcdclient.NewClient()
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//-------------------------------------------
	// Write
	//-------------------------------------------
	fmt.Println("=== PUT Operation ===")

	res1, err := cli.Put(ctx, "/my-key", "my-value")
	if err != nil {
		log.Fatal(err)
	}

	pretty, _ := json.MarshalIndent(res1, "", "  ")
	fmt.Printf("Write response:\n%s\n\n", pretty)

	//-------------------------------------------
	// Read
	//-------------------------------------------
	fmt.Println("=== GET Operation ===")

	res2, err := cli.Get(ctx, "/my-key")
	if err != nil {
		log.Fatal(err)
	}

	for _, kv := range res2.Kvs {
		fmt.Printf("%s -> %s\n", kv.Key, kv.Value)
	}

	pretty, _ = json.MarshalIndent(res2, "", "  ")
	fmt.Printf("\nGet full response:\n%s\n\n", pretty)

	//-------------------------------------------
	// Delete
	//-------------------------------------------
	fmt.Println("=== DEL Operation ===")

	res3, err := cli.Delete(ctx, "/my-key")
	if err != nil {
		log.Fatal(err)
	}

	pretty, _ = json.MarshalIndent(res3, "", "  ")
	fmt.Printf("\nDel response:\n%s\n", pretty)

	res4, _ := cli.Get(ctx, "/my-key")
	for _, kv := range res4.Kvs {
		fmt.Printf("%s -> %s\n", kv.Key, kv.Value)
	}
	pretty, _ = json.MarshalIndent(res4, "", "  ")
	fmt.Printf("\nGet after Del response:\n%s\n\n", pretty)
}
