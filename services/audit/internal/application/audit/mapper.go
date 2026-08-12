package audit

import (
	"fmt"
	"strings"
	"time"

	platformevents "github.com/halcyon-io/halcyon/pkg/platform/events"
	"github.com/halcyon-io/halcyon/services/audit/internal/domain"
)

// MapEnvelope maps a platform domain event envelope to an audit event.
// Returns false when the event type should not produce an audit record.
func MapEnvelope(envelope platformevents.Envelope, sourceService string) (domain.AuditEvent, bool, error) {
	if sourceService == "" {
		sourceService = envelope.Producer
	}
	orgID := envelope.OrganizationID
	if orgID == "" {
		orgID = domain.SystemOrganizationID
	}

	base := func(eventType, action string, actorType domain.ActorType, actorID string, targetType domain.TargetType, targetID string, result domain.Result, metadata map[string]string) domain.AuditEvent {
		id, _ := domain.NewAuditEventID()
		return domain.AuditEvent{
			ID:             id,
			OrganizationID: orgID,
			EventType:      eventType,
			Action:         action,
			ActorType:      actorType,
			ActorID:        actorID,
			TargetType:     targetType,
			TargetID:       targetID,
			Result:         result,
			OccurredAt:     envelope.OccurredAt.UTC(),
			RequestID:      envelope.RequestID,
			CorrelationID:  envelope.CorrelationID,
			SourceService:  sourceService,
			Metadata:       domain.SanitizeMetadata(metadata),
		}
	}

	actorID := envelope.ActorID
	if actorID == "" {
		actorID = envelope.SubjectID
	}

	switch envelope.EventType {
	case platformevents.TypeUserLoggedIn:
		p, err := decodePayload[platformevents.UserLoggedInPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventAuthLoginSucceeded, "login", domain.ActorUser, p.UserID, domain.TargetUser, p.UserID, domain.ResultSuccess, map[string]string{
			"email_domain": emailDomain(p.Email),
		}), true, nil

	case platformevents.TypeUserLoginFailed:
		p, err := decodePayload[platformevents.UserLoginFailedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		actor := domain.ActorAnonymous
		target := domain.TargetUser
		targetID := ""
		if p.UserID != "" {
			actor = domain.ActorUser
			targetID = p.UserID
		}
		return base(domain.EventAuthLoginFailed, "login", actor, actorIDOrEmpty(p.UserID), target, targetID, domain.ResultFailure, map[string]string{
			"reason": p.Reason,
		}), true, nil

	case platformevents.TypeOrganizationCreated:
		p, err := decodePayload[platformevents.OrganizationCreatedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventOrganizationCreated, "create", actorTypeFor(actorID), actorID, domain.TargetOrganization, p.OrganizationID, domain.ResultSuccess, map[string]string{
			"name": p.Name,
			"slug": p.Slug,
		}), true, nil

	case platformevents.TypeOrganizationUpdated:
		p, err := decodePayload[platformevents.OrganizationUpdatedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventOrganizationUpdated, "update", actorTypeFor(actorID), actorID, domain.TargetOrganization, p.OrganizationID, domain.ResultSuccess, map[string]string{
			"name": p.Name,
		}), true, nil

	case platformevents.TypeOrganizationDeleted:
		p, err := decodePayload[platformevents.OrganizationDeletedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventOrganizationDeleted, "delete", actorTypeFor(actorID), actorID, domain.TargetOrganization, p.OrganizationID, domain.ResultSuccess, nil), true, nil

	case platformevents.TypeMembershipCreated, platformevents.TypeMemberJoined:
		p, err := decodeMembershipCreated(envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventMembershipCreated, "create", actorTypeFor(actorID), actorID, domain.TargetMembership, p.UserID, domain.ResultSuccess, map[string]string{
			"role": p.Role,
		}), true, nil

	case platformevents.TypeMembershipRoleChanged:
		p, err := decodePayload[platformevents.MembershipRoleChangedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventMembershipRoleChanged, "role_change", actorTypeFor(actorID), actorID, domain.TargetMembership, p.UserID, domain.ResultSuccess, map[string]string{
			"from_role": p.FromRole,
			"to_role":   p.ToRole,
		}), true, nil

	case platformevents.TypeMembershipRemoved:
		p, err := decodePayload[platformevents.MembershipRemovedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventMembershipRemoved, "remove", actorTypeFor(actorID), actorID, domain.TargetMembership, p.UserID, domain.ResultSuccess, nil), true, nil

	case platformevents.TypeInvitationCreated, platformevents.TypeMemberInvited:
		p, err := decodeInvitationCreated(envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventInvitationCreated, "create", actorTypeFor(actorID), actorID, domain.TargetInvitation, envelope.AggregateID, domain.ResultSuccess, map[string]string{
			"role": p.Role,
		}), true, nil

	case platformevents.TypeInvitationAccepted:
		p, err := decodePayload[platformevents.InvitationAcceptedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventInvitationAccepted, "accept", actorTypeFor(actorID), actorID, domain.TargetInvitation, envelope.AggregateID, domain.ResultSuccess, map[string]string{
			"role": p.Role,
		}), true, nil

	case platformevents.TypeInvitationRevoked:
		return base(domain.EventInvitationRevoked, "revoke", actorTypeFor(actorID), actorID, domain.TargetInvitation, envelope.AggregateID, domain.ResultSuccess, nil), true, nil

	case platformevents.TypeAPIKeyCreated:
		p, err := decodePayload[platformevents.APIKeyCreatedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventAPIKeyCreated, "create", actorTypeFor(actorID), actorID, domain.TargetAPIKey, p.APIKeyID, domain.ResultSuccess, map[string]string{
			"key_id": p.KeyID,
			"name":   p.Name,
		}), true, nil

	case platformevents.TypeAPIKeyRevoked:
		p, err := decodePayload[platformevents.APIKeyRevokedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventAPIKeyRevoked, "revoke", actorTypeFor(actorID), actorID, domain.TargetAPIKey, p.APIKeyID, domain.ResultSuccess, nil), true, nil

	case platformevents.TypeAPIKeyRotated:
		p, err := decodePayload[platformevents.APIKeyRotatedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventAPIKeyRotated, "rotate", actorTypeFor(actorID), actorID, domain.TargetAPIKey, p.NewAPIKeyID, domain.ResultSuccess, map[string]string{
			"old_api_key_id": p.OldAPIKeyID,
		}), true, nil

	case platformevents.TypeAPIKeyAuthenticationFailed:
		p, err := decodePayload[platformevents.APIKeyAuthenticationFailedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		targetID := p.APIKeyID
		if targetID == "" {
			targetID = p.KeyPrefix
		}
		return base(domain.EventAPIKeyAuthenticationFailed, "authenticate", domain.ActorAnonymous, "", domain.TargetAPIKey, targetID, domain.ResultFailure, map[string]string{
			"key_prefix": p.KeyPrefix,
			"reason":     p.Reason,
		}), true, nil

	case platformevents.TypeWorkflowExecutionStarted:
		return base(domain.EventWorkflowExecutionStarted, "start", actorTypeFor(actorID), actorID, domain.TargetWorkflowExecution, envelope.AggregateID, domain.ResultSuccess, map[string]string{
			"workflow_id": workflowIDFromAggregate(envelope),
		}), true, nil

	case platformevents.TypeWorkflowExecutionSucceeded:
		return base(domain.EventWorkflowExecutionSucceeded, "complete", actorTypeFor(actorID), actorID, domain.TargetWorkflowExecution, envelope.AggregateID, domain.ResultSuccess, nil), true, nil

	case platformevents.TypeWorkflowExecutionFailed:
		return base(domain.EventWorkflowExecutionFailed, "fail", actorTypeFor(actorID), actorID, domain.TargetWorkflowExecution, envelope.AggregateID, domain.ResultFailure, nil), true, nil

	case platformevents.TypeWorkflowExecutionCancelled:
		return base(domain.EventWorkflowExecutionCancelled, "cancel", actorTypeFor(actorID), actorID, domain.TargetWorkflowExecution, envelope.AggregateID, domain.ResultSuccess, nil), true, nil

	case platformevents.TypeSecurityInvalidToken:
		p, err := decodePayload[platformevents.SecurityInvalidTokenPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		ev := base(domain.EventSecurityInvalidToken, "validate_token", domain.ActorAnonymous, "", domain.TargetSystem, "auth", domain.ResultFailure, map[string]string{
			"reason": p.Reason,
		})
		ev.IPAddress = domain.SanitizeString(p.IPAddress)
		ev.UserAgent = domain.SanitizeString(p.UserAgent)
		return ev, true, nil

	case platformevents.TypeSecurityRateLimited:
		p, err := decodePayload[platformevents.SecurityRateLimitedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		ev := base(domain.EventSecurityRateLimited, "rate_limit", domain.ActorAnonymous, "", domain.TargetSystem, "gateway", domain.ResultDenied, map[string]string{
			"route_class": p.RouteClass,
		})
		ev.IPAddress = domain.SanitizeString(p.IPAddress)
		return ev, true, nil

	case platformevents.TypeSecurityAuthzDenied:
		p, err := decodePayload[platformevents.SecurityAuthzDeniedPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		return base(domain.EventAuthzDenied, "authorize", actorTypeFor(actorID), actorID, domain.TargetSystem, p.Resource, domain.ResultDenied, map[string]string{
			"permission": p.Permission,
		}), true, nil

	case platformevents.TypeSecuritySuspiciousRequest:
		p, err := decodePayload[platformevents.SecuritySuspiciousRequestPayload](envelope)
		if err != nil {
			return domain.AuditEvent{}, false, err
		}
		ev := base(domain.EventSecuritySuspiciousRequest, "inspect", domain.ActorAnonymous, "", domain.TargetSystem, "gateway", domain.ResultFailure, map[string]string{
			"reason": p.Reason,
		})
		ev.IPAddress = domain.SanitizeString(p.IPAddress)
		return ev, true, nil

	default:
		return domain.AuditEvent{}, false, nil
	}
}

func decodePayload[T any](envelope platformevents.Envelope) (T, error) {
	var zero T
	raw, err := platformevents.UnmarshalPayload(envelope.EventType, envelope.Payload)
	if err != nil {
		return zero, fmt.Errorf("decode %s: %w", envelope.EventType, err)
	}
	typed, ok := raw.(T)
	if !ok {
		return zero, fmt.Errorf("unexpected payload type for %s", envelope.EventType)
	}
	return typed, nil
}

func decodeMembershipCreated(envelope platformevents.Envelope) (platformevents.MembershipCreatedPayload, error) {
	switch envelope.EventType {
	case platformevents.TypeMemberJoined:
		p, err := decodePayload[platformevents.MemberJoinedPayload](envelope)
		if err != nil {
			return platformevents.MembershipCreatedPayload{}, err
		}
		return platformevents.MembershipCreatedPayload{
			OrganizationID: p.OrganizationID,
			UserID:         p.UserID,
			Role:           p.Role,
		}, nil
	default:
		return decodePayload[platformevents.MembershipCreatedPayload](envelope)
	}
}

func decodeInvitationCreated(envelope platformevents.Envelope) (platformevents.InvitationCreatedPayload, error) {
	switch envelope.EventType {
	case platformevents.TypeMemberInvited:
		p, err := decodePayload[platformevents.MemberInvitedPayload](envelope)
		if err != nil {
			return platformevents.InvitationCreatedPayload{}, err
		}
		return platformevents.InvitationCreatedPayload{
			OrganizationID: p.OrganizationID,
			Email:          "",
			Role:           p.Role,
		}, nil
	default:
		return decodePayload[platformevents.InvitationCreatedPayload](envelope)
	}
}

func actorTypeFor(actorID string) domain.ActorType {
	if actorID == "" {
		return domain.ActorSystem
	}
	if strings.HasPrefix(actorID, "svc_") {
		return domain.ActorService
	}
	if strings.HasPrefix(actorID, "key_") {
		return domain.ActorAPIKey
	}
	return domain.ActorUser
}

func actorIDOrEmpty(id string) string {
	if id == "" {
		return ""
	}
	return id
}

func emailDomain(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return ""
	}
	return parts[1]
}

func workflowIDFromAggregate(envelope platformevents.Envelope) string {
	if envelope.AggregateType == "workflow" {
		return envelope.AggregateID
	}
	return ""
}

// MapEnvelopeWithOccurredAt allows tests to override occurred time.
func MapEnvelopeWithOccurredAt(envelope platformevents.Envelope, sourceService string, at time.Time) (domain.AuditEvent, bool, error) {
	event, ok, err := MapEnvelope(envelope, sourceService)
	if ok {
		event.OccurredAt = at.UTC()
	}
	return event, ok, err
}
