package codegen

import (
	"fmt"
	"strings"

	"github.com/gertd/go-pluralize"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	caser = cases.Title(language.English, cases.NoLower)
)

type CommandParameterType string

func (c CommandParameterType) String() string {
	return string(c)
}

const (
	CommandParameterTypeString CommandParameterType = "Text"
	CommandParameterTypeInt    CommandParameterType = "Number"
	CommandParameterTypeEnum   CommandParameterType = "Combo"
)

type Command struct {
	Id               string             `json:"id"`
	CommandName      *string            `json:"commandName,omitempty"`
	FriendlyName     string             `json:"friendlyName"`
	Text             string             `json:"text"`
	Description      string             `json:"description"`
	Parameters       []CommandParameter `json:"parameters,omitempty"`
	Response         *Response          `json:"response,omitempty"`
	InlineParameters *bool              `json:"inline,omitempty"`
}

func (c Command) Name() string {
	return c.Id
}

func (c Command) Imports() Imports {
	res := Imports{}
	res.Add("context", "context")

	// a bit hacky way to extract types from the go standard library. Let's hope there will never ever be a type
	// from outside the std library, as this is most likely not remotely supported.
	for _, t := range c.ReturnTypeDefinitions() {
		if strings.Contains(t.AliasedType, ".") {
			pkg := strings.SplitN(t.AliasedType, ".", 2)[0]
			res.Add(pkg, pkg)
		}
		for _, m := range t.Members {
			if s, ok := m.Type.(string); ok && strings.Contains(s, ".") {
				pkg := strings.SplitN(s, ".", 2)[0]
				res.Add(pkg, pkg)
			}
		}
	}
	return res
}

func (c Command) Params() (res Params) {
	res = append(res, Param{
		Name: "ctx",
		Type: "context.Context",
	})
	if c.InlineParameters != nil && *c.InlineParameters && len(c.Parameters) > 0 {
		p := c.Parameters[0]
		res = append(res, Param{
			Name: p.Id,
			Type: fmt.Sprintf("%s%s", c.Id, caser.String(c.Parameters[0].Id)),
		})
	} else {
		for _, p := range c.Parameters {
			res = append(res, Param{
				Name:       p.Id,
				Type:       c.ParameterGoType(p),
				FixedValue: p.FixedValue,
			})
		}
	}
	return
}

func (c Command) ParameterGoType(p CommandParameter) string {
	switch p.Type {
	case CommandParameterTypeString:
		return "string"
	case CommandParameterTypeInt:
		return "int"
	case CommandParameterTypeEnum:
		if p.IsMapNameType() {
			return p.Id
		}
		if !p.IsMapNameType() && p.Id == "MapName" {
			return "MapName"
		}
		return fmt.Sprintf("%s%s", c.Id, caser.String(p.Id))
	default:
		return "any"
	}
}

func (p CommandParameter) IsMapNameType() bool {
	// AddMapToRotation has a full list of available maps as valueMember, ChangeMap for example has only one map.
	// To not wrongfully advertise ChangeMap to take a single map as an argument, define the MapName as a global type.
	// it will then be used in other commands that take the same MapName parameter as well.
	return p.Type == CommandParameterTypeEnum && p.Id == "MapName" && len(p.ValueMember) > 2
}

func (c Command) Types() (res Types) {
	for _, p := range c.Parameters {
		if !p.IsMapNameType() && p.Id == "MapName" {
			continue
		}
		if len(p.ValueMember) > 0 {
			var members []TypeMember
			for i := range p.ValueMember {
				memberName := fmt.Sprintf("%s%s%s", c.Id, p.Id, caser.String(p.DisplayMember[i]))
				if p.IsMapNameType() {
					memberName = fmt.Sprintf("%s%s", p.Id, caser.String(p.DisplayMember[i]))
				}
				members = append(members, TypeMember{
					Name: memberName,
					Type: p.ValueMember[i],
				})
			}
			typeName := fmt.Sprintf("%s%s", c.Id, p.Id)
			if p.IsMapNameType() {
				typeName = p.Id
			}
			res = append(res, Type{
				Name:        typeName,
				AliasedType: aliasedType(p.ValueMember),
				Members:     members,
			})
		}
	}

	res = append(res, c.RequestType())
	if c.Response != nil {
		res = append(res, c.ReturnTypeDefinitions()...)
	}
	return res
}

func aliasedType(valueMembers []any, defaultType ...string) string {
	if len(valueMembers) == 0 {
		if len(defaultType) > 0 {
			return defaultType[0]
		}
		return ""
	}
	switch valueMembers[0].(type) {
	case string:
		return "string"
	case float64:
		return "int"
	default:
		return "any"
	}
}

func (c Command) RequestType() Type {
	var requestMembers []TypeMember
	if len(c.Parameters) == 1 && c.InlineParameters != nil && *c.InlineParameters {
		return Type{
			Name:        fmt.Sprintf("%s%s", c.Id, caser.String(c.Parameters[0].Id)),
			AliasedType: aliasedType(c.Parameters[0].ValueMember, "string"),
		}
	} else {
		for _, param := range c.Parameters {
			requestMembers = append(requestMembers, TypeMember{
				Name: caser.String(param.Id),
				Type: c.ParameterGoType(param),
			})
		}

		return Type{
			Name:    c.Name(),
			Members: requestMembers,
		}
	}
}

func (c Command) ResponseGoType(p ResponseElement) string {
	switch p.Type {
	case ResponseItemTypeString:
		return "string"
	case ResponseItemTypeInt:
		return "int"
	case ResponseItemTypeEnum:
		return fmt.Sprintf("%s%s", c.Id, caser.String(p.Name))
	case ResponseItemTypeBool:
		return "bool"
	case ResponseItemTypeTime:
		return "time.Time"
	case ResponseItemTypeComplex:
		return fmt.Sprintf("%s%s", c.Id, pluralize.NewClient().Singular(p.Name))
	case ResponseItemTypeList:
		return fmt.Sprintf("[]%s%s", c.Id, pluralize.NewClient().Singular(p.Name))
	default:
		return "any"
	}
}

func (c Command) ReturnType() *Type {
	if c.Response == nil {
		return nil
	}
	res := *c.Response
	var responseMembers []TypeMember
	for _, param := range res {
		responseMembers = append(responseMembers, TypeMember{
			Name: strings.ReplaceAll(param.Name, " ", ""),
			Type: c.ResponseGoType(param),
			Json: new(param.Id),
		})
	}
	return &Type{
		Name:      fmt.Sprintf("%sResponse", c.Id),
		IsPointer: true,
		Members:   responseMembers,
	}
}

func (c Command) ReturnTypeDefinitions() (res []Type) {
	if retval := c.ReturnType(); retval != nil {
		res = append(res, *retval)
	}
	if c.Response == nil {
		return
	}
	r := *c.Response
	for _, param := range r {
		if param.Type == ResponseItemTypeList && (param.ValueMember != nil || param.ListType == ResponseItemListTypeString) {
			res = append(res, c.buildResponseItemCombo(param)...)
		} else if param.Type == ResponseItemTypeList || param.Type == ResponseItemTypeComplex {
			res = append(res, c.buildSubTypes(param)...)
		} else if param.Type == ResponseItemTypeEnum {
			res = append(res, c.buildResponseItemCombo(param)...)
		}
	}
	return res
}

func (c Command) buildResponseItemCombo(param ResponseElement) (res []Type) {
	singularize := pluralize.NewClient()
	var members []TypeMember
	for i := range param.ValueMember {
		members = append(members, TypeMember{
			Name: fmt.Sprintf("%s%s%s", c.Id, caser.String(param.Id), singularize.Singular(caser.String(param.DisplayMember[i]))),
			Type: param.ValueMember[i],
		})
	}
	res = append(res, Type{
		Name:        fmt.Sprintf("%s%s", c.Id, singularize.Singular(param.Name)),
		AliasedType: aliasedType(param.ValueMember, "string"),
		Members:     members,
	})
	return
}

func (c Command) buildSubTypes(param ResponseElement) (res []Type) {
	var responseMembers []TypeMember
	for _, m := range param.Members {
		responseMembers = append(responseMembers, TypeMember{
			Name: strings.ReplaceAll(m.Name, " ", ""),
			Type: c.ResponseGoType(m),
			Json: new(m.Id),
		})
		if m.Type == ResponseItemTypeList || m.Type == ResponseItemTypeComplex {
			res = append(res, c.buildSubTypes(m)...)
		} else if m.Type == ResponseItemTypeEnum {
			res = append(res, c.buildResponseItemCombo(m)...)
		}
	}
	res = append(res, Type{
		Name:    fmt.Sprintf("%s%s", c.Id, pluralize.NewClient().Singular(param.Name)),
		Members: responseMembers,
	})
	return
}

func (c Command) EnumValues() map[string]any {
	res := make(map[string]any)
	for _, t := range c.Types() {
		if t.AliasedType == "" {
			continue
		}
		for _, member := range t.Members {
			res[member.EnumName()] = member.Type
		}
	}
	return res
}

type ResponseItemType string

func (c ResponseItemType) String() string {
	return string(c)
}

const (
	ResponseItemTypeString  ResponseItemType = "Text"
	ResponseItemTypeInt     ResponseItemType = "Number"
	ResponseItemTypeEnum    ResponseItemType = "Combo"
	ResponseItemTypeList    ResponseItemType = "List"
	ResponseItemTypeComplex ResponseItemType = "Complex"
	ResponseItemTypeBool    ResponseItemType = "Bool"
	ResponseItemTypeTime    ResponseItemType = "Time"
)

type ResponseItemListType string

const (
	ResponseItemListTypeString ResponseItemListType = "Text"
)

type Response []ResponseElement

type ResponseElement struct {
	Type ResponseItemType `json:"type"`
	// ListType is only applicable for ResponseItemTypeList and can be used to optionally define the type of list.
	// it has no effect when Members is given (ListType will be implicitly an object), as well as when ValueMember is
	// provided (ListType will be an Enum).
	ListType ResponseItemListType `json:"listType,omitempty"`
	Name     string               `json:"name"`
	Id       string               `json:"id"`

	// Only filled for ResponseItemTypeEnum
	DisplayMember []string `json:"displayMember,omitempty"`
	ValueMember   []any    `json:"valueMember,omitempty"`

	// Only filled for ResponseItemTypeList
	Members []ResponseElement `json:"members,omitempty"`
}

type CommandParameter struct {
	Type          CommandParameterType `json:"type"`
	Name          string               `json:"name"`
	Id            string               `json:"id"`
	DisplayMember []string             `json:"displayMember,omitempty"`
	ValueMember   []any                `json:"valueMember,omitempty"`
	FixedValue    *string              `json:"fixedValue"`
}

type Types []Type

type Type struct {
	Name        string
	AliasedType string
	IsPointer   bool
	Members     []TypeMember
	FixedValue  *string
}

type TypeMember struct {
	Name string
	Type any
	Json *string
}

func (t TypeMember) JsonName() string {
	if t.Json != nil {
		return *t.Json
	}
	return t.Name
}

func (t TypeMember) EnumName() string {
	s := strings.ReplaceAll(t.Name, "_", " ")
	s = cases.Title(language.English, cases.NoLower).String(s)
	return strings.ReplaceAll(s, " ", "")
}

func (t Type) AsTypeDefinition() string {
	if t.AliasedType != "" {
		return fmt.Sprintf(`type %s %s`, t.Name, t.AliasedType)
	}
	members := make([]string, len(t.Members))
	for i, member := range t.Members {
		members[i] = fmt.Sprintf("%s %s `json:\"%s\"`", caser.String(member.Name), member.Type, member.JsonName())
	}
	return fmt.Sprintf(`type %s struct {
	%s
}`, t.Name, strings.Join(members, "\n\t"))
}

func (t Type) AsTypeReference() string {
	if t.IsPointer {
		return fmt.Sprintf("*%s", t.Name)
	}
	return t.Name
}

type Params []Param

type Param struct {
	Name       string
	Type       string
	FixedValue *string
}

func (p Param) AsRequestAssignment() string {
	if p.FixedValue != nil {
		return fmt.Sprintf(`%s: "%s",`, caser.String(p.Name), *p.FixedValue)
	}
	return fmt.Sprintf(`%s: %s,`, caser.String(p.Name), caser.String(p.Name))
}

func (p Params) AsNamedArgsWithTypes() string {
	if len(p) == 0 {
		return ""
	}

	var params []string
	for _, param := range p {
		if param.FixedValue != nil {
			continue
		}
		params = append(params, param.Name+" "+param.Type)
	}
	return strings.Join(params, ", ")
}

func (c Command) Returns() Returns {
	ret := Returns{}
	if returnType := c.ReturnType(); returnType != nil {
		ret = append(ret, *returnType)
	}
	ret = append(ret, Type{
		Name: "error",
	})
	return ret
}

type Returns []Type

type Return struct {
	Type string
}

func (r Returns) String() string {
	if len(r) == 0 {
		return " "
	}
	if len(r) == 1 {
		return r[0].Name
	}
	var types []string
	for _, t := range r {
		if t.IsPointer {
			types = append(types, fmt.Sprintf("*%s", t.Name))
		} else {
			types = append(types, t.Name)
		}
	}
	return fmt.Sprintf(" (%s) ", strings.Join(types, ", "))
}

type Imports map[string]Import

func (i *Imports) Add(alias, pkgPath string) {
	if _, exists := (*i)[alias]; exists {
		return
	}
	(*i)[alias] = Import{Alias: alias, PackagePath: pkgPath}
}

type Import struct {
	Alias       string
	PackagePath string
}

func (i Import) String() string {
	if i.Alias == i.PackagePath {
		return `"` + i.Alias + `"`
	}
	return fmt.Sprintf(`%s "%s"`, i.Alias, i.PackagePath)
}
