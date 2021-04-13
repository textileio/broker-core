package util

import (
	"context"
	"crypto/tls"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// SetLogLevels sets levels for the given systems.
func SetLogLevels(systems map[string]logging.LogLevel) error {
	for sys, level := range systems {
		l := zapcore.Level(level)
		if sys == "*" {
			for _, s := range logging.GetSubsystems() {
				if err := logging.SetLogLevel(s, l.CapitalString()); err != nil {
					return err
				}
			}
		}
		if err := logging.SetLogLevel(sys, l.CapitalString()); err != nil {
			return err
		}
	}
	return nil
}

// GetClientRPCOpts dial options for target.
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
