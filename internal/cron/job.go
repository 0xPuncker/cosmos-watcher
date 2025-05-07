package cron

import (
	"fmt"
	"sync"
	"time"

	"github.com/p2p/devops-cosmos-watcher/pkg/types"
	"github.com/robfig/cron/v3"
	"github.com/sirupsen/logrus"
)

type Scheduler struct {
	cron   *cron.Cron
	logger *logrus.Logger
	jobs   map[string]struct {
		id          cron.EntryID
		schedule    string
		taskName    string
		enabled     bool
		description string
	}
	mu             sync.RWMutex
	started        bool
	tasks          map[string]func() error
	maxConcurrent  int
	activeJobs     int
	activeJobsLock sync.Mutex
}

func NewScheduler(logger *logrus.Logger, config types.JobConfig) *Scheduler {
	return &Scheduler{
		cron:          cron.New(cron.WithSeconds()),
		logger:        logger,
		maxConcurrent: config.MaxConcurrent,
		jobs: make(map[string]struct {
			id          cron.EntryID
			schedule    string
			taskName    string
			enabled     bool
			description string
		}),
		tasks: make(map[string]func() error),
	}
}

func (s *Scheduler) RegisterTask(name string, task func() error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[name] = task
}

func (s *Scheduler) LoadPredefinedJobs(jobs []types.Job) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear existing jobs
	for name, job := range s.jobs {
		s.cron.Remove(job.id)
		delete(s.jobs, name)
	}

	// Load predefined jobs
	for _, job := range jobs {
		if !job.Enabled {
			s.logger.Infof("Skipping disabled job: %s", job.Name)
			continue
		}

		task, exists := s.tasks[job.TaskName]
		if !exists {
			return fmt.Errorf("task %s not registered", job.TaskName)
		}

		wrapper := func() {
			s.activeJobsLock.Lock()
			if s.activeJobs >= s.maxConcurrent {
				s.activeJobsLock.Unlock()
				s.logger.Warnf("Max concurrent jobs reached, skipping job: %s", job.Name)
				return
			}
			s.activeJobs++
			s.activeJobsLock.Unlock()

			s.logger.WithFields(logrus.Fields{
				"job_name":    job.Name,
				"schedule":    job.Schedule,
				"task":        job.TaskName,
				"active_jobs": s.activeJobs,
			}).Info("Starting job execution")

			start := time.Now()

			if err := task(); err != nil {
				s.logger.WithFields(logrus.Fields{
					"job_name": job.Name,
					"error":    err.Error(),
					"duration": formatDuration(time.Since(start)),
				}).Error("Job execution failed")
			} else {
				s.logger.WithFields(logrus.Fields{
					"job_name": job.Name,
					"duration": formatDuration(time.Since(start)),
				}).Info("Job execution completed successfully")
			}

			s.activeJobsLock.Lock()
			s.activeJobs--
			s.activeJobsLock.Unlock()
		}

		id, err := s.cron.AddFunc(job.Schedule, wrapper)
		if err != nil {
			return fmt.Errorf("failed to schedule job %s: %w", job.Name, err)
		}

		s.jobs[job.Name] = struct {
			id          cron.EntryID
			schedule    string
			taskName    string
			enabled     bool
			description string
		}{
			id:          id,
			schedule:    job.Schedule,
			taskName:    job.TaskName,
			enabled:     job.Enabled,
			description: job.Description,
		}

		s.logger.WithFields(logrus.Fields{
			"job_name":    job.Name,
			"schedule":    job.Schedule,
			"task":        job.TaskName,
			"enabled":     job.Enabled,
			"description": job.Description,
		}).Info("Job scheduled successfully")
	}

	return nil
}

func formatDuration(d time.Duration) string {
	if d < time.Millisecond {
		return fmt.Sprintf("%.2fµs", float64(d.Microseconds()))
	} else if d < time.Second {
		return fmt.Sprintf("%.2fms", float64(d.Milliseconds()))
	} else if d < time.Minute {
		return fmt.Sprintf("%.2fs", d.Seconds())
	} else {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
}

func (s *Scheduler) GetJobStatus(name string) (bool, string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, exists := s.jobs[name]
	if !exists {
		return false, "", fmt.Errorf("job %s not found", name)
	}

	return job.enabled, job.description, nil
}

func (s *Scheduler) ListJobs() []types.Job {
	s.mu.RLock()
	defer s.mu.RUnlock()

	jobs := make([]types.Job, 0, len(s.jobs))
	for name, job := range s.jobs {
		jobs = append(jobs, types.Job{
			Name:        name,
			Schedule:    job.schedule,
			TaskName:    job.taskName,
			Enabled:     job.enabled,
			Description: job.description,
		})
	}

	return jobs
}

func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.started {
		return fmt.Errorf("scheduler already started")
	}

	s.cron.Start()
	s.started = true
	s.logger.Info("Scheduler started...")

	return nil
}

func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.started {
		return
	}

	ctx := s.cron.Stop()
	<-ctx.Done()
	s.started = false
	s.logger.Info("Scheduler stopped")
}

func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.started
}
