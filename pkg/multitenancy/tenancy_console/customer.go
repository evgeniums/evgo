package tenancy_console

const CustomerCmd string = "customer"
const CustomerDescription string = "Move tenancy to other customer"

func Customer() Handler {
	a := &CustomerHandler{}
	a.Init(CustomerCmd, CustomerDescription)
	return a
}

type CustomerData struct {
	TenancySelector
	TargetCustomer string `validate:"required,alphanum_|email" vmessage:"Invalid customer ID" long:"target-customer" description:"ID or name of a customer to move the tenancy to" required:"true"`
}

type CustomerHandler struct {
	HandlerBase
	CustomerData
}

func (a *CustomerHandler) Data() interface{} {
	return &a.CustomerData
}

func (a *CustomerHandler) Execute(args []string) error {

	ctx, sctx, controller, err := a.Context(a.Data())
	if err != nil {
		return err
	}
	defer ctx.Close(sctx)

	id, idIsDisplay := PrepareId(a.Id, a.Customer, a.Role)
	return controller.SetCustomer(sctx, id, a.TargetCustomer, idIsDisplay)
}
