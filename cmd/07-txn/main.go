package main

import (
	"context"
	"fmt"
	"time"

	"github.com/mabhi256/etcd-demo/internal/etcdclient"
	"go.etcd.io/etcd/api/v3/etcdserverpb"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {
	cli := etcdclient.NewClient()
	defer cli.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := "/my-key"
	//-------------------------------------------
	// Basic Txn
	//-------------------------------------------
	res1, _ := cli.Txn(ctx).
		Then(clientv3.OpPut(key, "value-1")). // A Txn with no If always succeeds
		Commit()

	if res1.Succeeded {
		fmt.Printf("Txn Success: %s -> value-1\n\n", key)
	} else {
		fmt.Printf("Txn Fail\n\n")
	}

	//-------------------------------------------
	// Txn + Condition (can be used for Optimistic Locking)
	//-------------------------------------------
	fmt.Printf("Txn succeeds if %s is xxxx\n", key)
	res2, _ := cli.Txn(ctx).
		If(clientv3.Compare(clientv3.Value(key), "=", "xxxx")).
		Then(clientv3.OpPut(key, "value-2")).
		Else(clientv3.OpGet(key)).
		Commit()

	if res2.Succeeded {
		fmt.Printf("Txn Success: %s updated\n\n", key)
	} else {
		getResp := res2.Responses[0].GetResponseRange()
		fmt.Printf("Txn Fail: Found %s -> %s\n\n", key, getResp.Kvs[0].Value)
	}

	//-------------------------------------------
	// Multiple ops in one transaction
	//-------------------------------------------
	fmt.Printf("Txn succeeds & Multiple Ops if %s is value-1\n", key)

	cli.Put(ctx, key+"/a", "AAA")
	res3, _ := cli.Txn(ctx).
		If(clientv3.Compare(clientv3.Value(key), "=", "value-1")).
		Then(
			clientv3.OpPut(key, "value-2"),
			clientv3.OpDelete(key+"/a"),
			clientv3.OpPut(key+"/b", "BBB"),
			clientv3.OpGet(key, clientv3.WithPrefix()),
		).
		Else(clientv3.OpGet(key)).
		Commit()

	if res3.Succeeded {
		// pretty, _ := json.MarshalIndent(res2, "", "  ")
		// fmt.Printf("Txn Success:\n%s\n\n", pretty)

		for _, r := range res3.Responses {
			switch v := r.Response.(type) {
			case *etcdserverpb.ResponseOp_ResponseRange:
				fmt.Println("GET range:")
				for _, kv := range v.ResponseRange.Kvs {
					fmt.Printf("  %s -> %s\n", kv.Key, kv.Value)
				}
			case *etcdserverpb.ResponseOp_ResponsePut:
				fmt.Printf("PUT revision: %d\n", v.ResponsePut.Header.Revision)
			case *etcdserverpb.ResponseOp_ResponseDeleteRange:
				fmt.Printf("DEL count: %d\n", v.ResponseDeleteRange.Deleted)
			}
		}
	} else {
		fmt.Printf("Txn Fail\n\n")
	}
}
