package common

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"strings"

	"github.com/textileio/broker-core/auth"
	pb "github.com/textileio/broker-core/auth/pb/auth"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// all the common stuff for tests (used by main.go as well)

// RPCCredentials implements PerRPCCredentials, allowing context values
// to be included in request metadata.
type RPCCredentials struct {
	Secure bool
}

func (c RPCCredentials) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	md := map[string]string{}
	// // token, ok := TokenFromContext(ctx)
	// if ok {
	// 	md["authorization"] = "bearer " + string(token)
	// }
	return md, nil
}
func (c RPCCredentials) RequireTransportSecurity() bool {
	return c.Secure
}

func GetServerAndProxy(listenAddr, listenAddrProxy string) (*grpc.Server, error) {
	server := grpc.NewServer()
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, err
	}
	go func() {
		pb.RegisterAPIServiceServer(server, auth.NewService())
		if err := server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			panic(fmt.Errorf("server error: %v", err))
		}
	}()
	return server, nil
}

func GetClientRPCOpts(target string) (opts []grpc.DialOption) {
	creds := RPCCredentials{}
	if strings.Contains(target, "443") {
		tcreds := credentials.NewTLS(&tls.Config{})
		opts = append(opts, grpc.WithTransportCredentials(tcreds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	opts = append(opts, grpc.WithPerRPCCredentials(creds))
	return opts
}
