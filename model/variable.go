package model

import "time"

type Variable struct {
	Name    string     `json:"name"`
	VarType string     `json:"type,omitempty"`
	Value   any        `json:"value,omitempty"`
	Created *time.Time `json:"created,omitempty"`
	Updated *time.Time `json:"updated,omitempty"`
}
