package diagnostic

type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
)

type Diagnostic struct {
	Severity Severity `json:"severity"`
	Rule     string   `json:"rule"`
	Key      string   `json:"key,omitempty"`
	Message  string   `json:"message"`
	Path     string   `json:"path,omitempty"`
	Line     int      `json:"line,omitempty"`
}

func (diagnostic Diagnostic) IsError() bool { return diagnostic.Severity == Error }
