package rpc

import (
	"context"
	"crypto/tls"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// GetClientOpts dial options for target.
func GetClientOpts(target string) (opts []grpc.DialOption) {
	creds := Credentials{}
	if strings.Contains(target, "443") {
		tcreds := credentials.NewTLS(&tls.Config{})
		opts = append(opts, grpc.WithTransportCredentials(tcreds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}
	opts = append(opts, grpc.WithPerRPCCredentials(creds))
	return opts
}

// Credentials implements PerRPCCredentials, allowing context values
// to be included in request metadata.
type Credentials struct {
	Secure bool
}

// GetRequestMetadata gets the current request metadata.
func (c Credentials) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	md := map[string]string{}
	// // token, ok := TokenFromContext(ctx)
	// if ok {
	// 	md["authorization"] = "bearer " + string(token)
	// }
	return md, nil
}

// RequireTransportSecurity indicates whether the credentials requires
// transport security.
func (c Credentials) RequireTransportSecurity() bool {
	return c.Secure
}

// StopServer attempts to gracefully stop the server without permanently blocking.
func StopServer(s *grpc.Server) {
	stopped := make(chan struct{})
	go func() {
		s.GracefulStop()
		close(stopped)
	}()
	timer := time.NewTimer(10 * time.Second)
	select {
	case <-timer.C:
		s.Stop()
	case <-stopped:
		timer.Stop()
	}
}
