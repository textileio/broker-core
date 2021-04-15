package util

import (
	"context"
	"crypto/tls"
	"log"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Flag describes a configuration flag.
type Flag struct {
	Name        string
	DefValue    interface{}
	Description string
}

// ConfigureCLI configures a Viper environment with flags and envs.
func ConfigureCLI(v *viper.Viper, envPrefix string, flags []Flag, rootCmd *cobra.Command) {
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	for _, flag := range flags {
		switch defval := flag.DefValue.(type) {
		case string:
			rootCmd.Flags().String(flag.Name, defval, flag.Description)
			v.SetDefault(flag.Name, defval)
		case bool:
			rootCmd.Flags().Bool(flag.Name, defval, flag.Description)
			v.SetDefault(flag.Name, defval)
		default:
			log.Fatalf("unknown flag type: %T", flag)
		}
		if err := v.BindPFlag(flag.Name, rootCmd.Flags().Lookup(flag.Name)); err != nil {
			log.Fatalf("binding flag %s: %s", flag.Name, err)
		}
	}
}

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

// GetRequestMetadata returns a map with all the metadata present in the RPC call.
func (c RPCCredentials) GetRequestMetadata(ctx context.Context, _ ...string) (map[string]string, error) {
	md := map[string]string{}
	// // token, ok := TokenFromContext(ctx)
	// if ok {
	// 	md["authorization"] = "bearer " + string(token)
	// }
	return md, nil
}

// RequireTransportSecurity returns if security is enabled.
func (c RPCCredentials) RequireTransportSecurity() bool {
	return c.Secure
}
