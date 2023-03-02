package types

type AuthProvider interface {
	CheckAccess(accessInfo TableAccessInfo, username string) bool
	CheckAuth(username, password string) bool
}
