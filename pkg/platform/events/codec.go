package events

import (
	"fmt"
	"time"

	"google.golang.org/protobuf/encoding/protowire"
)

func marshalEnvelope(e Envelope) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, e.EventID)
	buf = appendStringField(buf, 2, e.EventType)
	buf = appendStringField(buf, 3, e.EventVersion)
	buf = appendStringField(buf, 4, e.AggregateID)
	buf = appendStringField(buf, 5, e.AggregateType)
	buf = appendStringField(buf, 6, e.OrganizationID)
	buf = appendStringField(buf, 7, e.CorrelationID)
	buf = appendStringField(buf, 8, e.CausationID)
	buf = appendTimestampField(buf, 9, e.OccurredAt)
	buf = appendStringField(buf, 10, e.Producer)
	buf = appendBytesField(buf, 11, e.Payload)
	buf = appendStringField(buf, 12, e.TraceID)
	buf = appendStringField(buf, 13, e.SpanID)
	return buf, nil
}

func unmarshalEnvelope(data []byte) (Envelope, error) {
	var e Envelope
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return Envelope{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]

		switch typ {
		case protowire.VarintType, protowire.Fixed32Type, protowire.Fixed64Type:
			v, n := protowire.ConsumeVarint(data)
			if n < 0 {
				return Envelope{}, fmt.Errorf("invalid varint field %d", num)
			}
			data = data[n:]
			_ = v
		case protowire.BytesType:
			val, n := protowire.ConsumeBytes(data)
			if n < 0 {
				return Envelope{}, fmt.Errorf("invalid bytes field %d", num)
			}
			data = data[n:]
			switch num {
			case 1:
				e.EventID = string(val)
			case 2:
				e.EventType = string(val)
			case 3:
				e.EventVersion = string(val)
			case 4:
				e.AggregateID = string(val)
			case 5:
				e.AggregateType = string(val)
			case 6:
				e.OrganizationID = string(val)
			case 7:
				e.CorrelationID = string(val)
			case 8:
				e.CausationID = string(val)
			case 9:
				ts, err := unmarshalTimestamp(val)
				if err != nil {
					return Envelope{}, err
				}
				e.OccurredAt = ts
			case 10:
				e.Producer = string(val)
			case 11:
				e.Payload = append([]byte(nil), val...)
			case 12:
				e.TraceID = string(val)
			case 13:
				e.SpanID = string(val)
			}
		default:
			return Envelope{}, fmt.Errorf("unsupported wire type %d", typ)
		}
	}
	return e, nil
}

func marshalUserRegistered(p UserRegisteredPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.UserID)
	buf = appendStringField(buf, 2, p.Email)
	return buf, nil
}

func unmarshalUserRegistered(data []byte) (UserRegisteredPayload, error) {
	var p UserRegisteredPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return UserRegisteredPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return UserRegisteredPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return UserRegisteredPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.UserID = string(val)
		case 2:
			p.Email = string(val)
		}
	}
	return p, nil
}

func marshalUserLoggedIn(p UserLoggedInPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.UserID)
	buf = appendStringField(buf, 2, p.Email)
	return buf, nil
}

func unmarshalUserLoggedIn(data []byte) (UserLoggedInPayload, error) {
	var p UserLoggedInPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return UserLoggedInPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return UserLoggedInPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return UserLoggedInPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.UserID = string(val)
		case 2:
			p.Email = string(val)
		}
	}
	return p, nil
}

func marshalUserLoginFailed(p UserLoginFailedPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.UserID)
	buf = appendStringField(buf, 2, p.Reason)
	return buf, nil
}

func unmarshalUserLoginFailed(data []byte) (UserLoginFailedPayload, error) {
	var p UserLoginFailedPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return UserLoginFailedPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return UserLoginFailedPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return UserLoginFailedPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.UserID = string(val)
		case 2:
			p.Reason = string(val)
		}
	}
	return p, nil
}

func marshalSecurityInvalidToken(p SecurityInvalidTokenPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.Reason)
	buf = appendStringField(buf, 2, p.IPAddress)
	buf = appendStringField(buf, 3, p.UserAgent)
	return buf, nil
}

func unmarshalSecurityInvalidToken(data []byte) (SecurityInvalidTokenPayload, error) {
	var p SecurityInvalidTokenPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return SecurityInvalidTokenPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return SecurityInvalidTokenPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return SecurityInvalidTokenPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.Reason = string(val)
		case 2:
			p.IPAddress = string(val)
		case 3:
			p.UserAgent = string(val)
		}
	}
	return p, nil
}

func marshalSecurityRateLimited(p SecurityRateLimitedPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.RouteClass)
	buf = appendStringField(buf, 2, p.IPAddress)
	return buf, nil
}

func unmarshalSecurityRateLimited(data []byte) (SecurityRateLimitedPayload, error) {
	var p SecurityRateLimitedPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return SecurityRateLimitedPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return SecurityRateLimitedPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return SecurityRateLimitedPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.RouteClass = string(val)
		case 2:
			p.IPAddress = string(val)
		}
	}
	return p, nil
}

func marshalSecurityAuthzDenied(p SecurityAuthzDeniedPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.Resource)
	buf = appendStringField(buf, 2, p.Permission)
	return buf, nil
}

func unmarshalSecurityAuthzDenied(data []byte) (SecurityAuthzDeniedPayload, error) {
	var p SecurityAuthzDeniedPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return SecurityAuthzDeniedPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return SecurityAuthzDeniedPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return SecurityAuthzDeniedPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.Resource = string(val)
		case 2:
			p.Permission = string(val)
		}
	}
	return p, nil
}

func marshalSecuritySuspiciousRequest(p SecuritySuspiciousRequestPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.Reason)
	buf = appendStringField(buf, 2, p.IPAddress)
	return buf, nil
}

func unmarshalSecuritySuspiciousRequest(data []byte) (SecuritySuspiciousRequestPayload, error) {
	var p SecuritySuspiciousRequestPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return SecuritySuspiciousRequestPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return SecuritySuspiciousRequestPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return SecuritySuspiciousRequestPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.Reason = string(val)
		case 2:
			p.IPAddress = string(val)
		}
	}
	return p, nil
}

func marshalOrganizationCreated(p OrganizationCreatedPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.OrganizationID)
	buf = appendStringField(buf, 2, p.Name)
	buf = appendStringField(buf, 3, p.Slug)
	return buf, nil
}

func unmarshalOrganizationCreated(data []byte) (OrganizationCreatedPayload, error) {
	var p OrganizationCreatedPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return OrganizationCreatedPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return OrganizationCreatedPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return OrganizationCreatedPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.OrganizationID = string(val)
		case 2:
			p.Name = string(val)
		case 3:
			p.Slug = string(val)
		}
	}
	return p, nil
}

func marshalMemberInvited(p MemberInvitedPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.OrganizationID)
	buf = appendStringField(buf, 2, p.UserID)
	buf = appendStringField(buf, 3, p.Role)
	return buf, nil
}

func unmarshalMemberInvited(data []byte) (MemberInvitedPayload, error) {
	var p MemberInvitedPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return MemberInvitedPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return MemberInvitedPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return MemberInvitedPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.OrganizationID = string(val)
		case 2:
			p.UserID = string(val)
		case 3:
			p.Role = string(val)
		}
	}
	return p, nil
}

func marshalMemberJoined(p MemberJoinedPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.OrganizationID)
	buf = appendStringField(buf, 2, p.UserID)
	buf = appendStringField(buf, 3, p.Role)
	return buf, nil
}

func unmarshalMemberJoined(data []byte) (MemberJoinedPayload, error) {
	var p MemberJoinedPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return MemberJoinedPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return MemberJoinedPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return MemberJoinedPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.OrganizationID = string(val)
		case 2:
			p.UserID = string(val)
		case 3:
			p.Role = string(val)
		}
	}
	return p, nil
}

func marshalAPIKeyCreated(p APIKeyCreatedPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.APIKeyID)
	buf = appendStringField(buf, 2, p.KeyID)
	buf = appendStringField(buf, 3, p.ServiceAccountID)
	buf = appendStringField(buf, 4, p.OrganizationID)
	buf = appendStringField(buf, 5, p.Name)
	return buf, nil
}

func unmarshalAPIKeyCreated(data []byte) (APIKeyCreatedPayload, error) {
	var p APIKeyCreatedPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return APIKeyCreatedPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return APIKeyCreatedPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return APIKeyCreatedPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.APIKeyID = string(val)
		case 2:
			p.KeyID = string(val)
		case 3:
			p.ServiceAccountID = string(val)
		case 4:
			p.OrganizationID = string(val)
		case 5:
			p.Name = string(val)
		}
	}
	return p, nil
}

func marshalAPIKeyRevoked(p APIKeyRevokedPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.APIKeyID)
	buf = appendStringField(buf, 2, p.ServiceAccountID)
	buf = appendStringField(buf, 3, p.OrganizationID)
	return buf, nil
}

func unmarshalAPIKeyRevoked(data []byte) (APIKeyRevokedPayload, error) {
	var p APIKeyRevokedPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return APIKeyRevokedPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return APIKeyRevokedPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return APIKeyRevokedPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.APIKeyID = string(val)
		case 2:
			p.ServiceAccountID = string(val)
		case 3:
			p.OrganizationID = string(val)
		}
	}
	return p, nil
}

func marshalAPIKeyRotated(p APIKeyRotatedPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.OldAPIKeyID)
	buf = appendStringField(buf, 2, p.NewAPIKeyID)
	buf = appendStringField(buf, 3, p.ServiceAccountID)
	buf = appendStringField(buf, 4, p.OrganizationID)
	return buf, nil
}

func unmarshalAPIKeyRotated(data []byte) (APIKeyRotatedPayload, error) {
	var p APIKeyRotatedPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return APIKeyRotatedPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return APIKeyRotatedPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return APIKeyRotatedPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.OldAPIKeyID = string(val)
		case 2:
			p.NewAPIKeyID = string(val)
		case 3:
			p.ServiceAccountID = string(val)
		case 4:
			p.OrganizationID = string(val)
		}
	}
	return p, nil
}

func marshalAPIKeyAuthenticationFailed(p APIKeyAuthenticationFailedPayload) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, p.APIKeyID)
	buf = appendStringField(buf, 2, p.KeyPrefix)
	buf = appendStringField(buf, 3, p.OrganizationID)
	buf = appendStringField(buf, 4, p.Reason)
	return buf, nil
}

func unmarshalAPIKeyAuthenticationFailed(data []byte) (APIKeyAuthenticationFailedPayload, error) {
	var p APIKeyAuthenticationFailedPayload
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return APIKeyAuthenticationFailedPayload{}, fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return APIKeyAuthenticationFailedPayload{}, fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return APIKeyAuthenticationFailedPayload{}, fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		switch num {
		case 1:
			p.APIKeyID = string(val)
		case 2:
			p.KeyPrefix = string(val)
		case 3:
			p.OrganizationID = string(val)
		case 4:
			p.Reason = string(val)
		}
	}
	return p, nil
}

func appendStringField(buf []byte, field int, value string) []byte {
	if value == "" {
		return buf
	}
	buf = protowire.AppendTag(buf, protowire.Number(field), protowire.BytesType)
	return protowire.AppendString(buf, value)
}

func appendBytesField(buf []byte, field int, value []byte) []byte {
	if len(value) == 0 {
		return buf
	}
	buf = protowire.AppendTag(buf, protowire.Number(field), protowire.BytesType)
	return protowire.AppendBytes(buf, value)
}

func appendTimestampField(buf []byte, field int, value time.Time) []byte {
	if value.IsZero() {
		return buf
	}
	ts := value.UTC()
	var tsBuf []byte
	tsBuf = protowire.AppendTag(tsBuf, 1, protowire.VarintType)
	tsBuf = protowire.AppendVarint(tsBuf, uint64(ts.Unix()))
	if ts.Nanosecond() != 0 {
		tsBuf = protowire.AppendTag(tsBuf, 2, protowire.VarintType)
		tsBuf = protowire.AppendVarint(tsBuf, uint64(ts.Nanosecond()))
	}
	buf = protowire.AppendTag(buf, protowire.Number(field), protowire.BytesType)
	return protowire.AppendBytes(buf, tsBuf)
}

func unmarshalTimestamp(data []byte) (time.Time, error) {
	var seconds int64
	var nanos int32
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return time.Time{}, fmt.Errorf("invalid timestamp tag")
		}
		data = data[n:]
		if typ != protowire.VarintType {
			return time.Time{}, fmt.Errorf("invalid timestamp wire type")
		}
		v, n := protowire.ConsumeVarint(data)
		if n < 0 {
			return time.Time{}, fmt.Errorf("invalid timestamp varint")
		}
		data = data[n:]
		switch num {
		case 1:
			seconds = int64(v)
		case 2:
			nanos = int32(v)
		}
	}
	return time.Unix(seconds, int64(nanos)).UTC(), nil
}

// MarshalPayload encodes a registered payload type to protobuf bytes.
func MarshalPayload(eventType string, payload any) ([]byte, error) {
	switch eventType {
	case TypeUserRegistered:
		p, ok := payload.(UserRegisteredPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalUserRegistered(p)
	case TypeUserLoggedIn:
		p, ok := payload.(UserLoggedInPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalUserLoggedIn(p)
	case TypeUserLoginFailed:
		p, ok := payload.(UserLoginFailedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalUserLoginFailed(p)
	case TypeOrganizationCreated:
		p, ok := payload.(OrganizationCreatedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalOrganizationCreated(p)
	case TypeMemberInvited:
		p, ok := payload.(MemberInvitedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalMemberInvited(p)
	case TypeMemberJoined:
		p, ok := payload.(MemberJoinedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalMemberJoined(p)
	case TypeAPIKeyCreated:
		p, ok := payload.(APIKeyCreatedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalAPIKeyCreated(p)
	case TypeAPIKeyRevoked:
		p, ok := payload.(APIKeyRevokedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalAPIKeyRevoked(p)
	case TypeAPIKeyRotated:
		p, ok := payload.(APIKeyRotatedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalAPIKeyRotated(p)
	case TypeAPIKeyAuthenticationFailed:
		p, ok := payload.(APIKeyAuthenticationFailedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalAPIKeyAuthenticationFailed(p)
	case TypeOrganizationUpdated:
		p, ok := payload.(OrganizationUpdatedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalOrganizationUpdated(p)
	case TypeOrganizationDeleted:
		p, ok := payload.(OrganizationDeletedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalOrganizationDeleted(p)
	case TypeMembershipCreated:
		p, ok := payload.(MembershipCreatedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalMembershipCreated(p)
	case TypeMembershipRoleChanged:
		p, ok := payload.(MembershipRoleChangedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalMembershipRoleChanged(p)
	case TypeMembershipSuspended:
		p, ok := payload.(MembershipSuspendedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalMembershipSuspended(p)
	case TypeMembershipRemoved:
		p, ok := payload.(MembershipRemovedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalMembershipRemoved(p)
	case TypeInvitationCreated:
		p, ok := payload.(InvitationCreatedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalInvitationCreated(p)
	case TypeInvitationAccepted:
		p, ok := payload.(InvitationAcceptedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalInvitationAccepted(p)
	case TypeInvitationRevoked:
		p, ok := payload.(InvitationRevokedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalInvitationRevoked(p)
	case TypeSecurityInvalidToken:
		p, ok := payload.(SecurityInvalidTokenPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalSecurityInvalidToken(p)
	case TypeSecurityRateLimited:
		p, ok := payload.(SecurityRateLimitedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalSecurityRateLimited(p)
	case TypeSecurityAuthzDenied:
		p, ok := payload.(SecurityAuthzDeniedPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalSecurityAuthzDenied(p)
	case TypeSecuritySuspiciousRequest:
		p, ok := payload.(SecuritySuspiciousRequestPayload)
		if !ok {
			return nil, fmt.Errorf("invalid payload type for %s", eventType)
		}
		return marshalSecuritySuspiciousRequest(p)
	default:
		return nil, fmt.Errorf("unknown event type %q", eventType)
	}
}

// UnmarshalPayload decodes protobuf bytes for a registered event type.
func UnmarshalPayload(eventType string, data []byte) (any, error) {
	switch eventType {
	case TypeUserRegistered:
		return unmarshalUserRegistered(data)
	case TypeUserLoggedIn:
		return unmarshalUserLoggedIn(data)
	case TypeUserLoginFailed:
		return unmarshalUserLoginFailed(data)
	case TypeOrganizationCreated:
		return unmarshalOrganizationCreated(data)
	case TypeMemberInvited:
		return unmarshalMemberInvited(data)
	case TypeMemberJoined:
		return unmarshalMemberJoined(data)
	case TypeAPIKeyCreated:
		return unmarshalAPIKeyCreated(data)
	case TypeAPIKeyRevoked:
		return unmarshalAPIKeyRevoked(data)
	case TypeAPIKeyRotated:
		return unmarshalAPIKeyRotated(data)
	case TypeAPIKeyAuthenticationFailed:
		return unmarshalAPIKeyAuthenticationFailed(data)
	case TypeOrganizationUpdated:
		return unmarshalOrganizationUpdated(data)
	case TypeOrganizationDeleted:
		return unmarshalOrganizationDeleted(data)
	case TypeMembershipCreated:
		return unmarshalMembershipCreated(data)
	case TypeMembershipRoleChanged:
		return unmarshalMembershipRoleChanged(data)
	case TypeMembershipSuspended:
		return unmarshalMembershipSuspended(data)
	case TypeMembershipRemoved:
		return unmarshalMembershipRemoved(data)
	case TypeInvitationCreated:
		return unmarshalInvitationCreated(data)
	case TypeInvitationAccepted:
		return unmarshalInvitationAccepted(data)
	case TypeInvitationRevoked:
		return unmarshalInvitationRevoked(data)
	case TypeSecurityInvalidToken:
		return unmarshalSecurityInvalidToken(data)
	case TypeSecurityRateLimited:
		return unmarshalSecurityRateLimited(data)
	case TypeSecurityAuthzDenied:
		return unmarshalSecurityAuthzDenied(data)
	case TypeSecuritySuspiciousRequest:
		return unmarshalSecuritySuspiciousRequest(data)
	default:
		return nil, fmt.Errorf("unknown event type %q", eventType)
	}
}
