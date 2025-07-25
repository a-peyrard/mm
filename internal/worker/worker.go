package worker

import (
	"context"
	"github.com/rs/zerolog"
	"sync"
)

type (
	Factory[P any] func(ctx context.Context, workerIdx int) (Worker[P], error)

	Worker[P any] interface {
		Handle(ctx context.Context, param P) error
		WaitAndClose() error
	}

	Group[P any] struct {
		ctx     context.Context
		work    chan P
		workers []Worker[P]
	}
)

func NewGroup[P any](ctx context.Context, nbWorkers int, factory Factory[P]) (*Group[P], error) {
	logger := zerolog.Ctx(ctx)

	work := make(chan P)
	workers := make([]Worker[P], nbWorkers)
	for i := 0; i < nbWorkers; i++ {
		worker, err := factory(ctx, i)
		if err != nil {
			return nil, err
		}
		workers[i] = worker

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case param, ok := <-work:
					if !ok {
						return
					}
					if err := worker.Handle(ctx, param); err != nil {
						logger.Error().Err(err).Msgf("worker failed to handle parameter: %v", param)
						return
					}
				}
			}
		}()
	}

	return &Group[P]{
		ctx:     ctx,
		work:    work,
		workers: workers,
	}, nil
}

func (g Group[P]) Submit(s P) error {
	select {
	case <-g.ctx.Done():
		return g.ctx.Err()
	case g.work <- s:
	}
	return nil
}

func (g Group[P]) WaitAndClose() error {
	close(g.work)

	wg := sync.WaitGroup{}
	for _, worker := range g.workers {
		wg.Add(1)
		go func(w Worker[P]) {
			defer wg.Done()
			if err := w.WaitAndClose(); err != nil {
				zerolog.Ctx(g.ctx).Error().Err(err).Msg("worker failed to close")
			}
		}(worker)
	}

	wg.Wait()

	return nil
}
