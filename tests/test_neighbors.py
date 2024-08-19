from api import *
from elements import *
from uuid import uuid4
from connection_data import *

def test_neighbors_entities_with_simple_links():
    assert check_api_connection(), "no connection to patterns"
    
    token = generate_token(test_username, test_userpass)
    assert token is not None, "user connection failed"

    # clean it all before test
    assert clear_all_graphs(token)

    graph_id = create_graph(token,"first graph")
    assert graph_id is not None, "failed to create graph"

    france = Element(id = str(uuid4()))
    france.traits = ["Country"]
    france.activity = ["]-oo;+oo["]
    france.add_attribute_value("name", "France")
    upsert_element_in_graph(token, graph_id, france)
 
    paris = Element(id = str(uuid4()))
    paris.traits = ["City"]
    paris.activity = ["]-oo;+oo["]
    paris.add_attribute_value("name", "Paris")
    upsert_element_in_graph(token, graph_id, paris)
 
    capitale = Element(id = str(uuid4()))
    capitale.traits = ["Capitale"]
    capitale.activity = ["]-oo;+oo["]
    capitale.add_role_value("subject", paris.element_id)
    capitale.add_role_value("object", france.element_id)
    upsert_element_in_graph(token, graph_id, capitale)

    # case 1: mismatch
    result_graph = find_neighbors_of_requested_entities(token, "City", {"name": "I don't 'exist"})
    assert result_graph is not None and len(result_graph) == 0
    result_graph = find_neighbors_of_requested_entities(token, "Not the correct trait...", {"name": "Paris"})
    assert result_graph is not None and len(result_graph) == 0

    # case 2: match
    result_graph = find_neighbors_of_requested_entities(token, "City", {"name": "Paris"})
    original_graph = load_graph_by_id(token, graph_id) 
    assert original_graph is not None
    assert result_graph is not None 
    # cheating a little bit to avoid a long list of comparisons 
    result_graph.graph_id = original_graph.graph_id 
    result_graph.name = original_graph.name 
    assert result_graph == original_graph

    assert clear_all_graphs(token)


def test_neighbors_more_links():
    assert check_api_connection(), "no connection to patterns"
    
    token = generate_token(test_username, test_userpass)
    assert token is not None, "user connection failed"

    # clean it all before test
    assert clear_all_graphs(token)

    graph_id = create_graph(token,"first graph")
    assert graph_id is not None, "failed to create graph"

    france = Element(id = str(uuid4()))
    france.traits = ["Country"]
    france.activity = ["]-oo;+oo["]
    france.add_attribute_value("name", "France")
    upsert_element_in_graph(token, graph_id, france)
 
    paris = Element(id = str(uuid4()))
    paris.traits = ["City"]
    paris.activity = ["]-oo;+oo["]
    paris.add_attribute_value("name", "Paris")
    upsert_element_in_graph(token, graph_id, paris)
 
    capitale = Element(id = str(uuid4()))
    capitale.traits = ["Capitale"]
    capitale.activity = ["]-oo;+oo["]
    capitale.add_role_value("subject", paris.element_id)
    capitale.add_role_value("object", france.element_id)
    upsert_element_in_graph(token, graph_id, capitale)

    # someone likes Paris. 
    # Should not appear when looking for general information about France
    someone = Element(id = str(uuid4()))
    someone.traits = ["Human"]
    someone.activity = ["]-oo;+oo["]
    someone.add_attribute_value("name", "Someone")
    upsert_element_in_graph(token, graph_id, someone)

    somelink = Element(id = str(uuid4()))
    somelink.traits = ["Likes"]
    somelink.activity = ["]-oo;+oo["]
    somelink.add_role_value("subject", someone.element_id)
    somelink.add_role_value("object", paris.element_id)
    upsert_element_in_graph(token, graph_id, somelink)

    # Asking for country should lead to avoid someone likes Paris
    result_graph = find_neighbors_of_requested_entities(token, "Country")
    assert result_graph is not None 

    expected_graph = Graph(result_graph.graph_id, result_graph.name)
    expected_graph.add_node(france, graph_id, True)
    expected_graph.add_node(paris, graph_id, True)
    expected_graph.add_node(capitale, graph_id, True)
    assert result_graph == expected_graph

    # City is the center of the graph, should lead to get the full graph
    result_graph = find_neighbors_of_requested_entities(token, "City")
    assert result_graph is not None 

    expected_graph = Graph(result_graph.graph_id, result_graph.name)
    expected_graph.add_node(france, graph_id, True)
    expected_graph.add_node(paris, graph_id, True)
    expected_graph.add_node(capitale, graph_id, True)
    expected_graph.add_node(someone, graph_id, True)
    expected_graph.add_node(somelink, graph_id, True)
    assert result_graph == expected_graph

    assert clear_all_graphs(token)