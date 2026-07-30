package codegen

import (
	"fmt"
	"strings"

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
		if len(p.ValueMember) == 0 {
			continue
		}
		var members []TypeMember
		for i, member := range p.ValueMember {
			memberName := fmt.Sprintf("%s%s%s", c.Id, p.Id, caser.String(p.DisplayMember[i]))
			if p.IsMapNameType() {
				memberName = fmt.Sprintf("%s%s", p.Id, caser.String(p.DisplayMember[i]))
			}
			members = append(members, TypeMember{
				Name: memberName,
				Type: member,
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

type CommandParameter struct {
	Type          CommandParameterType `json:"type"`
	Name          string               `json:"name"`
	Id            string               `json:"id"`
	DisplayMember []string             `json:"displayMember,omitempty"`
	ValueMember   []any                `json:"valueMember,omitempty"`
	FixedValue    *string              `json:"fixedValue"`
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
