package domain

import (
	"fmt"
	"math"
	"time"
)

const (
	maxTunnelSpeed        = 350.0
	maxAbsoluteAngle      = 25.0
	maxStructuralLoad     = 120.0
	maxAllowedTemperature = 85.0
)

func EvaluateEnvelope(releaseID string, value OperatingEnvelope) OperatingEnvelope {
	value.ReleaseID = releaseID
	value.Violations = make([]string, 0)
	value.Checks = make([]EnvelopeCheck, 0)
	check := func(field, rule, threshold string, candidate, margin float64, passed bool, message string) {
		value.Checks = append(value.Checks, EnvelopeCheck{Field: field, Rule: rule, Threshold: threshold, Candidate: candidate, Margin: margin, Passed: passed})
		if !passed {
			value.Violations = append(value.Violations, message)
		}
	}
	finite := []struct {
		name string
		v    float64
	}{
		{"speed_min", value.SpeedMin}, {"speed_max", value.SpeedMax},
		{"attack_angle_min", value.AttackAngleMin}, {"attack_angle_max", value.AttackAngleMax},
		{"load_limit", value.LoadLimit}, {"temperature_limit", value.TemperatureLimit},
	}
	for _, field := range finite {
		if math.IsNaN(field.v) || math.IsInf(field.v, 0) {
			check(field.name, "finite", "有限数值", field.v, 0, false, field.name+" 必须是有限数值")
		}
	}
	check("speed_min", "lower_bound", ">= 0 m/s", value.SpeedMin, value.SpeedMin, true, "")
	if value.SpeedMin < 0 {
		check("speed_min", "lower_bound", ">= 0 m/s", value.SpeedMin, value.SpeedMin, false, "最低速度不得小于 0 m/s")
	}
	check("speed_max", "tunnel_limit", "<= 350 m/s", value.SpeedMax, maxTunnelSpeed-value.SpeedMax, value.SpeedMax <= maxTunnelSpeed, fmt.Sprintf("最高速度 %.1f m/s 超过风洞限制 %.1f m/s", value.SpeedMax, maxTunnelSpeed))
	check("speed_range", "ascending", "speed_max > speed_min", value.SpeedMax, value.SpeedMax-value.SpeedMin, value.SpeedMax > value.SpeedMin, "最高速度必须大于最低速度")
	check("attack_angle_range", "ascending", "attack_angle_max > attack_angle_min", value.AttackAngleMax, value.AttackAngleMax-value.AttackAngleMin, value.AttackAngleMax > value.AttackAngleMin, "最大攻角必须大于最小攻角")
	angleMargin := maxAbsoluteAngle - math.Max(math.Abs(value.AttackAngleMin), math.Abs(value.AttackAngleMax))
	check("attack_angle", "hard_limit", "abs <= 25 deg", math.Max(math.Abs(value.AttackAngleMin), math.Abs(value.AttackAngleMax)), angleMargin, angleMargin >= 0, "攻角绝对值不得超过 25 deg")
	loadMargin := maxStructuralLoad - value.LoadLimit
	check("load_limit", "hard_limit", "0 < load <= 120 kN", value.LoadLimit, loadMargin, value.LoadLimit > 0 && value.LoadLimit <= maxStructuralLoad, "载荷限制必须在 (0, 120] kN 内")
	tempMargin := math.Min(value.TemperatureLimit-10, maxAllowedTemperature-value.TemperatureLimit)
	check("temperature_limit", "hard_limit", "10 <= temperature <= 85 C", value.TemperatureLimit, tempMargin, value.TemperatureLimit >= 10 && value.TemperatureLimit <= maxAllowedTemperature, "温度限制必须在 [10, 85] C 内")
	if len(value.Violations) == 0 {
		value.EvaluationStatus = "passed"
	} else {
		value.EvaluationStatus = "blocked"
	}
	return value
}

func (r *TestRelease) SetEnvelope(value OperatingEnvelope) error {
	if err := r.RequireStatus(StatusDraft); err != nil {
		return err
	}
	evaluated := EvaluateEnvelope(r.ID, value)
	r.Envelope = &evaluated
	if evaluated.EvaluationStatus != "passed" {
		return Invalid("envelope", fmt.Sprintf("运行边界被阻断：%v", evaluated.Violations))
	}
	r.Status = StatusMeasurement
	return nil
}

func (r *TestRelease) TrialEnvelope(value OperatingEnvelope, now time.Time) OperatingEnvelope {
	evaluated := EvaluateEnvelope(r.ID, value)
	previous := OperatingEnvelope{SpeedMin: 20, SpeedMax: 180, AttackAngleMin: -10, AttackAngleMax: 15, LoadLimit: 80, TemperatureLimit: 60}
	if r.Envelope != nil {
		previous = *r.Envelope
	}
	trial := &EnvelopeTrial{Candidate: evaluated, CreatedAt: now.UTC(), Compared: []FieldDifference{}, RangeImpact: map[string]string{}}
	add := func(field string, old, candidate any, direction float64) {
		change := "unchanged"
		if direction > 0 {
			change = "expanded"
		}
		if direction < 0 {
			change = "narrowed"
		}
		trial.Compared = append(trial.Compared, FieldDifference{Field: field, Previous: old, Candidate: candidate, Change: change})
	}
	add("speed_min", previous.SpeedMin, value.SpeedMin, previous.SpeedMin-value.SpeedMin)
	add("speed_max", previous.SpeedMax, value.SpeedMax, value.SpeedMax-previous.SpeedMax)
	add("attack_angle_min", previous.AttackAngleMin, value.AttackAngleMin, previous.AttackAngleMin-value.AttackAngleMin)
	add("attack_angle_max", previous.AttackAngleMax, value.AttackAngleMax, value.AttackAngleMax-previous.AttackAngleMax)
	add("load_limit", previous.LoadLimit, value.LoadLimit, value.LoadLimit-previous.LoadLimit)
	add("temperature_limit", previous.TemperatureLimit, value.TemperatureLimit, value.TemperatureLimit-previous.TemperatureLimit)
	for _, typ := range []string{"pressure", "strain", "torque"} {
		oldMin, oldMax := requiredRange(typ, &previous)
		newMin, newMax := requiredRange(typ, &evaluated)
		trial.RangeImpact[typ] = fmt.Sprintf("%.1f..%.1f -> %.1f..%.1f", oldMin, oldMax, newMin, newMax)
	}
	evaluated.LastTrial = trial
	return evaluated
}
