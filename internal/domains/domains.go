package domains

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type ChallengeName struct {
	value string
}

func NewChallengeName(name string) (ChallengeName, error) {
	trimmed := strings.TrimSpace(name)

	if trimmed == "" {
		return ChallengeName{}, errors.New("challenge name cannot be empty")
	}

	if len(trimmed) > 1000 {
		return ChallengeName{}, fmt.Errorf("challenge name is too long (%d/1000 characters)", len(trimmed))
	}

	if strings.Contains(trimmed, "#") {
		return ChallengeName{}, errors.New("challenge name cannot contain the '#' symbol")
	}

	return ChallengeName{value: trimmed}, nil
}

func (c ChallengeName) String() string {
	return c.value
}

func (c *ChallengeName) Scan(value any) error {
	if value == nil {
		return errors.New("challenge name cannot be null")
	}

	sv, ok := value.(string)
	if !ok {
		bv, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type *domains.ChallengeName", value)
		}
		sv = string(bv)
	}

	name, err := NewChallengeName(sv)
	if err != nil {
		return err
	}

	*c = name
	return nil
}

func (c ChallengeName) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.value)
}
