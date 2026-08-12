package resolver

import (
	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
)

// Recipient is a notification target user.
type Recipient struct {
	UserID         string
	OrganizationID string
	Email          string
}

// Resolve determines notification recipients for an event envelope.
func Resolve(envelope platformevents.Envelope) []Recipient {
	orgID := envelope.OrganizationID
	if orgID == "" {
		return nil
	}

	switch envelope.EventType {
	case platformevents.TypeWorkflowExecutionSucceeded,
		platformevents.TypeWorkflowExecutionFailed:
		if uid := actorOrSubject(envelope); uid != "" {
			return []Recipient{{UserID: uid, OrganizationID: orgID}}
		}
	case platformevents.TypeMembershipRoleChanged,
		platformevents.TypeMembershipCreated,
		platformevents.TypeMembershipRemoved,
		platformevents.TypeMemberJoined:
		if uid := payloadUserID(envelope); uid != "" {
			return []Recipient{{UserID: uid, OrganizationID: orgID}}
		}
	case platformevents.TypeAPIKeyCreated, platformevents.TypeAPIKeyRevoked:
		if uid := actorOrSubject(envelope); uid != "" {
			return []Recipient{{UserID: uid, OrganizationID: orgID}}
		}
	case platformevents.TypeUserLoggedIn:
		if uid := envelope.AggregateID; uid != "" {
			email := loginEmail(envelope)
			return []Recipient{{UserID: uid, OrganizationID: orgID, Email: email}}
		}
	}
	return nil
}

func actorOrSubject(envelope platformevents.Envelope) string {
	if envelope.ActorID != "" {
		return envelope.ActorID
	}
	return envelope.SubjectID
}

func payloadUserID(envelope platformevents.Envelope) string {
	if envelope.SubjectID != "" {
		return envelope.SubjectID
	}
	return envelope.ActorID
}

func loginEmail(envelope platformevents.Envelope) string {
	raw, err := platformevents.UnmarshalPayload(envelope.EventType, envelope.Payload)
	if err != nil {
		return ""
	}
	if p, ok := raw.(platformevents.UserLoggedInPayload); ok {
		return p.Email
	}
	return ""
}
