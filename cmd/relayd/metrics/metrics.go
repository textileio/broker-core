package metrics

import (
	"context"
	"sync/atomic"

	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/cmd/brokerd/metrics"
	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

const prefix = "relayd"

var (
	log   = logger.Logger("metrics")
	meter = metric.Must(global.Meter(prefix))
)

type hostNotifee struct {
	activeConns   int64
	activeStreams int64
}

func Register(h host.Host) {
	var mActiveConns metric.Int64ValueObserver
	var mActiveStreams metric.Int64ValueObserver

	var cn hostNotifee
	batchObs := metrics.Meter.NewBatchObserver(func(ctx context.Context, result metric.BatchObserverResult) {
		result.Observe(
			[]attribute.KeyValue{},
			mActiveConns.Observation(atomic.LoadInt64(&cn.activeConns)),
			mActiveStreams.Observation(atomic.LoadInt64(&cn.activeStreams)),
		)
	})
	h.Network().Notify(&cn)
	mActiveConns = batchObs.NewInt64ValueObserver(prefix + ".active_conns")
	mActiveStreams = batchObs.NewInt64ValueObserver(prefix + ".active_streams")
}

func (n *hostNotifee) Connected(_ network.Network, ne network.Conn) {
	atomic.AddInt64(&n.activeConns, 1)
	log.Debugf("%s connected", ne.RemotePeer())
}
func (n *hostNotifee) Disconnected(_ network.Network, ne network.Conn) {
	atomic.AddInt64(&n.activeConns, -1)
	log.Debugf("%s disconnected", ne.RemotePeer())
}
func (n *hostNotifee) OpenedStream(_ network.Network, s network.Stream) {
	atomic.AddInt64(&n.activeStreams, 1)
	log.Debugf("%s opened stream %s (%s)", s.Conn().RemotePeer(), s.ID(), s.Protocol())
}
func (n *hostNotifee) ClosedStream(_ network.Network, s network.Stream) {
	atomic.AddInt64(&n.activeStreams, -1)
	log.Debugf("%s closed stream %s (%s)", s.Conn().RemotePeer(), s.ID(), s.Protocol())
}
func (n *hostNotifee) ListenClose(_ network.Network, _ multiaddr.Multiaddr) {}
func (n *hostNotifee) Listen(net network.Network, ma multiaddr.Multiaddr)   {}
