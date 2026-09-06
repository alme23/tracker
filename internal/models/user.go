package models

// UserInfo contains information about the current user
type UserInfo struct {
	Username     string `json:"username"`       // Username in DOMAIN\Username format
	FullName     string `json:"full_name"`      // Full display name
	Domain       string `json:"domain"`         // Short domain name (empty for local users)
	DomainFull   string `json:"domain_full"`    // Full domain name (empty for workgroup)
	Workgroup    string `json:"workgroup"`      // Workgroup name (if not in a domain)
	ProfilePath  string `json:"profile_path"`   // User profile path
	IsAdmin      bool   `json:"is_admin"`       // Whether user is an administrator
	IsDomainUser bool   `json:"is_domain_user"` // Whether user is a domain user
	IsLocalUser  bool   `json:"is_local_user"`  // Whether user is a local user
}
