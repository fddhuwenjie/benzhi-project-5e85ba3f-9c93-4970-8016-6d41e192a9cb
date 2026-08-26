package domain

type Status string

const (
	StatusDraft         Status = "draft"
	StatusMeasurement   Status = "measurement_verification"
	StatusInterlock     Status = "interlock_drill"
	StatusWitness       Status = "witness_review"
	StatusAuthorization Status = "pending_authorization"
	StatusReleased      Status = "released"
)

func (s Status) Label() string {
	switch s {
	case StatusDraft:
		return "草案与边界评估"
	case StatusMeasurement:
		return "测量链核验"
	case StatusInterlock:
		return "联锁演练"
	case StatusWitness:
		return "独立见证"
	case StatusAuthorization:
		return "待授权"
	case StatusReleased:
		return "已放行"
	default:
		return "未知状态"
	}
}

func (s Status) Mutable() bool { return s != StatusReleased }

type Role string

const (
	RoleOwner      Role = "owner"
	RoleEngineer   Role = "engineer"
	RoleWitness    Role = "witness"
	RoleAuthorizer Role = "authorizer"
)

func RequireRole(actual Role, allowed ...Role) error {
	for _, role := range allowed {
		if actual == role {
			return nil
		}
	}
	return ErrUnauthorizedRole
}
