package app

import (
	"encoding/json"
	"regexp"
	"strings"
)

var mobileNumber = regexp.MustCompile(`1[3-9][0-9]{9}`)

func maskPhoneText(value string) string {
	return mobileNumber.ReplaceAllStringFunc(value, maskPhone)
}

func maskPhone(phone string) string {
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + strings.Repeat("*", len(phone)-7) + phone[len(phone)-4:]
}

func maskValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		masked := make(map[string]any, len(typed))
		for key, item := range typed {
			if key == "phone" || key == "purePhoneNumber" || key == "phoneNumber" {
				if phone, ok := item.(string); ok {
					masked[key] = maskPhone(phone)
					continue
				}
			}
			masked[key] = maskValue(item)
		}
		return masked
	case []any:
		masked := make([]any, len(typed))
		for i, item := range typed {
			masked[i] = maskValue(item)
		}
		return masked
	case string:
		return maskPhoneText(typed)
	default:
		return value
	}
}

func maskStaffData(value any) any {
	raw, err := json.Marshal(value)
	if err != nil {
		return value
	}
	var decoded any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		return value
	}
	return maskValue(decoded)
}

func maskAuditMap(value map[string]interface{}) map[string]interface{} {
	if value == nil {
		return nil
	}
	return maskValue(value).(map[string]interface{})
}
