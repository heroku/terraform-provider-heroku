package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

const (
	sentinelIndex = "*"
)

// The below few functions were copied from the AWS provider.

// TestCheckTypeSetElemAttr is a resource.TestCheckFunc that accepts a resource
// name, an attribute path, which should use the sentinel value '*' for indexing
// into a TypeSet. The function verifies that an element matches the provided
// value.
//
// Use this function over SDK provided TestCheckFunctions when validating a
// TypeSet where its elements are a simple value
func TestCheckTypeSetElemAttr(name, attr, value string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		is, err := instanceState(s, name)
		if err != nil {
			return err
		}

		err = testCheckTypeSetElem(is, attr, value)
		if err != nil {
			return fmt.Errorf("%q error: %s", name, err)
		}

		return nil
	}
}

// TestCheckTypeSetElemAttrPair is a TestCheckFunc that verifies a pair of name/key
// combinations are equal where the first uses the sentinel value to index into a
// TypeSet.
//
// E.g., tfawsresource.TestCheckTypeSetElemAttrPair("aws_autoscaling_group.bar", "availability_zones.*", "data.aws_availability_zones.available", "names.0")
func TestCheckTypeSetElemAttrPair(nameFirst, keyFirst, nameSecond, keySecond string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		isFirst, err := instanceState(s, nameFirst)
		if err != nil {
			return err
		}

		isSecond, err := instanceState(s, nameSecond)
		if err != nil {
			return err
		}

		vSecond, okSecond := isSecond.Attributes[keySecond]
		if !okSecond {
			return fmt.Errorf("%s: Attribute %q not set, cannot be checked against TypeSet", nameSecond, keySecond)
		}

		return testCheckTypeSetElem(isFirst, keyFirst, vSecond)
	}
}

// instanceState returns the primary instance state for the given
// resource name in the root module.
func instanceState(s *terraform.State, name string) (*terraform.InstanceState, error) {
	ms := s.RootModule()
	rs, ok := ms.Resources[name]
	if !ok {
		return nil, fmt.Errorf("Not found: %s in %s", name, ms.Path)
	}

	is := rs.Primary
	if is == nil {
		return nil, fmt.Errorf("No primary instance: %s in %s", name, ms.Path)
	}

	return is, nil
}

func testCheckTypeSetElem(is *terraform.InstanceState, attr, value string) error {
	attrParts := strings.Split(attr, ".")
	if attrParts[len(attrParts)-1] != sentinelIndex {
		return fmt.Errorf("%q does not end with the special value %q", attr, sentinelIndex)
	}
	for stateKey, stateValue := range is.Attributes {
		if stateValue == value {
			stateKeyParts := strings.Split(stateKey, ".")
			if len(stateKeyParts) == len(attrParts) {
				for i := range attrParts {
					if attrParts[i] != stateKeyParts[i] && attrParts[i] != sentinelIndex {
						break
					}
					if i == len(attrParts)-1 {
						return nil
					}
				}
			}
		}
	}

	return fmt.Errorf("no TypeSet element %q, with value %q in state: %#v", attr, value, is.Attributes)
}

func Sleep(t *testing.T, amount time.Duration) func() {
	return func() {
		time.Sleep(amount * time.Second)
	}
}
