package nodes_test

import (
	"testing"
	"time"

	"github.com/zefrenchwan/patterns.git/nodes"
)

func TestAreEquivalentEntitiesNoValue(t *testing.T) {
	entity := nodes.NewEntity([]string{"city"})
	otherEntity := nodes.NewEntity([]string{"city", "capital"})
	// traits differ
	if nodes.AreSameElements(&entity, &otherEntity) {
		t.Error("traits equivalence mismatch")
	}

	// same traits
	otherEntity = nodes.NewEntity([]string{"city"})
	if !nodes.AreSameElements(&entity, &otherEntity) {
		t.Error("traits equivalence mismatch")
	}

	// different activities
	now := time.Now()
	periodToRemove := nodes.NewPeriod(nodes.NewLeftInfiniteTimeInterval(now, false))
	otherEntity.RemoveActivePeriod(periodToRemove)
	if nodes.AreSameElements(&entity, &otherEntity) {
		t.Error("period equivalence mismatch")
	}
}

func TestAreEquivalentEntitiesWithValue(t *testing.T) {
	entity := nodes.NewEntity([]string{"city"})
	otherEntity := nodes.NewEntity([]string{"city"})
	entity.AddValue("test", "value", nodes.NewFullPeriod())

	// values count differ
	if nodes.AreSameElements(&entity, &otherEntity) {
		t.Error("values equivalence mismatch")
	}

	// different values, same keys
	otherEntity.AddValue("test", "other value", nodes.NewFullPeriod())
	if nodes.AreSameElements(&entity, &otherEntity) {
		t.Error("values equivalence mismatch")
	}

	// same values, same periods
	otherEntity.AddValue("test", "value", nodes.NewFullPeriod())
	if !nodes.AreSameElements(&entity, &otherEntity) {
		t.Error("values equivalence mismatch")
	}

	// same values, not same period
	now := time.Now()
	period := nodes.NewPeriod(nodes.NewLeftInfiniteTimeInterval(now, false))
	entity = nodes.NewEntity([]string{"city"})
	otherEntity = nodes.NewEntity([]string{"city"})
	entity.AddValue("test", "value", nodes.NewFullPeriod())
	otherEntity.AddValue("test", "value", period)
	if nodes.AreSameElements(&entity, &otherEntity) {
		t.Error("values equivalence mismatch")
	}
}
