package deck

import (
	"context"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/gofiber/fiber/v2"
	"github.com/sheenazien8/deck/internal/deck/ui"
	"gorm.io/gorm"
)

// QueryParams holds pagination and filter parameters
type QueryParams struct {
	Page     int
	PerPage  int
	Search   string
	Filters  map[string]string
	OrderBy  string
	OrderDir string
}

// PaginatedResult holds paginated data with metadata
type PaginatedResult[T any] struct {
	Data       []T
	Page       int
	PerPage    int
	Total      int64
	TotalPages int
	HasNext    bool
	HasPrev    bool
}

type Resource[T any] struct {
	Name           string
	Query          func() ([]T, error)
	QueryPaginated func(QueryParams) (*PaginatedResult[T], error)
	FindBy         func(id uint) (*T, error)
	Create         func(*T) error
	Update         func(*T) error
	Delete         func(id uint) error
	Table          *Table[*T]
	Fields         []Field
	Filters        []Filter
	db             *gorm.DB
	perPage        int
}

// Filter represents a filterable field
type Filter struct {
	Name     string
	Label    string
	Type     string
	Options  []FilterOption
	Field    string
	Operator string
}

// FilterOption for select filters
type FilterOption struct {
	Value string
	Label string
}

// NewResource creates a new Resource with automatic GORM CRUD operations
// Usage: deck.NewResource[models.User](db, "users")
func NewResource[T any](db *gorm.DB, name string) *Resource[T] {
	r := &Resource[T]{
		Name:    name,
		db:      db,
		perPage: 10,
	}

	r.Query = func() ([]T, error) {
		var items []T
		if err := db.Find(&items).Error; err != nil {
			return nil, err
		}
		return items, nil
	}

	r.QueryPaginated = func(params QueryParams) (*PaginatedResult[T], error) {
		var items []T
		var total int64

		query := db.Model(new(T))

		if params.Search != "" {
			var searchConditions []string
			var searchValues []any

			typ := reflect.TypeOf(*new(T))
			for i := range typ.NumField() {
				field := typ.Field(i)
				if field.Type.Kind() == reflect.String {
					searchConditions = append(searchConditions, fmt.Sprintf("%s LIKE ?", field.Name))
					searchValues = append(searchValues, "%"+params.Search+"%")
				}
			}

			if len(searchConditions) > 0 {
				query = query.Where(strings.Join(searchConditions, " OR "), searchValues...)
			}
		}

		for key, value := range params.Filters {
			if value != "" {
				for _, filter := range r.Filters {
					if filter.Name == key {
						switch filter.Operator {
						case "LIKE":
							query = query.Where(filter.Field+" LIKE ?", "%"+value+"%")
						default:
							query = query.Where(filter.Field+" = ?", value)
						}
						break
					}
				}
			}
		}

		if err := query.Count(&total).Error; err != nil {
			return nil, err
		}

		orderBy := "id DESC"
		if params.OrderBy != "" {
			orderDir := "ASC"
			if params.OrderDir == "desc" {
				orderDir = "DESC"
			}
			orderBy = params.OrderBy + " " + orderDir
		}
		query = query.Order(orderBy)

		page := max(params.Page, 1)
		perPage := params.PerPage
		if perPage < 1 {
			perPage = r.perPage
		}

		offset := (page - 1) * perPage
		if err := query.Offset(offset).Limit(perPage).Find(&items).Error; err != nil {
			return nil, err
		}

		totalPages := int((total + int64(perPage) - 1) / int64(perPage))

		return &PaginatedResult[T]{
			Data:       items,
			Page:       page,
			PerPage:    perPage,
			Total:      total,
			TotalPages: totalPages,
			HasNext:    page < totalPages,
			HasPrev:    page > 1,
		}, nil
	}

	r.FindBy = func(id uint) (*T, error) {
		var item T
		if err := db.First(&item, id).Error; err != nil {
			return nil, err
		}
		return &item, nil
	}

	r.Create = func(item *T) error {
		return db.Create(item).Error
	}

	r.Update = func(item *T) error {
		return db.Save(item).Error
	}

	r.Delete = func(id uint) error {
		var item T
		return db.Delete(&item, id).Error
	}

	return r
}

// WithQuery allows overriding the default Query operation
func (r *Resource[T]) WithQuery(query func() ([]T, error)) *Resource[T] {
	r.Query = query
	return r
}

// WithQueryPaginated allows overriding the default QueryPaginated operation
func (r *Resource[T]) WithQueryPaginated(query func(QueryParams) (*PaginatedResult[T], error)) *Resource[T] {
	r.QueryPaginated = query
	return r
}

// WithFindBy allows overriding the default FindBy operation
func (r *Resource[T]) WithFindBy(findBy func(id uint) (*T, error)) *Resource[T] {
	r.FindBy = findBy
	return r
}

// WithCreate allows overriding the default Create operation
func (r *Resource[T]) WithCreate(create func(*T) error) *Resource[T] {
	r.Create = create
	return r
}

// WithUpdate allows overriding the default Update operation
func (r *Resource[T]) WithUpdate(update func(*T) error) *Resource[T] {
	r.Update = update
	return r
}

// WithDelete allows overriding the default Delete operation
func (r *Resource[T]) WithDelete(delete func(id uint) error) *Resource[T] {
	r.Delete = delete
	return r
}

// WithTable sets the table configuration
func (r *Resource[T]) WithTable(table *Table[*T]) *Resource[T] {
	r.Table = table
	return r
}

// WithFields sets the form fields
func (r *Resource[T]) WithFields(fields ...Field) *Resource[T] {
	r.Fields = fields
	return r
}

// WithFilters sets the filterable fields
func (r *Resource[T]) WithFilters(filters ...Filter) *Resource[T] {
	r.Filters = filters
	return r
}

// WithPerPage sets the default items per page
func (r *Resource[T]) WithPerPage(perPage int) *Resource[T] {
	r.perPage = perPage
	return r
}

// RenderTable renders the table component with data from the Query
func (r *Resource[T]) RenderTable() (templ.Component, error) {
	data, err := r.Query()
	if err != nil {
		return nil, err
	}

	rows := make([]*T, len(data))
	for i := range data {
		rows[i] = &data[i]
	}

	headers := make([]string, len(r.Table.Columns))
	for i, col := range r.Table.Columns {
		headers[i] = col.Label
	}

	if len(r.Table.Actions) > 0 {
		headers = append(headers, "Actions")
	}

	// Debug: print headers to help diagnose oversized header issue
	fmt.Printf("deck: RenderTable headers=%#v\n", headers)

	renderRow := func(item *T) templ.Component {
		return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
			for _, col := range r.Table.Columns {
				value := col.Value(item)
				component := col.Render(value)
				if err := component.Render(ctx, w); err != nil {
					return err
				}
			}

			if len(r.Table.Actions) > 0 {
				actionsCell := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
					w.Write([]byte(`<td class="px-3 py-2 text-sm"><div class="flex gap-2">`))
					for _, action := range r.Table.Actions {
						if err := action.Render(item).Render(ctx, w); err != nil {
							return err
						}
					}
					w.Write([]byte(`</div></td>`))
					return nil
				})
				if err := actionsCell.Render(ctx, w); err != nil {
					return err
				}
			}

			return nil
		})
	}

	return ui.Table(headers, rows, renderRow), nil
}

// RegisterRoutes registers CRUD routes for this resource
func (r *Resource[T]) RegisterRoutes(app *fiber.App) {
	basePath := "/" + r.Name

	app.Get(basePath, func(c *fiber.Ctx) error {
		ctx := templ.InitializeContext(c.UserContext())

		page, _ := strconv.Atoi(c.Query("page", "1"))
		perPage, _ := strconv.Atoi(c.Query("per_page", strconv.Itoa(r.perPage)))
		search := c.Query("search", "")

		filters := make(map[string]string)
		for _, filter := range r.Filters {
			if val := c.Query(filter.Name); val != "" {
				filters[filter.Name] = val
			}
		}

		params := QueryParams{
			Page:    page,
			PerPage: perPage,
			Search:  search,
			Filters: filters,
		}

		result, err := r.QueryPaginated(params)
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		rows := make([]*T, len(result.Data))
		for i := range result.Data {
			rows[i] = &result.Data[i]
		}

		headers := make([]string, len(r.Table.Columns))
		for i, col := range r.Table.Columns {
			headers[i] = col.Label
		}

		if len(r.Table.Actions) > 0 {
			headers = append(headers, "Actions")
		}

		// Debug: print headers when serving list page
		fmt.Printf("deck: RegisterRoutes headers=%#v\n", headers)

		renderRow := func(item *T) templ.Component {
			return templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
				for _, col := range r.Table.Columns {
					value := col.Value(item)
					component := col.Render(value)
					if err := component.Render(ctx, w); err != nil {
						return err
					}
				}

				if len(r.Table.Actions) > 0 {
					actionsCell := templ.ComponentFunc(func(ctx context.Context, w io.Writer) error {
						w.Write([]byte(`<td class="px-3 py-2 text-sm"><div class="flex gap-2">`))
						for _, action := range r.Table.Actions {
							if err := action.Render(item).Render(ctx, w); err != nil {
								return err
							}
						}
						w.Write([]byte(`</div></td>`))
						return nil
					})
					if err := actionsCell.Render(ctx, w); err != nil {
						return err
					}
				}

				return nil
			})
		}

		tableComponent := ui.Table(headers, rows, renderRow)

		b := templ.GetBuffer()
		defer templ.ReleaseBuffer(b)

		searchHtml := ""
		searchBuf := templ.GetBuffer()
		defer templ.ReleaseBuffer(searchBuf)
		if err := ui.SearchBar(basePath, search, "Search...").Render(ctx, searchBuf); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}
		searchHtml = searchBuf.String()

		if err := tableComponent.Render(ctx, b); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		paginationHtml := ""
		if result.TotalPages > 1 {
			paginationBuf := templ.GetBuffer()
			defer templ.ReleaseBuffer(paginationBuf)

			baseUrl := basePath
			if search != "" {
				baseUrl += "?search=" + search
			}

			if err := ui.Pagination(result.Page, result.TotalPages, baseUrl, result.HasNext, result.HasPrev).Render(ctx, paginationBuf); err != nil {
				return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
			}
			paginationHtml = paginationBuf.String()
		}

		createBtn := templ.GetBuffer()
		defer templ.ReleaseBuffer(createBtn)
		if err := ui.CreateButton(r.Name).Render(ctx, createBtn); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		html := fmt.Sprintf(`<!doctype html>
<html>
<head>
	<meta charset="utf-8">
	<title>%s - Deck UI</title>
	<script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="p-8">
	<div class="flex justify-between items-center mb-4">
		<h1 class="text-2xl font-bold">%s</h1>
		%s
	</div>
	%s
	%s
	%s
	<div class="mt-4 text-sm text-gray-600">
		Showing %d of %d results
	</div>
</body>
</html>`, r.Name, r.Name, createBtn.String(), searchHtml, b.String(), paginationHtml, len(result.Data), result.Total)

		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	})

	app.Get(basePath+"/new", func(c *fiber.Ctx) error {
		ctx := templ.InitializeContext(c.UserContext())

		fieldComponents := make([]templ.Component, len(r.Fields))
		for i, field := range r.Fields {
			opts := field.Options()
			label := opts.Label
			if label == "" {
				label = field.Name()
			}
			inputComp := ui.Input(field.Name(), field.Name(), "", field.Type(), map[string]string{})
			fieldComponents[i] = ui.FormField(label, inputComp)
		}

		formComponent := ui.Form(basePath, "POST", fieldComponents, "Create", basePath)

		b := templ.GetBuffer()
		defer templ.ReleaseBuffer(b)

		if err := formComponent.Render(ctx, b); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		html := fmt.Sprintf(`<!doctype html>
<html>
<head>
	<meta charset="utf-8">
	<title>Create %s - Deck UI</title>
	<script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="p-8">
	<h1 class="text-2xl font-bold mb-4">Create %s</h1>
	%s
</body>
</html>`, r.Name, r.Name, b.String())

		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	})

	app.Post(basePath, func(c *fiber.Ctx) error {
		var model T

		for _, field := range r.Fields {
			value := c.FormValue(field.Name())
			rv := reflect.ValueOf(&model).Elem()
			f := rv.FieldByNameFunc(func(n string) bool {
				return strings.EqualFold(n, field.Name())
			})
			if f.IsValid() && f.CanSet() {
				f.SetString(value)
			}
		}

		if err := r.Create(&model); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		return c.Redirect(basePath)
	})

	app.Get(basePath+"/:id/edit", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid ID")
		}

		model, err := r.FindBy(uint(id))
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("Not found")
		}

		ctx := templ.InitializeContext(c.UserContext())

		fieldComponents := make([]templ.Component, len(r.Fields))
		rv := reflect.ValueOf(model).Elem()
		for i, field := range r.Fields {
			opts := field.Options()
			label := opts.Label
			if label == "" {
				label = field.Name()
			}

			f := rv.FieldByNameFunc(func(n string) bool {
				return strings.EqualFold(n, field.Name())
			})
			value := ""
			if f.IsValid() {
				value = fmt.Sprint(f.Interface())
			}

			inputComp := ui.Input(field.Name(), field.Name(), value, field.Type(), map[string]string{})
			fieldComponents[i] = ui.FormField(label, inputComp)
		}

		formComponent := ui.Form(fmt.Sprintf("%s/%d", basePath, id), "POST", fieldComponents, "Update", basePath)

		b := templ.GetBuffer()
		defer templ.ReleaseBuffer(b)

		if err := formComponent.Render(ctx, b); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		html := fmt.Sprintf(`<!doctype html>
<html>
<head>
	<meta charset="utf-8">
	<title>Edit %s - Deck UI</title>
	<script src="https://cdn.tailwindcss.com"></script>
</head>
<body class="p-8">
	<h1 class="text-2xl font-bold mb-4">Edit %s</h1>
	%s
</body>
</html>`, r.Name, r.Name, b.String())

		c.Set("Content-Type", "text/html; charset=utf-8")
		return c.SendString(html)
	})

	app.Post(basePath+"/:id", func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid ID")
		}

		model, err := r.FindBy(uint(id))
		if err != nil {
			return c.Status(fiber.StatusNotFound).SendString("Not found")
		}

		rv := reflect.ValueOf(model).Elem()
		for _, field := range r.Fields {
			value := c.FormValue(field.Name())
			f := rv.FieldByNameFunc(func(n string) bool {
				return strings.EqualFold(n, field.Name())
			})
			if f.IsValid() && f.CanSet() {
				f.SetString(value)
			}
		}

		if err := r.Update(model); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		return c.Redirect(basePath)
	})

	deleteHandler := func(c *fiber.Ctx) error {
		id, err := c.ParamsInt("id")
		if err != nil {
			return c.Status(fiber.StatusBadRequest).SendString("Invalid ID")
		}

		if err := r.Delete(uint(id)); err != nil {
			return c.Status(fiber.StatusInternalServerError).SendString(err.Error())
		}

		return c.Redirect(basePath)
	}

	app.Delete(basePath+"/:id", deleteHandler)
	app.Post(basePath+"/:id/delete", deleteHandler)
}

type Table[T any] struct {
	Columns []Column[T]
	Actions []Action[T]
}

type Column[T any] struct {
	Label  string
	Value  func(T) any
	Render func(value any) templ.Component
}

func TextColumn[T any](label string, value func(T) string) Column[T] {
	return Column[T]{
		Label: label,
		Value: func(t T) any { return value(t) },
		Render: func(v any) templ.Component {
			return ui.Text(fmt.Sprint(v))
		},
	}
}

type Action[T any] struct {
	Label  string
	Render func(T) templ.Component
}

func EditAction[T interface{ GetID() uint }](resource string) Action[T] {
	return Action[T]{
		Label: "Edit",
		Render: func(t T) templ.Component {
			return ui.EditButton(resource, t.GetID())
		},
	}
}

func DeleteAction[T interface{ GetID() uint }](resource string) Action[T] {
	return Action[T]{
		Label: "Delete",
		Render: func(t T) templ.Component {
			return ui.DeleteButton(resource, t.GetID())
		},
	}
}

func NewTable[T any]() *Table[T] {
	return &Table[T]{}
}

func (t *Table[T]) Column(c Column[T]) *Table[T] {
	t.Columns = append(t.Columns, c)
	return t
}

func (t *Table[T]) SetActions(a ...Action[T]) *Table[T] {
	t.Actions = a
	return t
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
