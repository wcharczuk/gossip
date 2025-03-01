package types

type MetricSinkSubmission struct {
	Values []MetricSinkSubmissionValue
}

type MetricSinkSubmissionValue struct {
	Entity   string
	Hostname string
	Value    int64
}
