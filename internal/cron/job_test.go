package cron

import (
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/p2p/devops-cosmos-watcher/pkg/types"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestScheduler(t *testing.T) {
	logger := logrus.New()
	var counter int
	var mu sync.Mutex

	config := types.JobConfig{
		MaxConcurrent: 10,
		Predefined: []types.Job{
			{
				Name:     "test-job",
				Schedule: "*/1 * * * * *",
				TaskName: "test-task",
				Enabled:  true,
			},
		},
	}

	scheduler := NewScheduler(logger, config)

	scheduler.RegisterTask("test-task", func() error {
		mu.Lock()
		counter++
		mu.Unlock()
		return nil
	})

	err := scheduler.Start()
	assert.NoError(t, err)

	time.Sleep(3 * time.Second)

	scheduler.Stop()

	mu.Lock()
	assert.Greater(t, counter, 0)
	mu.Unlock()

	jobs := scheduler.ListJobs()
	assert.Len(t, jobs, 1)
	assert.Equal(t, "test-job", jobs[0].Name)
	assert.Equal(t, "*/1 * * * * *", jobs[0].Schedule)
}

func TestSchedulerErrors(t *testing.T) {
	logger := logrus.New()
	config := types.JobConfig{
		MaxConcurrent: 10,
		Predefined: []types.Job{
			{
				Name:     "invalid-job",
				Schedule: "invalid-schedule",
				TaskName: "non-existent-task",
				Enabled:  true,
			},
		},
	}

	scheduler := NewScheduler(logger, config)

	err := scheduler.Start()
	assert.NoError(t, err)

	err = scheduler.Start()
	assert.Error(t, err)

	scheduler.Stop()
}

func TestConcurrentJobExecution(t *testing.T) {
	logger := logrus.New()
	var counter int
	var mu sync.Mutex

	config := types.JobConfig{
		MaxConcurrent: 2,
		Predefined: []types.Job{
			{
				Name:     "concurrent-job",
				Schedule: "*/1 * * * * *",
				TaskName: "concurrent-task",
				Enabled:  true,
			},
		},
	}

	scheduler := NewScheduler(logger, config)

	scheduler.RegisterTask("concurrent-task", func() error {
		mu.Lock()
		counter++
		mu.Unlock()
		return nil
	})

	err := scheduler.Start()
	assert.NoError(t, err)

	time.Sleep(5 * time.Second)

	scheduler.Stop()

	mu.Lock()
	assert.Greater(t, counter, 0)
	mu.Unlock()
}

func TestJobErrorHandling(t *testing.T) {
	logger := logrus.New()
	expectedErr := errors.New("test error")
	var errorCount int
	var mu sync.Mutex

	config := types.JobConfig{
		MaxConcurrent: 10,
		Predefined: []types.Job{
			{
				Name:     "error-job",
				Schedule: "*/1 * * * * *",
				TaskName: "error-task",
				Enabled:  true,
			},
		},
	}

	scheduler := NewScheduler(logger, config)

	scheduler.RegisterTask("error-task", func() error {
		mu.Lock()
		errorCount++
		mu.Unlock()
		return expectedErr
	})

	err := scheduler.Start()
	assert.NoError(t, err)

	time.Sleep(3 * time.Second)

	scheduler.Stop()

	mu.Lock()
	assert.Greater(t, errorCount, 0)
	mu.Unlock()
}

func TestJobDisabling(t *testing.T) {
	logger := logrus.New()
	var counter int
	var mu sync.Mutex

	config := types.JobConfig{
		MaxConcurrent: 10,
		Predefined: []types.Job{
			{
				Name:     "disabled-job",
				Schedule: "*/1 * * * * *",
				TaskName: "disabled-task",
				Enabled:  false,
			},
		},
	}

	scheduler := NewScheduler(logger, config)

	scheduler.RegisterTask("disabled-task", func() error {
		mu.Lock()
		counter++
		mu.Unlock()
		return nil
	})

	err := scheduler.Start()
	assert.NoError(t, err)

	time.Sleep(2 * time.Second)

	mu.Lock()
	assert.Equal(t, 0, counter)
	mu.Unlock()

	scheduler.Stop()
}

func TestSchedulerState(t *testing.T) {
	logger := logrus.New()
	config := types.JobConfig{
		MaxConcurrent: 10,
		Predefined:    []types.Job{},
	}

	scheduler := NewScheduler(logger, config)

	assert.False(t, scheduler.IsRunning())

	err := scheduler.Start()
	assert.NoError(t, err)
	assert.True(t, scheduler.IsRunning())

	scheduler.Stop()
	assert.False(t, scheduler.IsRunning())
}
