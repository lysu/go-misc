package cb

import "golang.org/x/net/context"

type RepChannels struct {
	Success chan interface{}
	Failure chan error
}

type Service func(ctx context.Context, req interface{}) RepChannels

type Filter func(ctx context.Context, req interface{}, service Service) RepChannels

func (f Filter) AndThenService(s Service) Service {
	return func(ctx context.Context, req interface{}) RepChannels {
		return f(ctx, req, s)
	}
}

func (f Filter) AndThenFilter(nf Filter) Filter {
	return func(ctx context.Context, req interface{}, service Service) RepChannels {
		filteredService := nf.AndThenService(service)
		return f.AndThenService(filteredService)(ctx, req)
	}
}


type ClientConnection struct {

}

type ServiceChannel struct {
	Success chan Service
	Failure chan error
}

type ServiceFactory func(ctx context.Context, connection ClientConnection) ServiceChannel
