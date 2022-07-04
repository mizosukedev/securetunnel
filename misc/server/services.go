package server

type Services struct {
	NeedAuth bool
	Auth     Authorizer
}

func (svc *Services) Start() error {

	return nil
}
