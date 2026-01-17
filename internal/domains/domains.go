package domains

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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

type ChallengeID struct {
	value int
}

func NewChallengeID(id int) (ChallengeID, error) {
	if id <= 0 {
		return ChallengeID{}, errors.New("challenge id must be positive")
	}
	return ChallengeID{value: id}, nil
}

func NewChallengeIDFromString(id string) (ChallengeID, error) {
	val, err := strconv.Atoi(id)
	if err != nil {
		return ChallengeID{}, fmt.Errorf("invalid challenge id format: %w", err)
	}
	return NewChallengeID(val)
}

func (c ChallengeID) String() string {
	return strconv.Itoa(c.value)
}

func (c ChallengeID) Int() int {
	return c.value
}

func (c *ChallengeID) Scan(value any) error {
	if value == nil {
		return errors.New("challenge id cannot be null")
	}

	switch v := value.(type) {
	case int64:
		cid, err := NewChallengeID(int(v))
		if err != nil {
			return err
		}
		*c = cid
	case int32:
		cid, err := NewChallengeID(int(v))
		if err != nil {
			return err
		}
		*c = cid
	case int:
		cid, err := NewChallengeID(v)
		if err != nil {
			return err
		}
		*c = cid
	default:
		return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type *domains.ChallengeID", value)
	}

	return nil
}

func (c ChallengeID) MarshalJSON() ([]byte, error) {
	return json.Marshal(strconv.Itoa(c.value))
}

type UserName struct {
	value string
}

func NewUserName(name string) (UserName, error) {
	if len(name) < 3 || len(name) > 50 {
		return UserName{}, fmt.Errorf("username length must be between 3 and 50 characters, got %d", len(name))
	}

	if strings.TrimSpace(name) != name {
		return UserName{}, errors.New("username cannot have leading or trailing whitespace")
	}

	return UserName{value: name}, nil
}

func (u UserName) String() string {
	return u.value
}

func (u *UserName) Scan(value any) error {
	if value == nil {
		return errors.New("username cannot be null")
	}

	sv, ok := value.(string)
	if !ok {
		bv, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type *domains.UserName", value)
		}
		sv = string(bv)
	}

	name, err := NewUserName(sv)
	if err != nil {
		return err
	}

	*u = name
	return nil
}

func (u UserName) MarshalJSON() ([]byte, error) {
	return json.Marshal(u.value)
}

type Category struct {
	value string
}

func NewCategory(name string) (Category, error) {
	validCategories := map[string]bool{
		"web hacking":         true,
		"embedded hacking":    true,
		"reverse engineering": true,
		"crypto challenge":    true,
		"forensics":           true,
	}

	if !validCategories[name] {
		return Category{}, fmt.Errorf("invalid category: %s", name)
	}

	return Category{value: name}, nil
}

func (c Category) String() string {
	return c.value
}

func (c *Category) Scan(value any) error {
	if value == nil {
		return errors.New("category cannot be null")
	}

	sv, ok := value.(string)
	if !ok {
		bv, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type *domains.Category", value)
		}
		sv = string(bv)
	}

	cat, err := NewCategory(sv)
	if err != nil {
		return err
	}

	*c = cat
	return nil
}

func (c Category) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.value)
}

type Content struct {
	value string
}

func NewContent(content string) (Content, error) {
	if strings.TrimSpace(content) == "" {
		return Content{}, errors.New("content cannot be empty")
	}
	return Content{value: content}, nil
}

func (c Content) String() string {
	return c.value
}

func (c *Content) Scan(value any) error {
	if value == nil {
		return errors.New("content cannot be null")
	}

	sv, ok := value.(string)
	if !ok {
		bv, ok := value.([]byte)
		if !ok {
			return fmt.Errorf("unsupported Scan, storing driver.Value type %T into type *domains.Content", value)
		}
		sv = string(bv)
	}

	content, err := NewContent(sv)
	if err != nil {
		return err
	}

	*c = content
	return nil
}

func (c Content) MarshalJSON() ([]byte, error) {
	return json.Marshal(c.value)
}
