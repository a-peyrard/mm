package worker

import (
	"context"
	"github.com/rs/zerolog"
	"log"
	"sync"
)

type (
	Factory[P any] func(ctx context.Context, workerIdx int) (Worker[P], error)

	Worker[P any] interface {
		WaitReady(ctx context.Context) error
		Handle(ctx context.Context, param P) error
		WaitAndClose() error
	}

	Group[P any] struct {
		ctx     context.Context
		work    chan P
		workers []Worker[P]

		workersInProgress *sync.WaitGroup
	}
)

func NewGroup[P any](ctx context.Context, nbWorkers int, factory Factory[P]) (*Group[P], error) {
	logger := zerolog.Ctx(ctx)

	work := make(chan P)
	workers := make([]Worker[P], nbWorkers)
	workersInCreation := sync.WaitGroup{}
	workersInProgress := sync.WaitGroup{}
	for i := 0; i < nbWorkers; i++ {
		workersInCreation.Add(1)
		workersInProgress.Add(1)
		go func(i int) {
			defer workersInProgress.Done()

			worker, err := factory(ctx, i)
			if err != nil {
				logger.Error().Err(err).Msgf("failed to create worker %d", i)
				return
			}
			workers[i] = worker

			workersInCreation.Done()

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
		}(i)
	}

	// wait for all workers to be created before returning the group
	workersInCreation.Wait()

	return &Group[P]{
		ctx:               ctx,
		work:              work,
		workers:           workers,
		workersInProgress: &workersInProgress,
	}, nil
}

func (p Group[P]) WaitAllWorkersToBeReady(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, worker := range p.workers {
		wg.Add(1)
		go func(w Worker[P]) {
			defer wg.Done()

			if w == nil {
				zerolog.Ctx(p.ctx).Error().Msg("worker is nil")
				log.Printf("workers: %v\n", p.workers)
				return
			}
			err := w.WaitReady(ctx)
			if err != nil {
				zerolog.Ctx(p.ctx).Error().Err(err).Msg("worker failed to be ready")
			}
		}(worker)
	}

	wg.Wait()
	return nil
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
	g.workersInProgress.Wait()

	closingWg := sync.WaitGroup{}
	for _, worker := range g.workers {
		closingWg.Add(1)
		go func(w Worker[P]) {
			defer closingWg.Done()
			if err := w.WaitAndClose(); err != nil {
				zerolog.Ctx(g.ctx).Error().Err(err).Msg("worker failed to close")
			}
		}(worker)
	}

	closingWg.Wait()

	return nil
}
