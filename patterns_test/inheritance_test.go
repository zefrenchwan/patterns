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
