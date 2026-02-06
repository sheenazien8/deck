package deck

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/a-h/templ"
	"github.com/sheenazien8/deck/internal/deck/ui"
)

type Resource struct {
	Name  string
	Model any
	Form  []Field
	Table []Column
}

type Field interface {
	Name() string
	Type() string
	Options() FieldOptions
}

type FieldOptions struct {
	Required bool
	Label    string
	Default  any
}

type Column interface {
	Name() string
	Render(value any) string
}

type BaseField struct {
	name    string
	kind    string
	options FieldOptions
}

func (f *BaseField) Name() string {
	return f.name
}

func (f *BaseField) Type() string {
	return f.kind
}

func (f *BaseField) Options() FieldOptions {
	return f.options
}

func (f *BaseField) Required() *BaseField {
	f.options.Required = true
	return f
}

func (f *BaseField) Label(label string) *BaseField {
	f.options.Label = label
	return f
}

func Text(name string) *BaseField {
	return &BaseField{
		name: name,
		kind: "text",
	}
}

func Email(name string) *BaseField {
	return &BaseField{
		name: name,
		kind: "email",
	}
}

func (f *BaseField) Bind(modelType reflect.Type) BoundField {
	field, ok := modelType.FieldByNameFunc(func(n string) bool {
		return strings.EqualFold(n, f.name)
	})
	if !ok {
		panic("field not found: " + f.name)
	}

	return BoundField{
		Name:  f.name,
		Label: f.options.Label,
		Get: func(model any) any {
			return reflect.ValueOf(model).Elem().FieldByIndex(field.Index).String()
		},
		Set: func(model any, value any) {
			reflect.ValueOf(model).Elem().FieldByIndex(field.Index).SetString(value.(string))
		},
		Render: func(v any) templ.Component {
			return ui.Input(f.name, f.name, "", f.kind, map[string]string{})
		},
	}
}

type BoundField struct {
	Name   string
	Label  string
	Get    func(any) any
	Set    func(any, any)
	Render func(any) templ.Component
}

type BaseColumn struct {
	name string
}

func (c *BaseColumn) Name() string {
	return c.name
}

func (c *BaseColumn) Render(value any) string {
	return fmt.Sprintf("%v", value)
}

func Col(name string) *BaseColumn {
	return &BaseColumn{name: name}
}
