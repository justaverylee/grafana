package notifiers

import (
	"testing"

	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/services/alerting"
	encryptionservice "github.com/grafana/grafana/pkg/services/encryption/service"

	"github.com/stretchr/testify/require"
)

func TestGoogleChatNotifier(t *testing.T) {
	encryptionService := encryptionservice.SetupTestService(t)

	t.Run("Parsing alert notification from settings", func(t *testing.T) {
		t.Run("empty settings should return error", func(t *testing.T) {
			json := `{ }`

			settingsJSON, _ := simplejson.NewJson([]byte(json))
			model := &alerting.AlertNotification{
				Name:     "ops",
				Type:     "googlechat",
				Settings: settingsJSON,
			}

			_, err := newGoogleChatNotifier(model, encryptionService.GetDecryptedValue, nil)
			require.Error(t, err)
		})

		t.Run("from settings", func(t *testing.T) {
			json := `
				{
          			"url": "http://google.com"
				}`

			settingsJSON, _ := simplejson.NewJson([]byte(json))
			model := &alerting.AlertNotification{
				Name:     "ops",
				Type:     "googlechat",
				Settings: settingsJSON,
			}

			not, err := newGoogleChatNotifier(model, encryptionService.GetDecryptedValue, nil)
			webhookNotifier := not.(*GoogleChatNotifier)

			require.Nil(t, err)
			require.Equal(t, "ops", webhookNotifier.Name)
			require.Equal(t, "googlechat", webhookNotifier.Type)
			require.Equal(t, "http://google.com", webhookNotifier.URL)
		})
	})
}
