//go:build as_proxy

// Copyright 2014-2022 Aerospike, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aerospike

import (
	"context"
	"math/rand"
	"time"

	kvs "github.com/aerospike/aerospike-client-go/v7/proto/kvs"
)

// newGRPCExecuteTask initializes task with fields needed to query server nodes.
func newGRPCExecuteTask(clnt *ProxyClient, statement *Statement) *ExecuteTask {
	return &ExecuteTask{
		baseTask: newTask(nil),
		taskID:   statement.TaskId,
		scan:     statement.IsScan(),
		clnt:     clnt,
	}
}

func (etsk *ExecuteTask) grpcIsDone() (bool, Error) {
	statusReq := &kvs.BackgroundTaskStatusRequest{
		TaskId: int64(etsk.taskID),
		IsScan: etsk.scan,
	}

	req := kvs.AerospikeRequestPayload{
		Id:                          rand.Uint32(),
		Iteration:                   1,
		BackgroundTaskStatusRequest: statusReq,
	}

	clnt := etsk.clnt.(*ProxyClient)
	conn, err := clnt.grpcConn()
	if err != nil {
		return false, err
	}

	client := kvs.NewQueryClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), NewInfoPolicy().Timeout)
	defer cancel()

	streamRes, gerr := client.BackgroundTaskStatus(ctx, &req)
	if gerr != nil {
		return false, newGrpcError(true, gerr, gerr.Error())
	}

	for {
		time.Sleep(time.Second)

		res, gerr := streamRes.Recv()
		if gerr != nil {
			e := newGrpcError(true, gerr)
			return false, e
		}

		if res.GetStatus() != 0 {
			e := newGrpcStatusError(res)
			clnt.returnGrpcConnToPool(conn)
			return false, e
		}

		switch res.GetBackgroundTaskStatus() {
		case kvs.BackgroundTaskStatus_COMPLETE:
			clnt.returnGrpcConnToPool(conn)
			return true, nil
		default:
			clnt.returnGrpcConnToPool(conn)
			return false, nil
		}
	}
}
