package manager

import "encoding/json"

type Template interface {
	TemplateName() string
	Build() (kind, apiVersion, namespace, name string, spec json.RawMessage, err error)
	Decompile(spec json.RawMessage) error
}
