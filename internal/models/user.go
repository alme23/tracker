// tracker/internal/models/user.go
package models

// UserInfo содержит информацию о текущем пользователе
type UserInfo struct {
	Username     string `json:"username"`       // Имя пользователя в формате DOMAIN\Username
	FullName     string `json:"full_name"`      // Полное имя
	Domain       string `json:"domain"`         // Короткое имя домена (или пусто для локальных)
	DomainFull   string `json:"domain_full"`    // Полное имя домена (пусто для рабочей группы)
	Workgroup    string `json:"workgroup"`      // Рабочая группа (если не в домене)
	ProfilePath  string `json:"profile_path"`   // Путь к профилю
	IsAdmin      bool   `json:"is_admin"`       // Администратор ли
	IsDomainUser bool   `json:"is_domain_user"` // Доменный ли пользователь
	IsLocalUser  bool   `json:"is_local_user"`  // Локальный ли пользователь
}
