package patterns_test

import (
	"slices"
	"testing"

	"github.com/zefrenchwan/patterns.git/patterns"
)

func TestDictionary(t *testing.T) {
	dict := patterns.NewDictionary("test")
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

func TestDictionaryMerge(t *testing.T) {
	a := patterns.NewDictionary("test")
	b := patterns.NewDictionary("test")

	a.AddRelationWithObject("couple", []string{"Person"}, []string{"Person"})
	b.AddRelationWithObject("married", []string{"Person"}, []string{"Person"})
	a.AddTrait("Man")
	b.AddTrait("Woman")

	errMerge := a.Merge(b)
	if errMerge != nil {
		t.Fail()
	}

	if !a.HasEntityTrait("Man") {
		t.Fail()
	} else if !a.HasRelationTrait("couple") {
		t.Fail()
	} else if !b.HasRelationTrait("married") {
		t.Fail()
	} else if !a.HasEntityTrait("Person") {
		t.Fail()
	} else if !a.HasEntityTrait("Woman") {
		t.Fail()
	}

}
