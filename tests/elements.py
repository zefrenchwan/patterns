class AbstractValue:
    def __init__(self, value: str, periods: set[str]):
        self.value = value
        self.periods:list[str] = sorted(list((periods)))

    def __eq__(self, other) -> bool:
        if other is None or type(self) != type(other):
            return False 
        return self.value == other.value and self.periods == other.periods
    
    def clean_periods(self):
        self.periods = list()

    def __hash__(self) -> int:
        return hash((self.value, self.periods))
    
    def __repr__(self):
        return str(self)
    
    def __str__(self):
        return str(self.__dict__)
    
    def to_shorten_str(self):
        return self.value + "=>[" + ",".join(sorted(self.periods))+"]"


class AttributeValue(AbstractValue):

    def __init__(self, value: str, periods: set[str]):
        AbstractValue.__init__(self, value, periods)

    def copy_values(self) -> 'AttributeValue':
        return AttributeValue(str(self.value), list(self.periods))
    

class RoleValue(AbstractValue):

    def __init__(self, value: str, periods: set[str]):
        AbstractValue.__init__(self, value, periods)

    def copy_values(self) -> 'RoleValue':
        return RoleValue(str(self.value), list(self.periods))


class Element: 

    def __init__(self, id: str):
        self.element_id = id
        self.traits: list[str] = list()
        self.activity: list[str] = list()
        self.attributes: dict[str, list[AttributeValue]] = dict()
        self.roles: dict[str,list[RoleValue]] = dict()

    def add_attribute_value(self, name: str, value: str, validity: list[str] = ["]-oo;+oo["]):
        previous = self.attributes.get(name)
        if previous is None:
            self.attributes[name] = [AttributeValue(value, validity)]
        else:
            self.attributes[name].append(AttributeValue(value, validity))
        self.attributes[name] = sorted(self.attributes[name], key = lambda a: a.to_shorten_str())

    def add_role_value(self, name:str, value:str, validity: list[str] = ["]-oo;+oo["]):
        previous = self.roles.get(name)
        if previous is None:
            self.roles[name] = [RoleValue(value, validity)]
        else:
            self.roles[name].append(RoleValue(value, validity))
        self.roles[name] = sorted(self.roles[name], key = lambda a: a.to_shorten_str())

    def copy_values(self) -> 'Element':
        result = Element(self.element_id)
        result.traits = list(self.traits)
        result.activity = list(self.activity)
        for k, v in self.attributes.items(): 
            result.attributes[k] = [e.copy_values() for e in v]
        for k, v in self.roles.items():
            result.roles[k] = [e.copy_values() for e in v]
        return result 

    def clean_periods(self):
        self.activity = list()
        for k, v in self.attributes.items():
            for attr in v:
                attr.clean_periods()
        for k,v in self.roles.items():
            for role in v:
                role.clean_periods()

    def remove_attribute_value(self, name, value):
        if self.attributes.get(name) is not None:
            values = self.attributes[name]
            self.attributes[name] = [attr for attr in values if attr.value != value ]

    def remove_role_value(self, name, value):
        if self.roles.get(name) is not None:
            values = self.roles[name]
            self.roles[name] = [link for link in values if link.value != value ]        

    def __str__(self) -> str:
        return str(self.__dict__)
    
    def __repr__(self) -> str:
        return str(self)

    def __eq__(self, other) -> bool:
        if other is None:
            return False 
        if self.element_id != other.element_id:
            return False 
        if other.traits != self.traits:
            return False 
        if other.activity != self.activity:
            return False
        # case entity, validate attributes 
        if len(self.attributes) != len(other.attributes):
            return False
        for key, value in self.attributes.items():
            other_value = other.attributes.get(key)
            if other_value is None:
                return False
            value = sorted(value, key = lambda a:a.to_shorten_str())
            other_value = sorted(other_value, key = lambda a:a.to_shorten_str())
            if value != other_value:
                return False
        # case relation: validate roles
        if len(self.roles) != len(other.roles):
            return False 
        for key, value in self.roles.items():
            other_value = other.roles.get(key)
            if other_value is None:
                return False 
            value = sorted(value, key = lambda a:a.to_shorten_str())
            other_value = sorted(other_value, key = lambda a:a.to_shorten_str())
            if value != other_value:
                return False
        return True        

    def to_json(self) -> dict:
        result = {"id": self.element_id}
        if len(self.traits) != 0:
            result["traits"] = sorted(list(self.traits))
        else:
            result["traits"] = list()
        if len(self.activity) != 0:
            result["activity"] = sorted(list(self.activity))
        else:
            result["activity"] = list()
        
        # serialize attributes
        attributes = list()
        for attr, values in self.attributes.items():
            for value in values:
                attribute = {
                    "attribute": attr, 
                    "value": value.value, 
                    "validity": sorted(list(value.periods))
                }
                attributes.append(attribute)
        if len(attributes) != 0:
            result["attributes"] = attributes
        
        # serialize roles 
        roles_content = dict()
        for name, values in self.roles.items():
            role_values = list()
            for value in values:
                role_value = {"operand":value.value, "validity": value.periods}
                role_values.append(role_value)
            if len(role_values) != 0:
                roles_content[name] = role_values        
        if len(roles_content) != 0:
            result["roles"] = roles_content
        
        return result

def element_from_json(value: dict) -> Element: 
    id = value["id"]
    traits = value.get("traits")
    activity = value.get("activity") 
    activity = activity if activity is not None else list()
    result = Element(id=id)
    if traits is not None:
        result.traits = sorted(list(traits)) 
    else:
        result.traits = list()
    result.activity = activity 
    attributes = value.get("attributes")
    if attributes is not None:
        for attribute in attributes:
            attr_name = attribute["attribute"]
            attr_value = attribute["value"]
            attr_period = attribute.get("validity") 
            attr_period = attr_period if attr_period is not None else set()
            attribute_value = AttributeValue(attr_value, attr_period)
            if result.attributes.get(attr_name) is None:
                result.attributes[attr_name] = [attribute_value]
            else:
                result.attributes[attr_name].append(attribute_value)
    
    roles = value.get("roles")
    if roles is not None:
        for role, role_values in roles.items():
            new_role_values = list()
            for role_value in role_values:
                value = role_value["operand"]
                periods = role_value.get("validity")
                periods = periods if periods is not None else set()
                new_role_values.append(RoleValue(value, periods))
            if len(new_role_values) != 0:
                result.roles[role] = new_role_values
    return result