package metrics

import "github.com/prometheus/client_golang/prometheus"

type Metrics struct {
	EventsTotal prometheus.Counter
	EventsDrop  *prometheus.CounterVec
	TopReads    prometheus.Counter
}

func New(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		EventsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wb_search_events_total",
			Help: "Total search events accepted by the service.",
		}),
		EventsDrop: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "wb_search_events_dropped_total",
			Help: "Total search events dropped by reason.",
		}, []string{"reason"}),
		TopReads: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "wb_search_top_reads_total",
			Help: "Total top read requests.",
		}),
	}

	reg.MustRegister(m.EventsTotal, m.EventsDrop, m.TopReads)
	return m
}
