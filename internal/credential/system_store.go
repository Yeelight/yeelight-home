package credential

import (
	"errors"
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

type SystemStore struct {
	service string
}

func NewSystemStore(service string) SystemStore {
	service = strings.TrimSpace(service)
	if service == "" {
		service = "yeelight-home"
	}
	return SystemStore{service: service}
}

func (store SystemStore) Save(record TokenRecord) error {
	profile, err := normalizeProfile(record.Profile)
	if err != nil {
		return err
	}
	if runtime.GOOS != "darwin" {
		return errors.New("system credential store is not implemented on this platform")
	}
	account := store.accountName(profile)
	if err := exec.Command("security", "delete-generic-password", "-s", store.service, "-a", account).Run(); err != nil {
		// 删除旧条目失败通常表示首次保存，没有必要中断。
	}
	if err := exec.Command("security", "add-generic-password", "-U", "-s", store.service, "-a", account, "-w", record.AccessToken).Run(); err != nil {
		return fmt.Errorf("save token to system credential store: %w", err)
	}
	return nil
}

func (store SystemStore) Load(profile string) (TokenRecord, bool, error) {
	normalized, err := normalizeProfile(profile)
	if err != nil {
		return TokenRecord{}, false, err
	}
	if runtime.GOOS != "darwin" {
		return TokenRecord{}, false, errors.New("system credential store is not implemented on this platform")
	}
	output, err := exec.Command("security", "find-generic-password", "-s", store.service, "-a", store.accountName(normalized), "-w").Output()
	if err != nil {
		return TokenRecord{}, false, nil
	}
	return TokenRecord{Profile: normalized, AccessToken: strings.TrimSpace(string(output))}, true, nil
}

func (store SystemStore) Delete(profile string) error {
	normalized, err := normalizeProfile(profile)
	if err != nil {
		return err
	}
	if runtime.GOOS != "darwin" {
		return errors.New("system credential store is not implemented on this platform")
	}
	if err := exec.Command("security", "delete-generic-password", "-s", store.service, "-a", store.accountName(normalized)).Run(); err != nil {
		return nil
	}
	return nil
}

func (store SystemStore) accountName(profile string) string {
	return store.service + ":" + strings.TrimSpace(profile)
}
