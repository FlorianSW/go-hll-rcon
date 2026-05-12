package rconv2

import (
	"fmt"
	"slices"
)

var (
	requiresValue = []GetServerInformationName{
		GetServerInformationNamePlayer,
	}
)

func (s GetServerInformation) Validate() error {
	if slices.Contains(requiresValue, s.Name) && s.Value == "" {
		return fmt.Errorf("%s command requires a Value", s.Name)
	}
	return nil
}
