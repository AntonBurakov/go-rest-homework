package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type Registry struct {
	mu       sync.Mutex
	requests map[string]int64
	errors   map[string]int64
	duration map[string][]float64
}

func NewRegistry() *Registry {
	return &Registry{
		requests: map[string]int64{},
		errors:   map[string]int64{},
		duration: map[string][]float64{},
	}
}

func (r *Registry) Observe(method, path string, statusCode int, durationSeconds float64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	status := fmt.Sprintf("%d", statusCode)
	requestKey := labels(method, path, status)
	durationKey := labels(method, path, "")

	r.requests[requestKey]++
	r.duration[durationKey] = append(r.duration[durationKey], durationSeconds)
	if statusCode >= 400 {
		r.errors[requestKey]++
	}
}

func (r *Registry) Render() string {
	r.mu.Lock()
	defer r.mu.Unlock()

	var b strings.Builder
	b.WriteString("# HELP http_requests_total Total number of HTTP requests\n")
	b.WriteString("# TYPE http_requests_total counter\n")
	writeCounter(&b, "http_requests_total", r.requests)

	b.WriteString("# HELP http_errors_total Total number of HTTP error responses\n")
	b.WriteString("# TYPE http_errors_total counter\n")
	writeCounter(&b, "http_errors_total", r.errors)

	b.WriteString("# HELP http_request_duration_seconds HTTP request duration in seconds\n")
	b.WriteString("# TYPE http_request_duration_seconds summary\n")
	writeDuration(&b, r.duration)
	return b.String()
}

func writeCounter(b *strings.Builder, name string, values map[string]int64) {
	for _, key := range sortedKeys(values) {
		b.WriteString(fmt.Sprintf("%s{%s} %d\n", name, key, values[key]))
	}
}

func writeDuration(b *strings.Builder, values map[string][]float64) {
	for _, key := range sortedKeys(values) {
		var sum float64
		for _, item := range values[key] {
			sum += item
		}
		b.WriteString(fmt.Sprintf("http_request_duration_seconds_sum{%s} %f\n", key, sum))
		b.WriteString(fmt.Sprintf("http_request_duration_seconds_count{%s} %d\n", key, len(values[key])))
	}
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func labels(method, path, statusCode string) string {
	parts := []string{
		fmt.Sprintf(`method="%s"`, method),
		fmt.Sprintf(`path="%s"`, path),
	}
	if statusCode != "" {
		parts = append(parts, fmt.Sprintf(`status_code="%s"`, statusCode))
	}
	return strings.Join(parts, ",")
}
