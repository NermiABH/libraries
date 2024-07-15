package inform

import (
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"strconv"
	"time"
)

// ERROR

type ErrLevel string

type ErrMetrics struct {
	gv *prometheus.GaugeVec
}

var (
	WARN           ErrLevel = "WARN"
	CRIT           ErrLevel = "CRIT"
	PANIC          ErrLevel = "PANIC"
	FATAL          ErrLevel = "FATAL"
	generalMetrics *Metrics
)

func errMetricsInit(namespace, subsystem string) (*ErrMetrics, error) {
	gv := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "wbkeepers_errors",
			Help:      "errors",
		}, []string{"level"})

	if err := prometheus.Register(gv); err != nil {
		return nil, fmt.Errorf("can't register errGauge: %v", err)
	}
	return &ErrMetrics{gv: gv}, nil
}

func (em *ErrMetrics) ErrGaugeInc(level ErrLevel) {
	em.gv.WithLabelValues(string(level)).Inc()
}

// GENERAL

type Metrics struct {
	ResponseTimeHistogram *prometheus.HistogramVec
	RequestsCounter       *prometheus.CounterVec
	PayloadCounter        *prometheus.CounterVec
	MultipurposeCounter   *prometheus.CounterVec
	PushTimeHistogram     *prometheus.HistogramVec
	MultipurposeGauge     *prometheus.GaugeVec
}

func IncRequest(method, path string) {
	generalMetrics.RequestsCounter.WithLabelValues(method, path).Inc()
}

func IncResponse(method, path, query string, code int, dur time.Duration) {
	generalMetrics.ResponseTimeHistogram.WithLabelValues(method, path, query, strconv.Itoa(code)).Observe(dur.Seconds())
}

func AddPayload(method, path string, bytesQuantity int) {
	generalMetrics.PayloadCounter.WithLabelValues(method, path).Add(float64(bytesQuantity) / 1048576)
}

func IncCounter(t string) {
	generalMetrics.MultipurposeCounter.WithLabelValues(t).Inc()
}

func AddCounter(t string, v float64) {
	generalMetrics.MultipurposeCounter.WithLabelValues(t).Add(v)
}

func IncPush(address, topic string, dur time.Duration) {
	generalMetrics.PushTimeHistogram.WithLabelValues(address, topic).Observe(dur.Seconds())
}

func IncGauge(t string) {
	generalMetrics.MultipurposeGauge.WithLabelValues(t).Inc()
}

func AddGauge(t string, v float64) {
	generalMetrics.MultipurposeGauge.WithLabelValues(t).Add(v)
}

func DecGauge(t string) {
	generalMetrics.MultipurposeGauge.WithLabelValues(t).Dec()
}

func SubGauge(t string, v float64) {
	generalMetrics.MultipurposeGauge.WithLabelValues(t).Sub(v)
}

func SetGauge(t string, v float64) {
	generalMetrics.MultipurposeGauge.WithLabelValues(t).Set(v)
}

func initGeneral(namespace, subsystem string) (*Metrics, error) {
	m := new(Metrics)
	m.ResponseTimeHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "wbkeepers_handler_http_response_time",
		Help:      "duration of HTTP response by method, path, query and status code",
		Buckets:   []float64{0.000001, 0.00001, 0.0001, 0.001, 0.01, 0.05, 0.1, 0.5, 1, 2, 3, 5, 10, 100},
	}, []string{"method", "path", "query", "code"})
	if err := prometheus.Register(m.ResponseTimeHistogram); err != nil {
		return nil, fmt.Errorf("can't register: %v", err)
	}
	m.RequestsCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "wbkeepers_handler_http_requests_count",
		Help:      "count of registered HTTP requests in handler",
	}, []string{"method", "path"})
	if err := prometheus.Register(m.RequestsCounter); err != nil {
		return nil, fmt.Errorf("can't register: %v", err)
	}
	m.PayloadCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "wbkeepers_handler_http_requests_payload_count",
		Help:      "volume of registered HTTP request bodies in handler (mb)",
	}, []string{"method", "path"})
	if err := prometheus.Register(m.PayloadCounter); err != nil {
		return nil, fmt.Errorf("can't register: %v", err)
	}
	m.MultipurposeCounter = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "wbkeepers_multipurpose_count",
		Help:      "multipurpose counter",
	}, []string{"type"})
	if err := prometheus.Register(m.MultipurposeCounter); err != nil {
		return nil, fmt.Errorf("can't register: %v", err)
	}
	m.PushTimeHistogram = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "wbkeepers_kafka_push_time",
		Help:      "duration of kafka push by address and topic",
		Buckets:   []float64{0.000001, 0.00001, 0.0001, 0.001, 0.01, 0.05, 0.1, 0.5, 1, 2, 3, 5, 10, 100},
	}, []string{"address", "topic"})
	if err := prometheus.Register(m.PushTimeHistogram); err != nil {
		return nil, fmt.Errorf("can't register: %v", err)
	}
	m.MultipurposeGauge = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "wbkeepers_multipurpose_gauge",
		Help:      "multipurpose counter",
	}, []string{"type"})
	if err := prometheus.Register(m.MultipurposeGauge); err != nil {
		return nil, fmt.Errorf("can't register: %v", err)
	}
	return m, nil
}

func EnableMetrics(namespace, subsystem string) error {
	m, err := initGeneral(namespace, subsystem)
	if err != nil {
		return err
	}
	generalMetrics = m
	return nil
}
