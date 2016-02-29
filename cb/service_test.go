package cb_test

import (
	"fmt"
	"github.com/lysu/go-misc/cb"
	"golang.org/x/net/context"
	"testing"
)

func TestAbc(t *testing.T) {

	var s cb.Service = func(ctx context.Context, req interface{}) cb.RepChannels {
		rep := cb.RepChannels{
			Success: make(chan interface{}, 1),
			Failure: make(chan error, 1),
		}
		go func() {
			fmt.Println("Service called")
			rep.Success <- struct{}{}
		}()
		return rep
	}

	var f cb.Filter = func(ctx context.Context, req interface{}, service cb.Service) cb.RepChannels {
		rep := cb.RepChannels{
			Success: make(chan interface{}, 1),
			Failure: make(chan error, 1),
		}
		fmt.Println("Filter1 called")
		if req == nil {
			rep.Failure <- fmt.Errorf("niled")
			return rep
		}
		return service(ctx, req)
	}

	f = f.AndThenFilter(func(ctx context.Context, req interface{}, service cb.Service) cb.RepChannels {
		rep := cb.RepChannels{
			Success: make(chan interface{}, 1),
			Failure: make(chan error, 1),
		}
		fmt.Println("Filter2 called")
		if req == nil {
			rep.Failure <- fmt.Errorf("niled")
			return rep
		}
		return service(ctx, req)
	})

	s = f.AndThenService(s)

	ctx := context.Background()
	var req interface{} = struct{}{}
	repChannels := s(ctx, req)

	select {
	case sucResult := <-repChannels.Success:
		fmt.Println(sucResult)
	case failResult := <-repChannels.Failure:
		fmt.Println(failResult)
	}

}
