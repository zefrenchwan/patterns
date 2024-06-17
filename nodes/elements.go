package nodes

// Element is a relation or an entity.
// A relation links elements together (recursive definition on purpose).
// An entity describes a real world object
type Element interface {
	// Id returns the (unique) id of the element
	Id() string

	// ActivePeriod returns the period the element was active during
	ActivePeriod() Period
	// AddActivePeriod flags given period as active
	AddActivePeriod(Period) error
	// RemoveActivePeriod flags given period as inactive
	RemoveActivePeriod(Period)
	// SetActivePeriod forces the activity for that element
	SetActivePeriod(Period) error
	// IsActiveDuring returns true if the element is active at least one moment during the period
	IsActiveDuring(Period) bool

	// Traits returns all the traits an element implements
	Traits() []string
	// AddTrait adds a trait if not already present
	AddTrait(string) error
	// RemoveTrait removes a trait to an element
	RemoveTrait(string)
}

// FormalInstance defines an element with time dependent attributes
type FormalInstance interface {
	Element

	// ContainsAttribute returns true if the element is not nil and contains a value for that attribute
	ContainsAttribute(string) bool
	// Attributes returns all the attributes of the element
	Attributes() []string
	// SetValue sets the value for that attribute during the full period
	SetValue(string, string) error
	// AddValue adds to the attribute and value the period
	AddValue(string, string, Period) error
	// SetPeriodForValue sets the value and the period for that attribute.
	// AddValue adds the period for that value to existing period, but setPeriodForValue just forces given period
	SetPeriodForValue(string, string, Period) error
	// RemovePeriodForAttribute just removes period, no matter the value, for that attribute
	RemovePeriodForAttribute(string, Period) error
	// ValuesForAttribute returns the value of the attribute during the period activity
	ValuesForAttribute(attribute string) ([]string, error)
	// PeriodValuesForAttribute returns the values and matching period for a given attribute
	PeriodValuesForAttribute(attribute string) (map[string]Period, error)
}

type FormalRelation interface {
	Element

	// ValuesPerRole returns the values per role, including the subject, for that relation
	ValuesPerRole() map[string][]string
	// SetValuesForRole sets the values for a given role
	SetValuesForRole(role string, linkedIds []string) error
}
