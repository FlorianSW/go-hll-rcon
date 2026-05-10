package codegen

import (
	"fmt"
	"strings"

	"github.com/gertd/go-pluralize"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
	Id           string             `json:"id"`
	CommandName  *string            `json:"commandName,omitempty"`
	FriendlyName string             `json:"friendlyName"`
	Text         string             `json:"text"`
	Description  string             `json:"description"`
	Parameters   []CommandParameter `json:"parameters,omitempty"`
	Response     *Response          `json:"response,omitempty"`
}

func (c Command) Name() string {
	return c.Id
}

func (c Command) Imports() Imports {
	res := Imports{}
	res.Add("context", "context")
	return res
}

func (c Command) Params() (res Params) {
	res = append(res, Param{
		Name: "ctx",
		Type: "context.Context",
	})
	for _, p := range c.Parameters {
		res = append(res, Param{
			Name:       p.Id,
			Type:       c.ParameterGoType(p),
			FixedValue: p.FixedValue,
		})
	}
	return
}

func (c Command) ParameterGoType(p CommandParameter) string {
	switch p.Type {
	case CommandParameterTypeString:
		return "string"
	case CommandParameterTypeInt:
		return "int32"
	case CommandParameterTypeEnum:
		if p.IsMapNameType() {
			return p.Id
		}
		if !p.IsMapNameType() && p.Id == "MapName" {
			return "MapName"
		}
		return fmt.Sprintf("%s%s", c.Id, p.Id)
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
	caser := cases.Title(language.English)
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
				AliasedType: "string",
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

func (c Command) RequestType() Type {
	var requestMembers []TypeMember
	for _, param := range c.Parameters {
		requestMembers = append(requestMembers, TypeMember{
			Name: param.Id,
			Type: c.ParameterGoType(param),
		})
	}

	return Type{
		Name:    c.Name(),
		Members: requestMembers,
	}
}

func (c Command) ResponseGoType(p ResponseElement) string {
	caser := cases.Title(language.English)
	switch p.Type {
	case ResponseItemTypeString:
		return "string"
	case ResponseItemTypeInt:
		return "int32"
	case ResponseItemTypeEnum:
		return fmt.Sprintf("%s%s", c.Id, caser.String(p.Name))
	case ResponseItemTypeComplex:
		return fmt.Sprintf("%s%s", c.Id, pluralize.NewClient().Singular(p.Name))
	case ResponseItemTypeList:
		return fmt.Sprintf("[]%s%s", c.Id, pluralize.NewClient().Singular(p.Name))
	default:
		return "any"
	}
}

func (c Command) ReturnType() Type {
	if c.Response == nil {
		return Type{
			Name: "any",
		}
	}
	res := *c.Response
	var responseMembers []TypeMember
	for _, param := range res {
		responseMembers = append(responseMembers, TypeMember{
			Name: param.Name,
			Type: c.ResponseGoType(param),
		})
	}
	return Type{
		Name:      fmt.Sprintf("%sResponse", c.Id),
		IsPointer: true,
		Members:   responseMembers,
	}
}

func (c Command) ReturnTypeDefinitions() (res []Type) {
	res = append(res, c.ReturnType())
	if c.Response == nil {
		return
	}
	r := *c.Response
	for _, param := range r {
		if param.Type == ResponseItemTypeList || param.Type == ResponseItemTypeComplex {
			res = append(res, c.buildSubTypes(param)...)
		}
	}
	return res
}

func (c Command) buildSubTypes(param ResponseElement) (res []Type) {
	var responseMembers []TypeMember
	caser := cases.Title(language.English)
	for _, m := range param.Members {
		responseMembers = append(responseMembers, TypeMember{
			Name: strings.ReplaceAll(m.Name, " ", ""),
			Type: c.ResponseGoType(m),
			Json: new(m.Id),
		})
		if m.Type == ResponseItemTypeList || m.Type == ResponseItemTypeComplex {
			res = append(res, c.buildSubTypes(m)...)
		} else if m.Type == ResponseItemTypeEnum {
			var members []TypeMember
			for i := range m.ValueMember {
				members = append(members, TypeMember{
					Name: fmt.Sprintf("%s%s%s", c.Id, caser.String(m.Id), caser.String(m.DisplayMember[i])),
					Type: m.ValueMember[i],
				})
			}
			res = append(res, Type{
				Name:        fmt.Sprintf("%s%s", c.Id, caser.String(m.Name)),
				AliasedType: "string",
				Members:     members,
			})
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
)

type Response []ResponseElement

type ResponseElement struct {
	Type ResponseItemType `json:"type"`
	Name string           `json:"name"`
	Id   string           `json:"id"`

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
	ValueMember   []string             `json:"valueMember,omitempty"`
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
		members[i] = fmt.Sprintf("%s %s `json:\"%s\"`", member.Name, member.Type, member.JsonName())
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
		return fmt.Sprintf(`%s: "%s",`, p.Name, *p.FixedValue)
	}
	return fmt.Sprintf(`%s: %s,`, p.Name, p.Name)
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
	return Returns{
		c.ReturnType(),
		{
			Name: "error",
		},
	}
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
	if imp, exists := (*i)[alias]; exists {
		panic(fmt.Sprintf("duplicate import %s, already exists with path %s", alias, imp.PackagePath))
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
