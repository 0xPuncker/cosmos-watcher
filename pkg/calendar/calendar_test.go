package calendar

import (
	"testing"
	"time"

	"github.com/0xPuncker/cosmos-watcher/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestCreateEventURL(t *testing.T) {
	tests := []struct {
		name        string
		title       string
		startTime   time.Time
		endTime     time.Time
		description string
		expectError bool
	}{
		{
			name:        "valid event",
			title:       "Test Event",
			startTime:   time.Now(),
			endTime:     time.Now().Add(1 * time.Hour),
			description: "Test Description",
			expectError: false,
		},
		{
			name:        "empty title",
			title:       "",
			startTime:   time.Now(),
			endTime:     time.Now().Add(1 * time.Hour),
			description: "Test Description",
			expectError: true,
		},
		{
			name:        "end time before start time",
			title:       "Test Event",
			startTime:   time.Now(),
			endTime:     time.Now().Add(-1 * time.Hour),
			description: "Test Description",
			expectError: true,
		},
		{
			name:        "same start and end time",
			title:       "Test Event",
			startTime:   time.Now(),
			endTime:     time.Now(),
			description: "Test Description",
			expectError: true,
		},
		{
			name:        "very long title",
			title:       "This is a very long title that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long title that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long title that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long title that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long title that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long title that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long title that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long title that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long title that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long title that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters.",
			startTime:   time.Now(),
			endTime:     time.Now().Add(1 * time.Hour),
			description: "Test Description",
			expectError: true,
		},
	}

	service := NewCalendarService()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := service.CreateEventURL(tt.title, tt.description, tt.startTime, tt.endTime, "")
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, url, "https://calendar.google.com/calendar/render")
				assert.Contains(t, url, "action=TEMPLATE")
				assert.Contains(t, url, "text="+tt.title)
				assert.Contains(t, url, "dates=")
				assert.Contains(t, url, "details="+tt.description)
			}
		})
	}
}

func TestCreateUpgradeEvent(t *testing.T) {
	tests := []struct {
		name        string
		chainName   string
		upgradeInfo *types.UpgradeInfo
		expectError bool
	}{
		{
			name:      "valid upgrade info",
			chainName: "cosmoshub",
			upgradeInfo: &types.UpgradeInfo{
				Name:      "v1.0.0",
				Height:    1000000,
				Info:      "Major upgrade",
				Time:      time.Now().Add(24 * time.Hour),
				Estimated: true,
			},
			expectError: false,
		},
		{
			name:        "nil upgrade info",
			chainName:   "cosmoshub",
			upgradeInfo: nil,
			expectError: true,
		},
		{
			name:      "empty chain name",
			chainName: "",
			upgradeInfo: &types.UpgradeInfo{
				Name:      "v1.0.0",
				Height:    1000000,
				Info:      "Major upgrade",
				Time:      time.Now().Add(24 * time.Hour),
				Estimated: true,
			},
			expectError: true,
		},
		{
			name:      "upgrade in the past",
			chainName: "cosmoshub",
			upgradeInfo: &types.UpgradeInfo{
				Name:      "v1.0.0",
				Height:    1000000,
				Info:      "Major upgrade",
				Time:      time.Now().Add(-24 * time.Hour),
				Estimated: true,
			},
			expectError: true,
		},
		{
			name:      "very long chain name",
			chainName: "This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters.",
			upgradeInfo: &types.UpgradeInfo{
				Name:      "v1.0.0",
				Height:    1000000,
				Info:      "Major upgrade",
				Time:      time.Now().Add(24 * time.Hour),
				Estimated: true,
			},
			expectError: true,
		},
	}

	service := NewCalendarService()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := service.CreateUpgradeEvent(tt.chainName, tt.upgradeInfo)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, url, "https://calendar.google.com/calendar/render")
				assert.Contains(t, url, "action=TEMPLATE")
				assert.Contains(t, url, "text="+tt.chainName+" Network Upgrade")
				assert.Contains(t, url, "dates=")
				assert.Contains(t, url, "details=")
			}
		})
	}
}

func TestCreateUpgradeCalendarURL(t *testing.T) {
	tests := []struct {
		name        string
		chainName   string
		upgradeInfo *types.UpgradeInfo
		expectError bool
	}{
		{
			name:      "valid upgrade info",
			chainName: "cosmoshub",
			upgradeInfo: &types.UpgradeInfo{
				Name:      "v1.0.0",
				Height:    1000000,
				Info:      "Major upgrade",
				Time:      time.Now().Add(24 * time.Hour),
				Estimated: true,
			},
			expectError: false,
		},
		{
			name:        "nil upgrade info",
			chainName:   "cosmoshub",
			upgradeInfo: nil,
			expectError: true,
		},
		{
			name:      "empty chain name",
			chainName: "",
			upgradeInfo: &types.UpgradeInfo{
				Name:      "v1.0.0",
				Height:    1000000,
				Info:      "Major upgrade",
				Time:      time.Now().Add(24 * time.Hour),
				Estimated: true,
			},
			expectError: true,
		},
		{
			name:      "upgrade in the past",
			chainName: "cosmoshub",
			upgradeInfo: &types.UpgradeInfo{
				Name:      "v1.0.0",
				Height:    1000000,
				Info:      "Major upgrade",
				Time:      time.Now().Add(-24 * time.Hour),
				Estimated: true,
			},
			expectError: true,
		},
		{
			name:      "very long chain name",
			chainName: "This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters. This is a very long chain name that exceeds the maximum length allowed by Google Calendar for event titles which is 1024 characters.",
			upgradeInfo: &types.UpgradeInfo{
				Name:      "v1.0.0",
				Height:    1000000,
				Info:      "Major upgrade",
				Time:      time.Now().Add(24 * time.Hour),
				Estimated: true,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			url, err := CreateUpgradeCalendarURL(tt.chainName, tt.upgradeInfo)
			if tt.expectError {
				assert.Error(t, err)
				assert.Empty(t, url)
			} else {
				assert.NoError(t, err)
				assert.Contains(t, url, "https://calendar.google.com/calendar/render")
				assert.Contains(t, url, "action=TEMPLATE")
				assert.Contains(t, url, "text="+tt.chainName+" Network Upgrade")
				assert.Contains(t, url, "dates=")
				assert.Contains(t, url, "details=")
			}
		})
	}
}
