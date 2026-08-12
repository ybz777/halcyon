package eventtopics

// TopicForEventType maps an event type to its Kafka topic.
func TopicForEventType(eventType string) string {
	switch {
	case hasPrefix(eventType, "auth."):
		return "halcyon.auth"
	case hasPrefix(eventType, "organizations."):
		return "halcyon.organizations"
	case hasPrefix(eventType, "incidents."):
		return "halcyon.incidents"
	case hasPrefix(eventType, "workflow."):
		return "halcyon.workflows"
	case hasPrefix(eventType, "security."):
		return "halcyon.audit"
	case hasPrefix(eventType, "audit."):
		return "halcyon.audit"
	default:
		return "halcyon.events"
	}
}

// DLQTopicFor returns the dead-letter topic for a source topic.
func DLQTopicFor(topic string) string {
	return topic + ".dlq"
}

func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}
