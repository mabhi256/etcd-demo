package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/mabhi256/etcd-demo/internal/etcdclient"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.etcd.io/etcd/client/v3/concurrency"
)

// How leader election works in etcd:
//
//  1. Each candidate creates a Session (backed by a lease).
//  2. Candidates call Campaign(); this writes a key under the election
//     prefix tied to their lease. The candidate with the lowest create
//     revision wins; the others watch that key and wait.
//  3. When the leader Resigns or its lease expires (crash / disconnect),
//     the key is deleted and the next-lowest-revision candidate wins.
//
// Simulating members  → goroutines, each with its own Session + Election.
// Simulating failure  → Resign() for graceful handoff;
//                       or skip Resign and let the lease TTL expire
//                       to simulate a crash (slow), or call
//                       cli.Revoke(ctx, s.Lease()) to expire it instantly.

const (
	electionPrefix = "/demo-election"
	sessionTTL     = 3
)

type Winner struct {
	session  *concurrency.Session
	election *concurrency.Election
	cancelKA context.CancelFunc
}

func (w *Winner) currentLeader(ctx context.Context) string {
	return string((<-w.election.Observe(ctx)).Kvs[0].Value)
}

// member campaigns until elected, then sends itself on the channel.
func member(ctx context.Context, cli *clientv3.Client, id int, ch chan<- Winner) {
	keepaliveCtx, cancelKA := context.WithCancel(context.Background())
	session, err := concurrency.NewSession(cli,
		concurrency.WithTTL(sessionTTL),
		concurrency.WithContext(keepaliveCtx),
	)
	if err != nil {
		cancelKA()
		log.Printf("Member-%d session error: %v", id, err)
		return
	}

	election := concurrency.NewElection(session, electionPrefix)
	fmt.Printf("[Member-%d] campaigning...\n", id)

	// Campaign blocks until this node becomes the leader.
	// The value is stored in etcd so observers can see who is leading.
	if err := election.Campaign(ctx, fmt.Sprintf("Member-%d", id)); err != nil {
		cancelKA()
		session.Close()
		return
	}
	ch <- Winner{session, election, cancelKA}
}

func checkLease(cli *clientv3.Client, id clientv3.LeaseID, label string) {
	resp, err := cli.TimeToLive(context.Background(), id)
	if err != nil {
		fmt.Printf("  [lease-%d] %s: error: %v\n", id, label, err)
		return
	}
	if resp.TTL == -1 {
		fmt.Printf("  [lease-%d] %s: TTL=-1 (EXPIRED)\n", id, label)
	} else {
		fmt.Printf("  [lease-%d] %s: TTL=%ds (ALIVE)\n", id, label, resp.TTL)
	}
}

func main() {
	cli := etcdclient.NewClient()
	defer cli.Close()

	cli.Delete(context.Background(), electionPrefix, clientv3.WithPrefix())

	programCtx, cancelProgram := context.WithCancel(context.Background())
	defer cancelProgram()

	ch := make(chan Winner, 1)
	for i := range 5 {
		go member(programCtx, cli, i, ch)
	}

	obsCtx, cancelObs := context.WithCancel(context.Background())
	defer cancelObs()

	next := func() Winner {
		w := <-ch
		fmt.Println("Leader:", w.currentLeader(obsCtx))
		time.Sleep(time.Second)
		return w
	}

	// Election 1: crash - stop heartbeat, wait for TTL expiry.
	w := next()
	fmt.Println("...crashing leader, wait for TTL")
	w.cancelKA()

	// Election 2: session revoke.
	// Forcibly expires the lease, which deletes all keys tied to the lease.
	// Chooses the next leader immediately (next lowest create revision).
	w = next()
	fmt.Println("...revoking leader")
	lease := w.session.Lease()
	cli.Revoke(context.Background(), lease)
	checkLease(cli, lease, "after Revoke()")

	// Election 3: graceful resign.
	// Only deletes the leader's key from etcd. Underlying lease and session continue to exist.
	// Used when a leader intentionally wants to hand off (rolling restart, planned maintenance).
	w = next()
	fmt.Println("...resigning leader")
	lease = w.session.Lease()
	w.election.Resign(context.Background())
	checkLease(cli, lease, "after Resign()")

	// Election 4.
	next()
}
