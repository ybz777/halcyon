package domain

// OrganizationStatus represents organization lifecycle state.
type OrganizationStatus string

const (
	OrganizationStatusActive    OrganizationStatus = "active"
	OrganizationStatusSuspended OrganizationStatus = "suspended"
	OrganizationStatusDeleted   OrganizationStatus = "deleted"
)

func isValidOrganizationStatus(status OrganizationStatus) bool {
	switch status {
	case OrganizationStatusActive, OrganizationStatusSuspended, OrganizationStatusDeleted:
		return true
	default:
		return false
	}
}
