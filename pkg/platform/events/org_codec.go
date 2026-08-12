package events

import (
	"fmt"

	"google.golang.org/protobuf/encoding/protowire"
)

func marshalOrgTriple(id, name, slug string) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, id)
	buf = appendStringField(buf, 2, name)
	buf = appendStringField(buf, 3, slug)
	return buf, nil
}

func marshalOrgPair(id, value string) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, id)
	buf = appendStringField(buf, 2, value)
	return buf, nil
}

func marshalOrgSingle(id string) ([]byte, error) {
	return appendStringField(nil, 1, id), nil
}

func marshalMembership(orgID, userID, role string) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, orgID)
	buf = appendStringField(buf, 2, userID)
	buf = appendStringField(buf, 3, role)
	return buf, nil
}

func marshalRoleChange(orgID, userID, fromRole, toRole string) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, orgID)
	buf = appendStringField(buf, 2, userID)
	buf = appendStringField(buf, 3, fromRole)
	buf = appendStringField(buf, 4, toRole)
	return buf, nil
}

func marshalInvitation(orgID, email, role string) ([]byte, error) {
	var buf []byte
	buf = appendStringField(buf, 1, orgID)
	buf = appendStringField(buf, 2, email)
	buf = appendStringField(buf, 3, role)
	return buf, nil
}

func unmarshalStringFields(data []byte, fields map[int]*string) error {
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return fmt.Errorf("invalid tag")
		}
		data = data[n:]
		if typ != protowire.BytesType {
			return fmt.Errorf("unsupported wire type")
		}
		val, n := protowire.ConsumeBytes(data)
		if n < 0 {
			return fmt.Errorf("invalid bytes")
		}
		data = data[n:]
		if ptr, ok := fields[int(num)]; ok {
			*ptr = string(val)
		}
	}
	return nil
}

func marshalOrganizationUpdated(p OrganizationUpdatedPayload) ([]byte, error) {
	return marshalOrgPair(p.OrganizationID, p.Name)
}

func marshalOrganizationDeleted(p OrganizationDeletedPayload) ([]byte, error) {
	return marshalOrgSingle(p.OrganizationID)
}

func marshalMembershipCreated(p MembershipCreatedPayload) ([]byte, error) {
	return marshalMembership(p.OrganizationID, p.UserID, p.Role)
}

func marshalMembershipRoleChanged(p MembershipRoleChangedPayload) ([]byte, error) {
	return marshalRoleChange(p.OrganizationID, p.UserID, p.FromRole, p.ToRole)
}

func marshalMembershipSuspended(p MembershipSuspendedPayload) ([]byte, error) {
	return marshalOrgPair(p.OrganizationID, p.UserID)
}

func marshalMembershipRemoved(p MembershipRemovedPayload) ([]byte, error) {
	return marshalOrgPair(p.OrganizationID, p.UserID)
}

func marshalInvitationCreated(p InvitationCreatedPayload) ([]byte, error) {
	return marshalInvitation(p.OrganizationID, p.Email, p.Role)
}

func marshalInvitationAccepted(p InvitationAcceptedPayload) ([]byte, error) {
	return marshalMembership(p.OrganizationID, p.UserID, p.Role)
}

func marshalInvitationRevoked(p InvitationRevokedPayload) ([]byte, error) {
	return marshalOrgPair(p.OrganizationID, p.Email)
}

func unmarshalOrganizationUpdated(data []byte) (OrganizationUpdatedPayload, error) {
	var p OrganizationUpdatedPayload
	err := unmarshalStringFields(data, map[int]*string{1: &p.OrganizationID, 2: &p.Name})
	return p, err
}

func unmarshalOrganizationDeleted(data []byte) (OrganizationDeletedPayload, error) {
	var p OrganizationDeletedPayload
	err := unmarshalStringFields(data, map[int]*string{1: &p.OrganizationID})
	return p, err
}

func unmarshalMembershipCreated(data []byte) (MembershipCreatedPayload, error) {
	var p MembershipCreatedPayload
	err := unmarshalStringFields(data, map[int]*string{1: &p.OrganizationID, 2: &p.UserID, 3: &p.Role})
	return p, err
}

func unmarshalMembershipRoleChanged(data []byte) (MembershipRoleChangedPayload, error) {
	var p MembershipRoleChangedPayload
	err := unmarshalStringFields(data, map[int]*string{1: &p.OrganizationID, 2: &p.UserID, 3: &p.FromRole, 4: &p.ToRole})
	return p, err
}

func unmarshalMembershipSuspended(data []byte) (MembershipSuspendedPayload, error) {
	var p MembershipSuspendedPayload
	err := unmarshalStringFields(data, map[int]*string{1: &p.OrganizationID, 2: &p.UserID})
	return p, err
}

func unmarshalMembershipRemoved(data []byte) (MembershipRemovedPayload, error) {
	var p MembershipRemovedPayload
	err := unmarshalStringFields(data, map[int]*string{1: &p.OrganizationID, 2: &p.UserID})
	return p, err
}

func unmarshalInvitationCreated(data []byte) (InvitationCreatedPayload, error) {
	var p InvitationCreatedPayload
	err := unmarshalStringFields(data, map[int]*string{1: &p.OrganizationID, 2: &p.Email, 3: &p.Role})
	return p, err
}

func unmarshalInvitationAccepted(data []byte) (InvitationAcceptedPayload, error) {
	var p InvitationAcceptedPayload
	err := unmarshalStringFields(data, map[int]*string{1: &p.OrganizationID, 2: &p.UserID, 3: &p.Role})
	return p, err
}

func unmarshalInvitationRevoked(data []byte) (InvitationRevokedPayload, error) {
	var p InvitationRevokedPayload
	err := unmarshalStringFields(data, map[int]*string{1: &p.OrganizationID, 2: &p.Email})
	return p, err
}
