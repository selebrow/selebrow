package quota

type QuotaQueue interface {
	QueueLimit() int
	QueueSize() int
}
