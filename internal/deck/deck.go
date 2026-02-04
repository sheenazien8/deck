package deck

import "fmt"

type Resource struct {
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

