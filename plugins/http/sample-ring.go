package httpplugin

type Sample struct {
	Timestamp int64 `json:"timestamp_us"`
	Value     int16 `json:"value"`
}

type sampleRing struct {
	samples []Sample
	next    int
	count   int
}

func newSampleRing(capacity int) *sampleRing {
	return &sampleRing{samples: make([]Sample, capacity)}
}

func (ring *sampleRing) Append(sample Sample) {
	ring.samples[ring.next] = sample
	ring.next = (ring.next + 1) % len(ring.samples)
	if ring.count < len(ring.samples) {
		ring.count++
	}
}

func (ring *sampleRing) Snapshot() []Sample {
	result := make([]Sample, ring.count)
	start := ring.next - ring.count
	if start < 0 {
		start += len(ring.samples)
	}
	for index := range result {
		result[index] = ring.samples[(start+index)%len(ring.samples)]
	}
	return result
}
