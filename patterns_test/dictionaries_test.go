package patterns_test

import (
	"slices"
	"testing"

	"github.com/zefrenchwan/patterns.git/patterns"
)

func TestDictionary(t *testing.T) {
	dict := patterns.NewDictionary()
	dict.AddRelationWithObject("knows", []string{"Person"}, []string{"Place", "Person"})

	// place and persons should be valid traits
	if v := dict.DirectSuperTraits("Place"); v == nil || len(v) != 0 {
		t.Error("expecting empty non nil slice")
	} else if v = dict.DirectSubTraits("Person"); v == nil || len(v) != 0 {
		t.Error("expecting empty non nil slice")
	}

	relationDetails := dict.GetRelationRoles("knows")
	if len(relationDetails) != 2 {
		t.Error("expecting subject and object")
	} else if slices.Compare(relationDetails[patterns.SUBJECT_ROLE], []string{"Person"}) != 0 {
		t.Error("subject should be inserted subject only")
	} else if slices.Compare(relationDetails[patterns.OBJECT_ROLE], []string{"Person", "Place"}) != 0 {
		t.Error("objects should be sorted slices of values")
	}
}
