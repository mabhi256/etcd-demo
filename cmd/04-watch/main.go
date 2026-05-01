package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mabhi256/etcd-demo/internal/etcdclient"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	cli := etcdclient.NewClient()
	defer cli.Close()

	//-------------------------------------------
	// Watch Single Key
	//-------------------------------------------
	fmt.Println("=== Watch Single Key ===")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := "/my-key"

	cli.Put(ctx, key, "value-1")
	res, _ := cli.Get(ctx, key)
	fmt.Printf("Init: %s -> %s\n\n", key, res.Kvs[0].Value)

	// Launch a goroutine to update the key after 1s
	go func() {
		time.Sleep(1 * time.Second)
		cli.Put(context.Background(), key, "value-2")
	}()

	// Launch a goroutine to delete the key after 3s
	go func() {
		time.Sleep(3 * time.Second)
		cli.Delete(context.Background(), key)
	}()

	watchChan := cli.Watch(ctx, key)
	for watchRes := range watchChan {
		for _, event := range watchRes.Events {
			switch event.Type {
			case mvccpb.PUT:
				fmt.Printf("PUT %s = %s\n", event.Kv.Key, event.Kv.Value)
			case mvccpb.DELETE:
				fmt.Printf("DEL %s\n", event.Kv.Key)
			}
		}
	}

	//-------------------------------------------
	// Watch Prefix
	//-------------------------------------------
	fmt.Println("\n=== Watch Prefix '/app' ===")

	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()

	go func() {
		time.Sleep(1 * time.Second)
		cli.Put(context.Background(), "/app/version", "1.1.0")
	}()

	// Launch a goroutine to delete the key after 3s
	go func() {
		time.Sleep(2 * time.Second)
		cli.Put(context.Background(), "/app/host", "10.1.1.0")
	}()

	watchPrefix := cli.Watch(ctx2, "/app", clientv3.WithPrefix())
	for watchRes := range watchPrefix {
		for _, event := range watchRes.Events {
			switch event.Type {
			case mvccpb.PUT:
				fmt.Printf("PUT %s = %s\n", event.Kv.Key, event.Kv.Value)
			case mvccpb.DELETE:
				fmt.Printf("DEL %s\n", event.Kv.Key)
			}
		}
	}

	//-------------------------------------------
	// Watch with Revision
	//
	// cli.Put(...)        ← revision 10
	// cli.Get(...)        ← reads at revision 10
	//   ... time passes, revision 11 happens HERE (missed!) ...
	// cli.Watch(...)      ← starts at revision 12, never sees 11
	//-------------------------------------------
	fmt.Println("\n=== Watch with Revision ===")

	ctx3, cancel3 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel3()

	key3 := "/my-key-3"
	res3, _ := cli.Put(ctx3, key3, "value-1")
	startRev := res3.Header.Revision
	cli.Put(ctx3, key3, "value-2") // gap: written before Watch starts

	go func() {
		time.Sleep(1 * time.Second)
		cli.Put(context.Background(), key3, "value-3")
	}()

	fmt.Println("Watch (no rev) — misses value-2 written in the gap:")
	for watchRes := range cli.Watch(ctx3, key3) {
		for _, event := range watchRes.Events {
			fmt.Printf("  %s %s = %s\n", event.Type, event.Kv.Key, event.Kv.Value)
		}
	}

	// Use a fresh context for the second watch
	ctx4, cancel4 := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel4()

	fmt.Printf("\nWatch (rev=%d):\n", startRev+1)
	for watchRes := range cli.Watch(ctx4, key3, clientv3.WithRev(startRev+1)) {
		for _, event := range watchRes.Events {
			fmt.Printf("  %s %s = %s\n", event.Type, event.Kv.Key, event.Kv.Value)
		}
	}
}
