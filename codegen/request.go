package codegen

import (
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

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

func (c Command) RequestType() Type {
	if len(c.Parameters) == 1 && toBool(c.InlineParameters) {
		return Type{
			Name:        fmt.Sprintf("%s%s%s", c.TypePrefix, c.Id, caser.String(c.Parameters[0].Id)),
			AliasedType: aliasedType(c.Parameters[0].ValueMember, "string"),
		}
	}

	var requestMembers []TypeMember
	for _, param := range c.Parameters {
		requestMembers = append(requestMembers, TypeMember{
			Name: caser.String(param.Id),
			Type: c.ParameterGoType(param),
		})
	}

	return Type{
		Name:    fmt.Sprintf("%s%s", c.TypePrefix, c.Name()),
		Members: requestMembers,
	}
}

func toBool(v *bool) bool {
	if v == nil {
		return false
	}
	return *v
}
