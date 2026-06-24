package auth

import (
	"context"
	"errors"
	"time"
)

type QRClient interface {
	Create(ctx context.Context, device string) (QRInfo, error)
	Check(ctx context.Context, qrCodeID string) (QRInfo, error)
}

type QRLoginOptions struct {
	Client       QRClient
	Device       string
	HouseID      string
	NoWait       bool
	PollInterval time.Duration
	Timeout      time.Duration
	Sleep        func(context.Context, time.Duration) error
	OnCreated    func(QRLoginResult)
}

type QRLoginResult struct {
	OK          bool              `json:"ok"`
	Status      string            `json:"status"`
	QRCodeID    string            `json:"qrCodeId"`
	Device      string            `json:"device"`
	Payload     string            `json:"payload"`
	ExpireAt    int64             `json:"expireAt,omitempty"`
	Credentials *LoginCredentials `json:"credentials"`
}

func RunQRLoginFlow(ctx context.Context, options QRLoginOptions) (QRLoginResult, error) {
	client := options.Client
	if client == nil {
		client = NewQRLoginClient("", nil)
	}
	device := NormalizeDeviceMAC(options.Device)
	if device == "" {
		return QRLoginResult{}, errors.New("QR login device is required")
	}
	created, err := client.Create(ctx, device)
	if err != nil {
		return QRLoginResult{}, err
	}
	if created.QRCodeID == "" {
		return QRLoginResult{}, errors.New("QR login API did not return qrCodeId")
	}
	result := QRLoginResult{
		OK:       true,
		Status:   firstNonEmpty(created.Status, "CREATED"),
		QRCodeID: created.QRCodeID,
		Device:   device,
		Payload:  BuildQRPayload(created.QRCodeID, device, options.HouseID),
		ExpireAt: created.ExpireAt,
	}
	if options.NoWait {
		return result, nil
	}
	if options.OnCreated != nil {
		options.OnCreated(result)
	}

	timeout := options.Timeout
	if timeout <= 0 {
		timeout = time.Duration(DefaultQRLoginTimeoutMS) * time.Millisecond
	}
	pollInterval := options.PollInterval
	if pollInterval <= 0 {
		pollInterval = time.Duration(DefaultQRLoginPollIntervalMS) * time.Millisecond
	}
	sleep := options.Sleep
	if sleep == nil {
		sleep = sleepContext
	}
	deadline := time.Now().Add(timeout)
	for !time.Now().After(deadline) {
		if err := sleep(ctx, pollInterval); err != nil {
			return QRLoginResult{}, err
		}
		checked, err := client.Check(ctx, created.QRCodeID)
		if err != nil {
			return QRLoginResult{}, err
		}
		result.Status = firstNonEmpty(checked.Status, result.Status)
		if checked.ExpireAt != 0 {
			result.ExpireAt = checked.ExpireAt
		}
		if IsQRExpiredStatus(result.Status) || isExpiredAt(result.ExpireAt) {
			return QRLoginResult{}, errors.New("QR code expired")
		}
		if !IsQRLoginStatus(result.Status) {
			continue
		}
		credentials := ExtractQRLoginCredentials(checked)
		if credentials.Authorization == "" {
			return QRLoginResult{}, errors.New("QR login response did not contain access token")
		}
		result.Credentials = &credentials
		return result, nil
	}
	return QRLoginResult{}, errors.New("waiting for QR login timed out")
}

func sleepContext(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func isExpiredAt(expireAt int64) bool {
	return expireAt > 0 && expireAt <= time.Now().UnixMilli()
}
