package types

type AuthProvider interface {
	CheckAccess(accessInfo TableAccessInfo, username string) bool
}
