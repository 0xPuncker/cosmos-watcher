package calendar

import (
	"fmt"
	"net/url"
	"time"

	"github.com/0xPuncker/cosmos-watcher/pkg/types"
)

type CalendarService struct{}

func NewCalendarService() *CalendarService {
	return &CalendarService{}
}

func (s *CalendarService) CreateEventURL(title, description string, startTime, endTime time.Time, location string) (string, error) {
	if title == "" {
		return "", fmt.Errorf("title cannot be empty")
	}

	if endTime.Before(startTime) {
		return "", fmt.Errorf("end time cannot be before start time")
	}

	if startTime.Equal(endTime) {
		return "", fmt.Errorf("start time and end time cannot be the same")
	}

	start := startTime.UTC().Format("20060102T150405Z")
	end := endTime.UTC().Format("20060102T150405Z")

	u := url.URL{
		Scheme: "https",
		Host:   "calendar.google.com",
		Path:   "calendar/render",
	}

	params := url.Values{}
	params.Add("action", "TEMPLATE")
	params.Add("text", title)
	params.Add("details", description)
	params.Add("dates", fmt.Sprintf("%s/%s", start, end))
	params.Add("location", location)

	u.RawQuery = params.Encode()

	return u.String(), nil
}

func (s *CalendarService) CreateUpgradeEvent(chainName string, upgradeInfo *types.UpgradeInfo) (string, error) {
	if upgradeInfo == nil {
		return "", fmt.Errorf("upgrade info cannot be nil")
	}

	if chainName == "" {
		return "", fmt.Errorf("chain name cannot be empty")
	}

	if upgradeInfo.Time.Before(time.Now()) {
		return "", fmt.Errorf("upgrade time cannot be in the past")
	}

	title := fmt.Sprintf("%s Network Upgrade", chainName)
	description := fmt.Sprintf("Chain: %s\nUpgrade Name: %s\nUpgrade Height: %d\nInfo: %s\nEstimated: %v",
		chainName, upgradeInfo.Name, upgradeInfo.Height, upgradeInfo.Info, upgradeInfo.Estimated)

	endTime := upgradeInfo.Time.Add(1 * time.Hour)

	return s.CreateEventURL(title, description, upgradeInfo.Time, endTime, "Cosmos Network")
}

func CreateUpgradeCalendarURL(chainName string, upgradeInfo *types.UpgradeInfo) (string, error) {
	service := NewCalendarService()
	return service.CreateUpgradeEvent(chainName, upgradeInfo)
}
