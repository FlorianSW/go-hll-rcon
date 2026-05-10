package codegen

import (
	"fmt"
	"strings"

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
	CommandName  *string            `json:"commandName"`
	FriendlyName string             `json:"friendlyName"`
	Text         string             `json:"text"`
	Description  string             `json:"description"`
	Parameters   []CommandParameter `json:"parameters"`
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

func (c Command) ReturnType() Type {
	return Type{
		Name: "any",
	}
}

func (c Command) EnumValues() map[string]string {
	res := make(map[string]string)
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

type CommandParameter struct {
	Type          CommandParameterType `json:"type"`
	Name          string               `json:"name"`
	Id            string               `json:"id"`
	DisplayMember []string             `json:"displayMember"`
	ValueMember   []string             `json:"valueMember"`
	FixedValue    *string              `json:"fixedValue"`
}

type Types []Type

type Type struct {
	Name        string
	AliasedType string
	Members     []TypeMember
	FixedValue  *string
}

type TypeMember struct {
	Name string
	Type string
}

func (t TypeMember) EnumName() string {
	s := strings.ReplaceAll(t.Name, "_", " ")
	s = cases.Title(language.English, cases.NoLower).String(s)
	return strings.ReplaceAll(s, " ", "")
}

func (t Type) String() string {
	if t.AliasedType != "" {
		return fmt.Sprintf(`type %s %s`, t.Name, t.AliasedType)
	}
	members := make([]string, len(t.Members))
	for i, member := range t.Members {
		members[i] = fmt.Sprintf("%s %s `json:\"%s\"`", member.Name, member.Type, member.Name)
	}
	return fmt.Sprintf(`type %s struct {
	%s
}`, t.Name, strings.Join(members, "\n\t"))
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
		types = append(types, t.Name)
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
