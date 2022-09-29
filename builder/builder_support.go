package builder

import (
	"fmt"
)

type PublishStrategyOption struct {
	Name         string
	description  string
	defaultValue string
}

func (o *PublishStrategyOption) ToString() string {
	if o.defaultValue == "" {
		return fmt.Sprintf("%s: %s", o.Name, o.description)
	}
	return fmt.Sprintf("%s: %s, default %s", o.Name, o.description, o.defaultValue)
}
