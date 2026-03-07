package admin_api_service

import (
	"github.com/evgeniums/evgo/pkg/admin"
	"github.com/evgeniums/evgo/pkg/admin/admin_api"
	"github.com/evgeniums/evgo/pkg/api/api_server"
	"github.com/evgeniums/evgo/pkg/user/user_api/user_service"
)

type AdminService = user_service.UserService[*admin.Admin]

func NewAdminService(admins *admin.Manager) *AdminService {
	s := user_service.NewUserService[*admin.Admin](admins, admin_api.NewAdminFieldsSetter, "admin")

	adminTableConfig := &api_server.DynamicTableConfig{Model: &admin.Admin{}, Operation: s.ListOperation()}
	s.AddDynamicTables(adminTableConfig)

	return s
}
