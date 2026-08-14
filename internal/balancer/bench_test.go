package balancer

import "testing"

func BenchmarkRoundRobinNext(b *testing.B) {
	roundRobin := newTestRoundRobin(b, "A", "B", "C", "D", "E")

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if _, err := roundRobin.Next(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLeastConnectionsNext(b *testing.B) {
	leastConnections := newTestLeastConnections(b, "A", "B", "C", "D", "E")

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if _, err := leastConnections.Next(); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLeastConnectionsNextWithActiveRequests(b *testing.B) {
	leastConnections, backends := newTestLeastConnectionsWithBackends(b, "A", "B", "C", "D", "E")
	backends[0].IncrementActive()
	backends[0].IncrementActive()
	backends[1].IncrementActive()
	backends[3].IncrementActive()

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		if _, err := leastConnections.Next(); err != nil {
			b.Fatal(err)
		}
	}
}
