package codegen

import (
	"fmt"
	"strings"
)

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
