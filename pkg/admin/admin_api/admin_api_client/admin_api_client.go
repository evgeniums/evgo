package admin_api_client

import (
	"github.com/evgeniums/go-utils/pkg/admin"
	"github.com/evgeniums/go-utils/pkg/api"
	"github.com/evgeniums/go-utils/pkg/api/api_client"
	"github.com/evgeniums/go-utils/pkg/user/user_api/user_client"
)

type AdminClient = user_client.UserClient[*admin.Admin]

func NewAdminClient(client api_client.Client, visitors ...api.OperationVisitors) *AdminClient {
	c := user_client.NewUserClient(client, admin.NewAdmin, "admin")
	c.ApplyVisitors(visitors...)
	return c
}
