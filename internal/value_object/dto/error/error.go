package errordto

type Details struct {
	Fields map[string]FieldDetail `json:"fields,omitempty"`
}

type FieldDetail struct {
	Key     string `json:"key,omitempty"` // i18n key for this field error
	Rule    string `json:"rule"`
	Param   string `json:"param,omitempty"`
	Message string `json:"message,omitempty"` // English fallback
}
