package main

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mabhi256/etcd-demo/internal/etcdclient"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	cli := etcdclient.NewClient()
	defer cli.Close()

	// cacheRev tracks the highest etcd revision the watcher has processed.
	var cacheRev atomic.Int64
	prefix := "/config/"

	watchCtx, watchCancel := context.WithCancel(context.Background())
	defer watchCancel()

	// WithProgressNotify makes etcd send periodic heartbeats even when no keys
	// change, so cacheRev advances even during quiet periods.
	watchChan := cli.Watch(watchCtx, prefix,
		clientv3.WithPrefix(),
		clientv3.WithProgressNotify(),
	)

	pause := make(chan struct{})
	close(pause) // start unpaused so <-ch won't block
	var pauseMu sync.Mutex

	go func() {
		for resp := range watchChan {
			// Block here while paused; unblocks when pause is closed again.
			pauseMu.Lock()
			ch := pause
			pauseMu.Unlock()
			<-ch

			if resp.IsProgressNotify() {
				// etcd only sends progress notifications on a timer (every 10m by default).
				// RequestProgress short-circuits this process. The server responds
				// immediately with the current cluster revision in the header.
				cacheRev.Store(resp.Header.Revision)
				fmt.Printf("[watch] progress rev=%d\n", resp.Header.Revision)
				continue
			}
			for _, ev := range resp.Events {
				cacheRev.Store(ev.Kv.ModRevision)
				fmt.Printf("[watch] %s %s=%s rev=%d\n", ev.Type, ev.Kv.Key, ev.Kv.Value, ev.Kv.ModRevision)
			}
		}
	}()

	// check compares cacheRev against a linearizable (quorum) read to detect staleness.
	check := func(label string) {
		// RequestProgress triggers an immediate progress notify so cacheRev is
		// as fresh as possible before we compare it to the quorum revision.
		cli.RequestProgress(watchCtx)
		time.Sleep(50 * time.Millisecond)

		resp, _ := cli.Get(context.Background(), prefix, clientv3.WithPrefix())
		quorum, cache := resp.Header.Revision, cacheRev.Load()
		fmt.Printf("=== %s ===\n  quorum=%d  cache=%d", label, quorum, cache)
		if cache >= quorum {
			fmt.Println("  [up-to-date]")
		} else {
			fmt.Printf("  [STALE by %d]\n", quorum-cache)
		}
	}

	ctx := context.Background()
	cli.Put(ctx, "/config/host", "10.0.0.1")
	cli.Put(ctx, "/config/port", "5432")
	time.Sleep(100 * time.Millisecond)
	check("baseline")

	// In production, a remote cache lags naturally due to network latency.
	// Since everything is local here, we block the watcher goroutine to
	// artificially create the same stale-cache condition.
	fmt.Println("\n...pausing watcher")
	pauseMu.Lock()
	pause = make(chan struct{}) // unclosed channel blocks the consumer goroutine
	pauseMu.Unlock()

	cli.Put(ctx, "/config/host", "10.0.0.2")
	cli.Put(ctx, "/config/timeout", "30s")
	fmt.Println("...wrote 2 keys while paused")
	check("while paused") // cache should lag behind quorum here

	// Thaw the watcher; it drains the buffered events and catches up.
	fmt.Println("\n...resuming watcher")
	pauseMu.Lock()
	close(pause)
	pauseMu.Unlock()

	time.Sleep(100 * time.Millisecond)
	check("after resume") // cache should be caught up now
}
