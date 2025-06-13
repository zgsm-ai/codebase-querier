package job

import (
	"context"
	"github.com/zgsm-ai/codebase-indexer/internal/svc"
)

type Scheduler struct {
	jobs []Job
}

func NewScheduler(serverCtx context.Context, svcCtx *svc.ServiceContext) (*Scheduler, error) {
	cleaner, err := newCleaner(serverCtx, svcCtx)
	if err != nil {
		return nil, err
	}
	indexJob, err := newIndexJob(serverCtx, svcCtx)
	if err != nil {
		return nil, err
	}
	jobs := []Job{
		cleaner,
		indexJob,
	}
	return &Scheduler{
		jobs: jobs,
	}, nil
}

func (s *Scheduler) Schedule() {
	for _, job := range s.jobs {
		job.Start()
	}
}

func (s *Scheduler) Close() {
	for _, job := range s.jobs {
		if job == nil {
			continue
		}
		job.Close()
	}
}
