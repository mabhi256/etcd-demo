package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/mabhi256/etcd-demo/internal/etcdclient"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	cli := etcdclient.NewClient()
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	//-------------------------------------------
	// Lease Grant
	//-------------------------------------------
	fmt.Println("=== Lease Grant ===")

	lease, _ := cli.Grant(ctx, 3) // 3 sec TTL

	// Add lease to a key
	key := "/my-key"
	cli.Put(ctx, key, "value-1", clientv3.WithLease(lease.ID))

	fmt.Println("Get key:")
	res, _ := cli.Get(ctx, key)
	fmt.Printf("%s -> %s (Lease: %d)\n\n", key, res.Kvs[0].Value, lease.ID)

	// Check remaining time to expiry
	fmt.Println("...waiting for lease expiry")
	time.Sleep(1 * time.Second)

	ttlRes, _ := cli.TimeToLive(ctx, lease.ID, clientv3.WithAttachedKeys())
	fmt.Printf("Lease ID: %d, Granted TTL: %d, Remaining: %d\n", lease.ID, ttlRes.GrantedTTL, ttlRes.TTL)

	keys := make([]string, len(ttlRes.Keys))
	for i, b := range ttlRes.Keys {
		keys[i] = string(b)
	}
	fmt.Println("  Keys:", keys)
	time.Sleep(2 * time.Second)

	// Verify key expired
	fmt.Println("\nGet response after lease expire:")
	res, _ = cli.Get(ctx, key)
	pretty, _ := json.MarshalIndent(res, "", "  ")
	fmt.Printf("%s\n\n", pretty)

	//-------------------------------------------
	// Lease Revoke
	//-------------------------------------------
	fmt.Println("=== Lease Revoke ===")

	ctx2, cancel2 := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel2()

	lease, _ = cli.Grant(ctx2, 2)
	cli.Put(ctx2, key, "value-2", clientv3.WithLease(lease.ID))

	fmt.Println("Get key:")
	res, _ = cli.Get(ctx2, key)
	fmt.Printf("%s -> %s\n\n", key, res.Kvs[0].Value)

	fmt.Println("...revoking")
	cli.Revoke(ctx2, lease.ID)

	// Verify key revoked
	fmt.Println("\nGet response after lease revoke:")
	res, _ = cli.Get(ctx2, key)
	pretty, _ = json.MarshalIndent(res, "", "  ")
	fmt.Printf("%s\n\n", pretty)

	//-------------------------------------------
	// Lease keep-alive (forever)
	//-------------------------------------------
	fmt.Println("=== Lease keep-alive (forever) ===")

	ctx3, cancel3 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel3()

	lease, _ = cli.Grant(ctx3, 2)
	cli.Put(ctx3, key, "value-3", clientv3.WithLease(lease.ID))

	time.Sleep(1 * time.Second)
	cli.KeepAlive(ctx3, lease.ID)
	time.Sleep(2 * time.Second)

	fmt.Println("Get key:")
	res, _ = cli.Get(ctx3, key)
	fmt.Printf("%s -> %s\n\n", key, res.Kvs[0].Value)
}
