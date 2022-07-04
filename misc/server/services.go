package server

type Services struct {
	NeedAuth bool
	Auth     Authorizer
	Store    Store
	Notifier Notifier
}

func (svc *Services) Start() error {

	return nil
}
