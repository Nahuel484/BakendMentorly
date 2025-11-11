package models

type Plan struct {
	ID          int     `json:"id_plan"`
	Nombre      string  `json:"nombre_plan" binding:"required"`
	Precio      float64 `json:"precio" binding:"required,gte=0"`
	Descripcion string  `json:"descripcion"`
	Activo      bool    `json:"activo"`
}
