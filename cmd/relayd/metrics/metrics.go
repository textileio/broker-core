package metrics

import (
	"context"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/multiformats/go-multiaddr"
	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

const prefix = "relayd"

var (
	log   = logger.Logger("metrics")
	meter = metric.Must(global.Meter(prefix))
)

type hostNotifee struct {
	activeConns   metric.Int64UpDownCounter
	activeStreams metric.Int64UpDownCounter
}

// Register exports libp2p connection metrics about the host.
func Register(h host.Host) {
	cn := hostNotifee{
		activeConns:   meter.NewInt64UpDownCounter(prefix + ".active_conns"),
		activeStreams: meter.NewInt64UpDownCounter(prefix + ".active_streams"),
	}
	h.Network().Notify(&cn)
}

func (n *hostNotifee) Connected(_ network.Network, ne network.Conn) {
	n.activeConns.Add(context.Background(), 1)
	log.Debugf("%s connected", ne.RemotePeer())
}
func (n *hostNotifee) Disconnected(_ network.Network, ne network.Conn) {
	n.activeConns.Add(context.Background(), -1)
	log.Debugf("%s disconnected", ne.RemotePeer())
}
func (n *hostNotifee) OpenedStream(_ network.Network, s network.Stream) {
	n.activeStreams.Add(context.Background(), 1)
	log.Debugf("%s opened stream %s (%s)", s.Conn().RemotePeer(), s.ID(), s.Protocol())
}
func (n *hostNotifee) ClosedStream(_ network.Network, s network.Stream) {
	n.activeStreams.Add(context.Background(), -1)
	log.Debugf("%s closed stream %s (%s)", s.Conn().RemotePeer(), s.ID(), s.Protocol())
}
func (n *hostNotifee) ListenClose(_ network.Network, _ multiaddr.Multiaddr) {}
func (n *hostNotifee) Listen(net network.Network, ma multiaddr.Multiaddr)   {}
