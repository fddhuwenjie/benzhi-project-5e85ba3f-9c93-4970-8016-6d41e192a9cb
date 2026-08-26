package domain

import (
	"regexp"
	"strings"
	"unicode"
)

var modelCodePattern = regexp.MustCompile(`^[\p{L}\p{N}_-]+$`)

type ProfileValidation struct {
	Title            string            `json:"title"`
	Objective        string            `json:"objective"`
	ModelCode        string            `json:"model_code"`
	PlannedCondition string            `json:"planned_condition"`
	Owner            string            `json:"owner"`
	Errors           []ValidationError `json:"errors"`
}

func ValidateProfile(title, objective, modelCode, condition, owner string) ProfileValidation {
	values := map[string]*string{"title": &title, "objective": &objective, "model_code": &modelCode, "planned_condition": &condition, "owner": &owner}
	for _, v := range values {
		*v = strings.TrimSpace(*v)
	}
	result := ProfileValidation{Title: title, Objective: objective, ModelCode: modelCode, PlannedCondition: condition, Owner: owner}
	check := func(field, value string, max int) {
		if value == "" {
			result.Errors = append(result.Errors, ValidationError{Field: field, Message: "不能为空"})
			return
		}
		if len([]rune(value)) > max {
			result.Errors = append(result.Errors, ValidationError{Field: field, Message: "长度超过限制"})
		}
		for _, r := range value {
			if unicode.IsControl(r) {
				result.Errors = append(result.Errors, ValidationError{Field: field, Message: "包含不允许的控制字符"})
				break
			}
		}
	}
	check("title", title, 120)
	check("objective", objective, 1000)
	check("model_code", modelCode, 120)
	if modelCode != "" && !modelCodePattern.MatchString(modelCode) {
		result.Errors = append(result.Errors, ValidationError{Field: "model_code", Message: "仅允许字母、数字、下划线和连字符"})
	}
	check("planned_condition", condition, 500)
	check("owner", owner, 120)
	return result
}
