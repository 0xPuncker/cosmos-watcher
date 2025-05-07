package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/0xPuncker/cosmos-watcher/pkg/types"
	"github.com/0xPuncker/cosmos-watcher/pkg/utils"
	"github.com/sirupsen/logrus"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type SlackService struct {
	logger     *logrus.Logger
	webhookURL string
	client     *http.Client
}

type SlackMessage struct {
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

type Attachment struct {
	Color  string  `json:"color,omitempty"`
	Text   string  `json:"text,omitempty"`
	Fields []Field `json:"fields,omitempty"`
	Footer string  `json:"footer,omitempty"`
	Ts     int64   `json:"ts,omitempty"`
}

type Field struct {
	Title string `json:"title"`
	Value string `json:"value"`
	Short bool   `json:"short"`
}

func NewSlackService(logger *logrus.Logger) (*SlackService, error) {
	webhookURL := os.Getenv("SLACK_WEBHOOK_URL")
	if webhookURL == "" {
		return nil, fmt.Errorf("SLACK_WEBHOOK_URL environment variable is not set")
	}

	return &SlackService{
		logger:     logger,
		webhookURL: webhookURL,
		client:     &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func (s *SlackService) SendUpgradeNotification(chainName string, upgradeInfo *types.UpgradeInfo) error {
	timeUntilUpgrade := time.Until(upgradeInfo.Time)
	timeUntilStr := utils.FormatDuration(timeUntilUpgrade)

	color := "#36a64f"
	if timeUntilUpgrade < 24*time.Hour {
		color = "#ffcc00"
	}
	if timeUntilUpgrade < 1*time.Hour {
		color = "#ff0000"
	}

	mainMessage := fmt.Sprintf("🚀 New Upgrade Scheduled for %s\nUpgrade: %s",
		cases.Title(language.English).String(chainName),
		upgradeInfo.Version)

	fields := []Field{
		{
			Title: "Network Type",
			Value: upgradeInfo.Network,
			Short: true,
		},
		{
			Title: "Height",
			Value: fmt.Sprintf("%d", upgradeInfo.Height),
			Short: true,
		},
		{
			Title: "Estimated Time",
			Value: upgradeInfo.Time.Format(time.RFC1123),
			Short: true,
		},
		{
			Title: "Time Until Upgrade",
			Value: timeUntilStr,
			Short: true,
		},
	}

	// Add Cosmovisor folder if available
	if upgradeInfo.CosmovisorFolder != "" {
		fields = append(fields, Field{
			Title: "Cosmovisor Folder",
			Value: upgradeInfo.CosmovisorFolder,
			Short: true,
		})
	}

	var links []string

	if upgradeInfo.ProposalLink != "" {
		links = append(links, fmt.Sprintf("📋 <%s|View Proposal>", upgradeInfo.ProposalLink))
	}

	if upgradeInfo.Guide != "" {
		links = append(links, fmt.Sprintf("📚 <%s|View Guide>", upgradeInfo.Guide))
	}

	if upgradeInfo.BlockLink != "" {
		links = append(links, fmt.Sprintf("🔍 <%s|View Block>", upgradeInfo.BlockLink))
	}

	if upgradeInfo.Repo != "" {
		links = append(links, fmt.Sprintf("📦 <%s|View Code>", upgradeInfo.Repo))
	}

	if len(links) > 0 {
		fields = append(fields, Field{
			Title: "Links",
			Value: strings.Join(links, " | "),
			Short: false,
		})
	}

	message := SlackMessage{
		Text: mainMessage,
		Attachments: []Attachment{
			{
				Color:  color,
				Fields: fields,
				Footer: fmt.Sprintf("Chain: %s | Last Updated: %s",
					chainName,
					time.Now().Format("Mon, 02 Jan 2006 15:04:05 MST")),
				Ts: time.Now().Unix(),
			},
		},
	}

	if upgradeInfo.Info != "" {
		message.Attachments[0].Text = upgradeInfo.Info
	}

	return s.SendSlackMessage(&message)
}

func (s *SlackService) SendSlackMessage(message *SlackMessage) error {
	if s.webhookURL == "" {
		return fmt.Errorf("slack webhook URL not configured")
	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("error marshaling slack message: %w", err)
	}

	resp, err := http.Post(s.webhookURL, "application/json", bytes.NewBuffer(jsonMessage))
	if err != nil {
		return fmt.Errorf("error sending slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack API returned non-200 status code: %d", resp.StatusCode)
	}

	s.logger.Infof("Successfully sent message to Slack")
	return nil
}
