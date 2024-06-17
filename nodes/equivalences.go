package nodes

import "slices"

// AllSameElements returns true if each couple of elements are same, false otherwise
func AllSameElements(elements []Element) bool {
	size := len(elements)
	if size <= 1 {
		return true
	}

	// AreSameElements is an equivalence relation, so we may test only
	// one with all the others instead of each couple.
	previous := elements[0]
	for index := 1; index < size; index++ {
		if !AreSameElements(previous, elements[index]) {
			return false
		}
	}

	return true
}

// AreSameElements returns true if contents (not id, just content) are identical (same values), false otherwise
func AreSameElements(a, b Element) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}

	// do not test id, but traits and activity
	// Activity test
	aPeriod := a.ActivePeriod()
	bPeriod := b.ActivePeriod()
	if !aPeriod.IsSameAs(bPeriod) {
		return false
	}
	// traits test
	aTraits := a.Traits()
	bTraits := b.Traits()
	slices.Sort(aTraits)
	slices.Sort(bTraits)
	if slices.Compare(aTraits, bTraits) != 0 {
		return false
	}

	aFormalInstance, aFormalInstanceOk := a.(FormalInstance)
	bFormalInstance, bFormalInstanceOk := b.(FormalInstance)
	aFormalRelation, aFormalRelationOk := a.(FormalRelation)
	bFormalRelation, bFormalRelationOk := b.(FormalRelation)
	if aFormalInstanceOk != bFormalInstanceOk {
		return false
	} else if aFormalRelationOk != bFormalRelationOk {
		return false
	}

	if aFormalInstanceOk {
		aAttr := aFormalInstance.Attributes()
		bAttr := bFormalInstance.Attributes()
		if (aAttr == nil || bAttr == nil) && !(aAttr == nil && bAttr == nil) {
			return false
		}

		// test if same attributes
		slices.Sort(aAttr)
		slices.Sort(bAttr)
		if slices.Compare(aAttr, bAttr) != 0 {
			return false
		}

		for index := 0; index < len(aAttr); index++ {
			attribute := aAttr[index]
			aValues, _ := aFormalInstance.PeriodValuesForAttribute(attribute)
			bValues, _ := bFormalInstance.PeriodValuesForAttribute(attribute)
			if len(aValues) != len(bValues) {
				return false
			}

			for attr, period := range aValues {
				bPeriod, found := bValues[attr]
				if !found {
					return false
				}

				if !bPeriod.IsSameAs(period) {
					return false
				}
			}
		}
	}

	if aFormalRelationOk {
		aValues := aFormalRelation.ValuesPerRole()
		bValues := bFormalRelation.ValuesPerRole()
		if len(aValues) != len(bValues) {
			return false
		}

		for role, links := range aValues {
			bLinks, bLinksFound := bValues[role]
			if !bLinksFound {
				return false
			} else if len(bLinks) != len(links) {
				return false
			}

			slices.Sort(links)
			slices.Sort(bLinks)
			if slices.Compare(links, bLinks) != 0 {
				return false
			}
		}
	}

	return true
}
