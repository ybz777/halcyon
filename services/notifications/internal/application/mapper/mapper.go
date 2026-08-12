package mapper

import (
	"fmt"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/services/notifications/internal/domain"
)

// Spec is a notification to create for one recipient.
type Spec struct {
	NotificationType domain.NotificationType
	Title            string
	Body             string
	ResourceType     string
	ResourceID       string
	Metadata         map[string]string
}

// MapEnvelope maps a domain event to notification specs (without recipients).
func MapEnvelope(envelope platformevents.Envelope) ([]Spec, bool) {
	switch envelope.EventType {
	case platformevents.TypeWorkflowExecutionSucceeded:
		return []Spec{{
			NotificationType: domain.TypeWorkflowExecutionCompleted,
			Title:            "Workflow completed",
			Body:             "A workflow execution completed successfully.",
			ResourceType:     "workflow_execution",
			ResourceID:       envelope.AggregateID,
		}}, true
	case platformevents.TypeWorkflowExecutionFailed:
		return []Spec{{
			NotificationType: domain.TypeWorkflowExecutionFailed,
			Title:            "Workflow failed",
			Body:             "A workflow execution failed and may require attention.",
			ResourceType:     "workflow_execution",
			ResourceID:       envelope.AggregateID,
		}}, true
	case platformevents.TypeMembershipRoleChanged:
		return []Spec{{
			NotificationType: domain.TypeOrganizationRoleChanged,
			Title:            "Your role changed",
			Body:             "Your organization role was updated.",
			ResourceType:     "membership",
			ResourceID:       envelope.AggregateID,
		}}, true
	case platformevents.TypeMembershipCreated, platformevents.TypeMemberJoined:
		return []Spec{{
			NotificationType: domain.TypeOrganizationMemberAdded,
			Title:            "Welcome to the organization",
			Body:             "You have been added to an organization.",
			ResourceType:     "membership",
			ResourceID:       envelope.AggregateID,
		}}, true
	case platformevents.TypeMembershipRemoved:
		return []Spec{{
			NotificationType: domain.TypeOrganizationMemberRemoved,
			Title:            "Membership removed",
			Body:             "Your membership in an organization was removed.",
			ResourceType:     "membership",
			ResourceID:       envelope.AggregateID,
		}}, true
	case platformevents.TypeAPIKeyCreated:
		return []Spec{{
			NotificationType: domain.TypeSecurityAPIKeyCreated,
			Title:            "API key created",
			Body:             "A new API key was created for your organization.",
			ResourceType:     "api_key",
			ResourceID:       envelope.AggregateID,
		}}, true
	case platformevents.TypeAPIKeyRevoked:
		return []Spec{{
			NotificationType: domain.TypeSecurityAPIKeyRevoked,
			Title:            "API key revoked",
			Body:             "An API key was revoked for your organization.",
			ResourceType:     "api_key",
			ResourceID:       envelope.AggregateID,
		}}, true
	case platformevents.TypeUserLoggedIn:
		return []Spec{{
			NotificationType: domain.TypeSecurityLoginAlert,
			Title:            "New sign-in",
			Body:             fmt.Sprintf("A sign-in to your account was detected at %s.", envelope.OccurredAt.UTC().Format("15:04 UTC")),
			ResourceType:     "user",
			ResourceID:       envelope.AggregateID,
		}}, true
	default:
		return nil, false
	}
}
