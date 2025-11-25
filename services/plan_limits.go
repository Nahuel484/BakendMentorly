// BakendMentorly/services/plan_limits.go
package services

type PlanLimits struct {
	MaxPostulacionesMentor int // por mes
	MaxAnunciosEmprendedor int // anuncios activos
}

func GetPlanLimitsByID(planID int) PlanLimits {
	switch planID {
	case 0: // Gratis
		return PlanLimits{
			MaxPostulacionesMentor: 5,
			MaxAnunciosEmprendedor: 1,
		}
	case 1: // Pro
		return PlanLimits{
			MaxPostulacionesMentor: 30,
			MaxAnunciosEmprendedor: 10,
		}
	default: // Premium u otros planes superiores
		return PlanLimits{
			MaxPostulacionesMentor: 9999,
			MaxAnunciosEmprendedor: 9999,
		}
	}
}
