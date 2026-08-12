package domain

// Permission identifies a gateway-enforced authorization capability carried in JWTs.
type Permission string

const (
	PermissionAPIKeyManage Permission = "apikey:manage"
)

// String returns the permission identifier.
func (p Permission) String() string {
	return string(p)
}
