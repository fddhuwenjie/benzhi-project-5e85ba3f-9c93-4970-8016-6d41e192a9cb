package domain

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

var requiredChannelTypes = []string{"pressure", "strain", "torque"}

func (r *TestRelease) PutChannel(channel MeasurementChannel, now time.Time) error {
	if err := r.RequireStatus(StatusMeasurement); err != nil {
		return err
	}
	channel.ReleaseID = r.ID
	channel.ChannelType = strings.TrimSpace(channel.ChannelType)
	channel.SensorCode = strings.TrimSpace(channel.SensorCode)
	channel.EvidenceDigest = strings.TrimSpace(channel.EvidenceDigest)
	if channel.ID == "" || channel.SensorCode == "" || channel.EvidenceDigest == "" {
		return Invalid("channel", "通道 ID、传感器编号和校准证据摘要不能为空")
	}
	if !contains(requiredChannelTypes, channel.ChannelType) {
		return Invalid("channel_type", "仅支持 pressure、strain、torque")
	}
	if channel.RangeMax <= channel.RangeMin {
		return Invalid("range", "量程上限必须大于下限")
	}
	if channel.CalibratedAt.IsZero() || channel.ExpiresAt.IsZero() || !channel.ExpiresAt.After(channel.CalibratedAt) {
		return Invalid("calibration", "校准时间与有效期无效")
	}
	channel.VerificationStatus = channelStatus(channel, r.Envelope, now)
	for i := range r.Channels {
		if r.Channels[i].ID == channel.ID || r.Channels[i].ChannelType == channel.ChannelType {
			r.Channels[i] = channel
			return nil
		}
	}
	r.Channels = append(r.Channels, channel)
	sort.Slice(r.Channels, func(i, j int) bool { return r.Channels[i].ChannelType < r.Channels[j].ChannelType })
	return nil
}

type ChannelCheck struct {
	ID          string   `json:"id"`
	ChannelType string   `json:"channel_type"`
	Status      string   `json:"status"`
	Reasons     []string `json:"reasons,omitempty"`
	RequiredMin float64  `json:"required_min"`
	RequiredMax float64  `json:"required_max"`
}

func ValidateChannelBatch(channels []MeasurementChannel, envelope *OperatingEnvelope, now time.Time) ([]ChannelCheck, []ValidationError) {
	checks := make([]ChannelCheck, 0, len(channels))
	var format []ValidationError
	ids := map[string]bool{}
	types := map[string]bool{}
	sensors := map[string]bool{}
	if len(channels) == 0 {
		format = append(format, ValidationError{Field: "channels", Message: "至少登记一个通道"})
	}
	if len(channels) > 50 {
		format = append(format, ValidationError{Field: "channels", Message: "批次数量不得超过 50"})
	}
	for i, c := range channels {
		c.ChannelType = strings.TrimSpace(c.ChannelType)
		c.ID = strings.TrimSpace(c.ID)
		c.SensorCode = strings.TrimSpace(c.SensorCode)
		c.EvidenceDigest = strings.TrimSpace(c.EvidenceDigest)
		check := ChannelCheck{ID: c.ID, ChannelType: c.ChannelType, Status: "passed"}
		if c.ID == "" {
			format = append(format, ValidationError{Field: fmt.Sprintf("channels[%d].id", i), Message: "不能为空"})
		} else if ids[c.ID] {
			format = append(format, ValidationError{Field: fmt.Sprintf("channels[%d].id", i), Message: "批次内重复"})
		}
		ids[c.ID] = true
		if !contains(requiredChannelTypes, c.ChannelType) {
			format = append(format, ValidationError{Field: fmt.Sprintf("channels[%d].channel_type", i), Message: "未知通道类型"})
		} else if types[c.ChannelType] {
			format = append(format, ValidationError{Field: fmt.Sprintf("channels[%d].channel_type", i), Message: "必需类型只能登记一次"})
		}
		types[c.ChannelType] = true
		if c.SensorCode == "" {
			format = append(format, ValidationError{Field: fmt.Sprintf("channels[%d].sensor_code", i), Message: "不能为空"})
		} else if sensors[c.SensorCode] {
			format = append(format, ValidationError{Field: fmt.Sprintf("channels[%d].sensor_code", i), Message: "批次内重复占用"})
		}
		sensors[c.SensorCode] = true
		if c.RangeMax <= c.RangeMin {
			format = append(format, ValidationError{Field: fmt.Sprintf("channels[%d].range", i), Message: "量程上限必须大于下限"})
		}
		if c.CalibratedAt.IsZero() || c.ExpiresAt.IsZero() || !c.ExpiresAt.After(c.CalibratedAt) {
			format = append(format, ValidationError{Field: fmt.Sprintf("channels[%d].calibration", i), Message: "校准时间与有效期无效"})
		}
		if c.EvidenceDigest == "" {
			format = append(format, ValidationError{Field: fmt.Sprintf("channels[%d].evidence_digest", i), Message: "不能为空"})
		}
		if envelope != nil && contains(requiredChannelTypes, c.ChannelType) {
			check.RequiredMin, check.RequiredMax = requiredRange(c.ChannelType, envelope)
			check.Reasons = channelReasons(c, envelope, now)
			if len(check.Reasons) > 0 {
				check.Status = "blocked"
			}
		}
		checks = append(checks, check)
	}
	return checks, format
}

func channelReasons(channel MeasurementChannel, envelope *OperatingEnvelope, now time.Time) []string {
	var reasons []string
	if !contains(requiredChannelTypes, channel.ChannelType) {
		return []string{"未知通道类型"}
	}
	if envelope == nil {
		return []string{"缺少合格运行边界"}
	}
	if now.After(channel.ExpiresAt) {
		reasons = append(reasons, fmt.Sprintf("校准已过期（%s）", channel.ExpiresAt.UTC().Format(time.RFC3339)))
	}
	if channel.CalibratedAt.After(now) {
		reasons = append(reasons, "校准时间晚于当前时间")
	}
	min, max := requiredRange(channel.ChannelType, envelope)
	if channel.RangeMin > min || channel.RangeMax < max {
		reasons = append(reasons, fmt.Sprintf("量程不足，需要 [%.1f, %.1f]，安全裕度 %.1f", min, max, mathMin(channel.RangeMax-max, min-channel.RangeMin)))
	}
	return reasons
}
func mathMin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func (r *TestRelease) ReplaceChannels(channels []MeasurementChannel, now time.Time) ([]ChannelCheck, []ValidationError) {
	if err := r.RequireStatus(StatusMeasurement); err != nil {
		return nil, []ValidationError{{Field: "status", Message: err.Error()}}
	}
	checks, format := ValidateChannelBatch(channels, r.Envelope, now)
	if len(format) > 0 {
		return checks, format
	}
	r.Channels = make([]MeasurementChannel, 0, len(channels))
	for i, c := range channels {
		c.ReleaseID = r.ID
		c.ChannelType = strings.TrimSpace(c.ChannelType)
		c.ID = strings.TrimSpace(c.ID)
		c.SensorCode = strings.TrimSpace(c.SensorCode)
		c.EvidenceDigest = strings.TrimSpace(c.EvidenceDigest)
		c.RequiredMin, c.RequiredMax = checks[i].RequiredMin, checks[i].RequiredMax
		c.VerificationStatus = checks[i].Status
		c.VerificationReason = strings.Join(checks[i].Reasons, "；")
		r.Channels = append(r.Channels, c)
	}
	sort.Slice(r.Channels, func(i, j int) bool { return r.Channels[i].ChannelType < r.Channels[j].ChannelType })
	return checks, nil
}

func (r *TestRelease) RecheckChannels(now time.Time) []ChannelCheck {
	checks := make([]ChannelCheck, 0, len(r.Channels))
	for _, c := range r.Channels {
		min, max := requiredRange(c.ChannelType, r.Envelope)
		reasons := channelReasons(c, r.Envelope, now)
		checks = append(checks, ChannelCheck{ID: c.ID, ChannelType: c.ChannelType, Status: map[bool]string{true: "passed", false: "blocked"}[len(reasons) == 0], Reasons: reasons, RequiredMin: min, RequiredMax: max})
	}
	return checks
}

func channelStatus(channel MeasurementChannel, envelope *OperatingEnvelope, now time.Time) string {
	if envelope == nil || now.After(channel.ExpiresAt) || channel.CalibratedAt.After(now) {
		return "blocked"
	}
	neededMin, neededMax := requiredRange(channel.ChannelType, envelope)
	if channel.RangeMin > neededMin || channel.RangeMax < neededMax {
		return "blocked"
	}
	return "passed"
}

func requiredRange(channelType string, envelope *OperatingEnvelope) (float64, float64) {
	switch channelType {
	case "pressure":
		return 0, envelope.SpeedMax
	case "strain":
		return -envelope.LoadLimit, envelope.LoadLimit
	case "torque":
		return -envelope.LoadLimit / 2, envelope.LoadLimit / 2
	default:
		return 0, 0
	}
}

func VerifyChannels(channels []MeasurementChannel, envelope *OperatingEnvelope, now time.Time) []string {
	if envelope == nil {
		return []string{"缺少合格运行边界，无法核验测量通道"}
	}
	byType := make(map[string]MeasurementChannel, len(channels))
	for _, channel := range channels {
		byType[channel.ChannelType] = channel
	}
	var reasons []string
	for _, required := range requiredChannelTypes {
		channel, ok := byType[required]
		if !ok {
			reasons = append(reasons, "缺少必需通道 "+required)
			continue
		}
		if now.After(channel.ExpiresAt) {
			reasons = append(reasons, fmt.Sprintf("通道 %s 校准已过期", required))
			continue
		}
		if channel.CalibratedAt.After(now) {
			reasons = append(reasons, fmt.Sprintf("通道 %s 校准时间晚于核验时间", required))
			continue
		}
		min, max := requiredRange(required, envelope)
		if channel.RangeMin > min || channel.RangeMax < max {
			reasons = append(reasons, fmt.Sprintf("通道 %s 量程 [%.1f, %.1f] 未覆盖要求 [%.1f, %.1f]", required, channel.RangeMin, channel.RangeMax, min, max))
		}
		if channel.EvidenceDigest == "" {
			reasons = append(reasons, "通道 "+required+" 缺少校准证据摘要")
		}
	}
	return reasons
}

func (r *TestRelease) ConfirmChannels(now time.Time) error {
	if err := r.RequireStatus(StatusMeasurement); err != nil {
		return err
	}
	if reasons := VerifyChannels(r.Channels, r.Envelope, now); len(reasons) > 0 {
		return Invalid("channels", strings.Join(reasons, "；"))
	}
	r.Status = StatusInterlock
	return nil
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
