package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/mabhi256/etcd-demo/internal/etcdclient"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

func main() {
	cli := etcdclient.NewClient()
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var wg sync.WaitGroup

	// -------------------------------------------
	// Optimistic Lock (Txn + ModRev)
	// Use when: Contention is rare, short critical sections, you can retry
	// -------------------------------------------
	fmt.Println("=== Optimistic Lock ===")

	key := "/config"
	res, _ := cli.Put(ctx, key, "a")
	currRev := res.Header.Revision
	// res1, _ := cli.Get(ctx, key)
	// fmt.Printf("currRev(%d) == modRev(%d): %v\n", currRev, res1.Kvs[0].ModRevision, currRev ==

	for i := 1; i <= 3; i++ {
		wg.Go(func() {
			res2, _ := cli.Txn(ctx).
				If(clientv3.Compare(clientv3.ModRevision(key), "=", currRev)).
				Then(clientv3.OpPut(key, "b")).
				Else(). // If we don't include empty Else, other txns will block
				Commit()

			if res2.Succeeded {
				fmt.Printf("[Worker-%d] Txn Success\n", i)
			} else {
				fmt.Printf("[Worker-%d] Txn Fail\n", i)
			}
		})
	}

	wg.Wait()

	// -------------------------------------------
	// Pessimistic Lock (concurrency.Mutex)
	// Use when: Contention is expected, long operations, can't retry easily
	// -------------------------------------------
	fmt.Println("\n=== Pessimistic Lock ===")

	lockKey := "/my-lock"

	for i := 1; i <= 3; i++ {
		wg.Go(func() {
			// Each goroutine gets its own session (backed by its own lease)
			s, _ := concurrency.NewSession(cli, concurrency.WithTTL(10))
			defer s.Close()

			mu := concurrency.NewMutex(s, lockKey)

			fmt.Printf("[Worker-%d] waiting to acquire lock...\n", i)
			mu.Lock(context.Background())
			fmt.Printf("[Worker-%d] lock acquired\n", i)

			// Simulate critical section
			time.Sleep(1 * time.Second)
			fmt.Printf("[Worker-%d] releasing lock\n", i)

			mu.Unlock(context.Background())
		})
	}

	wg.Wait()
}
