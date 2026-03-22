package grpc

import (
	"context"
	"fmt"
	"io"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// forwardGRPC transparently proxies a unary gRPC call to the upstream target.
// rawReq contains the already-received request bytes from the client stream.
// The upstream is reached over plain h2c (no TLS).
func forwardGRPC(stream grpc.ServerStream, target, fullMethod string, rawReq []byte) error {
	// Dial the upstream using the same raw-bytes codec so frames are relayed verbatim.
	conn, err := grpc.NewClient(target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.ForceCodec(encoding.GetCodec("proto"))),
	)
	if err != nil {
		return status.Errorf(codes.Unavailable, "dial upstream %q: %v", target, err)
	}
	defer conn.Close()

	// Propagate incoming metadata (headers) to the upstream.
	ctx := stream.Context()
	if md, ok := metadata.FromIncomingContext(ctx); ok {
		ctx = metadata.NewOutgoingContext(ctx, md)
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Open a client stream to the upstream using the same full method path.
	clientStream, err := conn.NewStream(ctx, &grpc.StreamDesc{
		ServerStreams: false,
		ClientStreams: false,
	}, fullMethod)
	if err != nil {
		return status.Errorf(codes.Unavailable, "open upstream stream: %v", err)
	}

	// Send the raw request bytes upstream (*[]byte → rawCodec passes through unchanged).
	if err := clientStream.SendMsg(&rawReq); err != nil {
		return status.Errorf(codes.Internal, "send to upstream: %v", err)
	}
	if err := clientStream.CloseSend(); err != nil {
		return status.Errorf(codes.Internal, "close send to upstream: %v", err)
	}

	// Relay the response back to the original client.
	var rawResp []byte
	if err := clientStream.RecvMsg(&rawResp); err != nil {
		if err == io.EOF {
			return nil
		}
		// Preserve the upstream gRPC status code if available.
		if st, ok := status.FromError(err); ok {
			return st.Err()
		}
		return status.Errorf(codes.Internal, "recv from upstream: %v", err)
	}

	// Forward any trailing metadata from upstream to the client.
	if trailer := clientStream.Trailer(); len(trailer) > 0 {
		stream.SetTrailer(trailer)
	}

	// Propagate response headers.
	if header, err := clientStream.Header(); err == nil && len(header) > 0 {
		if err := stream.SendHeader(header); err != nil {
			return fmt.Errorf("send header: %w", err)
		}
	}

	if err := stream.SendMsg(&rawResp); err != nil {
		return status.Errorf(codes.Internal, "send response to client: %v", err)
	}

	return nil
}
