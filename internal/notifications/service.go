package notifications

import (
	"fmt"
	"strings"
	"time"

	"github.com/0xPuncker/cosmos-watcher/pkg/types"
)

type NotificationType string

const (
	TypeUpgrade NotificationType = "UPGRADE"
	TypeJob     NotificationType = "JOB"
	TypeAPI     NotificationType = "API"
)

type NotificationService struct {
	slackService *SlackService
}

func NewNotificationService(slackService *SlackService) *NotificationService {
	return &NotificationService{
		slackService: slackService,
	}
}

func (s *NotificationService) formatJobNotification(jobName string, status string, duration time.Duration, details string) *SlackMessage {
	var color string
	var icon string

	switch status {
	case "success":
		color = "good"
		icon = "✅"
	case "failed":
		color = "danger"
		icon = "❌"
	case "started":
		color = "warning"
		icon = "🚀"
	default:
		color = "#808080"
		icon = "ℹ️"
	}

	fields := []Field{
		{
			Title: "Job Name",
			Value: jobName,
			Short: true,
		},
		{
			Title: "Status",
			Value: status,
			Short: true,
		},
	}

	if duration > 0 {
		fields = append(fields, Field{
			Title: "Duration",
			Value: duration.String(),
			Short: true,
		})
	}

	if details != "" {
		fields = append(fields, Field{
			Title: "Details",
			Value: details,
			Short: false,
		})
	}

	return &SlackMessage{
		Text: fmt.Sprintf("%s Job Status Update", icon),
		Attachments: []Attachment{
			{
				Color:  color,
				Fields: fields,
				Ts:     time.Now().Unix(),
			},
		},
	}
}

func (s *NotificationService) formatAPINotification(endpoint string, status string, details string) *SlackMessage {
	var color string
	var icon string

	switch status {
	case "success":
		color = "good"
		icon = "✅"
	case "error":
		color = "danger"
		icon = "❌"
	default:
		color = "#808080"
		icon = "ℹ️"
	}

	fields := []Field{
		{
			Title: "Endpoint",
			Value: endpoint,
			Short: true,
		},
		{
			Title: "Status",
			Value: status,
			Short: true,
		},
	}

	if details != "" {
		fields = append(fields, Field{
			Title: "Details",
			Value: details,
			Short: false,
		})
	}

	return &SlackMessage{
		Text: fmt.Sprintf("%s API Event", icon),
		Attachments: []Attachment{
			{
				Color:  color,
				Fields: fields,
				Ts:     time.Now().Unix(),
			},
		},
	}
}

func (s *NotificationService) SendJobNotification(jobName string, status string, duration time.Duration, details string) error {
	message := s.formatJobNotification(jobName, status, duration, details)
	return s.slackService.SendSlackMessage(message)
}

func (s *NotificationService) SendAPINotification(endpoint string, status string, details string) error {
	message := s.formatAPINotification(endpoint, status, details)
	return s.slackService.SendSlackMessage(message)
}

func (s *NotificationService) SendUpgradeNotification(chainName string, upgrade *types.UpgradeInfo) error {
	message := s.formatUpgradeNotification(chainName, upgrade)
	return s.slackService.SendSlackMessage(message)
}

func (s *NotificationService) formatUpgradeNotification(chainName string, upgrade *types.UpgradeInfo) *SlackMessage {
	// Format time in the exact format shown: YYYY-MM-DD HH:mm:ss.SSSSSS +ZZZZ CEST m=+NNNNN.NNNNNN
	timeStr := fmt.Sprintf("%s CEST m=+%f",
		upgrade.Time.Format("2006-01-02 15:04:05.000000 -0700"),
		time.Until(upgrade.Time).Seconds())

	fields := []Field{
		{
			Title: "Chain",
			Value: chainName,
			Short: true,
		},
		{
			Title: "Name",
			Value: upgrade.Name,
			Short: true,
		},
		{
			Title: "Height",
			Value: fmt.Sprintf("%d", upgrade.Height),
			Short: true,
		},
		{
			Title: "Time",
			Value: timeStr,
			Short: true,
		},
	}

	if upgrade.Info != "" {
		fields = append(fields, Field{
			Title: "Info",
			Value: upgrade.Info,
			Short: false,
		})
	}

	// Add Links section with icons
	if upgrade.ProposalLink != "" || upgrade.Guide != "" || upgrade.BlockLink != "" {
		var links []string
		if upgrade.ProposalLink != "" {
			links = append(links, "📄 View Proposal")
		}
		if upgrade.Guide != "" {
			links = append(links, "📚 View Guide")
		}
		if upgrade.BlockLink != "" {
			links = append(links, "🔍 View Block")
		}
		fields = append(fields, Field{
			Title: "Links",
			Value: strings.Join(links, " | "),
			Short: false,
		})
	}

	if upgrade.Network != "" {
		fields = append(fields, Field{
			Title: "Network",
			Value: upgrade.Network,
			Short: true,
		})
	}

	// Add footer with current time in format "Hoje às HH:mm"
	footer := fmt.Sprintf("Hoje às %s", time.Now().Format("15:04"))

	return &SlackMessage{
		Text: "🌐 Chain Upgrade Detected",
		Attachments: []Attachment{
			{
				Color:  "warning",
				Fields: fields,
				Footer: footer,
				Ts:     time.Now().Unix(),
			},
		},
	}
}
