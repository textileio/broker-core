package rpc

import (
	"context"
	"crypto/tls"
	"strings"

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

func (c Credentials) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	md := map[string]string{}
	// // token, ok := TokenFromContext(ctx)
	// if ok {
	// 	md["authorization"] = "bearer " + string(token)
	// }
	return md, nil
}
func (c Credentials) RequireTransportSecurity() bool {
	return c.Secure
}
