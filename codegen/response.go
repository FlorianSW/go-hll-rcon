package codegen

import (
	"fmt"
	"strings"

	"github.com/gertd/go-pluralize"
)

type ResponseItemType string

func (c ResponseItemType) String() string {
	return string(c)
}

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

const (
	ResponseItemTypeString  ResponseItemType = "Text"
	ResponseItemTypeInt     ResponseItemType = "Number"
	ResponseItemTypeEnum    ResponseItemType = "Combo"
	ResponseItemTypeList    ResponseItemType = "List"
	ResponseItemTypeComplex ResponseItemType = "Complex"
	ResponseItemTypeBool    ResponseItemType = "Bool"
	ResponseItemTypeTime    ResponseItemType = "Time"
)

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
