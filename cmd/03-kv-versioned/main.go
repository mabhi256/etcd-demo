package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mabhi256/etcd-demo/internal/etcdclient"
	clientv3 "go.etcd.io/etcd/client/v3"
)

// Demonstrates etcd's MVCC model: every write bumps the cluster revision,
// and each key tracks its own version, createRevision, and modRevision.
func main() {
	cli := etcdclient.NewClient()
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key1 := "/config/app/version"
	key2 := "/my-key"

	// Initial writes. createRev == modRev for both keys.
	cli.Put(ctx, key1, "1.1.0")
	cli.Put(ctx, key2, "my-value-1")
	curr, _ := cli.Get(ctx, key1)
	kv := curr.Kvs[0]
	rev1, rev2 := kv.CreateRevision, kv.ModRevision
	fmt.Printf("%s -> %s (version:%d createRev:%d modRev:%d, clusterRev:%d)\n", kv.Key, kv.Value, kv.Version, rev1, rev2, curr.Header.Revision)

	// Update key2 only. key2's modRev advances, createRev stays the same.
	cli.Put(ctx, key2, "my-value-2")
	curr, _ = cli.Get(ctx, key2)
	kv = curr.Kvs[0]
	rev3, rev4 := kv.CreateRevision, kv.ModRevision
	fmt.Printf("%s -> %s (version:%d createRev:%d modRev:%d, clusterRev:%d)\n", kv.Key, kv.Value, kv.Version, rev3, rev4, curr.Header.Revision)

	// Update both keys again. each Put increments the cluster revision.
	cli.Put(ctx, key1, "2.0.0")
	cli.Put(ctx, key2, "my-value-3")
	curr, _ = cli.Get(ctx, key1)
	kv = curr.Kvs[0]
	rev5, rev6 := kv.CreateRevision, kv.ModRevision
	fmt.Printf("%s -> %s (version:%d createRev:%d modRev:%d, clusterRev:%d)\n", kv.Key, kv.Value, kv.Version, rev5, rev6, curr.Header.Revision)

	// Point-in-time reads: WithRev returns the key's state at a past cluster revision.
	fmt.Println()
	revisions := []int64{rev1, rev2, rev5, rev6}
	for _, rev := range revisions {
		resp, _ := cli.Get(ctx, key1, clientv3.WithRev(rev))
		kv = resp.Kvs[0]
		fmt.Printf("%s -> %s (Rev:%d, version:%d)\n", kv.Key, kv.Value, rev, kv.Version)
	}
}
