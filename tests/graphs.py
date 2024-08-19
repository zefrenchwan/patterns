from elements import *


class GraphNode:
    """
    GraphNode defines an entry of the graph to represent an element and its metadata. 
    Its content comes from the pattern web server.
    """
    def __init__(self, source:str, element: Element, editable: bool, equivalence_parent: str = "", equivalence_parent_graph:str = ""):
        self.source = source 
        self.element = element 
        self.editable = editable
        self.equivalence_parent = equivalence_parent 
        self.equivalence_parent_graph = equivalence_parent_graph


    def to_json(self):
        result = dict()
        result["element"] = self.element.to_json()
        result["editable"] = self.editable
        result["source"] = self.source
        if self.equivalence_parent != "":
            result["equivalence_parent"] = self.equivalence_parent
        if self.equivalence_parent_graph != "":
            result["equivalence_parent_graph"] = self.equivalence_parent_graph
        return result 

    def __eq__(self, value) -> bool:
        if value is None or not isinstance(value, GraphNode):
            return False 
        if self.source != value.source:
            return False 
        if self.editable != value.editable:
            return False 
        if self.equivalence_parent != value.equivalence_parent:
            return False 
        if self.equivalence_parent_graph != value.equivalence_parent_graph:
            return False 
        if self.element != value.element:
            return False 
        return True
        

class Graph:
    """
    Graph reprensents a graph coming from patterns web server
    """
    def __init__(self, id: str, name: str, description: str = "", metadata: dict[str,list[str]]= dict()):
        # graph metadata 
        self.graph_id = id 
        self.name = name 
        if description is None:
            self.description = ""
        else:
            self.description = description
        if metadata is None:
            self.metadata = dict()
        else:
            self.metadata = metadata
        # elements, key is element id
        self.elements: dict[str, GraphNode] = dict()

    def __len__(self):
        return len(self.elements)
    
    def add_node(self, element: Element, source: str, editable: bool, equivalence_parent: str = "", equivalence_parent_graph: str = ""):
        self.elements[element.element_id] = GraphNode(source, element, editable, equivalence_parent, equivalence_parent_graph)        

    def list_elements_per_source(self):
        """
        Returns elements per graph id
        """
        result = dict()
        for element_id, node in self.elements.items():
            graph_id = node.source
            previous_value = result.get(graph_id)
            if previous_value is None:
                result[graph_id] = list()
            result[graph_id].append(node.element)
        return result 
    
    def to_json(self):
        result = dict()
        result["id"] = self.graph_id
        result["name"] = self.name 
        result["description"] = self.description
        result["metadata"] = self.metadata
        if len(self.elements) == 0:
            return result 
        all_nodes = list()
        for element_id, node in self.elements.items():
            all_nodes.append(node.to_json())
        result["nodes"] = all_nodes
        return result 
    
    def __eq__(self, value) -> bool:
        if value is None or not isinstance(value, Graph):
            return False 
        if self.graph_id != value.graph_id:
            return False 
        if self.name != value.name:
            return False 
        if self.description != value.description:
            return False 
        if self.metadata != value.metadata:
            return False 
        if len(self.elements) != len(value.elements):
            return False 
        for key, node in self.elements.items():
            other_node = value.elements.get(key)
            if other_node is None:
                return False 
            if node != other_node:
                print(key)
                return False 
        return True
        

def graph_from_json(data: dict) -> Graph:
    """
    Given a json as a dict, builds the graph
    """
    result = Graph(id = data["id"], name = data["name"], description=data.get("description"), metadata=data.get("metadata"))
    nodes = data.get("nodes")
    if nodes is None: 
        return result 
    for node in nodes:
        source_graph = node["source"]
        source_element = node["element"]
        source_editable = node["editable"]
        source_equivalent_parent = node.get("equivalence_parent") or ""
        source_equivalent_parent_graph = node.get("equivalence_parent_graph") or ""
        element = element_from_json(source_element)
        result.elements[element.element_id] = GraphNode(source_graph, element, source_editable, source_equivalent_parent, source_equivalent_parent_graph)
    return result 

