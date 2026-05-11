//go:build as_proxy

package aerospike

import (
	"math/rand"

	kvs "github.com/aerospike/aerospike-client-go/v7/proto/kvs"
)

func (cmd *readCommand) ExecuteGRPC(clnt *ProxyClient) Error {
	defer cmd.grpcPutBufferBack()

	err := cmd.prepareBuffer(cmd, cmd.policy.deadline())
	if err != nil {
		return err
	}

	req := kvs.AerospikeRequestPayload{
		Id:         rand.Uint32(),
		Iteration:  1,
		Payload:    cmd.dataBuffer[:cmd.dataOffset],
		ReadPolicy: cmd.policy.grpc(),
	}

	conn, err := clnt.grpcConn()
	if err != nil {
		return err
	}

	client := kvs.NewKVSClient(conn)

	ctx, cancel := cmd.policy.grpcDeadlineContext()
	defer cancel()

	res, gerr := client.Read(ctx, &req)
	if gerr != nil {
		return newGrpcError(!cmd.isRead(), gerr, gerr.Error())
	}

	cmd.commandWasSent = true

	defer clnt.returnGrpcConnToPool(conn)

	if res.GetStatus() != 0 {
		return newGrpcStatusError(res)
	}

	cmd.conn = newGrpcFakeConnection(res.GetPayload(), nil)
	err = cmd.parseResult(cmd, cmd.conn)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *batchCommandOperate) ExecuteGRPC(clnt *ProxyClient) Error {
	defer cmd.grpcPutBufferBack()

	err := cmd.prepareBuffer(cmd, cmd.policy.deadline())
	if err != nil {
		return err
	}

	req := kvs.AerospikeRequestPayload{
		Id:          rand.Uint32(),
		Iteration:   1,
		Payload:     cmd.dataBuffer[:cmd.dataOffset],
		ReadPolicy:  cmd.policy.grpc(),
		WritePolicy: cmd.policy.grpc_write(),
	}

	conn, err := clnt.grpcConn()
	if err != nil {
		return err
	}

	client := kvs.NewKVSClient(conn)

	ctx, cancel := cmd.policy.grpcDeadlineContext()
	defer cancel()

	streamRes, gerr := client.BatchOperate(ctx, &req)
	if gerr != nil {
		return newGrpcError(!cmd.isRead(), gerr, gerr.Error())
	}

	cmd.commandWasSent = true

	readCallback := func() ([]byte, Error) {
		if cmd.grpcEOS {
			return nil, errGRPCStreamEnd
		}

		res, gerr := streamRes.Recv()
		if gerr != nil {
			e := newGrpcError(!cmd.isRead(), gerr)
			return nil, e
		}

		if res.GetStatus() != 0 {
			e := newGrpcStatusError(res)
			return res.GetPayload(), e
		}

		cmd.grpcEOS = !res.GetHasNext()

		return res.GetPayload(), nil
	}

	cmd.conn = newGrpcFakeConnection(nil, readCallback)
	err = cmd.parseResult(cmd, cmd.conn)
	if err != nil && err != errGRPCStreamEnd {
		return err
	}

	clnt.returnGrpcConnToPool(conn)

	return nil
}

func (cmd *deleteCommand) ExecuteGRPC(clnt *ProxyClient) Error {
	defer cmd.grpcPutBufferBack()

	err := cmd.prepareBuffer(cmd, cmd.policy.deadline())
	if err != nil {
		return err
	}

	req := kvs.AerospikeRequestPayload{
		Id:          rand.Uint32(),
		Iteration:   1,
		Payload:     cmd.dataBuffer[:cmd.dataOffset],
		WritePolicy: cmd.policy.grpc(),
	}

	conn, err := clnt.grpcConn()
	if err != nil {
		return err
	}

	client := kvs.NewKVSClient(conn)

	ctx, cancel := cmd.policy.grpcDeadlineContext()
	defer cancel()

	res, gerr := client.Delete(ctx, &req)
	if gerr != nil {
		return newGrpcError(!cmd.isRead(), gerr, gerr.Error())
	}

	cmd.commandWasSent = true

	defer clnt.returnGrpcConnToPool(conn)

	if res.GetStatus() != 0 {
		return newGrpcStatusError(res)
	}

	cmd.conn = newGrpcFakeConnection(res.GetPayload(), nil)
	err = cmd.parseResult(cmd, cmd.conn)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *executeCommand) ExecuteGRPC(clnt *ProxyClient) Error {
	defer cmd.grpcPutBufferBack()

	err := cmd.prepareBuffer(cmd, cmd.policy.deadline())
	if err != nil {
		return err
	}

	req := kvs.AerospikeRequestPayload{
		Id:          rand.Uint32(),
		Iteration:   1,
		Payload:     cmd.dataBuffer[:cmd.dataOffset],
		WritePolicy: cmd.policy.grpc(),
	}

	conn, err := clnt.grpcConn()
	if err != nil {
		return err
	}

	client := kvs.NewKVSClient(conn)

	ctx, cancel := cmd.policy.grpcDeadlineContext()
	defer cancel()

	res, gerr := client.Execute(ctx, &req)
	if gerr != nil {
		return newGrpcError(!cmd.isRead(), gerr, gerr.Error())
	}

	cmd.commandWasSent = true

	defer clnt.returnGrpcConnToPool(conn)

	if res.GetStatus() != 0 {
		return newGrpcStatusError(res)
	}

	cmd.conn = newGrpcFakeConnection(res.GetPayload(), nil)
	err = cmd.parseResult(cmd, cmd.conn)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *existsCommand) ExecuteGRPC(clnt *ProxyClient) Error {
	defer cmd.grpcPutBufferBack()

	err := cmd.prepareBuffer(cmd, cmd.policy.deadline())
	if err != nil {
		return err
	}

	req := kvs.AerospikeRequestPayload{
		Id:         rand.Uint32(),
		Iteration:  1,
		Payload:    cmd.dataBuffer[:cmd.dataOffset],
		ReadPolicy: cmd.policy.grpc(),
	}

	conn, err := clnt.grpcConn()
	if err != nil {
		return err
	}

	client := kvs.NewKVSClient(conn)

	ctx, cancel := cmd.policy.grpcDeadlineContext()
	defer cancel()

	res, gerr := client.Exists(ctx, &req)
	if gerr != nil {
		return newGrpcError(!cmd.isRead(), gerr, gerr.Error())
	}

	cmd.commandWasSent = true

	defer clnt.returnGrpcConnToPool(conn)

	if res.GetStatus() != 0 {
		return newGrpcStatusError(res)
	}

	cmd.conn = newGrpcFakeConnection(res.GetPayload(), nil)
	err = cmd.parseResult(cmd, cmd.conn)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *operateCommand) ExecuteGRPC(clnt *ProxyClient) Error {
	defer cmd.grpcPutBufferBack()

	err := cmd.prepareBuffer(cmd, cmd.policy.deadline())
	if err != nil {
		return err
	}

	req := kvs.AerospikeRequestPayload{
		Id:          rand.Uint32(),
		Iteration:   1,
		Payload:     cmd.dataBuffer[:cmd.dataOffset],
		WritePolicy: cmd.policy.grpc(),
	}

	conn, err := clnt.grpcConn()
	if err != nil {
		return err
	}

	client := kvs.NewKVSClient(conn)

	ctx, cancel := cmd.policy.grpcDeadlineContext()
	defer cancel()

	res, gerr := client.Operate(ctx, &req)
	if gerr != nil {
		return newGrpcError(!cmd.isRead(), gerr, gerr.Error())
	}

	cmd.commandWasSent = true

	defer clnt.returnGrpcConnToPool(conn)

	if res.GetStatus() != 0 {
		return newGrpcStatusError(res)
	}

	cmd.conn = newGrpcFakeConnection(res.GetPayload(), nil)
	err = cmd.parseResult(cmd, cmd.conn)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *readHeaderCommand) ExecuteGRPC(clnt *ProxyClient) Error {
	defer cmd.grpcPutBufferBack()

	err := cmd.prepareBuffer(cmd, cmd.policy.deadline())
	if err != nil {
		return err
	}

	req := kvs.AerospikeRequestPayload{
		Id:         rand.Uint32(),
		Iteration:  1,
		Payload:    cmd.dataBuffer[:cmd.dataOffset],
		ReadPolicy: cmd.policy.grpc(),
	}

	conn, err := clnt.grpcConn()
	if err != nil {
		return err
	}

	client := kvs.NewKVSClient(conn)

	ctx, cancel := cmd.policy.grpcDeadlineContext()
	defer cancel()

	res, gerr := client.GetHeader(ctx, &req)
	if gerr != nil {
		return newGrpcError(!cmd.isRead(), gerr, gerr.Error())
	}

	cmd.commandWasSent = true

	defer clnt.returnGrpcConnToPool(conn)

	if res.GetStatus() != 0 {
		return newGrpcStatusError(res)
	}

	cmd.conn = newGrpcFakeConnection(res.GetPayload(), nil)
	err = cmd.parseResult(cmd, cmd.conn)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *serverCommand) ExecuteGRPC(clnt *ProxyClient) Error {
	defer cmd.grpcPutBufferBack()

	err := cmd.prepareBuffer(cmd, cmd.policy.deadline())
	if err != nil {
		return err
	}

	execReq := &kvs.BackgroundExecuteRequest{
		Statement:   cmd.statement.grpc(cmd.policy, cmd.operations),
		WritePolicy: cmd.writePolicy.grpc_exec(cmd.policy.FilterExpression),
	}

	req := kvs.AerospikeRequestPayload{
		Id:                       rand.Uint32(),
		Iteration:                1,
		Payload:                  cmd.dataBuffer[:cmd.dataOffset],
		BackgroundExecuteRequest: execReq,
	}

	conn, err := clnt.grpcConn()
	if err != nil {
		return err
	}

	client := kvs.NewQueryClient(conn)

	ctx, cancel := cmd.policy.grpcDeadlineContext()
	defer cancel()

	streamRes, gerr := client.BackgroundExecute(ctx, &req)
	if gerr != nil {
		return newGrpcError(!cmd.isRead(), gerr, gerr.Error())
	}

	cmd.commandWasSent = true

	readCallback := func() ([]byte, Error) {
		res, gerr := streamRes.Recv()
		if gerr != nil {
			e := newGrpcError(!cmd.isRead(), gerr)
			return nil, e
		}

		if res.GetStatus() != 0 {
			e := newGrpcStatusError(res)
			return res.GetPayload(), e
		}

		if !res.GetHasNext() {
			return nil, errGRPCStreamEnd
		}

		return res.GetPayload(), nil
	}

	cmd.conn = newGrpcFakeConnection(nil, readCallback)
	err = cmd.parseResult(cmd, cmd.conn)
	if err != nil && err != errGRPCStreamEnd {
		return err
	}

	clnt.returnGrpcConnToPool(conn)

	return nil
}

func (cmd *touchCommand) ExecuteGRPC(clnt *ProxyClient) Error {
	defer cmd.grpcPutBufferBack()

	err := cmd.prepareBuffer(cmd, cmd.policy.deadline())
	if err != nil {
		return err
	}

	req := kvs.AerospikeRequestPayload{
		Id:          rand.Uint32(),
		Iteration:   1,
		Payload:     cmd.dataBuffer[:cmd.dataOffset],
		WritePolicy: cmd.policy.grpc(),
	}

	conn, err := clnt.grpcConn()
	if err != nil {
		return err
	}

	client := kvs.NewKVSClient(conn)

	ctx, cancel := cmd.policy.grpcDeadlineContext()
	defer cancel()

	res, gerr := client.Touch(ctx, &req)
	if gerr != nil {
		return newGrpcError(!cmd.isRead(), gerr, gerr.Error())
	}

	cmd.commandWasSent = true

	defer clnt.returnGrpcConnToPool(conn)

	if res.GetStatus() != 0 {
		return newGrpcStatusError(res)
	}

	cmd.conn = newGrpcFakeConnection(res.GetPayload(), nil)
	err = cmd.parseResult(cmd, cmd.conn)
	if err != nil {
		return err
	}

	return nil
}

func (cmd *writeCommand) ExecuteGRPC(clnt *ProxyClient) Error {
	defer cmd.grpcPutBufferBack()

	err := cmd.prepareBuffer(cmd, cmd.policy.deadline())
	if err != nil {
		return err
	}

	req := kvs.AerospikeRequestPayload{
		Id:          rand.Uint32(),
		Iteration:   1,
		Payload:     cmd.dataBuffer[:cmd.dataOffset],
		WritePolicy: cmd.policy.grpc(),
	}

	conn, err := clnt.grpcConn()
	if err != nil {
		return err
	}

	client := kvs.NewKVSClient(conn)

	ctx, cancel := cmd.policy.grpcDeadlineContext()
	defer cancel()

	res, gerr := client.Write(ctx, &req)
	if gerr != nil {
		return newGrpcError(!cmd.isRead(), gerr, gerr.Error())
	}

	cmd.commandWasSent = true

	defer clnt.returnGrpcConnToPool(conn)

	if res.GetStatus() != 0 {
		return newGrpcStatusError(res)
	}

	cmd.conn = newGrpcFakeConnection(res.GetPayload(), nil)
	err = cmd.parseResult(cmd, cmd.conn)
	if err != nil {
		return err
	}

	return nil
}
