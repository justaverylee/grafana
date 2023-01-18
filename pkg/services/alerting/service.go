package alerting

import (
	"context"
	"fmt"

	"github.com/grafana/grafana/pkg/infra/db"
	"github.com/grafana/grafana/pkg/services/encryption"
	"github.com/grafana/grafana/pkg/services/notifications"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/util"
)

type AlertNotificationService struct {
	SQLStore            AlertNotificationStore
	EncryptionService   encryption.Internal
	NotificationService *notifications.NotificationService
}

func ProvideService(store db.DB, encryptionService encryption.Internal,
	notificationService *notifications.NotificationService) *AlertNotificationService {
	s := &AlertNotificationService{
		SQLStore:            &sqlStore{db: store},
		EncryptionService:   encryptionService,
		NotificationService: notificationService,
	}

	return s
}

func (s *AlertNotificationService) GetAlertNotifications(ctx context.Context, query *GetAlertNotificationsQuery) error {
	return s.SQLStore.GetAlertNotifications(ctx, query)
}

func (s *AlertNotificationService) CreateAlertNotificationCommand(ctx context.Context, cmd *CreateAlertNotificationCommand) error {
	if util.IsShortUIDTooLong(cmd.Uid) {
		return ValidationError{Reason: "Invalid UID: Must be 40 characters or less"}
	}

	var err error
	cmd.EncryptedSecureSettings, err = s.EncryptionService.EncryptJsonData(ctx, cmd.SecureSettings, setting.SecretKey)
	if err != nil {
		return err
	}

	model := AlertNotification{
		Name:     cmd.Name,
		Type:     cmd.Type,
		Settings: cmd.Settings,
	}

	if err := s.validateAlertNotification(ctx, &model, cmd.SecureSettings); err != nil {
		return err
	}

	return s.SQLStore.CreateAlertNotificationCommand(ctx, cmd)
}

func (s *AlertNotificationService) UpdateAlertNotification(ctx context.Context, cmd *UpdateAlertNotificationCommand) error {
	if util.IsShortUIDTooLong(cmd.Uid) {
		return ValidationError{Reason: "Invalid UID: Must be 40 characters or less"}
	}

	var err error
	cmd.EncryptedSecureSettings, err = s.EncryptionService.EncryptJsonData(ctx, cmd.SecureSettings, setting.SecretKey)
	if err != nil {
		return err
	}

	model := AlertNotification{
		Id:       cmd.Id,
		OrgId:    cmd.OrgId,
		Name:     cmd.Name,
		Type:     cmd.Type,
		Settings: cmd.Settings,
	}

	if err := s.validateAlertNotification(ctx, &model, cmd.SecureSettings); err != nil {
		return err
	}

	return s.SQLStore.UpdateAlertNotification(ctx, cmd)
}

func (s *AlertNotificationService) DeleteAlertNotification(ctx context.Context, cmd *DeleteAlertNotificationCommand) error {
	return s.SQLStore.DeleteAlertNotification(ctx, cmd)
}

func (s *AlertNotificationService) GetAllAlertNotifications(ctx context.Context, query *GetAllAlertNotificationsQuery) error {
	return s.SQLStore.GetAllAlertNotifications(ctx, query)
}

func (s *AlertNotificationService) GetOrCreateAlertNotificationState(ctx context.Context, cmd *GetOrCreateNotificationStateQuery) error {
	return s.SQLStore.GetOrCreateAlertNotificationState(ctx, cmd)
}

func (s *AlertNotificationService) SetAlertNotificationStateToCompleteCommand(ctx context.Context, cmd *SetAlertNotificationStateToCompleteCommand) error {
	return s.SQLStore.SetAlertNotificationStateToCompleteCommand(ctx, cmd)
}

func (s *AlertNotificationService) SetAlertNotificationStateToPendingCommand(ctx context.Context, cmd *SetAlertNotificationStateToPendingCommand) error {
	return s.SQLStore.SetAlertNotificationStateToPendingCommand(ctx, cmd)
}

func (s *AlertNotificationService) GetAlertNotificationsWithUid(ctx context.Context, query *GetAlertNotificationsWithUidQuery) error {
	return s.SQLStore.GetAlertNotificationsWithUid(ctx, query)
}

func (s *AlertNotificationService) UpdateAlertNotificationWithUid(ctx context.Context, cmd *UpdateAlertNotificationWithUidCommand) error {
	if util.IsShortUIDTooLong(cmd.Uid) || util.IsShortUIDTooLong(cmd.NewUid) {
		return ValidationError{Reason: "Invalid UID: Must be 40 characters or less"}
	}

	return s.SQLStore.UpdateAlertNotificationWithUid(ctx, cmd)
}

func (s *AlertNotificationService) DeleteAlertNotificationWithUid(ctx context.Context, cmd *DeleteAlertNotificationWithUidCommand) error {
	return s.SQLStore.DeleteAlertNotificationWithUid(ctx, cmd)
}

func (s *AlertNotificationService) GetAlertNotificationsWithUidToSend(ctx context.Context, query *GetAlertNotificationsWithUidToSendQuery) error {
	return s.SQLStore.GetAlertNotificationsWithUidToSend(ctx, query)
}

func (s *AlertNotificationService) createNotifier(ctx context.Context, model *AlertNotification, secureSettings map[string]string) (Notifier, error) {
	secureSettingsMap := map[string]string{}

	if model.Id > 0 {
		query := &GetAlertNotificationsQuery{
			OrgId: model.OrgId,
			Id:    model.Id,
		}
		if err := s.SQLStore.GetAlertNotifications(ctx, query); err != nil {
			return nil, err
		}

		if query.Result == nil {
			return nil, fmt.Errorf("unable to find the alert notification")
		}

		if query.Result.SecureSettings != nil {
			var err error
			secureSettingsMap, err = s.EncryptionService.DecryptJsonData(ctx, query.Result.SecureSettings, setting.SecretKey)
			if err != nil {
				return nil, err
			}
		}
	}

	for k, v := range secureSettings {
		secureSettingsMap[k] = v
	}

	var err error
	model.SecureSettings, err = s.EncryptionService.EncryptJsonData(ctx, secureSettingsMap, setting.SecretKey)
	if err != nil {
		return nil, err
	}

	notifier, err := InitNotifier(model, s.EncryptionService.GetDecryptedValue, s.NotificationService)
	if err != nil {
		logger.Error("Failed to create notifier", "error", err.Error())
		return nil, err
	}

	return notifier, nil
}

func (s *AlertNotificationService) validateAlertNotification(ctx context.Context, model *AlertNotification, secureSettings map[string]string) error {
	_, err := s.createNotifier(ctx, model, secureSettings)
	return err
}
