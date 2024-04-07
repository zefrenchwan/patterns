package patterns

import (
	"slices"
	"testing"

	"github.com/zefrenchwan/patterns.git/patterns"
)

func TestInheritanceUsingDictionary(t *testing.T) {
	d := patterns.NewDictionary()
	// do it twice to ensure deduplication
	d.AddTraitsLink("cat", "animal")
	d.AddTraitsLink("cat", "animal")

	if d.DirectSubTraits("choucroute") != nil {
		t.Error("when not present, return nil")
	}

	if slices.Compare([]string{"animal"}, d.DirectSuperTraits("cat")) != 0 {
		t.Fail()
	} else if slices.Compare([]string{"cat"}, d.DirectSubTraits("animal")) != 0 {
		t.Fail()
	}
}

func TestRelationInheritanceUsingDictionary(t *testing.T) {
	d := patterns.NewDictionary()
	d.AddRelationWithObject("couple", []string{"Person"}, []string{"Person"})
	d.AddRelationWithObject("married", []string{"Person"}, []string{"Person"})
	d.AddRelationLink("married", "couple")

	if d.GetDirectSubRelations("couple") == nil {
		t.Error("couple exists, should return empty")
	} else if vsup := d.GetDirectSuperRelations("married"); len(vsup) != 1 {
		t.Fail()
	} else if vsup[0] != "couple" {
		t.Error("inheritance failure for relation")
	} else if vsub := d.GetDirectSubRelations("couple"); len(vsub) != 1 {
		t.Fail()
	} else if vsub[0] != "married" {
		t.Error("inheritance failure for relation")
	}
}
